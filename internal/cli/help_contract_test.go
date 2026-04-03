package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestTypedAPILeafCommandsProvideLongHelp(t *testing.T) {
	root := NewRootCommand("dev")

	for _, path := range [][]string{
		{"ticket"},
		{"status"},
		{"chat"},
		{"project"},
		{"repo"},
		{"workflow"},
		{"scheduled-job"},
		{"machine"},
		{"provider"},
		{"agent"},
		{"activity"},
		{"channel"},
		{"notification-rule"},
		{"skill"},
		{"watch"},
		{"stream"},
	} {
		command, _, err := root.Find(path)
		if err != nil {
			t.Fatalf("Find(%v) returned error: %v", path, err)
		}
		if command == nil {
			t.Fatalf("expected command %v", path)
		}

		walkCLILeaves(command, path, func(path []string, command *cobra.Command) {
			if strings.TrimSpace(command.Long) == "" {
				t.Fatalf("command %s is missing Long help", strings.Join(path, " "))
			}

			if path[0] == "watch" || path[0] == "stream" {
				for _, want := range []string{
					"keeps the connection open",
					"Use Ctrl-C to stop the stream",
				} {
					if !strings.Contains(command.Long, want) {
						t.Fatalf("command %s Long help missing %q: %q", strings.Join(path, " "), want, command.Long)
					}
				}
			}

			for _, positional := range bracketedUseParams(command.Use) {
				if strings.HasSuffix(strings.ToLower(positional), "id") && !strings.Contains(command.Long, "UUID") {
					t.Fatalf("command %s should document UUID semantics for %s: %q", strings.Join(path, " "), positional, command.Long)
				}
			}
		})
	}
}

func TestRootLeafCommandsProvideLongHelp(t *testing.T) {
	root := NewRootCommand("dev")
	walkCLILeaves(root, []string{root.Name()}, func(path []string, command *cobra.Command) {
		if strings.TrimSpace(command.Long) == "" {
			t.Fatalf("command %s is missing Long help", strings.Join(path, " "))
		}
	})
}

func TestPlatformLeafCommandsProvideStructuredHelp(t *testing.T) {
	tests := []struct {
		name          string
		root          *cobra.Command
		expectProject bool
	}{
		{
			name:          "ticket",
			root:          newAgentPlatformTicketCommandWithDeps(platformCommandDeps{}),
			expectProject: true,
		},
		{
			name: "project",
			root: newAgentPlatformProjectCommandWithDeps(platformCommandDeps{}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			walkCLILeaves(tt.root, []string{tt.root.Name()}, func(path []string, command *cobra.Command) {
				if strings.TrimSpace(command.Long) == "" {
					t.Fatalf("command %s is missing Long help", strings.Join(path, " "))
				}
				for _, want := range []string{"OPENASE_API_URL", "OPENASE_AGENT_TOKEN"} {
					if !strings.Contains(command.Long, want) {
						t.Fatalf("command %s Long help missing %q: %q", strings.Join(path, " "), want, command.Long)
					}
				}

				if tt.expectProject && usesProjectContext(path, command) && !strings.Contains(command.Long, "OPENASE_PROJECT_ID") {
					t.Fatalf("command %s should document OPENASE_PROJECT_ID fallback: %q", strings.Join(path, " "), command.Long)
				}
				if usesTicketContext(path, command) && !strings.Contains(command.Long, "OPENASE_TICKET_ID") {
					t.Fatalf("command %s should document OPENASE_TICKET_ID fallback: %q", strings.Join(path, " "), command.Long)
				}
			})
		})
	}
}

func TestCriticalCLICommandsProvideExamples(t *testing.T) {
	root := NewRootCommand("dev")
	for _, path := range [][]string{
		{"api"},
		{"watch", "project"},
		{"stream", "events"},
		{"ticket", "comment", "update"},
		{"machine", "refresh-health"},
		{"provider", "get"},
	} {
		command, _, err := root.Find(path)
		if err != nil {
			t.Fatalf("Find(%v) returned error: %v", path, err)
		}
		if command == nil {
			t.Fatalf("expected command %v", path)
		}
		if strings.TrimSpace(command.Example) == "" {
			t.Fatalf("command %v is missing Example help", path)
		}
	}

	for _, command := range []*cobra.Command{
		newAgentPlatformTicketCommandWithDeps(platformCommandDeps{}),
		newAgentPlatformProjectCommandWithDeps(platformCommandDeps{}),
	} {
		walkCLILeaves(command, []string{command.Name()}, func(path []string, command *cobra.Command) {
			if isHighRiskPlatformCommand(path) && strings.TrimSpace(command.Example) == "" {
				t.Fatalf("command %s is missing Example help", strings.Join(path, " "))
			}
		})
	}
}

func walkCLILeaves(command *cobra.Command, path []string, visit func([]string, *cobra.Command)) {
	children := visibleCLIChildren(command)
	if len(children) == 0 {
		visit(path, command)
		return
	}
	for _, child := range children {
		walkCLILeaves(child, append(path, child.Name()), visit)
	}
}

func visibleCLIChildren(command *cobra.Command) []*cobra.Command {
	children := make([]*cobra.Command, 0, len(command.Commands()))
	for _, child := range command.Commands() {
		if child.Hidden || child.Name() == "help" {
			continue
		}
		children = append(children, child)
	}
	return children
}

func bracketedUseParams(use string) []string {
	fields := strings.Fields(use)
	params := make([]string, 0, len(fields))
	for _, field := range fields {
		if strings.HasPrefix(field, "[") && strings.HasSuffix(field, "]") {
			params = append(params, strings.TrimSuffix(strings.TrimPrefix(field, "["), "]"))
		}
	}
	return params
}

func usesProjectContext(path []string, command *cobra.Command) bool {
	if len(path) == 0 {
		return false
	}
	if path[0] == "project" {
		return true
	}
	if len(path) >= 2 && path[0] == "ticket" {
		return path[1] == "list" || path[1] == "create"
	}
	for _, positional := range bracketedUseParams(command.Use) {
		if strings.EqualFold(positional, "project-id") || strings.EqualFold(positional, "projectId") {
			return true
		}
	}
	return false
}

func usesTicketContext(path []string, command *cobra.Command) bool {
	if len(path) == 0 {
		return false
	}
	if len(path) >= 2 && path[0] == "ticket" {
		if path[1] == "update" || path[1] == "report-usage" {
			return true
		}
		if path[1] == "comment" {
			return true
		}
	}
	for _, positional := range bracketedUseParams(command.Use) {
		if strings.EqualFold(positional, "ticket-id") || strings.EqualFold(positional, "ticketId") {
			return true
		}
	}
	return false
}

func isHighRiskPlatformCommand(path []string) bool {
	joined := strings.Join(path, " ")
	for _, item := range []string{
		"ticket update",
		"ticket report-usage",
		"project add-repo",
	} {
		if strings.Contains(joined, item) {
			return true
		}
	}
	return false
}
