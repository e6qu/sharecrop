//go:build integration

package integration_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/e6qu/sharecrop/internal/wasibridge/httpbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
	"github.com/e6qu/sharecrop/internal/wasibridge/storehost"
)

// TestGuestCarriesLargeAttachmentFrames covers payloads that cross the WASI
// bridge as a single length-prefixed frame. An HTTP request and its response are
// each one frame, with the body base64-encoded inside (~4/3 inflation). The app
// accepts request bodies up to 2 MiB with attachments up to 500 KiB each, but
// the frame limit was 1 MiB - below the request limit - so a valid task with two
// large attachments (which the native mux accepts) failed under the guest with a
// 502, in both directions: the request frame to create it and the response frame
// carrying the attachments back.
//
// The check creates a task with two ~480 KiB attachments through the guest and
// requires it to succeed and round-trip both attachments, exercising an
// over-1-MiB request frame and an over-1-MiB response frame.
func TestGuestCarriesLargeAttachmentFrames(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t)

	guestWASM, err := compileWASIGuest(t, "github.com/e6qu/sharecrop/cmd/sharecrop-wasi-app-guest")
	if err != nil {
		t.Fatalf("compile app guest: %v", err)
	}
	guestPool, err := rpc.NewPool(ctx, guestWASM, storehost.Dispatcher(pool), 2)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	guestPool.WithGuestEnv(map[string]string{"SHARECROP_ACCESS_TOKEN_SECRET": appRouteSecret})
	t.Cleanup(func() { _ = guestPool.Close(ctx) })
	guest := httpbridge.Handler(guestPool)

	do := func(method, path, authorization, body string) *http.Response {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if authorization != "" {
			req.Header.Set("Authorization", "Bearer "+authorization)
		}
		rec := httptest.NewRecorder()
		guest.ServeHTTP(rec, req)
		return rec.Result()
	}

	registerResp := do("POST", "/api/auth/register", "",
		fmt.Sprintf(`{"email":%q,"password":"correct horse battery staple"}`, uniqueIntegrationEmail(t, "large-attach")))
	if registerResp.StatusCode != http.StatusCreated {
		t.Fatalf("register: status %d, want 201", registerResp.StatusCode)
	}
	var registered struct {
		SubjectID   string `json:"subject_id"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(registerResp.Body).Decode(&registered); err != nil {
		t.Fatalf("decode register response: %v", err)
	}

	// Two attachments just under the 500 KiB cap: together their base64 data URLs
	// push the request and response bodies past 1 MiB.
	dataURL := "data:text/plain;base64," + base64.StdEncoding.EncodeToString(make([]byte, 480*1024))
	body := fmt.Sprintf(`{
		"owner": {"kind": "user", "user_id": %q},
		"title": "large attachment frame test",
		"description": "two large attachments",
		"visibility": {"kind": "public"},
		"participation": {"policy": "open", "assignee_scope": "user", "reservation_expiry_hours": 48},
		"reward": {"kind": "none", "credit_amount": 0, "collectible_ids": []},
		"response_schema_json": "{\"kind\":\"freeform\"}",
		"payload": {"kind": "none", "json": ""},
		"placement": {"kind": "standalone"},
		"attachments": [
			{"name": "a.txt", "content_type": "text/plain", "data_url": %q},
			{"name": "b.txt", "content_type": "text/plain", "data_url": %q}
		]
	}`, registered.SubjectID, dataURL, dataURL)

	createResp := do("POST", "/api/tasks", registered.AccessToken, body)
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("create task with two large attachments: status %d, want 201 - a >1 MiB frame did not cross the bridge", createResp.StatusCode)
	}
	var created struct {
		Attachments []struct {
			DataURL string `json:"data_url"`
		} `json:"attachments"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if len(created.Attachments) != 2 {
		t.Fatalf("created task carried %d attachments back, want 2", len(created.Attachments))
	}
	for i, attachment := range created.Attachments {
		if attachment.DataURL != dataURL {
			t.Errorf("attachment %d data URL did not round-trip across the bridge", i)
		}
	}
}
