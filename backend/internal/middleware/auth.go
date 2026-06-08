// Package middleware provides Fiber middleware used by CodeTasker's HTTP server.
// auth.go implements JWT authentication: it validates bearer tokens or httpOnly
// cookies, parses the claims, and exposes helper functions that controllers
// use to read the authenticated user identity from the request context.
package middleware

import (
	"errors"
	"strings"

	"github.com/codetasker/backend/internal/config"
	jwtware "github.com/gofiber/jwt/v3"
	"github.com/golang-jwt/jwt/v4"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// localKeyUserID is the Locals key under which the authenticated user's
// ObjectID is stored after a successful JWT validation.
const localKeyUserID = "userID"

// localKeyUsername is the Locals key for the authenticated user's GitHub login.
const localKeyUsername = "username"

// Protected returns a Fiber middleware that validates the JWT carried either in
// the "Authorization: Bearer <token>" header or in the "auth_token" httpOnly
// cookie. Downstream handlers can call GetUserID and GetUsername to read the
// validated identity without touching JWT claims directly.
//
// On a valid token the middleware stores the parsed userID and username in
// fiber.Ctx.Locals and calls c.Next(). On an invalid or missing token it
// returns 401 Unauthorized with a JSON error body.
func Protected(cfg *config.Config) fiber.Handler {
	// jwtware.New creates a Fiber-compatible middleware that:
	//   1. Extracts the token from the Authorization header (default behaviour).
	//   2. Falls back to a cookie extractor configured below.
	//   3. Validates the signature using HS256 + the configured JWTSecret.
	//   4. Stores the parsed *jwt.Token in c.Locals("user") on success.
	jwtMiddleware := jwtware.New(jwtware.Config{
		SigningKey:    []byte(cfg.JWTSecret),
		SigningMethod: "HS256",
		// TokenLookup specifies where to look for the token.
		// "header:Authorization" is tried first, then "cookie:auth_token".
		TokenLookup: "header:Authorization,cookie:auth_token",
		// AuthScheme is the prefix stripped from the Authorization header value.
		AuthScheme: "Bearer",
		// ErrorHandler returns a consistent 401 JSON response on any auth failure.
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "missing or invalid authentication token",
			})
		},
		// SuccessHandler is called after the token is validated. It parses our
		// custom claims and stores the typed values in Locals so controllers
		// never have to cast jwt.MapClaims themselves.
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

			// Extract the "sub" claim which holds the ObjectID hex string.
			sub, _ := claims["sub"].(string)
			objID, err := primitive.ObjectIDFromHex(strings.TrimSpace(sub))
			if err != nil {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error":   "invalid token subject",
					"message": "token sub claim is not a valid ObjectID",
				})
			}

			username, _ := claims["username"].(string)

			// Store typed values so controllers do not need to deal with jwt internals.
			c.Locals(localKeyUserID, objID)
			c.Locals(localKeyUsername, username)

			return c.Next()
		},
	})

	return jwtMiddleware
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
