package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/spf13/cobra"
)

// Agent type constants
const (
	AgentClaude   = "claude"
	AgentCursor   = "cursor"
	AgentCline    = "cline"
	AgentCodex    = "codex"
	AgentOpenCode = "opencode"
)

// Scope constants
const (
	ScopeUser    = "user"
	ScopeProject = "project"
	ScopeBoth    = "both"
)

// InstallPath holds user and project level installation paths for an agent
type InstallPath struct {
	User    string
	Project string
}

// AgentInfo contains metadata about a supported agent
type AgentInfo struct {
	Name        string
	DisplayName string
	Description string
	Paths       InstallPath
}

// TemplateData holds variables for template rendering
type TemplateData struct {
	Version   string
	Date      string
	Timestamp string
}

// Supported agents configuration
var supportedAgents = map[string]AgentInfo{
	AgentClaude: {
		Name:        AgentClaude,
		DisplayName: "Claude Code",
		Description: "Anthropic's Claude Code CLI agent",
		Paths: InstallPath{
			User:    filepath.Join(".claude", "commands", "db9.md"),
			Project: filepath.Join(".claude", "commands", "db9.md"),
		},
	},
	AgentCursor: {
		Name:        AgentCursor,
		DisplayName: "Cursor IDE",
		Description: "AI-powered code editor",
		Paths: InstallPath{
			User:    "", // Cursor only supports project scope
			Project: filepath.Join(".cursor", "rules", "db9.md"),
		},
	},
	AgentCline: {
		Name:        AgentCline,
		DisplayName: "Cline",
		Description: "VSCode Cline extension",
		Paths: InstallPath{
			User:    filepath.Join(".cline", "rules", "db9.md"),
			Project: filepath.Join(".cline", "rules", "db9.md"),
		},
	},
	AgentCodex: {
		Name:        AgentCodex,
		DisplayName: "OpenAI Codex",
		Description: "OpenAI's coding agent",
		Paths: InstallPath{
			User:    filepath.Join(".codex", "commands", "db9.md"),
			Project: filepath.Join(".codex", "commands", "db9.md"),
		},
	},
	AgentOpenCode: {
		Name:        AgentOpenCode,
		DisplayName: "OpenCode",
		Description: "Open-source AI coding agent",
		Paths: InstallPath{
			User:    filepath.Join(".config", "opencode", "commands", "db9.md"),
			Project: filepath.Join(".opencode", "commands", "db9.md"),
		},
	},
}

// Template definitions for each agent
var agentTemplates = map[string]string{
	AgentClaude:   "# DB9 Database Commands\n\nYou have access to the db9 CLI tool for database management.\n\n## Common Commands\n\n### Create a new database\n```bash\ndb9 create --name <database-name>\n```\n\n### Execute SQL\n```bash\ndb9 sql <database-id> \"<SQL query>\"\n```\n\n### File operations\n```bash\ndb9 fs upload <database-id> <local-path> <remote-path>\ndb9 fs download <database-id> <remote-path> <local-path>\ndb9 fs list <database-id> [path]\n```\n\n### Snapshot management\n```bash\ndb9 snapshot create <database-id> --name <snapshot-name>\ndb9 snapshot list <database-id>\ndb9 snapshot restore <database-id> <snapshot-id>\n```\n\n### Branch management\n```bash\ndb9 branch create <database-id> --name <branch-name>\ndb9 branch list <database-id>\ndb9 branch delete <database-id> <branch-id>\n```\n\n### Generate types\n```bash\ndb9 gen types <database-id> ./types --language ts\n```\n\n## Best Practices\n- Always use parameterized queries ($1, $2) to prevent SQL injection\n- Use snapshots before making schema changes\n- Use branches for experimental features\n- Check connection with `db9 status` before operations\n\n---\n*Installed by db9 onboard (v{{.Version}}) on {{.Date}}*",
	AgentCursor:   "# Open-DB9 Rules\n\nWhen working with databases in this project, use the db9 CLI tool:\n\n1. **Database Creation**: Use `db9 create` for new databases\n2. **SQL Execution**: Use `db9 sql` for queries (never connect directly)\n3. **File Storage**: Use `db9 fs upload/download/list` for file operations\n4. **Snapshots**: Always snapshot before migrations\n5. **Branches**: Use branches for feature development\n6. **Type Safety**: Generate types with `db9 gen types`\n\nNever hardcode connection strings. Always use db9 CLI.\n\n---\n*Installed by db9 onboard (v{{.Version}}) on {{.Date}}*",
	AgentCline:    "# Open-DB9 Rules\n\nWhen working with databases in this project, use the db9 CLI tool:\n\n1. **Database Creation**: Use `db9 create` for new databases\n2. **SQL Execution**: Use `db9 sql` for queries (never connect directly)\n3. **File Storage**: Use `db9 fs upload/download/list` for file operations\n4. **Snapshots**: Always snapshot before migrations\n5. **Branches**: Use branches for feature development\n6. **Type Safety**: Generate types with `db9 gen types`\n\nNever hardcode connection strings. Always use db9 CLI.\n\n---\n*Installed by db9 onboard (v{{.Version}}) on {{.Date}}*",
	AgentCodex:    "# DB9 Database Commands\n\nYou have access to the db9 CLI tool for database management.\n\n## Common Commands\n\n### Create a new database\n```bash\ndb9 create --name <database-name>\n```\n\n### Execute SQL\n```bash\ndb9 sql <database-id> \"<SQL query>\"\n```\n\n### File operations\n```bash\ndb9 fs upload <database-id> <local-path> <remote-path>\ndb9 fs download <database-id> <remote-path> <local-path>\ndb9 fs list <database-id> [path]\n```\n\n### Snapshot management\n```bash\ndb9 snapshot create <database-id> --name <snapshot-name>\ndb9 snapshot list <database-id>\ndb9 snapshot restore <database-id> <snapshot-id>\n```\n\n## Best Practices\n- Always use parameterized queries ($1, $2) to prevent SQL injection\n- Use snapshots before making schema changes\n- Check connection with `db9 status` before operations\n\n---\n*Installed by db9 onboard (v{{.Version}}) on {{.Date}}*",
	AgentOpenCode: "# DB9 Database Commands\n\nYou have access to the db9 CLI tool for database management.\n\n## Common Commands\n\n### Create a new database\n```bash\ndb9 create --name <database-name>\n```\n\n### Execute SQL\n```bash\ndb9 sql <database-id> \"<SQL query>\"\n```\n\n### File operations\n```bash\ndb9 fs upload <database-id> <local-path> <remote-path>\ndb9 fs download <database-id> <remote-path> <local-path>\ndb9 fs list <database-id> [path]\n```\n\n### Snapshot management\n```bash\ndb9 snapshot create <database-id> --name <snapshot-name>\ndb9 snapshot list <database-id>\ndb9 snapshot restore <database-id> <snapshot-id>\n```\n\n### Generate types\n```bash\ndb9 gen types <database-id> ./types --language ts\n```\n\n## Best Practices\n- Always use parameterized queries ($1, $2) to prevent SQL injection\n- Use snapshots before making schema changes\n- Check connection with `db9 status` before operations\n\n---\n*Installed by db9 onboard (v{{.Version}}) on {{.Date}}*",
}

// OnboardCmd represents the onboard command
var OnboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Install db9 skills for AI agents",
	Long: `Install db9 command documentation and best practices for various AI coding agents.

Supported agents:
  - claude: Claude Code (Anthropic)
  - cursor: Cursor IDE
  - cline: VSCode Cline extension
  - codex: OpenAI Codex
  - opencode: OpenCode

Examples:
  # Install for Claude Code (user-level)
  db9 onboard --agent claude

  # Install for Cursor (project-level only)
  db9 onboard --agent cursor --scope project

  # Install for all scopes
  db9 onboard --agent cline --scope both

  # List installed agents
  db9 onboard --list

  # Uninstall from Claude Code
  db9 onboard --uninstall claude`,
	RunE: runOnboard,
}

// Command flags
var (
	agentName      string
	scope          string
	listAgents     bool
	uninstallAgent string
	version        = "0.1.0" // Will be replaced during build
)

func init() {
	OnboardCmd.Flags().StringVarP(&agentName, "agent", "a", "",
		"Agent name to install (claude, cursor, cline, codex, opencode)")
	OnboardCmd.Flags().StringVarP(&scope, "scope", "s", "user",
		"Installation scope: user, project, or both")
	OnboardCmd.Flags().BoolVarP(&listAgents, "list", "l", false,
		"List all supported agents and their installation status")
	OnboardCmd.Flags().StringVarP(&uninstallAgent, "uninstall", "u", "",
		"Uninstall db9 skills from an agent")

	OnboardCmd.MarkFlagsMutuallyExclusive("agent", "list")
	OnboardCmd.MarkFlagsMutuallyExclusive("agent", "uninstall")
	OnboardCmd.MarkFlagsMutuallyExclusive("list", "uninstall")
}

// runOnboard executes the onboard command
func runOnboard(cmd *cobra.Command, args []string) error {
	// Handle list mode
	if listAgents {
		return listInstalledAgents()
	}

	// Handle uninstall mode
	if uninstallAgent != "" {
		return uninstallFromAgent(uninstallAgent)
	}

	// Handle install mode
	if agentName == "" {
		return fmt.Errorf("agent name is required for installation. Use --agent <name>")
	}

	return installToAgent(agentName, scope)
}

// installToAgent installs db9 skills for a specific agent
func installToAgent(agentName string, scope string) error {
	// Validate agent type
	agent, ok := supportedAgents[agentName]
	if !ok {
		return fmt.Errorf("unsupported agent: %s\nSupported agents: %s",
			agentName, getSupportedAgentNames())
	}

	// Validate scope
	if scope != ScopeUser && scope != ScopeProject && scope != ScopeBoth {
		return fmt.Errorf("invalid scope: %s. Must be 'user', 'project', or 'both'", scope)
	}

	// Prepare template data
	data := TemplateData{
		Version:   version,
		Date:      time.Now().Format("2006-01-02"),
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Get template for this agent
	tmplContent, ok := agentTemplates[agentName]
	if !ok {
		return fmt.Errorf("no template available for agent: %s", agentName)
	}

	// Render template
	rendered, err := renderTemplate(tmplContent, data)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Determine which scopes to install
	scopes := []string{}
	switch scope {
	case ScopeUser:
		scopes = []string{ScopeUser}
	case ScopeProject:
		scopes = []string{ScopeProject}
	case ScopeBoth:
		scopes = []string{ScopeUser, ScopeProject}
	}

	// Install to each scope
	successCount := 0
	for _, s := range scopes {
		var targetPath string
		var basePath string

		if s == ScopeUser {
			if agent.Paths.User == "" {
				fmt.Printf("  !  Agent '%s' does not support user-level installation, skipping\n",
					agent.DisplayName)
				continue
			}
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get user home directory: %w", err)
			}
			targetPath = filepath.Join(homeDir, agent.Paths.User)
			basePath = homeDir
		} else { // project
			if agent.Paths.Project == "" {
				fmt.Printf("  !  Agent '%s' does not support project-level installation, skipping\n",
					agent.DisplayName)
				continue
			}
			wd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}
			targetPath = filepath.Join(wd, agent.Paths.Project)
			basePath = wd
		}

		// Create directory if needed
		dir := filepath.Dir(targetPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("  X Failed to create directory %s: %v\n", dir, err)
			continue
		}

		// Write file
		if err := os.WriteFile(targetPath, []byte(rendered), 0644); err != nil {
			fmt.Printf("  X Failed to write file %s: %v\n", targetPath, err)
			continue
		}

		// Verify write
		written, err := os.ReadFile(targetPath)
		if err != nil || len(written) == 0 {
			fmt.Printf("  X Verification failed for %s\n", targetPath)
			continue
		}

		relPath, _ := filepath.Rel(basePath, targetPath)
		fmt.Printf("  [OK] Installed to %s (%s)\n", s, relPath)
		successCount++
	}

	if successCount == 0 {
		return fmt.Errorf("no installations succeeded for agent '%s'", agent.DisplayName)
	}

	fmt.Printf("\nSuccessfully installed db9 skills for %s (%d location(s))\n",
		agent.DisplayName, successCount)
	return nil
}

// uninstallFromAgent removes db9 skills from an agent
func uninstallFromAgent(agentName string) error {
	// Validate agent type
	agent, ok := supportedAgents[agentName]
	if !ok {
		return fmt.Errorf("unsupported agent: %s\nSupported agents: %s",
			agentName, getSupportedAgentNames())
	}

	successCount := 0

	// Try to uninstall from both user and project locations
	locations := []struct {
		path  string
		scope string
	}{
		{agent.Paths.User, ScopeUser},
		{agent.Paths.Project, ScopeProject},
	}

	for _, loc := range locations {
		if loc.path == "" {
			continue
		}

		var fullPath string
		var basePath string

		if loc.scope == ScopeUser {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				fmt.Printf("  ! Could not determine home directory: %v\n", err)
				continue
			}
			fullPath = filepath.Join(homeDir, loc.path)
			basePath = homeDir
		} else {
			wd, err := os.Getwd()
			if err != nil {
				fmt.Printf("  ! Could not determine working directory: %v\n", err)
				continue
			}
			fullPath = filepath.Join(wd, loc.path)
			basePath = wd
		}

		// Check if file exists
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			continue // Not installed at this location
		}

		// Delete file
		if err := os.Remove(fullPath); err != nil {
			fmt.Printf("  X Failed to remove %s: %v\n", fullPath, err)
			continue
		}

		relPath, _ := filepath.Rel(basePath, fullPath)
		fmt.Printf("  [OK] Removed from %s (%s)\n", loc.scope, relPath)

		// Try to clean up empty parent directories
		cleanupEmptyDirs(filepath.Dir(fullPath))

		successCount++
	}

	if successCount == 0 {
		fmt.Printf("No db9 skills found installed for %s\n", agent.DisplayName)
		return nil
	}

	fmt.Printf("\nSuccessfully uninstalled db9 skills from %s (%d location(s))\n",
		agent.DisplayName, successCount)
	return nil
}

// listInstalledAgents shows installation status of all agents
func listInstalledAgents() error {
	fmt.Println("Supported AI Agents:")
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("%-12s %-15s %-30s %s\n", "AGENT", "DISPLAY NAME", "STATUS", "PATHS")
	fmt.Println(strings.Repeat("-", 80))

	for name, agent := range supportedAgents {
		status := "Not Installed"
		paths := []string{}

		// Check user-level installation
		if agent.Paths.User != "" {
			homeDir, _ := os.UserHomeDir()
			userPath := filepath.Join(homeDir, agent.Paths.User)
			if _, err := os.Stat(userPath); err == nil {
				status = "Installed [OK]"
			}
			paths = append(paths, "user:"+agent.Paths.User)
		}

		// Check project-level installation
		if agent.Paths.Project != "" {
			wd, _ := os.Getwd()
			projectPath := filepath.Join(wd, agent.Paths.Project)
			if _, err := os.Stat(projectPath); err == nil {
				if status == "Not Installed" {
					status = "Installed [OK]"
				}
			}
			paths = append(paths, "project:"+agent.Paths.Project)
		}

		pathStr := strings.Join(paths, ", ")
		fmt.Printf("%-12s %-15s %-30s %s\n", name, agent.DisplayName, status, pathStr)
	}

	fmt.Println(strings.Repeat("-", 80))
	fmt.Println("\nUse 'db9 onboard --agent <name>' to install for an agent")
	return nil
}

// renderTemplate renders a template string with provided data
func renderTemplate(tmplStr string, data TemplateData) (string, error) {
	tmpl, err := template.New("agent").Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// cleanupEmptyDirs removes empty parent directories
func cleanupEmptyDirs(dir string) {
	// Walk up and remove empty directories
	for {
		isEmpty, err := isDirEmpty(dir)
		if err != nil || !isEmpty {
			break
		}

		if err := os.Remove(dir); err != nil {
			break
		}

		dir = filepath.Dir(dir)
		if dir == "." || dir == filepath.Dir(dir) {
			break // Stop at root or when path doesn't change
		}
	}
}

// isDirEmpty checks if a directory is empty
func isDirEmpty(dir string) (bool, error) {
	f, err := os.Open(dir)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

// getSupportedAgentNames returns comma-separated list of supported agent names
func getSupportedAgentNames() string {
	names := make([]string, 0, len(supportedAgents))
	for name := range supportedAgents {
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}
