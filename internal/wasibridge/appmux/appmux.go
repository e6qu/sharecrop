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

// New builds the app mux over the given access-token secret and the auth and
// notification stores. The stores are interfaces, so the guest passes the
// bridge-backed GuestStores and a test passes the real internal/db stores - the
// mux is identical either way. The auth service is live (backed by the auth
// store) so store-touching auth routes like GET /api/users work; token
// verification stays stateless via the same secret.
func New(secret auth.AccessTokenSecret, authStore auth.Store, notificationStore notification.Store) http.Handler {
	verifier := auth.NewAccessTokenVerifier(secret, auth.SystemClock{})

	authService, _ := auth.NewService(authStore, secret, auth.SystemClock{}).(auth.ServiceCreated)

	runtime := httpserver.DefaultRuntimeState(map[string]bool{})
	runtime.NotificationService = notification.NewService(notificationStore)

	return httpserver.NewWithRuntimeState(fstest.MapFS{}, authService.Value, verifier, nil, nil, nil, nil, nil, nil, nil, runtime)
}
