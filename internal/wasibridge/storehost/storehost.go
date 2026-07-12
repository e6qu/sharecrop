//go:build !wasip1

// Package storehost is the host-side counterpart to the store guest: it routes a
// guest's store call, by method prefix, to the matching generated bridge over
// the real internal/db stores. The app host and the integration tests share it
// so the set of bridged stores is defined in one place.
package storehost

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/internal/wasibridge/agentbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/assetsbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/auditbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/authbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/ledgerbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/moderationtriagebridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/notificationbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/orgbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/orgcredbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/platformadminbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/privacybridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/ratelimitbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
	"github.com/e6qu/sharecrop/internal/wasibridge/savedqueueviewbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/submissionbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/taskbridge"

	"github.com/jackc/pgx/v5/pgxpool"
)

// bootstrapAdmins parses SHARECROP_ADMIN_USER_IDS (comma-separated user ids) the
// same way the native server does, so the host-side platform-admin store seeds
// the same bootstrap admins. Malformed ids are skipped.
func bootstrapAdmins() map[string]bool {
	admins := map[string]bool{}
	for _, raw := range strings.Split(os.Getenv("SHARECROP_ADMIN_USER_IDS"), ",") {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		if _, matched := core.ParseUserID(trimmed).(core.UserIDCreated); matched {
			admins[trimmed] = true
		}
	}
	return admins
}

// Dispatcher builds an rpc.Dispatcher backed by the real db stores on pool.
func Dispatcher(pool *pgxpool.Pool) rpc.Dispatcher {
	agentStore := db.NewAgentStore(pool)
	assetsStore := db.NewCollectibleStore(pool)
	auditStore := db.NewAuditStore(pool)
	authStore := db.NewAuthStore(pool)
	ledgerStore := db.NewLedgerStore(pool)
	notificationStore := db.NewNotificationStore(pool)
	orgStore := db.NewOrgStore(pool)
	orgcredStore := db.NewOrgCredentialStore(pool)
	submissionStore := db.NewSubmissionStore(pool)
	taskStore := db.NewTaskStore(pool)
	savedQueueViewStore := db.NewSavedQueueViewStore(pool)
	platformAdminStore := db.NewPlatformAdminStore(pool, bootstrapAdmins())
	moderationTriageStore := db.NewModerationTriageStore(pool)
	privacyStore := db.NewPrivacyStore(pool)
	ipRateLimiter := db.NewRateLimiter(pool, "ip", httpserver.IPRateCapacity, httpserver.IPRateRefillPerSec)
	subjectRateLimiter := db.NewRateLimiter(pool, "subject", httpserver.MCPRateCapacity, httpserver.MCPRateRefillPerSec)

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
		case "ledger":
			return ledgerbridge.Dispatch(ctx, ledgerStore, method, args)
		case "notification":
			return notificationbridge.Dispatch(ctx, notificationStore, method, args)
		case "org":
			return orgbridge.Dispatch(ctx, orgStore, method, args)
		case "orgcred":
			return orgcredbridge.Dispatch(ctx, orgcredStore, method, args)
		case "submission":
			return submissionbridge.Dispatch(ctx, submissionStore, method, args)
		case "task":
			return taskbridge.Dispatch(ctx, taskStore, method, args)
		case "savedqueueview":
			return savedqueueviewbridge.Dispatch(ctx, savedQueueViewStore, method, args)
		case "platformadmin":
			return platformadminbridge.Dispatch(ctx, platformAdminStore, method, args)
		case "moderationtriage":
			return moderationtriagebridge.Dispatch(ctx, moderationTriageStore, method, args)
		case "privacy":
			return privacybridge.Dispatch(ctx, privacyStore, method, args)
		case "ratelimit":
			return ratelimitbridge.Dispatch(ctx, ipRateLimiter, subjectRateLimiter, method, args)
		default:
			return nil, fmt.Errorf("no bridge for method %q", method)
		}
	}
}
