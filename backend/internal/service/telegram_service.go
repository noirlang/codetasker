package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// TelegramService handles sending notification messages via the Telegram Bot API.
type TelegramService struct {
	log    *zap.Logger
	client *http.Client
}

// NewTelegramService constructs a TelegramService.
func NewTelegramService(log *zap.Logger) *TelegramService {
	return &TelegramService{
		log: log,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// telegramMessagePayload represents the request body for Telegram Bot API sendMessage.
type telegramMessagePayload struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"` // "HTML" or "MarkdownV2"
}

// send delivers a message payload to Telegram Bot API.
func (s *TelegramService) send(ctx context.Context, token, chatID, text string) error {
	if token == "" || chatID == "" {
		return fmt.Errorf("bot token and chat id are required")
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	payload := telegramMessagePayload{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "HTML",
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		s.log.Error("Telegram API returned error status",
			zap.Int("status_code", resp.StatusCode),
			zap.Any("response", errResp),
		)
		return fmt.Errorf("telegram API error status: %d", resp.StatusCode)
	}

	return nil
}

// SendTaskAssigned sends a Telegram message notifying a user that they've been assigned to a task.
func (s *TelegramService) SendTaskAssigned(ctx context.Context, token, chatID, toName, assignerName, taskContent, repoName string) error {
	msg := fmt.Sprintf(
		"🔔 <b>New Task Assigned</b>\n\n"+
			"Hi <b>%s</b>, you have been assigned to a task in <b>%s</b> by <b>%s</b>:\n\n"+
			"📝 <i>\"%s\"</i>",
		toName, repoName, assignerName, taskContent,
	)
	return s.send(ctx, token, chatID, msg)
}

// SendCommentNotification sends a Telegram message notifying a user of a new comment on their task.
func (s *TelegramService) SendCommentNotification(ctx context.Context, token, chatID, toName, commenterName, taskContent, commentContent, repoName string) error {
	msg := fmt.Sprintf(
		"💬 <b>New Comment on Task</b>\n\n"+
			"Hi <b>%s</b>, <b>%s</b> commented on a task in <b>%s</b>:\n\n"+
			"Task: <i>\"%s\"</i>\n"+
			"Comment: <b>\"%s\"</b>",
		toName, commenterName, repoName, taskContent, commentContent,
	)
	return s.send(ctx, token, chatID, msg)
}

// SendTaskCompleted sends a Telegram message notifying a maintainer that a task has been completed.
func (s *TelegramService) SendTaskCompleted(ctx context.Context, token, chatID, maintainerName, developerName, taskContent, repoName, filePath string) error {
	msg := fmt.Sprintf(
		"✅ <b>Task Completed</b>\n\n"+
			"Hi <b>%s</b>, <b>%s</b> has completed a task in <b>%s</b>:\n\n"+
			"Task: <i>\"%s\"</i>\n"+
			"File: <code>%s</code>",
		maintainerName, developerName, repoName, taskContent, filePath,
	)
	return s.send(ctx, token, chatID, msg)
}
