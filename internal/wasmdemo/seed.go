package wasmdemo

import (
	"context"
	"net/http"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/task"
)

// demoUserSeed is one of the fixed browser-demo accounts. The email addresses
// match the ones the demo's own UI already searches by (e.g. the collectible
// award-recipient picker), so they must stay in sync with the Elm client and
// the Playwright suite, not just this file.
type demoUserSeed struct {
	label string
	email string
}

// demoPassword is the known password for every seeded demo account - fine to
// publish since these are throwaway browser-local accounts with no real
// value behind them, and a fixed password lets the demo auto-login as the
// admin account without a visible login screen (see SeedDemoScenario).
const demoPassword = "sharecrop-demo-password-1"

var demoUsers = []demoUserSeed{
	{label: "mara", email: "mara@sharecrop.demo"},
	{label: "jules", email: "jules@sharecrop.demo"},
	{label: "ren", email: "ren@sharecrop.demo"},
	{label: "tala", email: "tala@sharecrop.demo"},
	{label: "sol", email: "sol@sharecrop.demo"},
}

// SeedResult reports the outcome of SeedDemoScenario: the seeded admin
// user's id (to grant bootstrap platform-admin status, which resets every
// page load since PlatformAdmins is in-memory) and a ready-to-replay
// refresh-token cookie so the demo's first /api/auth/refresh call succeeds
// immediately, matching the "already logged in" UX the browser demo has
// always had.
type SeedResult struct {
	Err               string
	AdminUserID       core.UserID
	AdminRefreshToken *http.Cookie
}

func seedErr(reason string) SeedResult { return SeedResult{Err: reason} }

// SeedDemoScenario seeds the fixed browser-demo cast of users, an
// organization, and a handful of tasks the first time it runs against fresh
// browser storage (detected by mara's account not existing yet), then logs
// mara in either way so the caller always gets a fresh refresh-token cookie
// - browser storage persists seeded data across page reloads, but the
// in-memory admin/session state built fresh each reload does not.
func SeedDemoScenario(ctx context.Context, authService auth.Service, organizationService org.Service, taskService task.Service, ledgerService ledger.Service) SeedResult {
	maraEmailResult := auth.NewEmailAddress(demoUsers[0].email)
	maraEmail, matched := maraEmailResult.(auth.EmailAddressAccepted)
	if !matched {
		return seedErr(maraEmailResult.(auth.EmailAddressRejected).Reason.Description())
	}
	passwordResult := auth.NewPasswordSecret(demoPassword)
	password, matched := passwordResult.(auth.PasswordSecretAccepted)
	if !matched {
		return seedErr(passwordResult.(auth.PasswordSecretRejected).Reason.Description())
	}

	registerResult := authService.Register(ctx, maraEmail.Value, password.Value)
	if accepted, matched := registerResult.(auth.RegisterAccepted); matched {
		if err := seedDemoScenarioData(ctx, authService, organizationService, taskService, ledgerService, accepted.Subject.ID, password.Value); err != "" {
			return seedErr(err)
		}
		return SeedResult{AdminUserID: accepted.Subject.ID, AdminRefreshToken: refreshCookie(accepted.RefreshToken)}
	}

	// Registration failing with "already registered" means this browser has
	// already been seeded in a prior session - log in instead of re-seeding.
	loginResult := authService.Login(ctx, maraEmail.Value, password.Value)
	accepted, matched := loginResult.(auth.LoginAccepted)
	if !matched {
		return seedErr(loginResult.(auth.LoginRejected).Reason.Description())
	}
	return SeedResult{AdminUserID: accepted.Subject.ID, AdminRefreshToken: refreshCookie(accepted.RefreshToken)}
}

// refreshCookie mirrors internal/http's setRefreshCookie shape (same cookie
// name), the piece SeedDemoScenario's caller preloads into the WASM
// binary's own request bridge so the first /api/auth/refresh call succeeds
// without a round trip through the JS host.
func refreshCookie(token auth.RefreshTokenPlain) *http.Cookie {
	return &http.Cookie{Name: "sharecrop_refresh_token", Value: token.String(), Path: "/"}
}

func seedDemoScenarioData(ctx context.Context, authService auth.Service, organizationService org.Service, taskService task.Service, ledgerService ledger.Service, maraID core.UserID, password auth.PasswordSecret) string {
	mara := auth.UserSubject{ID: maraID}
	memberIDs := make(map[string]core.UserID, len(demoUsers))
	memberIDs[demoUsers[0].label] = maraID

	for _, seed := range demoUsers[1:] {
		emailResult := auth.NewEmailAddress(seed.email)
		email, matched := emailResult.(auth.EmailAddressAccepted)
		if !matched {
			return emailResult.(auth.EmailAddressRejected).Reason.Description()
		}
		registerResult := authService.Register(ctx, email.Value, password)
		accepted, matched := registerResult.(auth.RegisterAccepted)
		if !matched {
			return registerResult.(auth.RegisterRejected).Reason.Description()
		}
		memberIDs[seed.label] = accepted.Subject.ID
	}

	organizationNameResult := org.NewOrganizationName("Field Operations")
	organizationName, matched := organizationNameResult.(org.OrganizationNameAccepted)
	if !matched {
		return organizationNameResult.(org.OrganizationNameRejected).Reason.Description()
	}
	organizationResult := organizationService.CreateOrganization(ctx, mara, organizationName.Value)
	organizationCreated, matched := organizationResult.(org.OrganizationCreated)
	if !matched {
		return organizationResult.(org.CreateOrganizationRejected).Reason.Description()
	}
	organizationID := organizationCreated.Value.ID

	for _, label := range []string{"jules", "ren"} {
		emailResult := auth.NewEmailAddress(emailFor(label))
		email := emailResult.(auth.EmailAddressAccepted).Value
		provisionResult := organizationService.ProvisionMember(ctx, mara, organizationID, email, []org.Role{org.RoleMember})
		if _, matched := provisionResult.(org.MemberProvisioned); !matched {
			return provisionResult.(org.ProvisionMemberRejected).Reason.Description()
		}
	}

	// task-ledger-review: mara's own public task, already funded and open,
	// so the demo shows a refundable task with no fund panel on load.
	if err := seedFundedOpenTask(ctx, taskService, ledgerService, mara, "Verify 10 ledger transfers for fraud signals",
		"Review ledger movements and flag suspicious transfers.", task.TaskTypeCodeReview,
		`{"kind":"freeform"}`, "", 30); err != "" {
		return err
	}

	// task-invoices: public, reservable, with the exact embedded reference
	// data and response schema the demo's detail-view Playwright test reads.
	julesSubject := auth.UserSubject{ID: memberIDs["jules"]}
	invoicesTask := taskService.Create(ctx, task.CreateCommand{
		Actor: julesSubject, Owner: task.UserOwner{UserID: memberIDs["jules"]},
		Title:       seedTitle("Extract line items from 6 vendor invoices"),
		Description: seedDescription("OCR'd text of 6 vendor invoices."),
		Type:        task.TaskTypeQATesting, Reference: task.ReferenceURL{},
		Reward: task.NoRewardSpec{}, Participation: task.ParticipationPolicyReservationRequired,
		AssigneeScope: task.AssigneeScopeUser, ReservationTTL: task.DefaultReservationTTL(),
		Visibility: task.PublicVisibility{}, Placement: task.StandalonePlacement{},
		ResponseSchema: seedSchema(`{"kind":"object","fields":[{"name":"invoices","presence":"required","schema":{"kind":"array","item":{"kind":"freeform"}}}]}`),
		Payload:        task.JSONDataPayload{Source: seedPayload(`{"vendor":"Birch Supply Co","fields":["invoice_id","total","due_date"]}`)},
	})
	invoicesCreated, matched := invoicesTask.(task.TaskCreated)
	if !matched {
		return invoicesTask.(task.CreateRejected).Reason.Description()
	}
	if result := taskService.Open(ctx, julesSubject, invoicesCreated.Value.ID); !isTaskStateChanged(result) {
		return "open task-invoices failed"
	}

	// task-support: public, owned and personally funded by ren (not mara),
	// so mara sees it as a plain worker view (no owner controls) - used by
	// the attachment-submission test, which submits to it as mara.
	renSubject := auth.UserSubject{ID: memberIDs["ren"]}
	supportTask := taskService.Create(ctx, task.CreateCommand{
		Actor: renSubject, Owner: task.UserOwner{UserID: memberIDs["ren"]},
		Title:       seedTitle("Classify 8 support tickets by category"),
		Description: seedDescription("Classify support tickets into billing, bug, account, feature_request, or other."),
		Type:        task.TaskTypeQATesting, Reference: task.ReferenceURL{},
		Reward: mustCreditReward(20), Participation: task.ParticipationPolicyOpen,
		AssigneeScope: task.AssigneeScopeUser, ReservationTTL: task.DefaultReservationTTL(),
		Visibility: task.PublicVisibility{}, Placement: task.StandalonePlacement{},
		ResponseSchema: seedSchema(`{"kind":"object","fields":[{"name":"labels","presence":"required","schema":{"kind":"array","item":{"kind":"string"}}}]}`),
		Payload:        task.JSONDataPayload{Source: seedPayload(`{"tickets":["billing question","bug report"]}`)},
	})
	supportCreated, matched := supportTask.(task.TaskCreated)
	if !matched {
		return supportTask.(task.CreateRejected).Reason.Description()
	}
	fundResult := ledgerService.FundTask(ctx, memberIDs["ren"], supportCreated.Value.ID, mustCreditAmount(20), mustIdempotencyKey("seed-fund-support"))
	if _, matched := fundResult.(ledger.TaskFunded); !matched {
		return "fund task-support failed"
	}
	if result := taskService.Open(ctx, renSubject, supportCreated.Value.ID); !isTaskStateChanged(result) {
		return "open task-support failed"
	}

	// task-release-notes: public, no reward, simple freeform discovery item.
	talaSubject := auth.UserSubject{ID: memberIDs["tala"]}
	releaseNotesTask := taskService.Create(ctx, task.CreateCommand{
		Actor: talaSubject, Owner: task.UserOwner{UserID: memberIDs["tala"]},
		Title:       seedTitle("Write release notes for 5 changelog entries"),
		Description: seedDescription("Convert changelog entries into concise release notes."),
		Type:        task.TaskTypeGeneral, Reference: task.ReferenceURL{},
		Reward: task.NoRewardSpec{}, Participation: task.ParticipationPolicyOpen,
		AssigneeScope: task.AssigneeScopeUser, ReservationTTL: task.DefaultReservationTTL(),
		Visibility: task.PublicVisibility{}, Placement: task.StandalonePlacement{},
		ResponseSchema: seedSchema(`{"kind":"freeform"}`),
		Payload:        task.JSONDataPayload{Source: seedPayload(`{"entries":["Added WASM demo backend"]}`)},
	})
	releaseNotesCreated, matched := releaseNotesTask.(task.TaskCreated)
	if !matched {
		return releaseNotesTask.(task.CreateRejected).Reason.Description()
	}
	if result := taskService.Open(ctx, talaSubject, releaseNotesCreated.Value.ID); !isTaskStateChanged(result) {
		return "open task-release-notes failed"
	}

	return ""
}

func emailFor(label string) string {
	for _, seed := range demoUsers {
		if seed.label == label {
			return seed.email
		}
	}
	return ""
}

// seedTitle/seedDescription/seedSchema/seedPayload panic on an invalid seed
// literal - these are fixed, developer-owned strings, not user input, so a
// validation failure here is a programming error worth failing loudly and
// immediately, not a runtime error path.
func seedTitle(raw string) task.Title {
	result := task.NewTitle(raw)
	accepted, matched := result.(task.TitleAccepted)
	if !matched {
		panic("invalid seed title")
	}
	return accepted.Value
}

func seedDescription(raw string) task.Description {
	result := task.NewDescription(raw)
	accepted, matched := result.(task.DescriptionAccepted)
	if !matched {
		panic("invalid seed description")
	}
	return accepted.Value
}

func seedSchema(raw string) task.ResponseSchemaSource {
	result := task.NewResponseSchemaSource(raw)
	accepted, matched := result.(task.ResponseSchemaSourceAccepted)
	if !matched {
		panic("invalid seed response schema")
	}
	return accepted.Value
}

func seedPayload(raw string) task.PayloadSource {
	result := task.NewPayloadSource(raw)
	accepted, matched := result.(task.PayloadSourceAccepted)
	if !matched {
		panic("invalid seed payload")
	}
	return accepted.Value
}

func mustCreditReward(amount int64) task.RewardSpec {
	result := task.NewCreditRewardAmount(amount)
	accepted, matched := result.(task.CreditRewardAmountAccepted)
	if !matched {
		panic("invalid seed credit reward amount")
	}
	return task.CreditRewardSpec{Amount: accepted.Value}
}

func mustCreditAmount(amount int64) ledger.CreditAmount {
	result := ledger.NewCreditAmount(amount)
	accepted, matched := result.(ledger.CreditAmountAccepted)
	if !matched {
		panic("invalid seed credit amount")
	}
	return accepted.Value
}

func mustIdempotencyKey(raw string) ledger.IdempotencyKey {
	result := ledger.NewIdempotencyKey(raw)
	accepted, matched := result.(ledger.IdempotencyKeyAccepted)
	if !matched {
		panic("invalid seed idempotency key")
	}
	return accepted.Value
}

func isTaskStateChanged(result task.ChangeStateResult) bool {
	_, matched := result.(task.TaskStateChanged)
	return matched
}

// seedFundedOpenTask creates a task owned by actor, funds it from the
// actor's own balance, and opens it - the shape task-ledger-review needs.
func seedFundedOpenTask(ctx context.Context, taskService task.Service, ledgerService ledger.Service, actor auth.UserSubject, title string, description string, taskType task.TaskType, schemaJSON string, payloadJSON string, amount int64) string {
	command := task.CreateCommand{
		Actor: actor, Owner: task.UserOwner{UserID: actor.ID},
		Title: seedTitle(title), Description: seedDescription(description),
		Type: taskType, Reference: task.ReferenceURL{},
		Reward: mustCreditReward(amount), Participation: task.ParticipationPolicyOpen,
		AssigneeScope: task.AssigneeScopeUser, ReservationTTL: task.DefaultReservationTTL(),
		Visibility: task.PublicVisibility{}, Placement: task.StandalonePlacement{},
		ResponseSchema: seedSchema(schemaJSON), Payload: task.NoDataPayload{},
	}
	if payloadJSON != "" {
		command.Payload = task.JSONDataPayload{Source: seedPayload(payloadJSON)}
	}
	createResult := taskService.Create(ctx, command)
	created, matched := createResult.(task.TaskCreated)
	if !matched {
		return createResult.(task.CreateRejected).Reason.Description()
	}
	fundResult := ledgerService.FundTask(ctx, actor.ID, created.Value.ID, mustCreditAmount(amount), mustIdempotencyKey("seed-fund-"+created.Value.ID.String()))
	if _, matched := fundResult.(ledger.TaskFunded); !matched {
		return "fund seeded task failed"
	}
	if openResult := taskService.Open(ctx, actor, created.Value.ID); !isTaskStateChanged(openResult) {
		return "open seeded task failed"
	}
	return ""
}
