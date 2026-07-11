//go:build !wasip1

// Package storehost is the host-side counterpart to the store guest: it routes a
// guest's store call, by method prefix, to the matching generated bridge over
// the real internal/db stores. The app host and the integration tests share it
// so the set of bridged stores is defined in one place.
package storehost

import (
	"context"
	"fmt"
	"strings"

	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/wasibridge/agentbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/assetsbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/auditbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/authbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/notificationbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/orgcredbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Dispatcher builds an rpc.Dispatcher backed by the real db stores on pool.
func Dispatcher(pool *pgxpool.Pool) rpc.Dispatcher {
	agentStore := db.NewAgentStore(pool)
	assetsStore := db.NewCollectibleStore(pool)
	auditStore := db.NewAuditStore(pool)
	authStore := db.NewAuthStore(pool)
	notificationStore := db.NewNotificationStore(pool)
	orgcredStore := db.NewOrgCredentialStore(pool)

	return func(ctx context.Context, method string, args []byte) ([]byte, error) {
		store, _, _ := strings.Cut(method, ".")
		switch store {
		case "agent":
			return agentbridge.Dispatch(ctx, agentStore, method, args)
		case "assets":
			return assetsbridge.Dispatch(ctx, assetsStore, method, args)
		case "audit":
			return auditbridge.Dispatch(ctx, auditStore, method, args)
		case "auth":
			return authbridge.Dispatch(ctx, authStore, method, args)
		case "notification":
			return notificationbridge.Dispatch(ctx, notificationStore, method, args)
		case "orgcred":
			return orgcredbridge.Dispatch(ctx, orgcredStore, method, args)
		default:
			return nil, fmt.Errorf("no bridge for method %q", method)
		}
	}
}
