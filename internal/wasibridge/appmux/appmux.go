// Package appmux assembles the real internal/http routing table for the WASI
// app guest around a live notification service, so the guest and the tests that
// check it against the native server build the exact same mux. Only the pieces
// the currently-bridged routes need are wired (a stateless access-token verifier
// and the notification service); the remaining services are nil, matching the
// Phase 4 slice.
package appmux

import (
	"net/http"
	"testing/fstest"

	"github.com/e6qu/sharecrop/internal/auth"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/internal/notification"
)

// New builds the app mux over the given access-token secret and notification
// store. The store is an interface, so the guest passes the bridge-backed
// GuestStore and a test passes the real internal/db store - the mux is
// identical either way.
func New(secret auth.AccessTokenSecret, notificationStore notification.Store) http.Handler {
	verifier := auth.NewAccessTokenVerifier(secret, auth.SystemClock{})

	runtime := httpserver.DefaultRuntimeState(map[string]bool{})
	runtime.NotificationService = notification.NewService(notificationStore)

	return httpserver.NewWithRuntimeState(fstest.MapFS{}, nil, verifier, nil, nil, nil, nil, nil, nil, nil, runtime)
}
