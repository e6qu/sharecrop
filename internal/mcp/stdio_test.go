package mcp

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/e6qu/sharecrop/internal/agent"
)

func TestServeStdioHandlesRequestsAndSkipsNotifications(t *testing.T) {
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"ping"}`,
	}, "\n")
	var out bytes.Buffer

	server := NewServer(fakeServices{})
	if err := ServeStdio(context.Background(), server, testSubject(t), agent.Credential{Scopes: allScopes()}, strings.NewReader(input), &out); err != nil {
		t.Fatalf("serve stdio: %v", err)
	}

	lines := splitNonEmptyLines(out.String())
	if len(lines) != 2 {
		t.Fatalf("response line count = %d, want 2 (notification skipped)", len(lines))
	}
	if !strings.Contains(lines[0], "sharecrop.list_tasks") {
		t.Fatalf("first response missing tools: %s", lines[0])
	}
}

func splitNonEmptyLines(value string) []string {
	lines := make([]string, 0)
	for _, line := range strings.Split(value, "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
