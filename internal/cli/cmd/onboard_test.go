package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRenderTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     TemplateData
		expected string
		wantErr  bool
	}{
		{
			name:     "basic template with version",
			template: "Version: {{.Version}}",
			data:     TemplateData{Version: "1.0.0"},
			expected: "Version: 1.0.0",
			wantErr:  false,
		},
		{
			name:     "template with date",
			template: "Date: {{.Date}}",
			data:     TemplateData{Date: "2026-01-01"},
			expected: "Date: 2026-01-01",
			wantErr:  false,
		},
		{
			name:     "template with all fields",
			template: "V: {{.Version}}, D: {{.Date}}, T: {{.Timestamp}}",
			data: TemplateData{
				Version:   "2.0.0",
				Date:      "2026-03-31",
				Timestamp: "2026-03-31T12:00:00Z",
			},
			expected: "V: 2.0.0, D: 2026-03-31, T: 2026-03-31T12:00:00Z",
			wantErr:  false,
		},
		{
			name:     "empty template",
			template: "",
			data:     TemplateData{},
			expected: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := renderTemplate(tt.template, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("renderTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("renderTemplate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAgentTemplatesExist(t *testing.T) {
	expectedAgents := []string{AgentClaude, AgentCursor, AgentCline, AgentCodex, AgentOpenCode}

	for _, agent := range expectedAgents {
		t.Run(agent+"_template_exists", func(t *testing.T) {
			tmpl, ok := agentTemplates[agent]
			if !ok {
				t.Errorf("No template found for agent %s", agent)
				return
			}
			if tmpl == "" {
				t.Errorf("Empty template for agent %s", agent)
			}

			// Test that template can be rendered
			data := TemplateData{
				Version:   "test-version",
				Date:      "2026-01-01",
				Timestamp: time.Now().Format(time.RFC3339),
			}
			result, err := renderTemplate(tmpl, data)
			if err != nil {
				t.Errorf("Failed to render template for %s: %v", agent, err)
				return
			}
			if !strings.Contains(result, "test-version") {
				t.Errorf("Template for %s does not contain version after rendering", agent)
			}
		})
	}
}

func TestSupportedAgentsConfig(t *testing.T) {
	// Verify all supported agents have proper configuration
	for name, agent := range supportedAgents {
		t.Run(name+"_config", func(t *testing.T) {
			if name != agent.Name {
				t.Errorf("Agent map key %s doesn't match Name field %s", name, agent.Name)
			}
			if agent.DisplayName == "" {
				t.Errorf("Agent %s has empty DisplayName", name)
			}
			if agent.Description == "" {
				t.Errorf("Agent %s has empty Description", name)
			}

			// At least one path should be set (user or project)
			if agent.Paths.User == "" && agent.Paths.Project == "" {
				t.Errorf("Agent %s has no installation paths configured", name)
			}
		})
	}
}

func TestInstallToAgent_InvalidAgent(t *testing.T) {
	err := installToAgent("invalid_agent", ScopeUser)
	if err == nil {
		t.Error("Expected error for invalid agent, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported agent") {
		t.Errorf("Error should mention 'unsupported agent', got: %v", err)
	}
}

func TestInstallToAgent_InvalidScope(t *testing.T) {
	err := installToAgent(AgentClaude, "invalid_scope")
	if err == nil {
		t.Error("Expected error for invalid scope, got nil")
	}
	if !strings.Contains(err.Error(), "invalid scope") {
		t.Errorf("Error should mention 'invalid scope', got: %v", err)
	}
}

func TestInstallAndUninstall_ProjectScope(t *testing.T) {
	// Create temp directory to simulate project root
	tmpDir, err := os.MkdirTemp("", "db9-onboard-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Install for Claude at project scope
	err = installToAgent(AgentClaude, ScopeProject)
	if err != nil {
		t.Fatalf("Installation failed: %v", err)
	}

	// Verify file exists
	expectedPath := filepath.Join(tmpDir, ".claude", "commands", "db9.md")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected file not created at %s", expectedPath)
	}

	// Verify file content contains version
	content, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read installed file: %v", err)
	}
	if !strings.Contains(string(content), version) {
		t.Errorf("Installed file doesn't contain version info")
	}

	// Uninstall
	err = uninstallFromAgent(AgentClaude)
	if err != nil {
		t.Fatalf("Uninstallation failed: %v", err)
	}

	// Verify file was removed
	if _, err := os.Stat(expectedPath); !os.IsNotExist(err) {
		t.Errorf("File should have been removed after uninstall")
	}
}

func TestInstall_CursorProjectOnly(t *testing.T) {
	// Cursor only supports project scope
	tmpDir, err := os.MkdirTemp("", "db9-cursor-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// This should work (project scope)
	err = installToAgent(AgentCursor, ScopeProject)
	if err != nil {
		t.Fatalf("Cursor project scope installation failed: %v", err)
	}

	// Verify file exists
	expectedPath := filepath.Join(tmpDir, ".cursor", "rules", "db9.md")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Cursor rules file not created at %s", expectedPath)
	}

	// Clean up
	os.RemoveAll(filepath.Join(tmpDir, ".cursor"))
}

func TestGetSupportedAgentNames(t *testing.T) {
	names := getSupportedAgentNames()

	// Should contain all expected agents
	expectedAgents := []string{AgentClaude, AgentCursor, AgentCline, AgentCodex, AgentOpenCode}
	for _, agent := range expectedAgents {
		if !strings.Contains(names, agent) {
			t.Errorf("Expected agent %s in supported agents list", agent)
		}
	}

	// Should be comma-separated
	parts := strings.Split(names, ", ")
	if len(parts) != len(expectedAgents) {
		t.Errorf("Expected %d agents in list, got %d", len(expectedAgents), len(parts))
	}
}

func TestCleanupEmptyDirs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "db9-cleanup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create nested directory structure
	nestedDir := filepath.Join(tmpDir, "a", "b", "c")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("Failed to create nested dirs: %v", err)
	}

	// Create a file in the deepest directory
	testFile := filepath.Join(nestedDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Cleanup should not remove directories with files
	cleanupEmptyDirs(nestedDir)
	if _, err := os.Stat(nestedDir); os.IsNotExist(err) {
		t.Error("Directory with file should not be removed")
	}

	// Remove the file and cleanup
	os.Remove(testFile)
	cleanupEmptyDirs(nestedDir)

	// All empty directories should be removed
	if _, err := os.Stat(filepath.Join(tmpDir, "a")); !os.IsNotExist(err) {
		t.Error("Empty parent directories should be cleaned up")
	}
}

func TestIsDirEmpty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "db9-empty-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Empty directory should return true
	isEmpty, err := isDirEmpty(tmpDir)
	if err != nil {
		t.Fatalf("isDirEmpty failed: %v", err)
	}
	if !isEmpty {
		t.Error("Newly created directory should be empty")
	}

	// Add a file and check again
	testFile := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	isEmpty, err = isDirEmpty(tmpDir)
	if err != nil {
		t.Fatalf("isDirEmpty failed: %v", err)
	}
	if isEmpty {
		t.Error("Directory with file should not be empty")
	}
}

func TestListInstalledAgents(t *testing.T) {
	// Create temp directory for testing
	tmpDir, err := os.MkdirTemp("", "db9-list-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// List without any installations - should not error
	err = listInstalledAgents()
	if err != nil {
		t.Fatalf("listInstalledAgents failed: %v", err)
	}

	// Install for one agent and list again
	err = installToAgent(AgentCline, ScopeProject)
	if err != nil {
		t.Fatalf("Installation failed: %v", err)
	}

	// List again - should show cline as installed
	err = listInstalledAgents()
	if err != nil {
		t.Fatalf("listInstalledAgents failed after installation: %v", err)
	}
}

func TestOnboardCommandFlags(t *testing.T) {
	// Test that command has required flags
	flags := OnboardCmd.Flags()

	// Check --agent flag
	agentFlag := flags.Lookup("agent")
	if agentFlag == nil {
		t.Error("--agent flag not found")
	}
	if agentFlag.Shorthand != "a" {
		t.Errorf("--agent shorthand should be 'a', got '%s'", agentFlag.Shorthand)
	}

	// Check --scope flag
	scopeFlag := flags.Lookup("scope")
	if scopeFlag == nil {
		t.Error("--scope flag not found")
	}
	if scopeFlag.Shorthand != "s" {
		t.Errorf("--scope shorthand should be 's', got '%s'", scopeFlag.Shorthand)
	}

	// Check --list flag
	listFlag := flags.Lookup("list")
	if listFlag == nil {
		t.Error("--list flag not found")
	}
	if listFlag.Shorthand != "l" {
		t.Errorf("--list shorthand should be 'l', got '%s'", listFlag.Shorthand)
	}

	// Check --uninstall flag
	uninstallFlag := flags.Lookup("uninstall")
	if uninstallFlag == nil {
		t.Error("--uninstall flag not found")
	}
	if uninstallFlag.Shorthand != "u" {
		t.Errorf("--uninstall shorthand should be 'u', got '%s'", uninstallFlag.Shorthand)
	}
}

func TestInstallBothScopes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "db9-both-scope-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Get home directory for user scope verification
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home dir: %v", err)
	}

	// Install for Cline with both scopes
	err = installToAgent(AgentCline, ScopeBoth)
	if err != nil {
		t.Fatalf("Both scope installation failed: %v", err)
	}

	// Verify both locations exist
	projectPath := filepath.Join(tmpDir, ".cline", "rules", "db9.md")
	userPath := filepath.Join(homeDir, ".cline", "rules", "db9.md")

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Errorf("Project scope file not created at %s", projectPath)
	}

	if _, err := os.Stat(userPath); os.IsNotExist(err) {
		t.Errorf("User scope file not created at %s", userPath)
	}

	// Cleanup user scope installation
	uninstallFromAgent(AgentCline)
}

func BenchmarkRenderTemplate(b *testing.B) {
	data := TemplateData{
		Version:   "1.0.0",
		Date:      "2026-01-01",
		Timestamp: time.Now().Format(time.RFC3339),
	}
	tmpl := agentTemplates[AgentClaude]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := renderTemplate(tmpl, data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func ExampleOnboardCmd() {
	fmt.Println("Usage:")
	fmt.Println("  db9 onboard --agent claude              # Install for Claude Code")
	fmt.Println("  db9 onboard --agent cursor --scope project  # Install for Cursor")
	fmt.Println("  db9 onboard --list                      # List all agents")
	fmt.Println("  db9 onboard --uninstall claude          # Uninstall from Claude")
	// Output:
	// Usage:
	//   db9 onboard --agent claude              # Install for Claude Code
	//   db9 onboard --agent cursor --scope project  # Install for Cursor
	//   db9 onboard --list                      # List all agents
	//   db9 onboard --uninstall claude          # Uninstall from Claude
}
