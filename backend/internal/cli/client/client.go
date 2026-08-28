package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/codetasker/backend/internal/cli/config"
	"github.com/codetasker/backend/internal/debt"
	"github.com/codetasker/backend/internal/domain"
)

// RepositoryInfo represents repository data returned by the API.
type RepositoryInfo struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	FullName      string    `json:"full_name"`
	DefaultBranch string    `json:"default_branch"`
	Private       bool      `json:"private"`
	Description   string    `json:"description"`
	Language      string    `json:"language"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// FileTreeNode represents a file/directory node in the repo tree.
type FileTreeNode struct {
	Path string `json:"path"`
	Type string `json:"type"` // "blob" or "tree"
	SHA  string `json:"sha"`
	Size int    `json:"size,omitempty"`
}

// SyncResponse holds the result of a sync operation.
type SyncResponse struct {
	RepoID       int64  `json:"repo_id"`
	RepoName     string `json:"repo_name"`
	TotalTasks   int    `json:"total_tasks"`
	NewTasks     int    `json:"new_tasks"`
	UpdatedTasks int    `json:"updated_tasks"`
	RemovedTasks int    `json:"removed_tasks"`
	ScannedFiles int    `json:"scanned_files"`
}

// InjectResponse holds the result of task injection and PR creation.
type InjectResponse struct {
	PRURL     string `json:"pr_url"`
	CommitSHA string `json:"commit_sha"`
	Branch    string `json:"branch"`
	TaskID    string `json:"task_id"`
}

// Client handles all CodeTasker API requests.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// NewClient creates a new API client using the provided config.
func NewClient(cfg *config.Config) *Client {
	baseURL := strings.TrimRight(cfg.ServerURL, "/")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	return &Client{
		BaseURL: baseURL,
		Token:   cfg.Token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) doRequest(ctx context.Context, method, endpoint string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	reqURL := fmt.Sprintf("%s%s", c.BaseURL, endpoint)
	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		if strings.HasPrefix(c.Token, "ct_app_") {
			req.Header.Set("X-App-Token", c.Token)
		} else {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
		}
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("network request failed (is backend running at %s?): %w", c.BaseURL, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(respBytes, &apiErr); err == nil {
			msg := apiErr.Error
			if msg == "" {
				msg = apiErr.Message
			}
			if msg != "" {
				return fmt.Errorf("API error (%d): %s", resp.StatusCode, msg)
			}
		}
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBytes))
	}

	if result != nil && len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, result); err != nil {
			return fmt.Errorf("unmarshal response (%s): %w", string(respBytes), err)
		}
	}

	return nil
}

// ── Auth Endpoints ───────────────────────────────────────────────────────────

func (c *Client) GetMe(ctx context.Context) (*domain.User, error) {
	var user domain.User
	if err := c.doRequest(ctx, http.MethodGet, "/auth/me", nil, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// ── Repo Endpoints ───────────────────────────────────────────────────────────

func (c *Client) ListRepos(ctx context.Context) ([]RepositoryInfo, error) {
	var repos []RepositoryInfo
	if err := c.doRequest(ctx, http.MethodGet, "/repos", nil, &repos); err != nil {
		return nil, err
	}
	return repos, nil
}

func (c *Client) GetRepoTree(ctx context.Context, owner, repo, branch string) ([]FileTreeNode, error) {
	endpoint := fmt.Sprintf("/repos/%s/%s/tree", url.PathEscape(owner), url.PathEscape(repo))
	if branch != "" {
		endpoint += "?branch=" + url.QueryEscape(branch)
	}
	var tree []FileTreeNode
	if err := c.doRequest(ctx, http.MethodGet, endpoint, nil, &tree); err != nil {
		return nil, err
	}
	return tree, nil
}

func (c *Client) GetRepoCollaborators(ctx context.Context, owner, repo string) ([]domain.Collaborator, error) {
	endpoint := fmt.Sprintf("/repos/%s/%s/collaborators", url.PathEscape(owner), url.PathEscape(repo))
	var collabs []domain.Collaborator
	if err := c.doRequest(ctx, http.MethodGet, endpoint, nil, &collabs); err != nil {
		return nil, err
	}
	return collabs, nil
}

func (c *Client) SyncRepo(ctx context.Context, owner, repo string) (*SyncResponse, error) {
	endpoint := fmt.Sprintf("/repos/%s/%s/sync", url.PathEscape(owner), url.PathEscape(repo))
	var syncResp SyncResponse
	if err := c.doRequest(ctx, http.MethodPost, endpoint, nil, &syncResp); err != nil {
		return nil, err
	}
	return &syncResp, nil
}

// ── Task Endpoints ───────────────────────────────────────────────────────────

func (c *Client) ListTasks(ctx context.Context, repoID int64, status, taskType string) ([]domain.Task, error) {
	endpoint := fmt.Sprintf("/repos/%d/tasks", repoID)
	params := url.Values{}
	if status != "" {
		params.Set("status", status)
	}
	if taskType != "" {
		params.Set("type", taskType)
	}
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	var tasks []domain.Task
	if err := c.doRequest(ctx, http.MethodGet, endpoint, nil, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (c *Client) InjectTask(ctx context.Context, req domain.InjectTaskRequest) (*InjectResponse, error) {
	var resp InjectResponse
	if err := c.doRequest(ctx, http.MethodPost, "/tasks/inject", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) UpdateTaskStatus(ctx context.Context, taskID string, status domain.TaskStatus) error {
	endpoint := fmt.Sprintf("/tasks/%s/status", url.PathEscape(taskID))
	body := map[string]string{"status": string(status)}
	return c.doRequest(ctx, http.MethodPatch, endpoint, body, nil)
}

func (c *Client) UpdateTaskAssignee(ctx context.Context, taskID, username string) error {
	endpoint := fmt.Sprintf("/tasks/%s/assignee", url.PathEscape(taskID))
	body := map[string]string{"username": username}
	return c.doRequest(ctx, http.MethodPatch, endpoint, body, nil)
}

func (c *Client) ListComments(ctx context.Context, taskID string) ([]domain.Comment, error) {
	endpoint := fmt.Sprintf("/tasks/%s/comments", url.PathEscape(taskID))
	var comments []domain.Comment
	if err := c.doRequest(ctx, http.MethodGet, endpoint, nil, &comments); err != nil {
		return nil, err
	}
	return comments, nil
}

func (c *Client) AddComment(ctx context.Context, taskID, content string) (*domain.Comment, error) {
	endpoint := fmt.Sprintf("/tasks/%s/comments", url.PathEscape(taskID))
	body := map[string]string{"content": content}
	var comment domain.Comment
	if err := c.doRequest(ctx, http.MethodPost, endpoint, body, &comment); err != nil {
		return nil, err
	}
	return &comment, nil
}

func (c *Client) ListProposals(ctx context.Context, taskID string) ([]domain.TaskProposal, error) {
	endpoint := fmt.Sprintf("/tasks/%s/proposals", url.PathEscape(taskID))
	var proposals []domain.TaskProposal
	if err := c.doRequest(ctx, http.MethodGet, endpoint, nil, &proposals); err != nil {
		return nil, err
	}
	return proposals, nil
}

func (c *Client) AddProposal(ctx context.Context, taskID, title, content string) (*domain.TaskProposal, error) {
	endpoint := fmt.Sprintf("/tasks/%s/proposals", url.PathEscape(taskID))
	body := map[string]string{"title": title, "content": content}
	var proposal domain.TaskProposal
	if err := c.doRequest(ctx, http.MethodPost, endpoint, body, &proposal); err != nil {
		return nil, err
	}
	return &proposal, nil
}

func (c *Client) VoteProposal(ctx context.Context, taskID, proposalID string, status domain.ProposalStatus) error {
	endpoint := fmt.Sprintf("/tasks/%s/proposals/%s/status", url.PathEscape(taskID), url.PathEscape(proposalID))
	body := map[string]string{"status": string(status)}
	return c.doRequest(ctx, http.MethodPatch, endpoint, body, nil)
}

// ── Technical Debt Endpoints ─────────────────────────────────────────────────

func (c *Client) AnalyzeDebtRemote(ctx context.Context, owner, repo string, days int, hourlyCost float64) (*debt.AnalysisResult, error) {
	endpoint := fmt.Sprintf("/repos/%s/%s/debt/analyze?days=%d&hourly_cost=%f", url.PathEscape(owner), url.PathEscape(repo), days, hourlyCost)
	var result debt.AnalysisResult
	if err := c.doRequest(ctx, http.MethodPost, endpoint, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ── Notification Endpoints ───────────────────────────────────────────────────

func (c *Client) ListNotifications(ctx context.Context, unreadOnly bool) ([]domain.Notification, error) {
	endpoint := "/notifications"
	if unreadOnly {
		endpoint += "?unread=true"
	}
	var notifications []domain.Notification
	if err := c.doRequest(ctx, http.MethodGet, endpoint, nil, &notifications); err != nil {
		return nil, err
	}
	return notifications, nil
}

func (c *Client) MarkNotificationRead(ctx context.Context, id string) error {
	endpoint := fmt.Sprintf("/notifications/%s/read", url.PathEscape(id))
	return c.doRequest(ctx, http.MethodPatch, endpoint, nil, nil)
}

func (c *Client) MarkAllNotificationsRead(ctx context.Context) error {
	return c.doRequest(ctx, http.MethodPatch, "/notifications/read-all", nil, nil)
}
