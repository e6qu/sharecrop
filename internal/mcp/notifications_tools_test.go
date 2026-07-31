package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/e6qu/sharecrop/internal/agent"
)

func TestToolsCallListNotificationsThreadsStateFilter(t *testing.T) {
	server := NewServer(fakeServices{})

	unfiltered := decodeToolText(t, server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: agent.NewScopeSet([]agent.Scope{agent.ScopeNotificationsRead})}, request(`1`, "tools/call", `{"name":"sharecrop.list_notifications","arguments":{}}`)))
	if !strings.Contains(unfiltered, `"subject_kind":"any_state"`) {
		t.Fatalf("missing state argument did not thread the anyState filter: %s", unfiltered)
	}

	unread := decodeToolText(t, server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: agent.NewScopeSet([]agent.Scope{agent.ScopeNotificationsRead})}, request(`2`, "tools/call", `{"name":"sharecrop.list_notifications","arguments":{"state":"unread"}}`)))
	if !strings.Contains(unread, `"subject_kind":"unread_only"`) {
		t.Fatalf("state=unread did not thread the unread filter: %s", unread)
	}

	invalid := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: agent.NewScopeSet([]agent.Scope{agent.ScopeNotificationsRead})}, request(`3`, "tools/call", `{"name":"sharecrop.list_notifications","arguments":{"state":"archived"}}`))
	if invalid.Error == nil || invalid.Error.Code != codeInvalidParams {
		t.Fatalf("unknown state filter did not raise invalid params: %+v", invalid.Error)
	}
}

func TestToolsCallGetUnreadNotificationCount(t *testing.T) {
	server := NewServer(fakeServices{})
	content := decodeToolText(t, server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: agent.NewScopeSet([]agent.Scope{agent.ScopeNotificationsRead})}, request(`1`, "tools/call", `{"name":"sharecrop.get_unread_notification_count","arguments":{}}`)))
	if !strings.Contains(content, `"unread_count":1`) {
		t.Fatalf("count content missing unread_count: %s", content)
	}
}

func TestToolsCallGetUnreadNotificationCountRequiresNotificationsReadScope(t *testing.T) {
	server := NewServer(fakeServices{})
	response := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: agent.NewScopeSet([]agent.Scope{agent.ScopeTasksRead})}, request(`1`, "tools/call", `{"name":"sharecrop.get_unread_notification_count","arguments":{}}`))
	if response.Error == nil {
		t.Fatalf("missing scope did not raise an error")
	}
}
