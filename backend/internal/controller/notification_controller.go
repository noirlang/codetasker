// Package controller implements the HTTP handler layer of CodeTasker.
// notification_controller.go exposes endpoints for listing, counting, and
// marking in-app notifications. All routes require a valid JWT.
package controller

import (
	"github.com/codetasker/backend/internal/middleware"
	"github.com/codetasker/backend/internal/repository"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// NotificationController handles notification-related HTTP endpoints.
type NotificationController struct {
	notifRepo *repository.NotificationRepository
}

// NewNotificationController constructs a NotificationController with its dependencies injected.
func NewNotificationController(notifRepo *repository.NotificationRepository) *NotificationController {
	return &NotificationController{notifRepo: notifRepo}
}

// RegisterRoutes mounts all notification routes onto the provided Fiber router group.
// The group is expected to already have the Protected JWT middleware applied.
func (nc *NotificationController) RegisterRoutes(group fiber.Router) {
	group.Get("/notifications", nc.ListNotifications)
	group.Get("/notifications/unread-count", nc.UnreadCount)
	// read-all must be registered before :id/read to avoid route conflicts.
	group.Patch("/notifications/read-all", nc.MarkAllRead)
	group.Patch("/notifications/:id/read", nc.MarkRead)
}

// ListNotifications returns the authenticated user's most recent notifications.
//
// Route: GET /api/notifications
func (nc *NotificationController) ListNotifications(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}
	notifs, err := nc.notifRepo.FindByUser(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"notifications": notifs, "count": len(notifs)})
}

// UnreadCount returns the number of unread notifications for the authenticated user.
//
// Route: GET /api/notifications/unread-count
func (nc *NotificationController) UnreadCount(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}
	count, err := nc.notifRepo.UnreadCount(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"count": count})
}

// MarkRead marks a single notification as read. Only the notification's owner
// may mark it read — ownership is verified before the update.
//
// Route: PATCH /api/notifications/:id/read
func (nc *NotificationController) MarkRead(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	// Verify ownership before marking read.
	n, err := nc.notifRepo.FindByID(c.Context(), id)
	if err != nil || n == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	if n.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	}
	if err := nc.notifRepo.MarkRead(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "notification marked as read"})
}

// MarkAllRead marks all of the authenticated user's unread notifications as read.
//
// Route: PATCH /api/notifications/read-all
func (nc *NotificationController) MarkAllRead(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}
	if err := nc.notifRepo.MarkAllRead(c.Context(), userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "all notifications marked as read"})
}
