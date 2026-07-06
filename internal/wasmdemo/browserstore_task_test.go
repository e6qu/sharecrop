package wasmdemo

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/task"
)

// noopOrganizationPermissions denies everything - none of these core-lifecycle
// tests exercise organization-owned tasks/teams.
type noopOrganizationPermissions struct{}

func (noopOrganizationPermissions) CheckOrganizationPermission(context.Context, core.OrganizationID, core.UserID, org.Permission) org.PermissionCheck {
	return org.PermissionDenied{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "not implemented in this test")}
}

func (noopOrganizationPermissions) CheckOrganizationTeamMembership(context.Context, core.OrganizationID, core.TeamID, core.UserID) org.PermissionCheck {
	return org.PermissionDenied{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "not implemented in this test")}
}

func (noopOrganizationPermissions) CheckTeamMembership(context.Context, core.TeamID, core.UserID) org.PermissionCheck {
	return org.PermissionDenied{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "not implemented in this test")}
}

func (noopOrganizationPermissions) GetTeam(context.Context, auth.Subject, core.TeamID) org.GetTeamResult {
	return org.GetTeamRejected{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "not implemented in this test")}
}

// allowAllOrganizationPermissions grants every check unconditionally, for
// tests exercising a team-scoped reservation where the permission-layer
// check (task.Service.ReserveForTeam etc.) isn't itself under test.
type allowAllOrganizationPermissions struct{}

func (allowAllOrganizationPermissions) CheckOrganizationPermission(context.Context, core.OrganizationID, core.UserID, org.Permission) org.PermissionCheck {
	return org.PermissionGranted{}
}

func (allowAllOrganizationPermissions) CheckOrganizationTeamMembership(context.Context, core.OrganizationID, core.TeamID, core.UserID) org.PermissionCheck {
	return org.PermissionGranted{}
}

func (allowAllOrganizationPermissions) CheckTeamMembership(context.Context, core.TeamID, core.UserID) org.PermissionCheck {
	return org.PermissionGranted{}
}

func (allowAllOrganizationPermissions) GetTeam(context.Context, auth.Subject, core.TeamID) org.GetTeamResult {
	return org.TeamGot{}
}

func testTaskTitle(t *testing.T, raw string) task.Title {
	t.Helper()
	result := task.NewTitle(raw)
	accepted, matched := result.(task.TitleAccepted)
	if !matched {
		t.Fatalf("new title %q failed", raw)
	}
	return accepted.Value
}

func testTaskDescription(t *testing.T, raw string) task.Description {
	t.Helper()
	result := task.NewDescription(raw)
	accepted, matched := result.(task.DescriptionAccepted)
	if !matched {
		t.Fatalf("new description %q failed", raw)
	}
	return accepted.Value
}

func testResponseSchema(t *testing.T) task.ResponseSchemaSource {
	t.Helper()
	result := task.NewResponseSchemaSource(`{"kind":"freeform"}`)
	accepted, matched := result.(task.ResponseSchemaSourceAccepted)
	if !matched {
		t.Fatalf("new response schema failed")
	}
	return accepted.Value
}

func testCreateCommand(t *testing.T, actorID core.UserID, reward task.RewardSpec, participation task.ParticipationPolicy) task.CreateCommand {
	t.Helper()
	return task.CreateCommand{
		Actor: auth.UserSubject{ID: actorID}, Owner: task.UserOwner{UserID: actorID},
		Title: testTaskTitle(t, "Test task"), Description: testTaskDescription(t, "A task for testing."),
		Type: task.TaskTypeGeneral, Reference: task.ReferenceURL{},
		Reward: reward, Participation: participation, AssigneeScope: task.AssigneeScopeUser,
		ReservationTTL: task.DefaultReservationTTL(), Visibility: task.PublicVisibility{},
		Placement: task.StandalonePlacement{}, ResponseSchema: testResponseSchema(t), Payload: task.NoDataPayload{},
	}
}

func newTaskTestEnv(t *testing.T) (task.Service, ledger.Service, *counterLedgerIDs, BrowserStorage) {
	t.Helper()
	storage := newTestBrowserStorage()
	ids := &counterLedgerIDs{}
	taskService := task.NewService(NewTaskBrowserStore(storage, ids), noopOrganizationPermissions{}, nil)
	ledgerService := ledger.NewService(NewLedgerBrowserStore(storage, ids))
	return taskService, ledgerService, ids, storage
}

func TestTaskBrowserStoreCreateAndGet(t *testing.T) {
	taskService, _, _, _ := newTaskTestEnv(t)
	ctx := context.Background()
	owner := auth.UserSubject{ID: testUserID(t, "owner")}

	createResult := taskService.Create(ctx, testCreateCommand(t, owner.ID, task.NoRewardSpec{}, task.ParticipationPolicyOpen))
	created, matched := createResult.(task.TaskCreated)
	if !matched {
		t.Fatalf("create: want TaskCreated, got %#v", createResult)
	}
	if created.Value.State != task.StateDraft {
		t.Fatalf("new task state = %v, want draft", created.Value.State)
	}

	getResult := taskService.Get(ctx, owner, created.Value.ID)
	got, matched := getResult.(task.TaskGot)
	if !matched {
		t.Fatalf("get: want TaskGot, got %#v", getResult)
	}
	if got.Value.Title.String() != "Test task" {
		t.Fatalf("got title = %q, want %q", got.Value.Title.String(), "Test task")
	}
}

// TestTaskBrowserStoreOpenRequiresFunding is the exact invariant this whole
// session's investigation started from: a credit reward must actually be
// escrowed before the task can open, not just declared.
func TestTaskBrowserStoreOpenRequiresFunding(t *testing.T) {
	taskService, _, _, _ := newTaskTestEnv(t)
	ctx := context.Background()
	owner := auth.UserSubject{ID: testUserID(t, "owner")}

	rewardResult := task.NewCreditRewardAmount(30)
	reward := rewardResult.(task.CreditRewardAmountAccepted).Value
	created := taskService.Create(ctx, testCreateCommand(t, owner.ID, task.CreditRewardSpec{Amount: reward}, task.ParticipationPolicyOpen)).(task.TaskCreated)

	openResult := taskService.Open(ctx, owner, created.Value.ID)
	if _, matched := openResult.(task.ChangeStateRejected); !matched {
		t.Fatalf("open unfunded credit-reward task: want ChangeStateRejected, got %#v", openResult)
	}
}

func TestTaskBrowserStoreOpenSucceedsAfterFunding(t *testing.T) {
	taskService, ledgerService, ids, storage := newTaskTestEnv(t)
	ctx := context.Background()
	owner := testUserID(t, "owner")
	ownerSubject := auth.UserSubject{ID: owner}
	NewAuthBrowserStore(storage, ids).insertSignupGrant("user", owner.String())

	rewardResult := task.NewCreditRewardAmount(30)
	reward := rewardResult.(task.CreditRewardAmountAccepted).Value
	created := taskService.Create(ctx, testCreateCommand(t, owner, task.CreditRewardSpec{Amount: reward}, task.ParticipationPolicyOpen)).(task.TaskCreated)

	fundResult := ledgerService.FundTask(ctx, owner, created.Value.ID, testCreditAmount(t, 30), testIdempotencyKey(t, "fund-1"))
	if _, matched := fundResult.(ledger.TaskFunded); !matched {
		t.Fatalf("fund: want TaskFunded, got %#v", fundResult)
	}

	openResult := taskService.Open(ctx, ownerSubject, created.Value.ID)
	opened, matched := openResult.(task.TaskStateChanged)
	if !matched {
		t.Fatalf("open funded task: want TaskStateChanged, got %#v", openResult)
	}
	if opened.Value.State != task.StateOpen {
		t.Fatalf("opened task state = %v, want open", opened.Value.State)
	}
}

func TestTaskBrowserStoreCancelRejectsWithHeldEscrow(t *testing.T) {
	taskService, ledgerService, ids, storage := newTaskTestEnv(t)
	ctx := context.Background()
	owner := testUserID(t, "owner")
	ownerSubject := auth.UserSubject{ID: owner}
	NewAuthBrowserStore(storage, ids).insertSignupGrant("user", owner.String())

	rewardResult := task.NewCreditRewardAmount(30)
	reward := rewardResult.(task.CreditRewardAmountAccepted).Value
	created := taskService.Create(ctx, testCreateCommand(t, owner, task.CreditRewardSpec{Amount: reward}, task.ParticipationPolicyOpen)).(task.TaskCreated)
	ledgerService.FundTask(ctx, owner, created.Value.ID, testCreditAmount(t, 30), testIdempotencyKey(t, "fund-1"))

	cancelResult := taskService.Cancel(ctx, ownerSubject, created.Value.ID)
	if _, matched := cancelResult.(task.ChangeStateRejected); !matched {
		t.Fatalf("cancel with held escrow: want ChangeStateRejected, got %#v", cancelResult)
	}
}

func TestTaskBrowserStoreReservationRequiredLifecycle(t *testing.T) {
	taskService, _, _, _ := newTaskTestEnv(t)
	ctx := context.Background()
	owner := auth.UserSubject{ID: testUserID(t, "owner")}
	worker := auth.UserSubject{ID: testUserID(t, "worker")}

	created := taskService.Create(ctx, testCreateCommand(t, owner.ID, task.NoRewardSpec{}, task.ParticipationPolicyReservationRequired)).(task.TaskCreated)
	openResult := taskService.Open(ctx, owner, created.Value.ID)
	if _, matched := openResult.(task.TaskStateChanged); !matched {
		t.Fatalf("open no-reward task: want TaskStateChanged, got %#v", openResult)
	}

	reserveResult := taskService.Reserve(ctx, worker, created.Value.ID)
	reserved, matched := reserveResult.(task.ReservationCreated)
	if !matched {
		t.Fatalf("reserve: want ReservationCreated, got %#v", reserveResult)
	}
	if reserved.Value.State != task.ReservationStateActive {
		t.Fatalf("reservation state = %v, want active (reservation_required skips approval)", reserved.Value.State)
	}

	eligibilityResult := taskService.CheckSubmissionEligibility(ctx, created.Value.ID, worker.ID)
	if _, matched := eligibilityResult.(task.SubmissionEligible); !matched {
		t.Fatalf("submission eligibility for reserving worker: want SubmissionEligible, got %#v", eligibilityResult)
	}

	otherWorker := testUserID(t, "other-worker")
	otherEligibilityResult := taskService.CheckSubmissionEligibility(ctx, created.Value.ID, otherWorker)
	if _, matched := otherEligibilityResult.(task.SubmissionEligibilityRejected); !matched {
		t.Fatalf("submission eligibility for non-reserving user: want SubmissionEligibilityRejected, got %#v", otherEligibilityResult)
	}

	cancelResult := taskService.CancelReservation(ctx, worker, created.Value.ID, reserved.Value.ID)
	cancelled, matched := cancelResult.(task.ReservationStateChanged)
	if !matched {
		t.Fatalf("worker cancels own reservation: want ReservationStateChanged, got %#v", cancelResult)
	}
	if cancelled.Value.State != task.ReservationStateCancelledByRequester {
		t.Fatalf("cancelled reservation state = %v, want cancelled_by_requester", cancelled.Value.State)
	}
}

func TestTaskBrowserStoreReservationBlockedByUnpublishedSeries(t *testing.T) {
	taskService, _, _, _ := newTaskTestEnv(t)
	ctx := context.Background()
	owner := auth.UserSubject{ID: testUserID(t, "owner")}
	worker := auth.UserSubject{ID: testUserID(t, "worker")}

	titleResult := task.NewSeriesTitle("Batch A")
	seriesTitle := titleResult.(task.SeriesTitleAccepted).Value
	positionResult := task.NewSeriesPosition(1)
	position := positionResult.(task.SeriesPositionAccepted).Value

	command := testCreateCommand(t, owner.ID, task.NoRewardSpec{}, task.ParticipationPolicyReservationRequired)
	command.Placement = task.NewSeriesPlacement{Title: seriesTitle, Position: position}
	created := taskService.Create(ctx, command).(task.TaskCreated)
	taskService.Open(ctx, owner, created.Value.ID)

	// The series starts in draft (unpublished) state, so the task inside it
	// cannot be reserved or submitted to yet.
	reserveResult := taskService.Reserve(ctx, worker, created.Value.ID)
	if _, matched := reserveResult.(task.ReservationRejected); !matched {
		t.Fatalf("reserve task in unpublished series: want ReservationRejected, got %#v", reserveResult)
	}
	eligibilityResult := taskService.CheckSubmissionEligibility(ctx, created.Value.ID, worker.ID)
	if _, matched := eligibilityResult.(task.SubmissionEligibilityRejected); !matched {
		t.Fatalf("submission eligibility in unpublished series: want SubmissionEligibilityRejected, got %#v", eligibilityResult)
	}

	placement := created.Value.Placement.(task.ExistingSeriesPlacement)
	publishResult := taskService.ChangeSeriesState(ctx, owner, placement.SeriesID, task.PublishSeriesState)
	if _, matched := publishResult.(task.SeriesMutated); !matched {
		t.Fatalf("publish series: want SeriesMutated, got %#v", publishResult)
	}

	reservedResult := taskService.Reserve(ctx, worker, created.Value.ID)
	if _, matched := reservedResult.(task.ReservationCreated); !matched {
		t.Fatalf("reserve task in published series: want ReservationCreated, got %#v", reservedResult)
	}
}

func TestTaskBrowserStoreSubmissionEligibleForAnyTeamMember(t *testing.T) {
	storage := newTestBrowserStorage()
	ids := &counterLedgerIDs{}
	taskStore := NewTaskBrowserStore(storage, ids)
	taskService := task.NewService(taskStore, allowAllOrganizationPermissions{}, nil)
	ctx := context.Background()
	owner := auth.UserSubject{ID: testUserID(t, "owner")}
	reservingMember := auth.UserSubject{ID: testUserID(t, "reserving-member")}
	otherMember := testUserID(t, "other-team-member")
	teamID := core.NewTeamID().(core.TeamIDCreated).Value

	command := testCreateCommand(t, owner.ID, task.NoRewardSpec{}, task.ParticipationPolicyReservationRequired)
	command.AssigneeScope = task.AssigneeScopeTeam
	created := taskService.Create(ctx, command).(task.TaskCreated)
	taskService.Open(ctx, owner, created.Value.ID)

	reserveResult := taskService.ReserveForTeam(ctx, reservingMember, created.Value.ID, teamID)
	if _, matched := reserveResult.(task.ReservationCreated); !matched {
		t.Fatalf("reserve for team: want ReservationCreated, got %#v", reserveResult)
	}

	// Not yet a recorded team member: not eligible.
	notYetEligible := taskService.CheckSubmissionEligibility(ctx, created.Value.ID, otherMember)
	if _, matched := notYetEligible.(task.SubmissionEligibilityRejected); !matched {
		t.Fatalf("submission eligibility before team membership: want SubmissionEligibilityRejected, got %#v", notYetEligible)
	}

	if _, matched := appendStringIndex(storage, teamMembersKey(teamID.String()), otherMember.String(), "team member").(stringIndexStored); !matched {
		t.Fatalf("write team member index failed")
	}

	// Any team member (not just whoever made the reservation) is eligible.
	eligibleResult := taskService.CheckSubmissionEligibility(ctx, created.Value.ID, otherMember)
	if _, matched := eligibleResult.(task.SubmissionEligible); !matched {
		t.Fatalf("submission eligibility for team member: want SubmissionEligible, got %#v", eligibleResult)
	}
}

func TestTaskBrowserStoreApprovalRequiredLifecycle(t *testing.T) {
	taskService, _, _, _ := newTaskTestEnv(t)
	ctx := context.Background()
	owner := auth.UserSubject{ID: testUserID(t, "owner")}
	worker := auth.UserSubject{ID: testUserID(t, "worker")}

	created := taskService.Create(ctx, testCreateCommand(t, owner.ID, task.NoRewardSpec{}, task.ParticipationPolicyApprovalRequired)).(task.TaskCreated)
	taskService.Open(ctx, owner, created.Value.ID)

	reserveResult := taskService.Reserve(ctx, worker, created.Value.ID)
	reserved, matched := reserveResult.(task.ReservationCreated)
	if !matched {
		t.Fatalf("reserve: want ReservationCreated, got %#v", reserveResult)
	}
	if reserved.Value.State != task.ReservationStateRequested {
		t.Fatalf("reservation state = %v, want requested (approval_required needs approval)", reserved.Value.State)
	}

	approveResult := taskService.ApproveReservation(ctx, owner, created.Value.ID, reserved.Value.ID)
	approved, matched := approveResult.(task.ReservationStateChanged)
	if !matched {
		t.Fatalf("approve: want ReservationStateChanged, got %#v", approveResult)
	}
	if approved.Value.State != task.ReservationStateActive {
		t.Fatalf("approved reservation state = %v, want active", approved.Value.State)
	}
}

func TestTaskBrowserStoreListTasksPublicScope(t *testing.T) {
	taskService, _, _, _ := newTaskTestEnv(t)
	ctx := context.Background()
	owner := auth.UserSubject{ID: testUserID(t, "owner")}

	created := taskService.Create(ctx, testCreateCommand(t, owner.ID, task.NoRewardSpec{}, task.ParticipationPolicyOpen)).(task.TaskCreated)
	taskService.Open(ctx, owner, created.Value.ID)

	viewer := auth.UserSubject{ID: testUserID(t, "viewer")}
	listResult := taskService.List(ctx, viewer, task.PublicListScope{ViewerID: viewer.ID}, task.NoListFilters(), testPage(t, 10, 0))
	listed, matched := listResult.(task.TasksListed)
	if !matched {
		t.Fatalf("list public tasks: want TasksListed, got %#v", listResult)
	}
	if len(listed.Values) != 1 || listed.Values[0].Task.ID != created.Value.ID {
		t.Fatalf("listed public tasks = %+v, want just %v", listed.Values, created.Value.ID)
	}
}
