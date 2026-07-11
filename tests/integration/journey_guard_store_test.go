//go:build integration

package integration_test

// Integration tests for the real-backend fixes from the journey correctness
// review: unknown tasks report not_found, cancel/refund are guarded while a
// submission is pending review, self-deactivation is guarded while the user
// still holds funded task rewards, and member provisioning reports clear
// not-found/duplicate errors.

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestTaskStoreFindTaskUnknownIsNotFound(t *testing.T) {
	pool := newPool(t)
	store := db.NewTaskStore(pool)

	result := store.FindTask(context.Background(), newTaskID(t))
	rejected, matched := result.(task.FindTaskStoreRejected)
	if !matched {
		t.Fatalf("find unknown task = %T, want FindTaskStoreRejected", result)
	}
	if rejected.Reason.Code() != core.ErrorCodeNotFound {
		t.Fatalf("find unknown task code = %v, want not_found", rejected.Reason.Code())
	}
}

func TestTaskStoreCancelWithPendingReviewSucceeds(t *testing.T) {
	pool := newPool(t)
	store := db.NewTaskStore(pool)
	owner := createUser(t, pool, "integration-cancel-guard-owner")
	worker := createUser(t, pool, "integration-cancel-guard-worker")

	taskID := insertTaskWithRewardKind(t, pool, owner, "open", "none")
	// A visibility scope is required for the task to be re-read after the state
	// change (ChangeTaskState returns the updated task).
	if _, err := pool.Exec(context.Background(), "insert into task_visibility_scopes (task_id, visibility_kind, scope_key) values ($1, 'public', 'public')", taskID.String()); err != nil {
		t.Fatalf("insert task visibility scope: %v", err)
	}
	insertSubmission(t, pool, taskID, worker)

	// Cancelling is allowed even with a submission pending review; the task is
	// simply cancelled (unfunded here, so nothing to settle).
	result := store.ChangeTaskState(context.Background(), taskID, task.StateCancelled)
	if _, matched := result.(task.ChangeTaskStateStoreAccepted); !matched {
		t.Fatalf("cancel with pending review = %T, want ChangeTaskStateStoreAccepted", result)
	}
}

func TestLedgerStoreRefundWithPendingReviewSucceeds(t *testing.T) {
	pool := newPool(t)
	store := db.NewLedgerStore(pool)
	owner := createUser(t, pool, "integration-refund-guard-owner")
	worker := createUser(t, pool, "integration-refund-guard-worker")

	taskID := insertTask(t, pool, owner, "draft", 30)
	if _, matched := store.FundTask(context.Background(), fundCommand(t, owner, taskID, 30, "fund-guard-"+taskID.String())).(ledger.TaskFunded); !matched {
		t.Fatalf("fund task failed")
	}
	setTaskState(t, pool, taskID, "open")
	insertSubmission(t, pool, taskID, worker)

	// Refunds are auto-granted: the owner can refund even with a submission
	// pending review. The allocated credits return to the owner's spendable
	// balance and the worker is not paid.
	result := store.RefundTask(context.Background(), refundCommand(t, owner, taskID, "refund-guard-"+taskID.String()))
	if _, matched := result.(ledger.TaskRefunded); !matched {
		t.Fatalf("refund with pending review = %T, want TaskRefunded", result)
	}
	if balance := mustBalance(t, store, owner); balance.Spendable() != 100 || balance.Allocated() != 0 {
		t.Fatalf("owner balance after refund = (spendable %d, allocated %d), want (100, 0)", balance.Spendable(), balance.Allocated())
	}
	if balance := mustBalance(t, store, worker); balance.Spendable() != 100 {
		t.Fatalf("worker balance after refund = %d, want 100 (not paid)", balance.Spendable())
	}
}

func TestAuthStoreDeactivateRejectsWhileHoldingFundedRewards(t *testing.T) {
	pool := newPool(t)
	authStore := db.NewAuthStore(pool)
	ledgerStore := db.NewLedgerStore(pool)
	owner := createUser(t, pool, "integration-deactivate-guard")

	taskID := insertTask(t, pool, owner, "draft", 30)
	if _, matched := ledgerStore.FundTask(context.Background(), fundCommand(t, owner, taskID, 30, "fund-deactivate-"+taskID.String())).(ledger.TaskFunded); !matched {
		t.Fatalf("fund task failed")
	}

	deactivateResult := authStore.DeactivateUser(context.Background(), owner)
	rejected, matched := deactivateResult.(auth.AccountMutationRejected)
	if !matched {
		t.Fatalf("deactivate with held escrow = %T, want AccountMutationRejected", deactivateResult)
	}
	if rejected.Reason.Code() != core.ErrorCodeConflict {
		t.Fatalf("deactivate rejection code = %v, want conflict", rejected.Reason.Code())
	}

	if _, matched := ledgerStore.RefundTask(context.Background(), refundCommand(t, owner, taskID, "refund-deactivate-"+taskID.String())).(ledger.TaskRefunded); !matched {
		t.Fatalf("refund failed")
	}
	if _, matched := authStore.DeactivateUser(context.Background(), owner).(auth.AccountMutationAccepted); !matched {
		t.Fatalf("deactivate after refund: want AccountMutationAccepted")
	}
}

func TestOrgStoreProvisionMemberNotFoundAndDuplicateErrors(t *testing.T) {
	pool := newPool(t)
	store := db.NewOrgStore(pool)
	ctx := context.Background()
	creator := createUser(t, pool, "integration-provision-creator")
	member := createUser(t, pool, "integration-provision-member")

	organizationID := core.NewOrganizationID().(core.OrganizationIDCreated).Value
	creatorMembershipID := core.NewOrganizationMembershipID().(core.OrganizationMembershipIDCreated).Value
	name := org.NewOrganizationName("Integration Provision Org").(org.OrganizationNameAccepted).Value
	if _, matched := store.CreateOrganization(ctx, organizationID, name, creator, creatorMembershipID).(org.CreateOrganizationStoreAccepted); !matched {
		t.Fatalf("create organization failed")
	}

	unknownEmail := auth.NewEmailAddress("nobody-" + organizationID.String() + "@example.com").(auth.EmailAddressAccepted).Value
	unknownResult := store.ProvisionMember(ctx, core.NewOrganizationMembershipID().(core.OrganizationMembershipIDCreated).Value, organizationID, unknownEmail, []org.Role{org.RoleMember})
	unknownRejected, matched := unknownResult.(org.ProvisionMemberStoreRejected)
	if !matched {
		t.Fatalf("provision unknown email = %T, want ProvisionMemberStoreRejected", unknownResult)
	}
	if unknownRejected.Reason.Code() != core.ErrorCodeNotFound {
		t.Fatalf("unknown email rejection code = %v, want not_found", unknownRejected.Reason.Code())
	}

	memberEmail := lookupUserEmail(t, pool, member)
	if _, matched := store.ProvisionMember(ctx, core.NewOrganizationMembershipID().(core.OrganizationMembershipIDCreated).Value, organizationID, memberEmail, []org.Role{org.RoleMember}).(org.MemberProvisioned); !matched {
		t.Fatalf("first provision failed")
	}
	duplicateResult := store.ProvisionMember(ctx, core.NewOrganizationMembershipID().(core.OrganizationMembershipIDCreated).Value, organizationID, memberEmail, []org.Role{org.RoleMember})
	duplicateRejected, matched := duplicateResult.(org.ProvisionMemberStoreRejected)
	if !matched {
		t.Fatalf("duplicate provision = %T, want ProvisionMemberStoreRejected", duplicateResult)
	}
	if duplicateRejected.Reason.Code() != core.ErrorCodeConflict {
		t.Fatalf("duplicate provision code = %v, want conflict", duplicateRejected.Reason.Code())
	}
	if duplicateRejected.Reason.Description() != "user is already a member of this organization" {
		t.Fatalf("duplicate provision = %q, want clear duplicate-member message", duplicateRejected.Reason.Description())
	}
}

func lookupUserEmail(t *testing.T, pool *pgxpool.Pool, userID core.UserID) auth.EmailAddress {
	t.Helper()
	var raw string
	if err := pool.QueryRow(context.Background(), "select email from users where id = $1", userID.String()).Scan(&raw); err != nil {
		t.Fatalf("read user email: %v", err)
	}
	return auth.NewEmailAddress(raw).(auth.EmailAddressAccepted).Value
}
