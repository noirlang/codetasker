package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/codetasker/backend/internal/cli/client"
	"github.com/codetasker/backend/internal/cli/config"
	"github.com/codetasker/backend/internal/cli/ui"
	"github.com/codetasker/backend/internal/debt"
	"github.com/codetasker/backend/internal/domain"
)

type activeTab int

const (
	tabTasks activeTab = iota
	tabFiles
	tabDebt
	tabNotifications
)

type modalMode int

const (
	modalNone modalMode = iota
	modalInject
	modalTaskDetail
)

// Model is the main Bubbletea TUI application model.
type Model struct {
	cfg        *config.Config
	client     *client.Client
	width      int
	height     int
	tab        activeTab
	modal      modalMode
	statusMsg  string

	// Repositories
	repos         []client.RepositoryInfo
	activeRepoIdx int

	// Tasks
	tasks         []domain.Task
	selectedTaskIdx int

	// Debt
	debtResult *debt.AnalysisResult

	// Notifications
	notifications []domain.Notification

	// Task Injector form inputs
	inputPath  textinput.Model
	inputLine  textinput.Model
	inputType  textinput.Model
	inputNote  textinput.Model
	injectFocus int
	injectSubmitting bool
}

// InitialModel creates the starting state of the TUI.
func InitialModel(cfg *config.Config) Model {
	p := textinput.New()
	p.Placeholder = "src/main.go"
	p.Focus()

	l := textinput.New()
	l.Placeholder = "42"

	t := textinput.New()
	t.Placeholder = "TODO"
	t.SetValue("TODO")

	n := textinput.New()
	n.Placeholder = "Describe the task or refactoring needed..."

	c := client.NewClient(cfg)

	return Model{
		cfg:         cfg,
		client:      c,
		tab:         tabTasks,
		modal:       modalNone,
		inputPath:   p,
		inputLine:   l,
		inputType:   t,
		inputNote:   n,
		injectFocus: 0,
	}
}

// Init starts initial data fetching commands.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.fetchReposCmd(),
		m.fetchNotificationsCmd(),
		m.fetchDebtCmd(),
	)
}

// ── Background commands ──────────────────────────────────────────────────────

type reposFetchedMsg []client.RepositoryInfo
type tasksFetchedMsg []domain.Task
type notificationsFetchedMsg []domain.Notification
type debtFetchedMsg *debt.AnalysisResult
type statusMsg string
type injectDoneMsg string

func (m Model) fetchReposCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		repos, err := m.client.ListRepos(ctx)
		if err != nil {
			return statusMsg(fmt.Sprintf("Error fetching repos: %v", err))
		}
		return reposFetchedMsg(repos)
	}
}

func (m Model) fetchTasksCmd(repoID int64) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		tasks, err := m.client.ListTasks(ctx, repoID, "", "")
		if err != nil {
			return statusMsg(fmt.Sprintf("Error fetching tasks: %v", err))
		}
		return tasksFetchedMsg(tasks)
	}
}

func (m Model) fetchNotificationsCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		notes, err := m.client.ListNotifications(ctx, false)
		if err != nil {
			return nil
		}
		return notificationsFetchedMsg(notes)
	}
}

func (m Model) fetchDebtCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		res, err := debt.AnalyzeLocalRepo(ctx, debt.Options{
			Repo:       ".",
			Days:       m.cfg.DefaultDays,
			HourlyCost: m.cfg.DefaultHourlyCost,
		})
		if err != nil {
			return nil
		}
		return debtFetchedMsg(&res)
	}
}

func (m Model) submitInjectCmd() tea.Cmd {
	return func() tea.Msg {
		if len(m.repos) == 0 {
			return statusMsg("Error: No repository active to inject into")
		}
		currRepo := m.repos[m.activeRepoIdx]
		parts := strings.Split(currRepo.FullName, "/")
		if len(parts) != 2 {
			return statusMsg("Error: Invalid repository full name")
		}

		line := 1
		fmt.Sscanf(m.inputLine.Value(), "%d", &line)
		if line <= 0 {
			line = 1
		}

		taskType := m.inputType.Value()
		if taskType == "" {
			taskType = "TODO"
		}

		req := domain.InjectTaskRequest{
			RepoOwner:   parts[0],
			RepoName:    parts[1],
			Branch:      currRepo.DefaultBranch,
			FilePath:    m.inputPath.Value(),
			LineNumber:  line,
			Type:        taskType,
			Description: m.inputNote.Value(),
			Locations: []domain.TaskLocation{
				{
					FilePath:    m.inputPath.Value(),
					LineNumber:  line,
					Description: m.inputNote.Value(),
				},
			},
		}

		ctx := context.Background()
		resp, err := m.client.InjectTask(ctx, req)
		if err != nil {
			return statusMsg(fmt.Sprintf("Inject failed: %v", err))
		}
		return injectDoneMsg(resp.PRURL)
	}
}

// ── Update ───────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case reposFetchedMsg:
		m.repos = msg
		if len(m.repos) > 0 {
			return m, m.fetchTasksCmd(m.repos[m.activeRepoIdx].ID)
		}
		return m, nil

	case tasksFetchedMsg:
		m.tasks = msg
		if m.selectedTaskIdx >= len(m.tasks) {
			m.selectedTaskIdx = max(0, len(m.tasks)-1)
		}
		return m, nil

	case notificationsFetchedMsg:
		m.notifications = msg
		return m, nil

	case debtFetchedMsg:
		m.debtResult = msg
		return m, nil

	case statusMsg:
		m.statusMsg = string(msg)
		return m, nil

	case injectDoneMsg:
		m.modal = modalNone
		m.injectSubmitting = false
		m.statusMsg = fmt.Sprintf("✓ PR Created: %s", string(msg))
		if len(m.repos) > 0 {
			return m, m.fetchTasksCmd(m.repos[m.activeRepoIdx].ID)
		}
		return m, nil

	case tea.KeyMsg:
		// Modal Key Handling
		if m.modal == modalInject {
			return m.handleInjectKey(msg)
		}

		// Global Key Handling
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "tab":
			m.tab = (m.tab + 1) % 4
			return m, nil

		case "1":
			m.tab = tabTasks
			return m, nil
		case "2":
			m.tab = tabFiles
			return m, nil
		case "3":
			m.tab = tabDebt
			return m, nil
		case "4":
			m.tab = tabNotifications
			return m, nil

		case "i":
			m.modal = modalInject
			m.inputPath.SetValue("")
			m.inputLine.SetValue("1")
			m.inputNote.SetValue("")
			m.inputPath.Focus()
			m.injectFocus = 0
			return m, nil

		case "r":
			m.statusMsg = "Refreshing data..."
			if len(m.repos) > 0 {
				return m, tea.Batch(
					m.fetchTasksCmd(m.repos[m.activeRepoIdx].ID),
					m.fetchNotificationsCmd(),
					m.fetchDebtCmd(),
				)
			}
			return m, m.fetchReposCmd()

		case "s":
			if len(m.repos) > 0 {
				curr := m.repos[m.activeRepoIdx]
				parts := strings.Split(curr.FullName, "/")
				if len(parts) == 2 {
					m.statusMsg = fmt.Sprintf("Syncing %s...", curr.FullName)
					go func() {
						_, _ = m.client.SyncRepo(context.Background(), parts[0], parts[1])
					}()
					return m, m.fetchTasksCmd(curr.ID)
				}
			}
			return m, nil

		case "j", "down":
			if m.tab == tabTasks && len(m.tasks) > 0 {
				if m.selectedTaskIdx < len(m.tasks)-1 {
					m.selectedTaskIdx++
				}
			}
			return m, nil

		case "k", "up":
			if m.tab == tabTasks && len(m.tasks) > 0 {
				if m.selectedTaskIdx > 0 {
					m.selectedTaskIdx--
				}
			}
			return m, nil

		case "space":
			// Toggle status: open -> in_progress -> resolved -> open
			if m.tab == tabTasks && len(m.tasks) > 0 {
				t := m.tasks[m.selectedTaskIdx]
				var nextStatus domain.TaskStatus
				switch t.Status {
				case domain.TaskStatusOpen:
					nextStatus = domain.TaskStatusInProgress
				case domain.TaskStatusInProgress:
					nextStatus = domain.TaskStatusResolved
				default:
					nextStatus = domain.TaskStatusOpen
				}

				m.tasks[m.selectedTaskIdx].Status = nextStatus
				go func(id string, st domain.TaskStatus) {
					_ = m.client.UpdateTaskStatus(context.Background(), id, st)
				}(t.ID.Hex(), nextStatus)

				m.statusMsg = fmt.Sprintf("Updated task status to %s", string(nextStatus))
			}
			return m, nil
		}
	}

	return m, nil
}

func (m Model) handleInjectKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.modal = modalNone
		return m, nil

	case "tab", "down":
		m.injectFocus = (m.injectFocus + 1) % 4
		m.updateInjectFocus()
		return m, nil

	case "shift+tab", "up":
		m.injectFocus = (m.injectFocus - 1 + 4) % 4
		m.updateInjectFocus()
		return m, nil

	case "enter":
		if m.injectFocus < 3 {
			m.injectFocus++
			m.updateInjectFocus()
			return m, nil
		}
		// Submit
		if m.inputPath.Value() == "" || m.inputNote.Value() == "" {
			m.statusMsg = "Error: File path and Task Description are required"
			return m, nil
		}
		m.injectSubmitting = true
		m.statusMsg = "Injecting task and opening Pull Request..."
		return m, m.submitInjectCmd()
	}

	var cmd tea.Cmd
	switch m.injectFocus {
	case 0:
		m.inputPath, cmd = m.inputPath.Update(msg)
	case 1:
		m.inputLine, cmd = m.inputLine.Update(msg)
	case 2:
		m.inputType, cmd = m.inputType.Update(msg)
	case 3:
		m.inputNote, cmd = m.inputNote.Update(msg)
	}

	return m, cmd
}

func (m *Model) updateInjectFocus() {
	m.inputPath.Blur()
	m.inputLine.Blur()
	m.inputType.Blur()
	m.inputNote.Blur()

	switch m.injectFocus {
	case 0:
		m.inputPath.Focus()
	case 1:
		m.inputLine.Focus()
	case 2:
		m.inputType.Focus()
	case 3:
		m.inputNote.Focus()
	}
}

// ── View ─────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.width == 0 {
		m.width = 100
	}
	if m.height == 0 {
		m.height = 30
	}

	// Top App Header
	header := m.renderHeader()

	// Main Content Panel
	var body string
	if m.modal == modalInject {
		body = m.renderInjectModal()
	} else {
		switch m.tab {
		case tabTasks:
			body = m.renderTasksTab()
		case tabFiles:
			body = m.renderFilesTab()
		case tabDebt:
			body = m.renderDebtTab()
		case tabNotifications:
			body = m.renderNotificationsTab()
		}
	}

	// Bottom Status Bar
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m Model) renderHeader() string {
	title := ui.BoldStyle.Render(" CODEMASTER / CODETASKER CLI ")

	tabs := []string{"[1] Tasks", "[2] Files", "[3] Technical Debt", "[4] Notifications"}
	var renderedTabs []string

	for i, t := range tabs {
		if activeTab(i) == m.tab {
			renderedTabs = append(renderedTabs, lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#ffffff")).
				Background(ui.Emerald).
				Padding(0, 1).
				Render(t))
		} else {
			renderedTabs = append(renderedTabs, lipgloss.NewStyle().
				Foreground(ui.Gray).
				Padding(0, 1).
				Render(t))
		}
	}

	repoName := "No active repo"
	if len(m.repos) > 0 {
		repoName = m.repos[m.activeRepoIdx].FullName
	}

	repoPill := lipgloss.NewStyle().
		Foreground(ui.Cyan).
		Bold(true).
		Render("📦 " + repoName)

	topLine := lipgloss.JoinHorizontal(lipgloss.Center, title, "  ", strings.Join(renderedTabs, " "), "  ", repoPill)
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(ui.DarkGray).
		Padding(0, 1).
		Width(m.width).
		Render(topLine)
}

func (m Model) renderTasksTab() string {
	if len(m.tasks) == 0 {
		return lipgloss.NewStyle().
			Padding(2).
			Render("No tasks found in active repository. Press [i] to inject a task, or [s] to sync.")
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("%-6s %-12s %-30s %-12s %s", "TYPE", "STATUS", "LOCATION", "ASSIGNEE", "CONTENT"))
	lines = append(lines, strings.Repeat("─", min(m.width-4, 110)))

	for i, t := range m.tasks {
		assignee := "-"
		if t.AssigneeUsername != "" {
			assignee = "@" + t.AssigneeUsername
		}

		typeBadge := ui.TaskTypeBadge(t.Type)
		statusBadge := ui.StatusBadge(string(t.Status))
		loc := fmt.Sprintf("%s:%d", t.FilePath, t.LineNumber)
		if len(loc) > 28 {
			loc = "..." + loc[len(loc)-25:]
		}

		content := t.Content
		if len(content) > 40 {
			content = content[:37] + "..."
		}

		line := fmt.Sprintf("%-6s %-12s %-30s %-12s %s", typeBadge, statusBadge, loc, assignee, content)

		if i == m.selectedTaskIdx {
			line = lipgloss.NewStyle().
				Background(lipgloss.Color("#1f2937")).
				Bold(true).
				Render("▶ " + line)
		} else {
			line = "  " + line
		}

		lines = append(lines, line)
	}

	return lipgloss.NewStyle().
		Padding(1, 2).
		Height(max(10, m.height-6)).
		Render(strings.Join(lines, "\n"))
}

func (m Model) renderFilesTab() string {
	return lipgloss.NewStyle().
		Padding(1, 2).
		Render(`📁 Repository File Explorer & Code Viewer
To browse files locally or open Monaco in the web UI, navigate to http://localhost:5173
Or run 'codetasker scan .' to scan local directory directly from terminal.`)
}

func (m Model) renderDebtTab() string {
	if m.debtResult == nil {
		return lipgloss.NewStyle().
			Padding(2).
			Render("Technical debt analysis in progress or no local git history detected...")
	}

	res := m.debtResult
	summary := fmt.Sprintf(`
── Technical Debt Metrics ──────────────────────────────────
  Files Analyzed:      %d files
  Estimated Cost:      %s/mo
  Risk Breakdown:      Critical: %d | High: %d | Medium: %d | Low: %d
`,
		res.Summary.FilesAnalyzed,
		ui.WarningStyle.Render(ui.FormatCost(res.Summary.EstimatedMonthlyCost)),
		res.Summary.Critical,
		res.Summary.High,
		res.Summary.Medium,
		res.Summary.Low,
	)

	var hotspots []string
	hotspots = append(hotspots, "── Top Hotspot Files ───────────────────────────────────────")
	for i, f := range res.Hotspots {
		if i >= 6 {
			break
		}
		hotspots = append(hotspots, fmt.Sprintf("  %-35s Score: %-4d Level: %-8s Cost: %s",
			f.File, f.DebtScore, f.Level, ui.FormatCost(f.EstimatedMonthlyCost)))
	}

	return lipgloss.NewStyle().
		Padding(1, 2).
		Render(summary + "\n" + strings.Join(hotspots, "\n"))
}

func (m Model) renderNotificationsTab() string {
	if len(m.notifications) == 0 {
		return lipgloss.NewStyle().
			Padding(2).
			Render("✓ No unread notifications.")
	}

	var lines []string
	lines = append(lines, "── Recent Notifications ───────────────────────────────────")
	for _, n := range m.notifications {
		st := ui.SubtleStyle.Render("READ")
		if !n.Read {
			st = ui.SuccessStyle.Render("NEW")
		}
		lines = append(lines, fmt.Sprintf("  [%s] %s: %s (%s)", st, ui.BoldStyle.Render(n.Title), n.Message, n.CreatedAt.Format("15:04")))
	}

	return lipgloss.NewStyle().
		Padding(1, 2).
		Render(strings.Join(lines, "\n"))
}

func (m Model) renderInjectModal() string {
	form := fmt.Sprintf(`
┌─ [ Interactive Task Injector ] ──────────────────────────┐
│                                                          │
│  File Path:                                              │
│  %s                                                      │
│                                                          │
│  Line Number:                                            │
│  %s                                                      │
│                                                          │
│  Task Type (TODO / FIXME / BUG / HACK / NOTE):           │
│  %s                                                      │
│                                                          │
│  Task Description / Note:                                │
│  %s                                                      │
│                                                          │
│  [Enter] Next / Submit  [Tab] Switch Field  [Esc] Cancel │
└──────────────────────────────────────────────────────────┘
`,
		m.inputPath.View(),
		m.inputLine.View(),
		m.inputType.View(),
		m.inputNote.View(),
	)

	return lipgloss.NewStyle().
		Padding(1, 4).
		Render(form)
}

func (m Model) renderFooter() string {
	status := m.statusMsg
	if status == "" {
		status = "Ready"
	}

	shortcuts := "[Tab] Switch Tab  [Space] Move Status  [i] Inject Task  [s] Sync  [r] Refresh  [q] Quit"

	bar := lipgloss.JoinHorizontal(lipgloss.Center,
		ui.SubtleStyle.Render("STATUS: "+status),
		"  │  ",
		ui.BoldStyle.Render(shortcuts),
	)

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(ui.DarkGray).
		Padding(0, 1).
		Width(m.width).
		Render(bar)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
