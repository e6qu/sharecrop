package wasmseed

import (
	"context"
	"net/http"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/attachment"
	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/notification"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/e6qu/sharecrop/internal/wasibridge/appmux"
)

// Seed builds the demo's domain services over the given stores and seeds the
// fixed browser-demo scenario. It is the entry point the browser demo uses so
// the stores it seeds are the same ones appmux.New serves requests over.
func Seed(ctx context.Context, secret auth.AccessTokenSecret, stores appmux.Stores) SeedResult {
	authServiceResult := auth.NewService(stores.Auth, secret, auth.SystemClock{})
	authService, matched := authServiceResult.(auth.ServiceCreated)
	if !matched {
		return seedErr(authServiceResult.(auth.ServiceRejected).Reason.Description())
	}
	organizationService := org.NewService(stores.Organization)
	agentService := agent.NewService(stores.Agent)
	// The seed goes through the same recorder-wired services production uses,
	// so seeded actions produce the same events and inbox notifications a live
	// user would see.
	recorder := event.NewRecorder(stores.Event, notification.NewService(stores.Notification))
	taskService := task.NewService(stores.Task, organizationService, agentService, recorder)
	submissionService := submission.NewService(stores.Submission, stores.Task, organizationService, recorder)
	ledgerService := ledger.NewService(stores.Ledger, recorder, audit.NewService(stores.Audit))
	assetService := assets.NewService(stores.Assets, recorder)

	return SeedDemoScenario(ctx, authService.Value, organizationService, taskService, ledgerService, submissionService, assetService)
}

// demoUserSeed is one of the fixed browser-demo accounts. The email addresses
// match the ones the demo's own UI already searches by (e.g. the collectible
// award-recipient picker), so they must stay in sync with the Elm client and
// the Playwright suite, not just this file.
type demoUserSeed struct {
	label string
	email string
	name  string
}

// demoPassword is the known password for every seeded demo account - fine to
// publish since these are throwaway browser-local accounts with no real
// value behind them, and a fixed password lets the demo auto-login as the
// admin account without a visible login screen (see SeedDemoScenario).
const demoPassword = "sharecrop-demo-password-1"

var demoUsers = []demoUserSeed{
	{label: "mara", email: "mara@sharecrop.demo", name: "Mara Ellison"},
	{label: "jules", email: "jules@sharecrop.demo", name: "Jules Arden"},
	{label: "ren", email: "ren@sharecrop.demo", name: "Ren Okafor"},
	{label: "tala", email: "tala@sharecrop.demo", name: "Tala Reyes"},
	{label: "sol", email: "sol@sharecrop.demo", name: "Sol Marchetti"},
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
func SeedDemoScenario(ctx context.Context, authService auth.Service, organizationService org.Service, taskService task.Service, ledgerService ledger.Service, submissionService submission.Service, assetService assets.Service) SeedResult {
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

	maraNameResult := auth.NewDisplayName(demoUsers[0].name)
	maraName, matched := maraNameResult.(auth.DisplayNameAccepted)
	if !matched {
		return seedErr(maraNameResult.(auth.DisplayNameRejected).Reason.Description())
	}
	registerResult := authService.Register(ctx, maraEmail.Value, password.Value, auth.ProvidedDisplayName{Value: maraName.Value})
	if accepted, matched := registerResult.(auth.RegisterAccepted); matched {
		if err := seedDemoScenarioData(ctx, authService, organizationService, taskService, ledgerService, submissionService, assetService, accepted.Subject.ID, password.Value); err != "" {
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

func seedDemoScenarioData(ctx context.Context, authService auth.Service, organizationService org.Service, taskService task.Service, ledgerService ledger.Service, submissionService submission.Service, assetService assets.Service, maraID core.UserID, password auth.PasswordSecret) string {
	mara := auth.UserSubject{ID: maraID}
	memberIDs := make(map[string]core.UserID, len(demoUsers))
	memberIDs[demoUsers[0].label] = maraID

	for _, seed := range demoUsers[1:] {
		emailResult := auth.NewEmailAddress(seed.email)
		email, matched := emailResult.(auth.EmailAddressAccepted)
		if !matched {
			return emailResult.(auth.EmailAddressRejected).Reason.Description()
		}
		nameResult := auth.NewDisplayName(seed.name)
		name, nameMatched := nameResult.(auth.DisplayNameAccepted)
		if !nameMatched {
			return nameResult.(auth.DisplayNameRejected).Reason.Description()
		}
		registerResult := authService.Register(ctx, email.Value, password, auth.ProvidedDisplayName{Value: name.Value})
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
	// so the demo shows a refundable task with no fund panel on load. It is
	// deliberately left free of pending submissions so the demo refund flow
	// still applies (a task with work pending review cannot be refunded).
	if _, fraudErr := seedFundedOpenTask(ctx, taskService, ledgerService, mara, "Verify 10 ledger transfers for fraud signals",
		"Review ledger movements and flag suspicious transfers.", task.TaskTypeCodeReview,
		`{"kind":"freeform"}`, "", 30); fraudErr != "" {
		return fraudErr
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

	// The demo is single-actor (always mara), so the review half of the
	// marketplace is only demonstrable if the seed targets mara with work
	// from the other seeded users: a submission waiting for her review, a
	// reservation request waiting for her approval, the matching inbox
	// notification, and a collectible in her holdings.

	// review-inbox: a separate mara-owned task with a pending submission from
	// ren, so mara has a real submission to accept/reject/request changes on,
	// plus the same inbox notification the HTTP layer writes for a live
	// submission. This is deliberately a *no-reward* task on its own (not the
	// fraud task): mara's balance stays predictable (95: the 100 signup grant,
	// minus the 30 fraud-task funding, plus ren's 25-credit send), and the fraud task
	// stays free of pending work so the demo's refund flow still applies to
	// it. renSubject is already declared above (it owns task-support).
	reviewTask := taskService.Create(ctx, task.CreateCommand{
		Actor: mara, Owner: task.UserOwner{UserID: maraID},
		Title:       seedTitle("Review 5 pull request diffs"),
		Description: seedDescription("Read each diff and note correctness, style, and test-coverage issues."),
		Type:        task.TaskTypeCodeReview, Reference: task.ReferenceURL{},
		Reward: task.NoRewardSpec{}, Participation: task.ParticipationPolicyOpen,
		AssigneeScope: task.AssigneeScopeUser, ReservationTTL: task.DefaultReservationTTL(),
		Visibility: task.PublicVisibility{}, Placement: task.StandalonePlacement{},
		ResponseSchema: seedSchema(`{"kind":"object","fields":[{"name":"findings","presence":"required","schema":{"kind":"array","item":{"kind":"freeform"}}}]}`),
		Payload:        task.JSONDataPayload{Source: seedPayload(`{"pull_requests":["#141","#142","#143","#144","#145"]}`)},
	})
	reviewCreated, matched := reviewTask.(task.TaskCreated)
	if !matched {
		return reviewTask.(task.CreateRejected).Reason.Description()
	}
	if result := taskService.Open(ctx, mara, reviewCreated.Value.ID); !isTaskStateChanged(result) {
		return "open task-review failed"
	}
	responseResult := submission.NewResponseSource(`{"findings":[{"pull_request":"#142","issue":"missing nil check on the reservation path"},{"pull_request":"#144","issue":"no test covers the refund-after-close case"}]}`)
	responseAccepted, matched := responseResult.(submission.ResponseSourceAccepted)
	if !matched {
		return "seed submission response was rejected"
	}
	submitResult := submissionService.Submit(ctx, submission.SubmitCommand{
		TaskID:         reviewCreated.Value.ID,
		SubmitterID:    renSubject.ID,
		ResponseSource: responseAccepted.Value,
		Attachments:    []attachment.Attachment{},
	})
	// The recorder-wired submission service also fans out mara's
	// submission_created inbox notification; no manual seed row is needed.
	if _, matched := submitResult.(submission.SubmissionCreated); !matched {
		return submitResult.(submission.SubmitRejected).Reason.Description()
	}

	// task-disputed: a ren-owned public task where MARA's own submission was
	// rejected, so the single-actor demo can show the rejected state and the
	// worker-side "File a dispute" affordance (and its Playwright test can
	// walk a dispute into the admin moderation queue).
	disputedTask := taskService.Create(ctx, task.CreateCommand{
		Actor: renSubject, Owner: task.UserOwner{UserID: memberIDs["ren"]},
		Title:       seedTitle("Tag 12 product photos with alt text"),
		Description: seedDescription("Write a one-sentence alt text for each of the 12 listed product photos."),
		Type:        task.TaskTypeGeneral, Reference: task.ReferenceURL{},
		Reward: task.NoRewardSpec{}, Participation: task.ParticipationPolicyOpen,
		AssigneeScope: task.AssigneeScopeUser, ReservationTTL: task.DefaultReservationTTL(),
		Visibility: task.PublicVisibility{}, Placement: task.StandalonePlacement{},
		ResponseSchema: seedSchema(`{"kind":"freeform"}`),
		Payload:        task.JSONDataPayload{Source: seedPayload(`{"photos":["p-01.jpg","p-02.jpg","p-03.jpg"]}`)},
	})
	disputedCreated, matched := disputedTask.(task.TaskCreated)
	if !matched {
		return disputedTask.(task.CreateRejected).Reason.Description()
	}
	if result := taskService.Open(ctx, renSubject, disputedCreated.Value.ID); !isTaskStateChanged(result) {
		return "open task-disputed failed"
	}
	disputedResponse := submission.NewResponseSource(`{"alt_texts":["Product photo","Product photo","Product photo"]}`)
	disputedResponseAccepted, matched := disputedResponse.(submission.ResponseSourceAccepted)
	if !matched {
		return "seed disputed submission response was rejected"
	}
	disputedSubmit := submissionService.Submit(ctx, submission.SubmitCommand{
		TaskID:         disputedCreated.Value.ID,
		SubmitterID:    maraID,
		ResponseSource: disputedResponseAccepted.Value,
		Attachments:    []attachment.Attachment{},
	})
	disputedSubmitted, matched := disputedSubmit.(submission.SubmissionCreated)
	if !matched {
		return disputedSubmit.(submission.SubmitRejected).Reason.Description()
	}
	rejectNoteResult := submission.NewRequiredReviewNote("Every photo got the same generic sentence - each alt text must describe its own photo.")
	rejectNote, matched := rejectNoteResult.(submission.ReviewNoteAccepted)
	if !matched {
		return "seed reject note was rejected"
	}
	rejectResult := ledgerService.RejectSubmission(ctx, ledger.UserReviewer{ID: memberIDs["ren"]}, disputedCreated.Value.ID, disputedSubmitted.Value.ID, mustIdempotencyKey("seed-reject-disputed"), rejectNote.Value, ledger.NoCreditReviewSelection{}, ledger.NoTipSelection{}, ledger.NoBanSelection{})
	if _, matched := rejectResult.(ledger.SubmissionRejected); !matched {
		return "seed reject of the disputed submission failed"
	}

	// task-reserved: a reservation-required task owned by mara that sol has
	// already reserved (reserving is immediate now that the approval gate is
	// gone), so the owner's reservation list and cancel control have
	// something real to act on.
	reservedTask := taskService.Create(ctx, task.CreateCommand{
		Actor: mara, Owner: task.UserOwner{UserID: maraID},
		Title:       seedTitle("Proofread 3 onboarding emails"),
		Description: seedDescription("Check the drafts for tone, typos, and broken links before they go out."),
		Type:        task.TaskTypeProductReview, Reference: task.ReferenceURL{},
		Reward: task.NoRewardSpec{}, Participation: task.ParticipationPolicyReservationRequired,
		AssigneeScope: task.AssigneeScopeUser, ReservationTTL: task.DefaultReservationTTL(),
		Visibility: task.PublicVisibility{}, Placement: task.StandalonePlacement{},
		ResponseSchema: seedSchema(`{"kind":"freeform"}`),
		Payload:        task.JSONDataPayload{Source: seedPayload(`{"drafts":["welcome.md","first-task.md","invite-team.md"]}`)},
	})
	reservedCreated, matched := reservedTask.(task.TaskCreated)
	if !matched {
		return reservedTask.(task.CreateRejected).Reason.Description()
	}
	if result := taskService.Open(ctx, mara, reservedCreated.Value.ID); !isTaskStateChanged(result) {
		return "open task-reserved failed"
	}
	solSubject := auth.UserSubject{ID: memberIDs["sol"]}
	if reserveResult := taskService.Reserve(ctx, solSubject, reservedCreated.Value.ID); !isReservationCreated(reserveResult) {
		return "seed reservation failed"
	}

	// A collectible in mara's holdings, awarded from the DB-backed catalog
	// the same way the admin award endpoint mints one, so the collectible
	// pages have something to show without minting first. Transferable, so
	// the demo trade flow also works on it.
	if mintResult := assetService.AwardFromCatalog(ctx, maraID, "harvest-star", assets.CollectibleOwnerKindUser, maraID.String(), ""); !isCollectibleMinted(mintResult) {
		return "seed collectible award failed"
	}

	// Catalog richness for the admin demo (mara is the platform admin):
	// a fully-minted unique, a partially-minted edition, and a withdrawn
	// entry, so the catalog gallery shows every state and count shape the
	// admin management UI can render.
	if err := assetService.AddCatalogEntry(ctx, assets.CatalogEntry{
		Slug: mustCatalogSlug("founders-medal"), Name: mustCollectibleName("Founders' Medal"),
		Kind: assets.CollectibleKindUnique, Policy: assets.TransferPolicyIssuerControlled,
		Art: "first-harvest-trophy", State: assets.CatalogEntryStateAvailable,
		Cap: assets.EditionCapOf{Limit: 1},
	}); !isCatalogMutated(err) {
		return "seed founders-medal catalog entry failed"
	}
	if mintResult := assetService.AwardFromCatalog(ctx, maraID, "founders-medal", assets.CollectibleOwnerKindUser, maraID.String(), ""); !isCollectibleMinted(mintResult) {
		return "seed founders-medal award failed"
	}
	if err := assetService.AddCatalogEntry(ctx, assets.CatalogEntry{
		Slug: mustCatalogSlug("harvest-festival-2026"), Name: mustCollectibleName("Harvest Festival 2026"),
		Kind: assets.CollectibleKindEdition, Policy: assets.TransferPolicyTransferableBetweenUsers,
		Art: "cornucopia", State: assets.CatalogEntryStateAvailable,
		Cap: assets.EditionCapOf{Limit: 25},
	}); !isCatalogMutated(err) {
		return "seed harvest-festival catalog entry failed"
	}
	for _, label := range []string{"mara", "ren", "jules"} {
		if mintResult := assetService.AwardFromCatalog(ctx, maraID, "harvest-festival-2026", assets.CollectibleOwnerKindUser, memberIDs[label].String(), ""); !isCollectibleMinted(mintResult) {
			return "seed harvest-festival award failed"
		}
	}
	if err := assetService.AddCatalogEntry(ctx, assets.CatalogEntry{
		Slug: mustCatalogSlug("retired-scarecrow"), Name: mustCollectibleName("Retired Scarecrow"),
		Kind: assets.CollectibleKindBadge, Policy: assets.TransferPolicyNonTransferableExceptPayout,
		Art: "scarecrow", State: assets.CatalogEntryStateAvailable,
		Cap: assets.NoEditionCap{},
	}); !isCatalogMutated(err) {
		return "seed retired-scarecrow catalog entry failed"
	}
	if err := assetService.WithdrawCatalogEntry(ctx, mustCatalogSlug("retired-scarecrow")); !isCatalogMutated(err) {
		return "seed retired-scarecrow withdrawal failed"
	}

	// A withdrawn instance in mara's own holdings: award a rain-drop, then
	// withdraw it, so the withdrawn state (and the admin Delete control) is
	// visible without a live mutation first.
	withdrawnMint := assetService.AwardFromCatalog(ctx, maraID, "rain-drop", assets.CollectibleOwnerKindUser, maraID.String(), "")
	withdrawnMinted, withdrawnMatched := withdrawnMint.(assets.CollectibleMinted)
	if !withdrawnMatched {
		return "seed rain-drop award failed"
	}
	if result := assetService.WithdrawCollectible(ctx, maraID, withdrawnMinted.Value.ID); !isCollectibleWithdrawn(result) {
		return "seed rain-drop withdrawal failed"
	}

	// A collectible owned by the Field Operations organization, so the
	// org page's award and send-to-user controls have something to move.
	if mintResult := assetService.AwardFromCatalog(ctx, maraID, "golden-egg", assets.CollectibleOwnerKindOrganization, organizationID.String(), organizationID.String()); !isCollectibleMinted(mintResult) {
		return "seed organization collectible failed"
	}

	// A received peer credit send: ren thanks mara with 25 credits, so the
	// ledger shows a peer_transfer row with its note and the inbox shows the
	// credits_received notification.
	sendNote := ledger.NewGrantNote("Thanks for the fraud-sweep review")
	sendNoteAccepted, sendNoteMatched := sendNote.(ledger.GrantNoteAccepted)
	if !sendNoteMatched {
		return "seed transfer note was rejected"
	}
	sendResult := ledgerService.SendCredits(ctx, memberIDs["ren"], ledger.TransferFromSelf{}, ledger.TransferToUser{ID: maraID},
		mustCreditAmount(25), ledger.TransferNoteProvided{Note: sendNoteAccepted.Value}, mustIdempotencyKey("seed-send-ren-mara"))
	if _, matched := sendResult.(ledger.CreditsSent); !matched {
		return "seed peer credit send failed"
	}

	return ""
}

func isCatalogMutated(result assets.CatalogMutationResult) bool {
	_, matched := result.(assets.CatalogMutated)
	return matched
}

func isCollectibleWithdrawn(result assets.WithdrawResult) bool {
	_, matched := result.(assets.CollectibleWithdrawn)
	return matched
}

func mustCatalogSlug(raw string) assets.CatalogSlug {
	result := assets.NewCatalogSlug(raw)
	accepted, matched := result.(assets.CatalogSlugAccepted)
	if !matched {
		panic("invalid seed catalog slug")
	}
	return accepted.Value
}

func mustCollectibleName(raw string) assets.CollectibleName {
	result := assets.NewCollectibleName(raw)
	accepted, matched := result.(assets.CollectibleNameAccepted)
	if !matched {
		panic("invalid seed collectible name")
	}
	return accepted.Value
}

func isReservationCreated(result task.ReservationResult) bool {
	_, matched := result.(task.ReservationCreated)
	return matched
}

func isCollectibleMinted(result assets.MintResult) bool {
	_, matched := result.(assets.CollectibleMinted)
	return matched
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
func seedFundedOpenTask(ctx context.Context, taskService task.Service, ledgerService ledger.Service, actor auth.UserSubject, title string, description string, taskType task.TaskType, schemaJSON string, payloadJSON string, amount int64) (core.TaskID, string) {
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
		return core.TaskID{}, createResult.(task.CreateRejected).Reason.Description()
	}
	fundResult := ledgerService.FundTask(ctx, actor.ID, created.Value.ID, mustCreditAmount(amount), mustIdempotencyKey("seed-fund-"+created.Value.ID.String()))
	if _, matched := fundResult.(ledger.TaskFunded); !matched {
		return core.TaskID{}, "fund seeded task failed"
	}
	if openResult := taskService.Open(ctx, actor, created.Value.ID); !isTaskStateChanged(openResult) {
		return core.TaskID{}, "open seeded task failed"
	}
	return created.Value.ID, ""
}
