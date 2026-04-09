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
	AgentClaude    = "claude"
	AgentCursor    = "cursor"
	AgentCline     = "cline"
	AgentCodex     = "codex"
	AgentOpenCode  = "opencode"
	AgentWorkBuddy = "workbuddy"
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
	AgentWorkBuddy: {
		Name:        AgentWorkBuddy,
		DisplayName: "WorkBuddy / OpenClaw",
		Description: "Open-source AI coding assistant (OpenClaw family)",
		Paths: InstallPath{
			User:    filepath.Join(".workbuddy", "skills", "db9", "SKILL.md"),
			Project: filepath.Join(".workbuddy", "skills", "db9", "SKILL.md"),
		},
	},
}

// Template definitions for each agent
var agentTemplates = map[string]string{
	AgentClaude:    "# DB9 Database Commands\n\nYou have access to the db9 CLI tool for database management.\n\n## Common Commands\n\n### Create a new database\n```bash\ndb9 create --name <database-name>\n```\n\n### Execute SQL\n```bash\ndb9 sql <database-id> \"<SQL query>\"\n```\n\n### File operations\n```bash\ndb9 fs upload <database-id> <local-path> <remote-path>\ndb9 fs download <database-id> <remote-path> <local-path>\ndb9 fs list <database-id> [path]\n```\n\n### Snapshot management\n```bash\ndb9 snapshot create <database-id> --name <snapshot-name>\ndb9 snapshot list <database-id>\ndb9 snapshot restore <database-id> <snapshot-id>\n```\n\n### Branch management\n```bash\ndb9 branch create <database-id> --name <branch-name>\ndb9 branch list <database-id>\ndb9 branch delete <database-id> <branch-id>\n```\n\n### Generate types\n```bash\ndb9 gen types <database-id> ./types --language ts\n```\n\n## Best Practices\n- Always use parameterized queries ($1, $2) to prevent SQL injection\n- Use snapshots before making schema changes\n- Use branches for experimental features\n- Check connection with `db9 status` before operations\n\n---\n*Installed by db9 onboard (v{{.Version}}) on {{.Date}}*",
	AgentCursor:    "# Open-DB9 Rules\n\nWhen working with databases in this project, use the db9 CLI tool:\n\n1. **Database Creation**: Use `db9 create` for new databases\n2. **SQL Execution**: Use `db9 sql` for queries (never connect directly)\n3. **File Storage**: Use `db9 fs upload/download/list` for file operations\n4. **Snapshots**: Always snapshot before migrations\n5. **Branches**: Use branches for feature development\n6. **Type Safety**: Generate types with `db9 gen types`\n\nNever hardcode connection strings. Always use db9 CLI.\n\n---\n*Installed by db9 onboard (v{{.Version}}) on {{.Date}}*",
	AgentCline:     "# Open-DB9 Rules\n\nWhen working with databases in this project, use the db9 CLI tool:\n\n1. **Database Creation**: Use `db9 create` for new databases\n2. **SQL Execution**: Use `db9 sql` for queries (never connect directly)\n3. **File Storage**: Use `db9 fs upload/download/list` for file operations\n4. **Snapshots**: Always snapshot before migrations\n5. **Branches**: Use branches for feature development\n6. **Type Safety**: Generate types with `db9 gen types`\n\nNever hardcode connection strings. Always use db9 CLI.\n\n---\n*Installed by db9 onboard (v{{.Version}}) on {{.Date}}*",
	AgentCodex:     "# DB9 Database Commands\n\nYou have access to the db9 CLI tool for database management.\n\n## Common Commands\n\n### Create a new database\n```bash\ndb9 create --name <database-name>\n```\n\n### Execute SQL\n```bash\ndb9 sql <database-id> \"<SQL query>\"\n```\n\n### File operations\n```bash\ndb9 fs upload <database-id> <local-path> <remote-path>\ndb9 fs download <database-id> <remote-path> <local-path>\ndb9 fs list <database-id> [path]\n```\n\n### Snapshot management\n```bash\ndb9 snapshot create <database-id> --name <snapshot-name>\ndb9 snapshot list <database-id>\ndb9 snapshot restore <database-id> <snapshot-id>\n```\n\n## Best Practices\n- Always use parameterized queries ($1, $2) to prevent SQL injection\n- Use snapshots before making schema changes\n- Check connection with `db9 status` before operations\n\n---\n*Installed by db9 onboard (v{{.Version}}) on {{.Date}}*",
	AgentOpenCode:  "# DB9 Database Commands\n\nYou have access to the db9 CLI tool for database management.\n\n## Common Commands\n\n### Create a new database\n```bash\ndb9 create --name <database-name>\n```\n\n### Execute SQL\n```bash\ndb9 sql <database-id> \"<SQL query>\"\n```\n\n### File operations\n```bash\ndb9 fs upload <database-id> <local-path> <remote-path>\ndb9 fs download <database-id> <remote-path> <local-path>\ndb9 fs list <database-id> [path]\n```\n\n### Snapshot management\n```bash\ndb9 snapshot create <database-id> --name <snapshot-name>\ndb9 snapshot list <database-id>\ndb9 snapshot restore <database-id> <snapshot-id>\n```\n\n### Generate types\n```bash\ndb9 gen types <database-id> ./types --language ts\n```\n\n## DB9 Memory Skill - Persistent Agent Memory\n\nUse these tools for persistent, searchable memory (better than memory.md):\n\n### Store a memory\n- MCP: `memory_store(content=\"...\", type=\"preference\", tags=[\"tech\"])\n- API: POST /api/v1/databases/:id/memories\n\n### Recall memories (semantic search)\n- MCP: `memory_recall(query=\"user preference\")`\n- API: POST /api/v1/databases/:id/memories/recall\n\n### List memories\n- MCP: `memory_list(agent_id=\"my-agent\")`\n- API: GET /api/v1/databases/:id/memories\n\nMemory types: fact, preference, context, decision, error, observation, plan\n\n---\n*Installed by db9 onboard (v{{.Version}}) on {{.Date}}*",
	AgentWorkBuddy: "---\nname: db9\ndescription: \"Open-DB9 Database Management & Agent Memory Layer\"\nversion: \"{{.Version}}\"\nauthor: \"Open-DB9 Team\"\nkeywords:\n  - database\n  - memory\n  - postgresql\n  - agent\n  - mcp\ncategory: \"Development Tools\"\n---\n\n# Open-DB9 Skills for WorkBuddy\n\nYou have access to the **db9 CLI tool** and **MCP Server** for database management and persistent memory storage.\n\n## Core Commands\n\n```bash\ndb9 create --name mydb          # Create database\ndb9 sql 1 \"SELECT * FROM users\" # Execute SQL\ndb9 fs upload 1 file.txt /data/  # Upload files\ndb9 snapshot create 1 --name backup\n```\n\n### Database Operations\n\n- **Create database**: `db9 create --name <database-name>`\n- **Execute SQL**: `db9 sql <database-id> \"<SQL query>\"`\n- **File operations**:\n  - Upload: `db9 fs upload <db-id> <local-path> <remote-path>`\n  - Download: `db9 fs download <db-id> <remote-path> <local-path>`\n  - List: `db9 fs list <db-id> [path]`\n- **Snapshot management**:\n  - Create: `db9 snapshot create <db-id> --name <snapshot-name>`\n  - List: `db9 snapshot list <db-id>`\n  - Restore: `db9 snapshot restore <db-id> <snapshot-id>`\n- **Branch management**:\n  - Create: `db9 branch create <db-id> --name <branch-name>`\n  - List: `db9 branch list <db-id>`\n  - Delete: `db9 branch delete <db-id> <branch-id>`\n- **Generate types**: `db9 gen types <db-id> ./types --language ts`\n\n## DB9 Memory Skill - Your Persistent Memory Layer\n\nReplace memory.md with structured, searchable memory storage. This is the **recommended approach** for persistent agent memory.\n\n### Why DB9 Memory over memory.md?\n\n- **Semantic search** (not just keyword matching)\n- **Structured data** (filter by type, tags, date)\n- **Unlimited size**, no file bloat\n- **SQL access** for analytics and reporting\n- **Shareable across sessions** and agents\n- **Type-safe** with proper categorization\n\n### Store a Memory\n\n**Using curl:**\n```bash\ncurl -X POST $API/api/v1/databases/$DB_ID/memories \\\n  -H \"Authorization: Bearer $TOKEN\" \\\n  -H \"Content-Type: application/json; charset=utf-8\" \\\n  -d '{\n    \"content\": \"User prefers TypeScript for frontend projects\",\n    \"type\": \"preference\",\n    \"tags\": [\"tech\", \"frontend\", \"typescript\"],\n    \"agent_id\": \"workbuddy\",\n    \"importance\": 0.8\n  }'\n```\n\n**Using PowerShell:**\n```powershell\n$body = @{\n    content = \"User prefers TypeScript for frontend projects\"\n    type = \"preference\"\n    tags = @(\"tech\", \"frontend\", \"typescript\")\n    agent_id = \"workbuddy\"\n    importance = 0.8\n} | ConvertTo-Json -Depth 3\n\n$bytes = [System.Text.Encoding]::UTF8.GetBytes($body)\nInvoke-RestMethod -Uri \"$API/api/v1/databases/$DB_ID/memories\" `\n    -Method Post `\n    -ContentType \"application/json; charset=utf-8\" `\n    -Body $bytes `\n    -Headers @{Authorization = \"Bearer $TOKEN\"}\n```\n\n### Semantic Recall (Search)\n\n**Using curl:**\n```bash\ncurl -X POST $API/api/v1/databases/$DB_ID/memories/recall \\\n  -H \"Authorization: Bearer $TOKEN\" \\\n  -d '{\n    \"query\": \"what does the user prefer for frontend?\",\n    \"top_k\": 5,\n    \"memory_type\": \"preference\"\n  }'\n```\n\n**Response:** Returns memories ranked by semantic similarity to your query.\n\n### List and Filter Memories\n\n```bash\n# List all memories for an agent\ncurl \"$API/api/v1/databases/$DB_ID/memories?agent_id=workbuddy\"\n\n# Filter by type\ncurl \"$API/api/v1/databases/$DB_ID/memories?agent_id=workbuddy&memory_type=fact\"\n\n# Filter by tags\ncurl \"$API/api/v1/databases/$DB_ID/memories?tags=personal,education\"\n\n# Delete a memory\ncurl -X DELETE $API/api/v1/databases/$DB_ID/memories/<memory-id>\n```\n\n## MCP Tools Reference\n\nIf MCP server is configured, use these tools directly:\n\n| Tool | Description | Parameters |\n|------|-------------|------------|\n| `memory_store` | Save a new memory | content, type, tags, agent_id, importance |\n| `memory_recall` | Search by semantic similarity | query, top_k, memory_type, tags |\n| `memory_list` | List memories with filters | agent_id, memory_type, tags, limit, offset |\n| `memory_delete` | Remove a memory | memory_id |\n\n## Memory Types\n\nUse these types to categorize memories:\n\n| Type | Description | Example Use Case |\n|------|-------------|------------------|\n| `fact` | Factual information | User's name, job title, location |\n| `preference` | User preferences | Prefers dark mode, likes TypeScript |\n| `context` | Project context | Current project goals, architecture decisions |\n| `decision` | Decisions made | Chose PostgreSQL over MongoDB |\n| `error` | Errors encountered | Fix: port 5432 already in use |\n| `observation` | Observations noticed | User seems frustrated with setup process |\n| `plan` | Plans and tasks | TODO: implement auth system next week |\n\n## ⚠️ UTF-8 Encoding for Chinese Content\n\n**CRITICAL**: When using PowerShell to store memories with Chinese characters, you MUST use UTF-8 encoding:\n\n```powershell\n# ✅ Correct way (UTF-8)\n$body = @{\n    content = \"黄译辉是北京交通大学的大二学生\"\n    type = \"fact\"\n    tags = @(\"personal\", \"education\")\n} | ConvertTo-Json -Depth 3\n\n$bytes = [System.Text.Encoding]::UTF8.GetBytes($body)  # <-- KEY STEP!\nInvoke-RestMethod -Uri \"$API/api/v1/databases/$DB_ID/memories\" `\n    -Method Post `\n    -ContentType \"application/json; charset=utf-8\" `\n    -Body $bytes `\n    -Headers @{Authorization = \"Bearer $TOKEN\"}\n```\n\n❌ **Do NOT use** the default encoding, it will cause garbled characters!\n\n## Best Practices\n\n### Tag Strategy\n- Use consistent, lowercase tags: `@(\"frontend\", \"react\", \"ui\")`\n- Create tag hierarchies: `project:name`, `feature:auth`\n- Limit to 3-5 tags per memory for optimal search\n- Reuse tags across related memories\n\n### Importance Scoring\nRate memories from 0.0 to 1.0:\n- **0.9-1.0**: Critical info (user credentials, key preferences)\n- **0.7-0.8**: Important context (project decisions, tech stack)\n- **0.5-0.6**: Useful observations (user behavior patterns)\n- **0.3-0.4**: Nice to know (casual mentions, minor details)\n\n### When to Store Memories\n✅ **DO store:**\n- User preferences that affect behavior\n- Project context and architecture decisions\n- Error solutions for future reference\n- Personal facts (with user consent)\n- Key decisions and their rationale\n\n❌ **DO NOT store:**\n- Temporary conversation details\n- Sensitive credentials or secrets\n- Information already in code/docs\n- Redundant data (update existing instead)\n\n### Security Notes\n- Never store passwords, API keys, or tokens in memories\n- Respect user privacy - only store what they explicitly share\n- Use appropriate memory types for sensitive info\n- Consider data retention policies\n\n---\n\n*Installed by db9 onboard (v{{.Version}}) on {{.Date}}*",
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
  - workbuddy: WorkBuddy / OpenClaw (Agent Memory Layer)

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
		"Agent name to install (claude, cursor, cline, codex, opencode, workbuddy)")
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
