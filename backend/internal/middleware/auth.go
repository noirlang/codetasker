// Package middleware provides Fiber middleware used by CodeTasker's HTTP server.
// auth.go implements JWT authentication: it validates bearer tokens or httpOnly
// cookies, parses the claims, and exposes helper functions that controllers
// use to read the authenticated user identity from the request context.
package middleware

import (
	"context"
	"errors"
	"strings"

	"github.com/codetasker/backend/internal/config"
	"github.com/codetasker/backend/internal/repository"
	jwtware "github.com/gofiber/jwt/v3"
	"github.com/golang-jwt/jwt/v4"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// localKeyUserID is the Locals key under which the authenticated user's
// ObjectID is stored after a successful JWT or App Token validation.
const localKeyUserID = "userID"

// localKeyUsername is the Locals key for the authenticated user's GitHub login.
const localKeyUsername = "username"

// Protected returns a Fiber middleware that validates either standard JWT tokens
// (via Authorization header / cookie) OR scoped App Tokens (via X-App-Token or Bearer ct_app_...).
//
// App Tokens are strictly restricted to reading notifications (GET /api/notifications).
// Any attempt to access other endpoints using an App Token returns 403 Forbidden.
func Protected(cfg *config.Config, appTokenRepo *repository.AppTokenRepository, userRepo *repository.UserRepository) fiber.Handler {
	jwtMiddleware := jwtware.New(jwtware.Config{
		SigningKey:    []byte(cfg.JWTSecret),
		SigningMethod: "HS256",
		TokenLookup:   "header:Authorization,cookie:auth_token",
		AuthScheme:    "Bearer",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "missing or invalid authentication token",
			})
		},
		SuccessHandler: func(c *fiber.Ctx) error {
			token, ok := c.Locals("user").(*jwt.Token)
			if !ok {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "invalid token type",
				})
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "invalid token claims",
				})
			}

			sub, _ := claims["sub"].(string)
			objID, err := primitive.ObjectIDFromHex(strings.TrimSpace(sub))
			if err != nil {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error":   "invalid token subject",
					"message": "token sub claim is not a valid ObjectID",
				})
			}

			username, _ := claims["username"].(string)

			c.Locals(localKeyUserID, objID)
			c.Locals(localKeyUsername, username)

			return c.Next()
		},
	})

	return func(c *fiber.Ctx) error {
		// 1. Check for App Token in headers or query
		rawToken := c.Get("X-App-Token")
		if rawToken == "" {
			authHeader := c.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ct_app_") {
				rawToken = strings.TrimPrefix(authHeader, "Bearer ")
			} else if strings.HasPrefix(authHeader, "ct_app_") {
				rawToken = authHeader
			}
		}

		// 2. If an App Token is present (starts with "ct_app_")
		if strings.HasPrefix(rawToken, "ct_app_") {
			if appTokenRepo == nil {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error":   "unauthorized",
					"message": "app tokens not supported",
				})
			}

			tokenHash := repository.HashToken(rawToken)
			tokenDoc, err := appTokenRepo.FindByTokenHash(c.Context(), tokenHash)
			if err != nil || tokenDoc == nil {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error":   "invalid_app_token",
					"message": "the provided app token is invalid or revoked",
				})
			}

			// If scope is explicitly set to notifications:read and request is not notifications, check scope
			if tokenDoc.Scope == domain.ScopeNotificationsRead {
				reqPath := c.Path()
				isNotificationPath := strings.HasPrefix(reqPath, "/api/notifications")
				isAllowedMethod := c.Method() == "GET" || c.Method() == "PATCH"
				if !isNotificationPath || !isAllowedMethod {
					// Also allow user profile / me check
					if !strings.HasPrefix(reqPath, "/api/auth/me") {
						return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
							"error":   "invalid_scope",
							"message": "this app token is restricted to notifications:read scope only (/api/notifications)",
						})
					}
				}
			}

			// Update last used timestamp
			go func() {
				_ = appTokenRepo.UpdateLastUsed(context.Background(), tokenDoc.ID)
			}()

			username := ""
			if userRepo != nil {
				u, _ := userRepo.FindByObjectID(c.Context(), tokenDoc.UserID)
				if u != nil {
					username = u.Username
				}
			}

			c.Locals(localKeyUserID, tokenDoc.UserID)
			c.Locals(localKeyUsername, username)
			c.Locals("isAppToken", true)

			return c.Next()
		}

		// 3. Otherwise, use standard JWT token validation
		return jwtMiddleware(c)
	}
}

// GetUserID retrieves the authenticated user's MongoDB ObjectID from Fiber's
// request-scoped Locals map. It returns an error if the middleware was not
// applied or the value is missing, making auth bugs explicit rather than
// causing downstream nil-pointer panics.
func GetUserID(c *fiber.Ctx) (primitive.ObjectID, error) {
	val := c.Locals(localKeyUserID)
	if val == nil {
		return primitive.NilObjectID, errors.New("GetUserID: userID not found in context — is the Protected middleware applied?")
	}

	objID, ok := val.(primitive.ObjectID)
	if !ok {
		return primitive.NilObjectID, errors.New("GetUserID: userID in context has unexpected type")
	}

	return objID, nil
}

// GetUsername retrieves the authenticated user's GitHub login handle from
// Fiber's request-scoped Locals map. Returns an empty string with an error
// if the middleware was not applied or the value is missing.
func GetUsername(c *fiber.Ctx) (string, error) {
	val := c.Locals(localKeyUsername)
	if val == nil {
		return "", errors.New("GetUsername: username not found in context — is the Protected middleware applied?")
	}

	username, ok := val.(string)
	if !ok {
		return "", errors.New("GetUsername: username in context has unexpected type")
	}

	return username, nil
}
