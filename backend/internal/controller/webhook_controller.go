// Package controller implements the HTTP handler layer of CodeTasker.
// webhook_controller.go receives GitHub webhook push events, verifies the
// HMAC-SHA256 signature (applied by middleware before reaching this handler),
// and delegates processing to TaskService.
package controller

import (
	"encoding/json"

	"github.com/codetasker/backend/internal/service"
	"github.com/gofiber/fiber/v2"
)

// WebhookController handles inbound GitHub webhook events.
// Signature verification is performed by the VerifyWebhookSignature middleware
// applied at the router level — by the time a request reaches these handlers,
// the payload has already been authenticated.
type WebhookController struct {
	taskService *service.TaskService
}

// NewWebhookController constructs a WebhookController with its service dependency.
func NewWebhookController(taskService *service.TaskService) *WebhookController {
	return &WebhookController{taskService: taskService}
}

// RegisterRoutes mounts the webhook endpoint. The VerifyWebhookSignature
// middleware must be applied by the caller before this group is registered.
func (wc *WebhookController) RegisterRoutes(group fiber.Router) {
	group.Post("/github", wc.HandleGithubWebhook)
}

// HandleGithubWebhook processes inbound GitHub webhook events.
// The handler reads the raw body from c.Locals("rawBody") (populated by the
// signature verification middleware), inspects the X-GitHub-Event header to
// determine the event type, and dispatches accordingly:
//
//   - "push"  — unmarshal payload, call TaskService.ProcessWebhookPush.
//   - "ping"  — GitHub sends this when a webhook is first configured; respond 200 OK.
//   - other   — log and ignore gracefully; always return 200 OK so GitHub does
//     not mark the webhook as failed and start retrying.
//
// Route: POST /api/webhooks/github
func (wc *WebhookController) HandleGithubWebhook(c *fiber.Ctx) error {
	// The raw body was stored by VerifyWebhookSignature.
	rawBody, ok := c.Locals("rawBody").([]byte)
	if !ok || len(rawBody) == 0 {
		// This should not happen if the middleware was correctly applied.
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_body",
			"message": "raw body not found in context — check middleware configuration",
		})
	}

	eventType := c.Get("X-GitHub-Event")

	switch eventType {
	case "push":
		return wc.handlePushEvent(c, rawBody)

	case "ping":
		// GitHub sends a ping immediately after a webhook is created/edited.
		// Responding with 200 confirms the endpoint is reachable.
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "pong",
		})

	default:
		// Unknown or unsupported events are acknowledged but not processed.
		// Returning 200 prevents GitHub from retrying the delivery.
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "event type not handled",
			"event":   eventType,
		})
	}
}

// handlePushEvent is the push-specific handler branch called from HandleGithubWebhook.
// It unmarshals the payload and delegates to the task processing pipeline.
func (wc *WebhookController) handlePushEvent(c *fiber.Ctx, rawBody []byte) error {
	var payload service.WebhookPushPayload

	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_payload",
			"message": "failed to parse push event JSON: " + err.Error(),
		})
	}

	// Validate minimal required fields so we fail fast with a clear message
	// rather than propagating a confusing zero-value into the service layer.
	if payload.Repository.ID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_payload",
			"message": "push event missing repository.id",
		})
	}

	// ProcessWebhookPush is designed to be fault-tolerant internally; it logs
	// per-file errors but returns a top-level error only for critical failures.
	if err := wc.taskService.ProcessWebhookPush(c.Context(), payload); err != nil {
		// Return 500 so GitHub retries the delivery — we want the data eventually.
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "processing_failed",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":    "push event processed",
		"repository": payload.Repository.FullName,
		"commit":     payload.After,
	})
}
