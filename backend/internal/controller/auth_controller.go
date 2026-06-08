// Package controller implements the HTTP handler layer of CodeTasker.
// auth_controller.go handles the GitHub OAuth 2.0 flow: initiating the
// redirect, handling the callback, issuing JWT cookies, and logging out.
package controller

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/codetasker/backend/internal/middleware"
	"github.com/codetasker/backend/internal/service"
	"github.com/gofiber/fiber/v2"
)

// AuthController handles all authentication-related HTTP endpoints.
// It delegates OAuth token exchange and JWT generation to AuthService.
type AuthController struct {
	authService *service.AuthService
}

// NewAuthController constructs an AuthController with its service dependency.
func NewAuthController(authService *service.AuthService) *AuthController {
	return &AuthController{authService: authService}
}

// RegisterRoutes mounts all auth routes onto the provided Fiber router.
// No JWT middleware is applied here — these routes must be publicly accessible.
func (ac *AuthController) RegisterRoutes(app *fiber.App) {
	auth := app.Group("/api/auth")
	auth.Get("/github", ac.InitiateOAuth)
	auth.Get("/github/callback", ac.HandleCallback)
	auth.Post("/logout", ac.Logout)
}

// RegisterProtectedRoutes mounts auth routes that require JWT verification.
func (ac *AuthController) RegisterProtectedRoutes(group fiber.Router) {
	group.Get("/auth/me", ac.GetMe)
}

// InitiateOAuth generates a cryptographically random state token, stores it
// in an httpOnly cookie (to prevent CSRF), and redirects the browser to
// GitHub's OAuth authorization page.
//
// The state parameter is verified in HandleCallback to ensure the callback is
// the authentic response to this specific authorization request and not a
// forged redirect injected by an attacker.
func (ac *AuthController) InitiateOAuth(c *fiber.Ctx) error {
	// Generate 16 random bytes → 32 hex character state token.
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "server_error",
			"message": "failed to generate OAuth state token",
		})
	}
	state := hex.EncodeToString(stateBytes)

	// Store the state in an httpOnly cookie so the callback can verify it.
	// SameSite=Lax allows the cookie to be sent on the GitHub → our server
	// redirect that completes the OAuth flow.
	c.Cookie(&fiber.Cookie{
		Name:     "oauth_state",
		Value:    state,
		HTTPOnly: true,
		SameSite: "Lax",
		Path:     "/",
		MaxAge:   int(10 * time.Minute / time.Second),
	})

	// Build the GitHub authorization URL with the state parameter.
	oauthCfg := ac.authService.GetOAuthConfig()
	redirectURL := oauthCfg.AuthCodeURL(state)

	return c.Redirect(redirectURL, fiber.StatusTemporaryRedirect)
}

// HandleCallback processes GitHub's OAuth callback:
//  1. Validates the state parameter against the cookie to prevent CSRF.
//  2. Exchanges the one-time code for a GitHub access token.
//  3. Fetches the authenticated user's GitHub profile.
//  4. Upserts the user in the database.
//  5. Issues a signed JWT as an httpOnly cookie.
//  6. Redirects the browser to the frontend dashboard.
func (ac *AuthController) HandleCallback(c *fiber.Ctx) error {
	// ── CSRF state validation ────────────────────────────────────────────────
	cookieState := c.Cookies("oauth_state")
	queryState := c.Query("state")

	if cookieState == "" || queryState == "" || cookieState != queryState {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_state",
			"message": "OAuth state mismatch — possible CSRF attack",
		})
	}

	// Clear the state cookie — it is single-use.
	c.Cookie(&fiber.Cookie{
		Name:    "oauth_state",
		Value:   "",
		MaxAge:  -1,
		Path:    "/",
		Expires: time.Now().Add(-time.Hour),
	})

	// ── Code exchange ────────────────────────────────────────────────────────
	code := c.Query("code")
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_code",
			"message": "no authorization code received from GitHub",
		})
	}

	token, err := ac.authService.ExchangeCodeForToken(c.Context(), code)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "token_exchange_failed",
			"message": err.Error(),
		})
	}

	// ── Fetch GitHub user profile ────────────────────────────────────────────
	ghUser, err := ac.authService.FetchGithubUser(c.Context(), token)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "fetch_user_failed",
			"message": err.Error(),
		})
	}

	// ── Upsert user in database ──────────────────────────────────────────────
	user, err := ac.authService.UpsertUser(c.Context(), ghUser, token)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "upsert_user_failed",
			"message": err.Error(),
		})
	}

	// ── Generate JWT ─────────────────────────────────────────────────────────
	jwtToken, err := ac.authService.GenerateJWT(user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "jwt_generation_failed",
			"message": err.Error(),
		})
	}

	// ── Set auth cookie ──────────────────────────────────────────────────────
	// httpOnly prevents JavaScript from reading the token.
	// Secure ensures the cookie is only sent over HTTPS in production.
	// SameSite=Strict prevents the cookie from being sent in cross-site requests.
	c.Cookie(&fiber.Cookie{
		Name:     "auth_token",
		Value:    jwtToken,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
		Path:     "/",
		Expires:  time.Now().Add(7 * 24 * time.Hour),
	})

	// ── Redirect to frontend dashboard ───────────────────────────────────────
	frontendDashboard := ac.authService.FrontendURL() + "/dashboard"
	return c.Redirect(frontendDashboard, fiber.StatusTemporaryRedirect)
}

// Logout clears the auth_token cookie, effectively ending the user's session.
// The JWT itself remains valid until it expires (server-side revocation is not
// implemented in this version), but clearing the cookie prevents the browser
// from sending it on subsequent requests.
func (ac *AuthController) Logout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:    "auth_token",
		Value:   "",
		HTTPOnly: true,
		Secure:  true,
		SameSite: "Strict",
		Path:    "/",
		MaxAge:  -1,
		Expires: time.Now().Add(-time.Hour),
	})

	return c.JSON(fiber.Map{
		"message": "logged out successfully",
	})
}

// GetMe returns the currently authenticated user's profile.
// It retrieves the user ID from Fiber context locals, looks up the user in
// the database, and returns the JSON representation.
func (ac *AuthController) GetMe(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": err.Error(),
		})
	}

	user, err := ac.authService.GetUserByID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "user_not_found",
			"message": err.Error(),
		})
	}

	return c.JSON(user)
}
