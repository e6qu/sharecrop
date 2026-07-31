package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/notification"
)

// notificationTestHandler builds the real mux over a live in-memory
// notification service seeded with one read and one unread row for the
// authenticated test user.
func notificationTestHandler(t *testing.T) http.Handler {
	t.Helper()
	service := notification.NewService(notification.NewMemoryStore())
	actor := core.NewUserID().(core.UserIDCreated).Value

	first, matched := service.Notify(context.Background(), stableTestUserID, actor, notification.KindSubmissionCreated, notification.Subject{Kind: "submission", ID: "submission-1"}, notification.EmptyMetadata()).(notification.NotificationCreated)
	if !matched {
		t.Fatalf("seed first notification rejected")
	}
	if _, matched := service.Notify(context.Background(), stableTestUserID, actor, notification.KindTaskFunded, notification.Subject{Kind: "task", ID: "task-1"}, notification.EmptyMetadata()).(notification.NotificationCreated); !matched {
		t.Fatalf("seed second notification rejected")
	}
	if _, matched := service.MarkRead(context.Background(), stableTestUserID, first.Value.ID).(notification.NotificationRead); !matched {
		t.Fatalf("seed mark-read rejected")
	}

	runtime := DefaultRuntimeState(map[string]bool{})
	runtime.NotificationService = service
	return NewWithRuntimeState(testStaticFiles(), testAuthService(), testVerifier{}, testOrganizationService{}, testTaskService{}, testSubmissionService{}, testLedgerService{}, testAgentService{}, testOrgCredentialService{}, testAssetService{}, runtime)
}

func TestListNotificationsUnreadFilter(t *testing.T) {
	handler := notificationTestHandler(t)

	request := httptest.NewRequest(http.MethodGet, "/api/notifications?state=unread", nil)
	request.Header.Set("Authorization", "Bearer test-access-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (%s)", response.Code, http.StatusOK, response.Body.String())
	}
	var body struct {
		Notifications []struct {
			Kind  string `json:"kind"`
			State string `json:"state"`
		} `json:"notifications"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Notifications) != 1 {
		t.Fatalf("unread listing returned %d rows, want 1", len(body.Notifications))
	}
	if body.Notifications[0].State != "unread" || body.Notifications[0].Kind != "task_funded" {
		t.Fatalf("unread row = %+v", body.Notifications[0])
	}
}

func TestListNotificationsRejectsUnknownStateFilter(t *testing.T) {
	handler := notificationTestHandler(t)

	request := httptest.NewRequest(http.MethodGet, "/api/notifications?state=archived", nil)
	request.Header.Set("Authorization", "Bearer test-access-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestUnreadNotificationCountEndpoint(t *testing.T) {
	handler := notificationTestHandler(t)

	request := httptest.NewRequest(http.MethodGet, "/api/notifications/unread-count", nil)
	request.Header.Set("Authorization", "Bearer test-access-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (%s)", response.Code, http.StatusOK, response.Body.String())
	}
	if response.Body.String() != "{\"unread_count\":1}\n" {
		t.Fatalf("body = %q, want {\"unread_count\":1}", response.Body.String())
	}
}

func TestUnreadNotificationCountRequiresAuth(t *testing.T) {
	handler := notificationTestHandler(t)

	request := httptest.NewRequest(http.MethodGet, "/api/notifications/unread-count", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}
