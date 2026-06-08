// Package middleware provides Fiber middleware used by CodeTasker's HTTP server.
// webhook_verify.go implements HMAC-SHA256 signature verification for GitHub
// webhook payloads, ensuring that only requests genuinely originating from
// GitHub (using the shared secret) are processed by the webhook handler.
//
// Security notes:
//   - The raw request body is read and buffered so that both the HMAC check
//     and the JSON unmarshalling in the handler see the exact same bytes.
//   - hmac.Equal is used for constant-time byte comparison, preventing
//     timing-oracle attacks that could allow an attacker to guess the HMAC.
//   - The verification rejects any request where the X-Hub-Signature-256
//     header is absent, malformed, or does not match.
package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// rawBodyLocalKey is the Locals key under which the raw request body bytes are
// stored by VerifyWebhookSignature so the downstream handler can unmarshal them
// without needing to read c.Body() again (which may already be consumed).
const rawBodyLocalKey = "rawBody"

// VerifyWebhookSignature returns a Fiber middleware that:
//  1. Reads the entire raw request body into memory.
//  2. Computes HMAC-SHA256 of the body using the provided secret.
//  3. Compares the computed digest with the value in X-Hub-Signature-256.
//  4. Stores the raw body in c.Locals("rawBody") for the handler.
//  5. Returns 403 Forbidden if the signature is missing or does not match.
//
// This middleware must be placed before the route handler in the middleware
// chain, and the secret must be the same value configured in the GitHub
// webhook settings.
func VerifyWebhookSignature(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Read the raw body; Fiber buffers it so this is safe to call before Next().
		body := c.Body()

		// Compute the expected HMAC-SHA256 digest.
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		expectedDigest := mac.Sum(nil)
		expectedSig := "sha256=" + hex.EncodeToString(expectedDigest)

		// Read the signature GitHub sends in the X-Hub-Signature-256 header.
		receivedSig := c.Get("X-Hub-Signature-256")

		if receivedSig == "" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "forbidden",
				"message": "missing X-Hub-Signature-256 header",
			})
		}

		// Normalise: GitHub sends "sha256=<hex>"; strip any extra whitespace.
		receivedSig = strings.TrimSpace(receivedSig)

		// Decode both signatures to raw bytes for constant-time comparison.
		// We compare the raw digest bytes, not the hex strings, to avoid any
		// potential short-circuit in string comparison.
		expectedBytes, err := hex.DecodeString(strings.TrimPrefix(expectedSig, "sha256="))
		if err != nil {
			// Should never happen since we just encoded it ourselves.
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("internal signature encoding error: %v", err),
			})
		}

		receivedHex := strings.TrimPrefix(receivedSig, "sha256=")
		receivedBytes, err := hex.DecodeString(receivedHex)
		if err != nil {
			// The header value is not valid hex — reject immediately.
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "forbidden",
				"message": "X-Hub-Signature-256 header is not valid hex",
			})
		}

		// Constant-time comparison prevents timing-oracle attacks.
		if !hmac.Equal(expectedBytes, receivedBytes) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "forbidden",
				"message": "webhook signature verification failed",
			})
		}

		// Store the raw body so the handler can unmarshal it without re-reading.
		// c.Body() returns the same slice for the lifetime of the request, but
		// storing it explicitly in Locals makes the dependency visible.
		c.Locals(rawBodyLocalKey, body)

		return c.Next()
	}
}
