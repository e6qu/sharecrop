package wasmseed

import (
	"github.com/e6qu/sharecrop/internal/db"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/internal/wasibridge/appmux"
)

// demoRegistrationRateCapacity is the browser-demo registration budget (see
// the RegistrationRateLimiter comment below).
const demoRegistrationRateCapacity = 10000

// StoresFromHandle builds the full app store set over one database handle. The
// browser demo uses it to run every store over its SQLite handle, the same set
// appmux.New serves requests over.
func StoresFromHandle(handle db.Beginner) appmux.Stores {
	return appmux.Stores{
		Auth:               db.NewAuthStoreFromHandle(handle),
		Event:              db.NewEventStoreFromHandle(handle),
		Webhook:            db.NewWebhookStoreFromHandle(handle),
		Notification:       db.NewNotificationStoreFromHandle(handle),
		Organization:       db.NewOrgStoreFromHandle(handle),
		Task:               db.NewTaskStoreFromHandle(handle),
		Submission:         db.NewSubmissionStoreFromHandle(handle),
		Ledger:             db.NewLedgerStoreFromHandle(handle),
		Agent:              db.NewAgentStoreFromHandle(handle),
		OrgCredential:      db.NewOrgCredentialStoreFromHandle(handle),
		Assets:             db.NewCollectibleStoreFromHandle(handle),
		Audit:              db.NewAuditStoreFromHandle(handle),
		SavedQueueViews:    db.NewSavedQueueViewStoreFromHandle(handle),
		PlatformAdmins:     db.NewPlatformAdminStoreFromHandle(handle, map[string]bool{}),
		ModerationTriage:   db.NewModerationTriageStoreFromHandle(handle),
		Privacy:            db.NewPrivacyStoreFromHandle(handle),
		IPRateLimiter:      db.NewRateLimiterFromHandle(handle, "ip", httpserver.IPRateCapacity, httpserver.IPRateRefillPerSec),
		SubjectRateLimiter: db.NewRateLimiterFromHandle(handle, "subject", httpserver.MCPRateCapacity, httpserver.MCPRateRefillPerSec),
		// The browser demo is a single-user local sandbox: every account lives
		// in this browser's own storage, so production's tight registration
		// budget would only block the demo's scenario actors. The demo keeps
		// the limiter mechanism but with a budget the scenario cannot exhaust.
		RegistrationRateLimiter: db.NewRateLimiterFromHandle(handle, "register", demoRegistrationRateCapacity, httpserver.RegistrationRateRefillPerSec),
		MCPSessions:             db.NewMCPSessionStoreFromHandle(handle),
	}
}
