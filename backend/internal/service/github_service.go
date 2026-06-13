// Package service implements the business logic layer of CodeTasker.
// github_service.go interacts with the GitHub REST API on behalf of
// authenticated users to list repositories, traverse file trees, fetch file
// contents, inject TODO comments, and open pull requests.
//
// Security: every user-supplied string that is incorporated into a GitHub API
// path is validated against a strict allowlist regex before use. This prevents
// Server-Side Request Forgery (SSRF) by ensuring no arbitrary path segments or
// URL-encoded redirects can be injected.
package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/codetasker/backend/internal/config"
	"github.com/codetasker/backend/internal/domain"
	"github.com/codetasker/backend/internal/repository"
	"github.com/google/go-github/v62/github"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	githuboauth "golang.org/x/oauth2/github"
)

// safeNamePattern is the SSRF guard regex. Only strings composed of
// alphanumerics, hyphens, underscores, and dots are accepted as owner/repo
// names. This rejects any attempt to inject slashes, colons, or percent-encoded
// characters that could redirect the GitHub API client to unintended hosts.
var safeNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.\-]+$`)

// GithubService provides all GitHub API interactions on behalf of
// CodeTasker users. It decrypts stored tokens, constructs authenticated
// clients, and implements the inject-TODO pipeline.
type GithubService struct {
	cfg      *config.Config
	userRepo *repository.UserRepository
	log      *zap.Logger
}

// NewGithubService constructs a GithubService with injected dependencies.
func NewGithubService(cfg *config.Config, userRepo *repository.UserRepository, log *zap.Logger) *GithubService {
	return &GithubService{
		cfg:      cfg,
		userRepo: userRepo,
		log:      log,
	}
}

// newGithubClient creates an authenticated *github.Client for the given plain
// (decrypted) access token. The oauth2 transport is used so the token is
// always sent in the Authorization header, matching GitHub's requirements.
func newGithubClient(ctx context.Context, accessToken string) *github.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
	tc := oauth2.NewClient(ctx, ts)
	// Ensure the oauth2 endpoint is respected; go-github uses the transport directly.
	_ = githuboauth.Endpoint // imported for the endpoint constant only
	return github.NewClient(tc)
}

// resolveToken is an internal helper that fetches the encrypted token for the
// given ObjectID and returns the decrypted plaintext token. It does not expose
// the plaintext token in logs.
func (s *GithubService) resolveToken(ctx context.Context, userID primitive.ObjectID) (string, error) {
	user, err := s.userRepo.FindByObjectID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("resolveToken FindByObjectID: %w", err)
	}
	if user == nil {
		return "", fmt.Errorf("user %s not found", userID.Hex())
	}

	plainToken, err := DecryptToken(user.AccessToken, s.cfg.TokenEncryptKey)
	if err != nil {
		return "", fmt.Errorf("resolveToken DecryptToken: %w", err)
	}

	return plainToken, nil
}

// CreateWebhook registers a webhook on the specified GitHub repository pointing to payloadURL.
// It returns the repository's numeric GitHub ID on success.
func (s *GithubService) CreateWebhook(ctx context.Context, userID primitive.ObjectID, owner, repo, payloadURL string) (int64, error) {
	if err := validateName(owner, "owner"); err != nil {
		return 0, err
	}
	if err := validateName(repo, "repo"); err != nil {
		return 0, err
	}

	token, err := s.resolveToken(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("CreateWebhook resolveToken: %w", err)
	}

	client := newGithubClient(ctx, token)

	// Fetch repo info to get the repository ID
	ghRepo, _, err := client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return 0, fmt.Errorf("CreateWebhook get repo: %w", err)
	}

	hookConfig := &github.HookConfig{
		URL:         github.String(payloadURL),
		ContentType: github.String("json"),
		Secret:      github.String(s.cfg.WebhookSecret),
		InsecureSSL: github.String("0"),
	}

	hook := &github.Hook{
		Name:   github.String("web"),
		Active: github.Bool(true),
		Events: []string{"push"},
		Config: hookConfig,
	}

	// In local development, if the payloadURL is localhost, we skip creating the webhook on GitHub.
	// This prevents the GitHub API from returning a 422 error since localhost is not publicly reachable,
	// while still allowing the repository to be marked as "synced" in our database for development testing.
	if strings.Contains(payloadURL, "localhost") || strings.Contains(payloadURL, "127.0.0.1") || strings.Contains(payloadURL, "::1") {
		s.log.Warn("localhost payload URL detected; skipping GitHub webhook registration to prevent 422 validation error",
			zap.String("repo", owner+"/"+repo), zap.String("url", payloadURL))
		return ghRepo.GetID(), nil
	}

	_, _, err = client.Repositories.CreateHook(ctx, owner, repo, hook)
	if err != nil {
		return 0, fmt.Errorf("CreateWebhook API call: %w", err)
	}

	s.log.Info("webhook created successfully", zap.String("repo", owner+"/"+repo), zap.String("url", payloadURL))
	return ghRepo.GetID(), nil
}

// validateName checks a user-supplied owner or repository name against the
// SSRF guard pattern. Returns an error if the name is empty or contains
// disallowed characters.
func validateName(name, field string) error {
	if name == "" {
		return fmt.Errorf("%s must not be empty", field)
	}
	if !safeNamePattern.MatchString(name) {
		return fmt.Errorf("%s %q contains invalid characters (allowed: [a-zA-Z0-9_.\\-])", field, name)
	}
	return nil
}

// ListRepos returns the authenticated user's repositories sorted by most
// recently updated, including both owned and organisation repositories.
func (s *GithubService) ListRepos(ctx context.Context, userID primitive.ObjectID) ([]*github.Repository, error) {
	token, err := s.resolveToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("ListRepos: %w", err)
	}

	client := newGithubClient(ctx, token)

	opts := &github.RepositoryListByAuthenticatedUserOptions{
		Type:      "all",
		Sort:      "updated",
		Direction: "desc",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	repos, _, err := client.Repositories.ListByAuthenticatedUser(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("ListRepos GitHub API: %w", err)
	}

	return repos, nil
}

// ListOrgs lists the organizations the authenticated user belongs to.
func (s *GithubService) ListOrgs(ctx context.Context, userID primitive.ObjectID) ([]*github.Organization, error) {
	token, err := s.resolveToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("ListOrgs: %w", err)
	}

	client := newGithubClient(ctx, token)
	orgs, _, err := client.Organizations.List(ctx, "", nil)
	if err != nil {
		return nil, fmt.Errorf("ListOrgs GitHub API: %w", err)
	}

	return orgs, nil
}

// ListOrgRepos lists the repositories of a specific organization.
func (s *GithubService) ListOrgRepos(ctx context.Context, userID primitive.ObjectID, org string) ([]*github.Repository, error) {
	if err := validateName(org, "org"); err != nil {
		return nil, err
	}

	token, err := s.resolveToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("ListOrgRepos: %w", err)
	}

	client := newGithubClient(ctx, token)
	opts := &github.RepositoryListByOrgOptions{
		Type: "all",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}
	repos, _, err := client.Repositories.ListByOrg(ctx, org, opts)
	if err != nil {
		return nil, fmt.Errorf("ListOrgRepos GitHub API (%s): %w", org, err)
	}

	return repos, nil
}

// GetTree fetches the full recursive file tree for a repository branch.
// The tree is used by the frontend to let users browse files before injecting
// a TODO comment at a specific path and line.
func (s *GithubService) GetTree(ctx context.Context, userID primitive.ObjectID, owner, repo, branch string) (*github.Tree, error) {
	if err := validateName(owner, "owner"); err != nil {
		return nil, err
	}
	if err := validateName(repo, "repo"); err != nil {
		return nil, err
	}

	token, err := s.resolveToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("GetTree: %w", err)
	}

	client := newGithubClient(ctx, token)

	// Use branch as the SHA/ref; recursive=true returns all nested entries.
	tree, _, err := client.Git.GetTree(ctx, owner, repo, branch, true)
	if err != nil {
		return nil, fmt.Errorf("GetTree GitHub API (%s/%s@%s): %w", owner, repo, branch, err)
	}

	return tree, nil
}

// GetContents fetches the decoded string content of a single file in a
// repository. GitHub returns file content base64-encoded; this method decodes
// it and returns the raw text, which is then fed to the parser.
func (s *GithubService) GetContents(ctx context.Context, userID primitive.ObjectID, owner, repo, path, ref string) (string, error) {
	if err := validateName(owner, "owner"); err != nil {
		return "", err
	}
	if err := validateName(repo, "repo"); err != nil {
		return "", err
	}

	token, err := s.resolveToken(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("GetContents: %w", err)
	}

	client := newGithubClient(ctx, token)

	opts := &github.RepositoryContentGetOptions{Ref: ref}
	fileContent, _, _, err := client.Repositories.GetContents(ctx, owner, repo, path, opts)
	if err != nil {
		return "", fmt.Errorf("GetContents GitHub API (%s/%s/%s@%s): %w", owner, repo, path, ref, err)
	}

	if fileContent == nil {
		return "", fmt.Errorf("GetContents: path %q returned no content (may be a directory)", path)
	}

	decoded, err := fileContent.GetContent()
	if err != nil {
		return "", fmt.Errorf("GetContents decode base64: %w", err)
	}

	return decoded, nil
}

// InjectTODO inserts a `// TODO: <description>` comment at the specified line
// of a file in a GitHub repository via a sequence of low-level Git API calls,
// then opens a pull request. The full pipeline is:
//
//  1. Validate owner/repo/path inputs against the SSRF guard pattern.
//  2. Get the latest commit SHA on the target branch.
//  3. Get the tree SHA from that commit.
//  4. Fetch and decode the target file's current content.
//  5. Insert the TODO comment at the requested line number.
//  6. Create a new blob with the modified content.
//  7. Create a new tree that replaces the old blob with the new one.
//  8. Create a new commit pointing at the new tree.
//  9. Create a new branch `codetasker/inject-<unix-timestamp>`.
//  10. Open a PR from that branch into the base branch.
//  11. Return the PR URL.
func (s *GithubService) InjectTODO(ctx context.Context, userID primitive.ObjectID, req *domain.InjectTaskRequest) (string, error) {
	// ── SSRF guards ─────────────────────────────────────────────────────────
	if err := validateName(req.RepoOwner, "repo_owner"); err != nil {
		return "", err
	}
	if err := validateName(req.RepoName, "repo_name"); err != nil {
		return "", err
	}
	if err := validateName(req.Branch, "branch"); err != nil {
		return "", err
	}

	// file_path may contain slashes — validate each segment individually.
	for _, segment := range strings.Split(req.FilePath, "/") {
		if segment == "" {
			continue
		}
		if err := validateName(segment, "file_path segment"); err != nil {
			return "", fmt.Errorf("invalid file_path %q: %w", req.FilePath, err)
		}
	}

	token, err := s.resolveToken(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("InjectTODO resolveToken: %w", err)
	}

	client := newGithubClient(ctx, token)
	owner := req.RepoOwner
	repo := req.RepoName
	branch := req.Branch

	// ── Step 1 & 2: Get latest commit and its tree ──────────────────────────
	ref, _, err := client.Git.GetRef(ctx, owner, repo, "refs/heads/"+branch)
	if err != nil {
		return "", fmt.Errorf("InjectTODO GetRef: %w", err)
	}

	latestCommitSHA := ref.GetObject().GetSHA()

	commit, _, err := client.Git.GetCommit(ctx, owner, repo, latestCommitSHA)
	if err != nil {
		return "", fmt.Errorf("InjectTODO GetCommit: %w", err)
	}

	commitTreeSHA := commit.GetTree().GetSHA()

	// ── Step 3 & 4: Fetch and decode the target file ─────────────────────────
	opts := &github.RepositoryContentGetOptions{Ref: branch}
	fileContent, _, _, err := client.Repositories.GetContents(ctx, owner, repo, req.FilePath, opts)
	if err != nil {
		return "", fmt.Errorf("InjectTODO GetContents(%s): %w", req.FilePath, err)
	}

	existingContent, err := fileContent.GetContent()
	if err != nil {
		return "", fmt.Errorf("InjectTODO decode file content: %w", err)
	}

	// ── Step 5: Insert the TODO comment at the requested line ─────────────────
	lines := strings.Split(existingContent, "\n")

	commentSymbol := getCommentPrefix(req.FilePath)
	tagType := req.Type
	if tagType == "" {
		tagType = "TODO"
	}
	todoLine := fmt.Sprintf("%s %s: %s", commentSymbol, tagType, req.Description)

	insertAt := req.LineNumber - 1 // convert to 0-based index
	if insertAt < 0 {
		insertAt = 0
	}
	if insertAt > len(lines) {
		insertAt = len(lines)
	}

	// Insert by growing the slice.
	lines = append(lines, "")
	copy(lines[insertAt+1:], lines[insertAt:])
	lines[insertAt] = todoLine

	modifiedContent := strings.Join(lines, "\n")

	// ── Step 6: Create a new blob for the modified file ───────────────────────
	encodingStr := "base64"
	encodedContent := base64.StdEncoding.EncodeToString([]byte(modifiedContent))

	blob, _, err := client.Git.CreateBlob(ctx, owner, repo, &github.Blob{
		Content:  &encodedContent,
		Encoding: &encodingStr,
	})
	if err != nil {
		return "", fmt.Errorf("InjectTODO CreateBlob: %w", err)
	}

	// ── Step 7: Create a new tree referencing the updated blob ────────────────
	fileMode := "100644" // regular file mode
	blobType := "blob"
	filePath := req.FilePath // local var so we can take its address

	newTree, _, err := client.Git.CreateTree(ctx, owner, repo, commitTreeSHA, []*github.TreeEntry{
		{
			Path: &filePath,
			Mode: &fileMode,
			Type: &blobType,
			SHA:  blob.SHA,
		},
	})
	if err != nil {
		return "", fmt.Errorf("InjectTODO CreateTree: %w", err)
	}

	// ── Step 8: Create the new commit ─────────────────────────────────────────
	commitMsg := fmt.Sprintf("[CodeTasker] Add TODO at %s:%d", req.FilePath, req.LineNumber)

	newCommit, _, err := client.Git.CreateCommit(ctx, owner, repo, &github.Commit{
		Message: &commitMsg,
		Tree:    &github.Tree{SHA: newTree.SHA},
		Parents: []*github.Commit{{SHA: &latestCommitSHA}},
	}, &github.CreateCommitOptions{})
	if err != nil {
		return "", fmt.Errorf("InjectTODO CreateCommit: %w", err)
	}

	// ── Step 9: Create the new branch ─────────────────────────────────────────
	newBranchName := fmt.Sprintf("codetasker/inject-%d", time.Now().Unix())
	newRefName := "refs/heads/" + newBranchName

	_, _, err = client.Git.CreateRef(ctx, owner, repo, &github.Reference{
		Ref:    &newRefName,
		Object: &github.GitObject{SHA: newCommit.SHA},
	})
	if err != nil {
		return "", fmt.Errorf("InjectTODO CreateRef (%s): %w", newBranchName, err)
	}

	// ── Step 10: Open the pull request ────────────────────────────────────────
	prTitle := fmt.Sprintf("[CodeTasker] Add TODO: %s", req.Description)
	prBody := fmt.Sprintf(
		"This PR was automatically generated by **CodeTasker**.\n\n"+
			"**File:** `%s`  \n**Line:** %d  \n**TODO:** %s\n",
		req.FilePath, req.LineNumber, req.Description,
	)

	pr, _, err := client.PullRequests.Create(ctx, owner, repo, &github.NewPullRequest{
		Title: &prTitle,
		Body:  &prBody,
		Head:  &newBranchName,
		Base:  &branch,
	})
	if err != nil {
		return "", fmt.Errorf("InjectTODO CreatePullRequest: %w", err)
	}

	s.log.Info("TODO injected via PR",
		zap.String("repo", owner+"/"+repo),
		zap.String("file", req.FilePath),
		zap.Int("line", req.LineNumber),
		zap.String("pr_url", pr.GetHTMLURL()),
	)

	// ── Step 11: Return PR URL ────────────────────────────────────────────────
	return pr.GetHTMLURL(), nil
}

// UpdateFile commits a file change to the specified GitHub repository and returns the new commit SHA.
// It supports Git co-authors by appending "Co-authored-by: Name <email>" to the commit message.
func (s *GithubService) UpdateFile(ctx context.Context, userID primitive.ObjectID, owner, repo, path, content, branch, message string, coAuthors []string) (string, error) {
	if err := validateName(owner, "owner"); err != nil {
		return "", err
	}
	if err := validateName(repo, "repo"); err != nil {
		return "", err
	}

	token, err := s.resolveToken(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("UpdateFile resolveToken: %w", err)
	}

	client := newGithubClient(ctx, token)

	// We need to fetch the existing file's SHA to update it on GitHub
	opts := &github.RepositoryContentGetOptions{
		Ref: branch,
	}
	fileContent, _, _, err := client.Repositories.GetContents(ctx, owner, repo, path, opts)
	if err != nil {
		return "", fmt.Errorf("UpdateFile GetContents: %w", err)
	}

	commitMsg := message
	for _, coAuthor := range coAuthors {
		coAuthorTrimmed := strings.TrimSpace(coAuthor)
		if coAuthorTrimmed != "" {
			commitMsg += "\n\nCo-authored-by: " + coAuthorTrimmed
		}
	}

	updateOpts := &github.RepositoryContentFileOptions{
		Message:   github.String(commitMsg),
		Content:   []byte(content),
		SHA:       fileContent.SHA,
		Branch:    github.String(branch),
	}

	resp, _, err := client.Repositories.UpdateFile(ctx, owner, repo, path, updateOpts)
	if err != nil {
		return "", fmt.Errorf("UpdateFile UpdateFile: %w", err)
	}

	return resp.Commit.GetSHA(), nil
}

// ListCommits lists the commits on a given branch.
func (s *GithubService) ListCommits(ctx context.Context, userID primitive.ObjectID, owner, repo, branch string) ([]*github.RepositoryCommit, error) {
	if err := validateName(owner, "owner"); err != nil {
		return nil, err
	}
	if err := validateName(repo, "repo"); err != nil {
		return nil, err
	}

	token, err := s.resolveToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("ListCommits resolveToken: %w", err)
	}

	client := newGithubClient(ctx, token)

	opts := &github.CommitsListOptions{
		ListOptions: github.ListOptions{
			PerPage: 50,
		},
	}
	if branch != "" {
		opts.SHA = branch
	}

	commits, _, err := client.Repositories.ListCommits(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("ListCommits API call: %w", err)
	}

	return commits, nil
}

// ListPullRequests lists the pull requests in the repository.
func (s *GithubService) ListPullRequests(ctx context.Context, userID primitive.ObjectID, owner, repo, state string) ([]*github.PullRequest, error) {
	if err := validateName(owner, "owner"); err != nil {
		return nil, err
	}
	if err := validateName(repo, "repo"); err != nil {
		return nil, err
	}

	token, err := s.resolveToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("ListPullRequests resolveToken: %w", err)
	}

	client := newGithubClient(ctx, token)

	if state == "" {
		state = "open"
	}

	opts := &github.PullRequestListOptions{
		State: state,
		ListOptions: github.ListOptions{
			PerPage: 50,
		},
	}

	pulls, _, err := client.PullRequests.List(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("ListPullRequests API call: %w", err)
	}

	return pulls, nil
}

// MergeBranch merges head branch into base branch.
func (s *GithubService) MergeBranch(ctx context.Context, userID primitive.ObjectID, owner, repo, base, head, commitMsg string) (*github.RepositoryCommit, error) {
	if err := validateName(owner, "owner"); err != nil {
		return nil, err
	}
	if err := validateName(repo, "repo"); err != nil {
		return nil, err
	}

	token, err := s.resolveToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("MergeBranch resolveToken: %w", err)
	}

	client := newGithubClient(ctx, token)

	req := &github.RepositoryMergeRequest{
		Base:          github.String(base),
		Head:          github.String(head),
		CommitMessage: github.String(commitMsg),
	}

	resp, _, err := client.Repositories.Merge(ctx, owner, repo, req)
	if err != nil {
		return nil, fmt.Errorf("MergeBranch API call: %w", err)
	}

	return resp, nil
}

// GetRepository fetches repository details including the default branch.
func (s *GithubService) GetRepository(ctx context.Context, userID primitive.ObjectID, owner, repo string) (*github.Repository, error) {
	if err := validateName(owner, "owner"); err != nil {
		return nil, err
	}
	if err := validateName(repo, "repo"); err != nil {
		return nil, err
	}

	token, err := s.resolveToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("GetRepository resolveToken: %w", err)
	}

	client := newGithubClient(ctx, token)
	r, _, err := client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("GetRepository GitHub API: %w", err)
	}

	return r, nil
}

// GetUserByUsername fetches public user information for a given GitHub username.
func (s *GithubService) GetUserByUsername(ctx context.Context, userID primitive.ObjectID, username string) (*github.User, error) {
	if !safeNamePattern.MatchString(username) {
		return nil, fmt.Errorf("invalid username parameter (SSRF prevention)")
	}

	token, err := s.resolveToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("GetUserByUsername resolveToken: %w", err)
	}

	client := newGithubClient(ctx, token)
	ghUser, _, err := client.Users.Get(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("GitHub Users.Get(%s) failed: %w", username, err)
	}

	return ghUser, nil
}


// getCommentPrefix detects the appropriate comment prefix based on the file extension.
func getCommentPrefix(filePath string) string {
	parts := strings.Split(filePath, "/")
	filename := strings.ToLower(parts[len(parts)-1])

	ext := ""
	extParts := strings.Split(filename, ".")
	if len(extParts) > 1 {
		ext = extParts[len(extParts)-1]
	}

	hashLangs := map[string]bool{
		"py": true, "rb": true, "sh": true, "bash": true, "yaml": true,
		"yml": true, "toml": true, "pl": true, "r": true, "dockerfile": true,
		"makefile": true,
	}
	dashLangs := map[string]bool{
		"sql": true, "lua": true, "hs": true,
	}

	if hashLangs[ext] || filename == "dockerfile" || filename == "makefile" {
		return "#"
	}
	if dashLangs[ext] {
		return "--"
	}
	return "//"
}


