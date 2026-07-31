package webhook

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
)

func TestNewEndpointURLAcceptsAbsoluteHTTPS(t *testing.T) {
	accepted, matched := NewEndpointURL("https://receiver.example.com/hooks?tag=1").(EndpointURLAccepted)
	if !matched {
		t.Fatalf("https url rejected")
	}
	if accepted.Value.String() != "https://receiver.example.com/hooks?tag=1" {
		t.Fatalf("url = %q", accepted.Value.String())
	}
}

func TestNewEndpointURLRejectsBadForms(t *testing.T) {
	cases := map[string]string{
		"http scheme":  "http://receiver.example.com/hooks",
		"no scheme":    "receiver.example.com/hooks",
		"empty":        "",
		"no host":      "https:///hooks",
		"userinfo":     "https://user:pw@receiver.example.com/hooks",
		"weird scheme": "ftp://receiver.example.com/hooks",
	}
	for name, raw := range cases {
		rejected, matched := NewEndpointURL(raw).(EndpointURLRejected)
		if !matched {
			t.Fatalf("%s: url %q was accepted", name, raw)
		}
		if rejected.Reason.Code() != core.ErrorCodeInvalidArgument {
			t.Fatalf("%s: code = %v", name, rejected.Reason.Code())
		}
	}
}

func TestNewSecretShapeAndRoundTrip(t *testing.T) {
	accepted, matched := NewSecret().(SecretAccepted)
	if !matched {
		t.Fatalf("secret generation rejected")
	}
	raw := accepted.Value.String()
	if !strings.HasPrefix(raw, "scrop_whsec_") {
		t.Fatalf("secret = %q, want scrop_whsec_ prefix", raw)
	}
	if len(strings.TrimPrefix(raw, "scrop_whsec_")) != 43 {
		t.Fatalf("secret entropy length = %d, want 43 base64url characters", len(strings.TrimPrefix(raw, "scrop_whsec_")))
	}
	if !HasSecretPrefix(raw) {
		t.Fatalf("HasSecretPrefix rejected a generated secret")
	}
	parsed, parsedMatched := ParseSecret(raw).(SecretAccepted)
	if !parsedMatched || parsed.Value.String() != raw {
		t.Fatalf("secret did not round-trip")
	}
	second := NewSecret().(SecretAccepted)
	if second.Value.String() == raw {
		t.Fatalf("two generated secrets were identical")
	}
}

func TestParseSecretRejectsForeignShapes(t *testing.T) {
	for _, raw := range []string{"", "scrop_org_abcdef", "scrop_whsec_!!!not-base64url!!!"} {
		if _, matched := ParseSecret(raw).(SecretRejected); !matched {
			t.Fatalf("secret %q was accepted", raw)
		}
	}
}

func TestNewKindFilterDeduplicatesAndRejectsEmpty(t *testing.T) {
	accepted, matched := NewKindFilter([]event.Kind{event.KindTaskOpened, event.KindTaskOpened, event.KindTipReceived}).(KindFilterAccepted)
	if !matched {
		t.Fatalf("kind filter rejected")
	}
	if len(accepted.Value.Values()) != 2 {
		t.Fatalf("kinds = %v, want deduplicated pair", accepted.Value.Values())
	}
	if !accepted.Value.Contains(event.KindTaskOpened) || accepted.Value.Contains(event.KindTaskCancelled) {
		t.Fatalf("Contains misreported membership")
	}
	if _, rejected := NewKindFilter([]event.Kind{}).(KindFilterRejected); !rejected {
		t.Fatalf("empty kind filter was accepted")
	}
}

func TestParseStateAndDeliveryState(t *testing.T) {
	if parsed, matched := ParseState("active").(StateAccepted); !matched || parsed.Value != StateActive {
		t.Fatalf("active state parse failed")
	}
	if parsed, matched := ParseState("revoked").(StateAccepted); !matched || parsed.Value != StateRevoked {
		t.Fatalf("revoked state parse failed")
	}
	if _, matched := ParseState("archived").(StateRejected); !matched {
		t.Fatalf("unknown state was accepted")
	}
	for raw, want := range map[string]DeliveryState{"pending": DeliveryStatePending, "delivered": DeliveryStateDelivered, "dead": DeliveryStateDead} {
		parsed, matched := ParseDeliveryState(raw).(DeliveryStateAccepted)
		if !matched || parsed.Value != want {
			t.Fatalf("delivery state %q parse failed", raw)
		}
	}
	if _, matched := ParseDeliveryState("expired").(DeliveryStateRejected); !matched {
		t.Fatalf("unknown delivery state was accepted")
	}
}

// TestRequiredScopeForKindIsTotal walks every kind so a new event kind
// cannot ship without an entitlement decision: the fallback branch maps to
// platform_admin, which no ordinary credential holds, and this test fails
// loudly if a kind lands there.
func TestRequiredScopeForKindIsTotal(t *testing.T) {
	expected := map[string]agent.Scope{
		"task_opened":                  agent.ScopeTasksRead,
		"task_funded":                  agent.ScopeLedgerRead,
		"task_cancelled":               agent.ScopeTasksRead,
		"task_expired":                 agent.ScopeTasksRead,
		"task_commented":               agent.ScopeTasksRead,
		"series_commented":             agent.ScopeTasksRead,
		"reservation_requested":        agent.ScopeTasksRead,
		"reservation_approved":         agent.ScopeTasksRead,
		"reservation_declined":         agent.ScopeTasksRead,
		"reservation_cancelled":        agent.ScopeTasksRead,
		"reservation_expired":          agent.ScopeTasksRead,
		"submission_created":           agent.ScopeSubmissionsRead,
		"submission_accepted":          agent.ScopeSubmissionsRead,
		"submission_changes_requested": agent.ScopeSubmissionsRead,
		"submission_rejected":          agent.ScopeSubmissionsRead,
		"submission_commented":         agent.ScopeSubmissionsRead,
		"payout_received":              agent.ScopeLedgerRead,
		"credit_granted":               agent.ScopeLedgerRead,
		"tip_received":                 agent.ScopeLedgerRead,
		"collectible_awarded":          agent.ScopeCollectiblesRead,
	}
	kinds := event.AllKinds()
	if len(kinds) != len(expected) {
		t.Fatalf("expected table covers %d kinds, event.AllKinds has %d", len(expected), len(kinds))
	}
	for _, kind := range kinds {
		want, listed := expected[kind.String()]
		if !listed {
			t.Fatalf("kind %s has no expected entitlement", kind.String())
		}
		if got := RequiredScopeForKind(kind); got != want {
			t.Fatalf("kind %s maps to %s, want %s", kind.String(), got.String(), want.String())
		}
	}
}

func TestMissingKindEntitlements(t *testing.T) {
	filter := NewKindFilter([]event.Kind{event.KindTaskOpened, event.KindPayoutReceived, event.KindCollectibleAwarded}).(KindFilterAccepted).Value

	full := agent.NewScopeSet([]agent.Scope{agent.ScopeTasksRead, agent.ScopeLedgerRead, agent.ScopeCollectiblesRead})
	if missing := MissingKindEntitlements(full, filter); len(missing) != 0 {
		t.Fatalf("fully entitled scopes reported missing kinds: %v", missing)
	}

	partial := agent.NewScopeSet([]agent.Scope{agent.ScopeTasksRead})
	missing := MissingKindEntitlements(partial, filter)
	if len(missing) != 2 {
		t.Fatalf("missing = %v, want the ledger and collectible kinds", missing)
	}
}

// TestDeliveryBodyWireShape pins the exact delivery body JSON, which is also
// the live feed's event wire shape; internal/http's contract fixtures pin
// the same struct from the other side.
func TestDeliveryBodyWireShape(t *testing.T) {
	eventID := core.ParseDomainEventID("0d9c1c1e-6f5a-4f4e-9e83-111111111111").(core.DomainEventIDCreated).Value
	taskID := core.ParseTaskID("0d9c1c1e-6f5a-4f4e-9e83-222222222222").(core.TaskIDCreated).Value
	actorID := core.ParseUserID("0d9c1c1e-6f5a-4f4e-9e83-333333333333").(core.UserIDCreated).Value
	subject := event.NoSubjectRefs()
	subject.Task = event.TaskSubject{ID: taskID}
	stored := event.StoredEvent{
		Event: event.Event{
			ID:         eventID,
			Kind:       event.KindTaskOpened,
			Actor:      event.ActorUser{ID: actorID},
			Subject:    subject,
			Metadata:   event.TaskMetadata(taskID),
			OccurredAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		},
		Cursor: event.CursorFromSequence(41),
	}
	stored = event.WithoutEnrichment(stored)
	actorName, actorNameOK := auth.NewDisplayName("ada").(auth.DisplayNameAccepted)
	if !actorNameOK {
		t.Fatalf("display name fixture rejected")
	}
	stored.ActorName = event.ActorNamed{DisplayName: actorName.Value}
	stored.TaskTitle = event.TaskTitled{Title: "Label receipts"}

	body, err := EncodeDeliveryBody(stored, "subscription-1")
	if err != nil {
		t.Fatalf("encode delivery body: %v", err)
	}
	want := `{"event":{"id":"0d9c1c1e-6f5a-4f4e-9e83-111111111111","kind":"task_opened","actor_kind":"user","actor_user_id":"0d9c1c1e-6f5a-4f4e-9e83-333333333333","actor_display_name":"ada","occurred_at":"2026-01-02T03:04:05Z","cursor":"41","task_id":"0d9c1c1e-6f5a-4f4e-9e83-222222222222","task_title":"Label receipts","submission_id":"","reservation_id":"","series_id":"","organization_id":"","collectible_id":"","metadata_json":"{\"task_id\":\"0d9c1c1e-6f5a-4f4e-9e83-222222222222\"}"},"subscription_id":"subscription-1"}`
	if string(body) != want {
		t.Fatalf("delivery body = %s, want %s", string(body), want)
	}
}

func TestServiceCreateReturnsSecretOnlyOnce(t *testing.T) {
	service := NewService(NewMemoryStore())
	owner := OwnerUser{ID: core.NewUserID().(core.UserIDCreated).Value}
	endpoint := NewEndpointURL("https://receiver.example.com/hooks").(EndpointURLAccepted).Value
	kinds := NewKindFilter([]event.Kind{event.KindTaskOpened}).(KindFilterAccepted).Value

	created, matched := service.Create(context.Background(), owner, endpoint, kinds, RecipientAudience{}).(SubscriptionCreated)
	if !matched {
		t.Fatalf("create rejected")
	}
	if created.Value.State != StateActive {
		t.Fatalf("created state = %v", created.Value.State)
	}
	if !HasSecretPrefix(created.Secret.String()) {
		t.Fatalf("created secret shape is wrong")
	}

	listed, listedMatched := service.List(context.Background(), owner, core.DefaultPage()).(SubscriptionsListed)
	if !listedMatched || len(listed.Values) != 1 {
		t.Fatalf("list = %#v", listed)
	}
	encoded, err := json.Marshal(listed.Values[0])
	if err != nil {
		t.Fatalf("marshal listed subscription: %v", err)
	}
	if strings.Contains(string(encoded), created.Secret.String()) {
		t.Fatalf("listing carries the secret")
	}

	stranger := OwnerUser{ID: core.NewUserID().(core.UserIDCreated).Value}
	strangerListed := service.List(context.Background(), stranger, core.DefaultPage()).(SubscriptionsListed)
	if len(strangerListed.Values) != 0 {
		t.Fatalf("stranger sees %d subscriptions", len(strangerListed.Values))
	}

	revoked, revokedMatched := service.Revoke(context.Background(), owner, created.Value.ID).(SubscriptionRevoked)
	if !revokedMatched || revoked.Value.State != StateRevoked {
		t.Fatalf("revoke = %#v", revoked)
	}
	if _, again := service.Revoke(context.Background(), owner, created.Value.ID).(RevokeRejected); !again {
		t.Fatalf("second revoke was accepted")
	}
}

func TestValidateAudienceKinds(t *testing.T) {
	taskOpened := NewKindFilter([]event.Kind{event.KindTaskOpened}).(KindFilterAccepted).Value
	mixed := NewKindFilter([]event.Kind{event.KindTaskOpened, event.KindTaskFunded}).(KindFilterAccepted).Value

	if _, ok := ValidateAudienceKinds(NewMarketplaceAudience(), taskOpened).(AudienceKindsAccepted); !ok {
		t.Fatalf("marketplace + task_opened was rejected")
	}
	if _, rejected := ValidateAudienceKinds(NewMarketplaceAudience(), mixed).(AudienceKindsRejected); !rejected {
		t.Fatalf("marketplace + non-task_opened kinds was accepted")
	}
	if _, ok := ValidateAudienceKinds(RecipientAudience{}, mixed).(AudienceKindsAccepted); !ok {
		t.Fatalf("recipient audience must accept every kind filter")
	}
}

func TestNewMinimumCreditRewardRequiresPositive(t *testing.T) {
	for _, amount := range []int64{0, -5} {
		if _, rejected := NewMinimumCreditReward(amount).(MinimumCreditRewardRejected); !rejected {
			t.Fatalf("NewMinimumCreditReward(%d) was accepted", amount)
		}
	}
	accepted, matched := NewMinimumCreditReward(25).(MinimumCreditRewardAccepted)
	if !matched || accepted.Value.Amount() != 25 {
		t.Fatalf("NewMinimumCreditReward(25) did not round-trip")
	}
}

func TestServiceCreateRejectsMarketplaceWithNonTaskOpenedKinds(t *testing.T) {
	service := NewService(NewMemoryStore())
	owner := OwnerUser{ID: core.NewUserID().(core.UserIDCreated).Value}
	endpoint := NewEndpointURL("https://receiver.invalid/hooks").(EndpointURLAccepted).Value
	mixed := NewKindFilter([]event.Kind{event.KindTaskFunded}).(KindFilterAccepted).Value

	if _, rejected := service.Create(context.Background(), owner, endpoint, mixed, NewMarketplaceAudience()).(CreateRejected); !rejected {
		t.Fatalf("marketplace subscription with task_funded kind was created")
	}

	taskOpened := NewKindFilter([]event.Kind{event.KindTaskOpened}).(KindFilterAccepted).Value
	created, matched := service.Create(context.Background(), owner, endpoint, taskOpened, NewMarketplaceAudience()).(SubscriptionCreated)
	if !matched {
		t.Fatalf("marketplace subscription with task_opened was rejected")
	}
	if _, marketplace := created.Value.Audience.(MarketplaceAudience); !marketplace {
		t.Fatalf("created subscription audience = %T, want MarketplaceAudience", created.Value.Audience)
	}
}
