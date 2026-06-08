// Package service implements the business logic layer of CodeTasker.
// auth_service.go handles the complete GitHub OAuth 2.0 flow: building the
// redirect URL, exchanging the code for a token, fetching the authenticated
// user's profile, persisting the user, and issuing a signed JWT for subsequent
// API calls.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/codetasker/backend/internal/config"
	"github.com/codetasker/backend/internal/domain"
	"github.com/codetasker/backend/internal/repository"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/go-github/v62/github"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	githuboauth "golang.org/x/oauth2/github"
)

// AuthService orchestrates the GitHub OAuth flow and JWT issuance.
// It depends on:
//   - *config.Config for OAuth credentials, JWT secret, and encryption key.
//   - *repository.UserRepository for persisting authenticated users.
//   - *zap.Logger for structured, production-grade logging.
type AuthService struct {
	cfg      *config.Config
	userRepo *repository.UserRepository
	log      *zap.Logger
}

// NewAuthService constructs an AuthService with its dependencies injected.
func NewAuthService(cfg *config.Config, userRepo *repository.UserRepository, log *zap.Logger) *AuthService {
	return &AuthService{
		cfg:      cfg,
		userRepo: userRepo,
		log:      log,
	}
}

// FrontendURL returns the configured frontend origin URL, used by the auth
// controller to build post-OAuth redirect targets without coupling the
// controller directly to the config package.
func (s *AuthService) FrontendURL() string {
	return s.cfg.FrontendURL
}

// GetOAuthConfig builds and returns the oauth2.Config struct configured for
// GitHub's authorization endpoint. The scopes requested are:
//   - "repo"        — read/write access to repositories (needed for file API).
//   - "read:user"   — read the authenticated user's profile.
//   - "user:email"  — read the user's primary email address.
func (s *AuthService) GetOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     s.cfg.GithubClientID,
		ClientSecret: s.cfg.GithubClientSecret,
		RedirectURL:  s.cfg.GithubRedirectURL,
		Scopes:       []string{"repo", "read:user", "user:email"},
		Endpoint:     githuboauth.Endpoint,
	}
}

// ExchangeCodeForToken trades the one-time authorization code received from
// GitHub's callback for a long-lived access token. The context allows the
// caller to cancel or time-out the HTTP exchange.
func (s *AuthService) ExchangeCodeForToken(ctx context.Context, code string) (*oauth2.Token, error) {
	oauthCfg := s.GetOAuthConfig()

	token, err := oauthCfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("ExchangeCodeForToken: %w", err)
	}

	return token, nil
}

// FetchGithubUser uses the provided OAuth token to call GitHub's
// /user endpoint and return the authenticated user's public profile.
// A dedicated HTTP client carrying the bearer token is created via the
// oauth2 transport, ensuring the token is always sent securely.
func (s *AuthService) FetchGithubUser(ctx context.Context, token *oauth2.Token) (*github.User, error) {
	oauthCfg := s.GetOAuthConfig()

	// Build a token-carrying HTTP client scoped to this context.
	httpClient := oauthCfg.Client(ctx, token)
	ghClient := github.NewClient(httpClient)

	// An empty string in GetAuthenticatedUser() means "the token owner".
	user, _, err := ghClient.Users.Get(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("FetchGithubUser: %w", err)
	}

	return user, nil
}

// UpsertUser maps a GitHub API user to a CodeTasker domain.User, encrypts the
// OAuth access token, and writes the record to MongoDB via the user repository.
// The encrypted token allows the service to make future GitHub API calls on
// behalf of the user without storing the raw token.
func (s *AuthService) UpsertUser(ctx context.Context, ghUser *github.User, token *oauth2.Token) (*domain.User, error) {
	// Encrypt the access token before it touches the database.
	encryptedToken, err := EncryptToken(token.AccessToken, s.cfg.TokenEncryptKey)
	if err != nil {
		return nil, fmt.Errorf("UpsertUser encrypt token: %w", err)
	}

	user := &domain.User{
		GithubID:    ghUser.GetID(),
		Username:    ghUser.GetLogin(),
		AvatarURL:   ghUser.GetAvatarURL(),
		AccessToken: encryptedToken,
	}

	if err := s.userRepo.Upsert(ctx, user); err != nil {
		return nil, fmt.Errorf("UpsertUser upsert: %w", err)
	}

	// After upsert the repository sets UpdatedAt; we need the ID from MongoDB.
	// Re-fetch the user so the caller gets the full document including _id.
	persisted, err := s.userRepo.FindByGithubID(ctx, user.GithubID)
	if err != nil {
		return nil, fmt.Errorf("UpsertUser re-fetch: %w", err)
	}

	s.log.Info("user upserted",
		zap.String("username", persisted.Username),
		zap.Int64("github_id", persisted.GithubID),
	)

	return persisted, nil
}

// jwtClaims defines the set of JWT claims embedded in tokens issued by CodeTasker.
// It extends RegisteredClaims with application-specific fields so the middleware
// can reconstruct the user identity without a database round-trip on every request.
type jwtClaims struct {
	jwt.RegisteredClaims

	// GithubID is included for informational purposes in the token payload.
	GithubID int64 `json:"github_id"`

	// Username is the GitHub login handle, used by the frontend to personalise UI.
	Username string `json:"username"`
}

// GenerateJWT creates a signed HS256 JWT for the given user.
// Claims embedded in the token:
//   - sub  — the user's MongoDB ObjectID in hex string form.
//   - exp  — seven days from now.
//   - iat  — current UTC time.
//   - github_id — GitHub numeric user ID.
//   - username  — GitHub login handle.
//
// The token is signed with the JWTSecret from config and is verified by the
// JWT middleware on every protected route.
func (s *AuthService) GenerateJWT(user *domain.User) (string, error) {
	now := time.Now().UTC()
	claims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.Hex(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour)),
		},
		GithubID: user.GithubID,
		Username: user.Username,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("GenerateJWT sign: %w", err)
	}

	return signed, nil
}

// GetUserByID looks up a user in the database by their MongoDB ObjectID.
// This is used by the /api/auth/me endpoint to fetch user details.
func (s *AuthService) GetUserByID(ctx context.Context, id primitive.ObjectID) (*domain.User, error) {
	user, err := s.userRepo.FindByObjectID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("GetUserByID: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("GetUserByID: user not found")
	}
	return user, nil
}

