package assets

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/event/eventtest"
)

func assetsRecipientsContain(recipients event.Recipients, user core.UserID) bool {
	for _, value := range recipients.Users {
		if value == user {
			return true
		}
	}
	return false
}

func newCollectibleID(t *testing.T) core.CollectibleID {
	t.Helper()
	created, matched := core.NewCollectibleID().(core.CollectibleIDCreated)
	if !matched {
		t.Fatalf("collectible id rejected")
	}
	return created.Value
}

func TestGiftCollectibleEmitsCollectibleAwarded(t *testing.T) {
	events := eventtest.NewCapturingStore()
	store := &memoryStore{events: events}
	service := NewService(store, eventtest.RecorderOver(events))
	from := newUserID(t)
	to := newUserID(t)
	collectible := newCollectibleID(t)

	if _, matched := service.GiftCollectible(context.Background(), from, to, collectible).(CollectibleGifted); !matched {
		t.Fatalf("gift rejected")
	}
	appended := events.Appended()
	if len(appended) != 1 || appended[0].Kind != event.KindCollectibleAwarded {
		t.Fatalf("emitted %d events, want one collectible_awarded", len(appended))
	}
	subject, matched := appended[0].Subject.Collectible.(event.CollectibleSubject)
	if !matched || subject.ID != collectible {
		t.Fatalf("collectible subject did not carry the gifted collectible")
	}
	recipients := events.RecipientsAt(0)
	if !assetsRecipientsContain(recipients, to) || !assetsRecipientsContain(recipients, from) {
		t.Fatalf("collectible_awarded recipients = %v, want recipient and gifter", recipients.Users)
	}
}

func TestWithdrawCollectibleEmitsCollectibleWithdrawn(t *testing.T) {
	events := eventtest.NewCapturingStore()
	store := &memoryStore{events: events}
	service := NewService(store, eventtest.RecorderOver(events))
	admin := newUserID(t)
	collectible := newCollectibleID(t)

	if _, matched := service.WithdrawCollectible(context.Background(), admin, collectible).(CollectibleWithdrawn); !matched {
		t.Fatalf("withdraw rejected")
	}
	appended := events.Appended()
	if len(appended) != 1 || appended[0].Kind != event.KindCollectibleWithdrawn {
		t.Fatalf("emitted %d events, want one collectible_withdrawn", len(appended))
	}
}

func TestReleaseCollectibleEmitsCollectibleReleased(t *testing.T) {
	events := eventtest.NewCapturingStore()
	store := &memoryStore{events: events}
	service := NewService(store, eventtest.RecorderOver(events))
	admin := newUserID(t)
	collectible := newCollectibleID(t)

	released, matched := service.ReleaseCollectible(context.Background(), admin, collectible).(CollectibleReleased)
	if !matched {
		t.Fatalf("release rejected")
	}
	if released.Value.State != CollectibleStateMinted {
		t.Fatalf("released state = %q, want minted", released.Value.State.String())
	}
	appended := events.Appended()
	if len(appended) != 1 || appended[0].Kind != event.KindCollectibleReleased {
		t.Fatalf("emitted %d events, want one collectible_released", len(appended))
	}
	subject, subjectMatched := appended[0].Subject.Collectible.(event.CollectibleSubject)
	if !subjectMatched || subject.ID != collectible {
		t.Fatalf("collectible subject did not carry the released collectible")
	}
	actor, actorMatched := appended[0].Actor.(event.ActorUser)
	if !actorMatched || actor.ID != admin {
		t.Fatalf("release actor = %v, want the acting admin", appended[0].Actor)
	}
	if !assetsRecipientsContain(events.RecipientsAt(0), admin) {
		t.Fatalf("collectible_released recipients missed the acting admin")
	}
}

func TestAwardOrganizationCollectibleEmitsCollectibleAwarded(t *testing.T) {
	events := eventtest.NewCapturingStore()
	store := &memoryStore{events: events}
	service := NewService(store, eventtest.RecorderOver(events))
	organizationID := newOrganizationID(t)
	recipient := newUserID(t)
	collectible := newCollectibleID(t)

	if _, matched := service.AwardOrganizationCollectible(context.Background(), organizationID, collectible, recipient).(CollectibleGifted); !matched {
		t.Fatalf("award rejected")
	}
	appended := events.Appended()
	if len(appended) != 1 || appended[0].Kind != event.KindCollectibleAwarded {
		t.Fatalf("emitted %d events, want one collectible_awarded", len(appended))
	}
	organizationSubject, matched := appended[0].Subject.Organization.(event.OrganizationSubject)
	if !matched || organizationSubject.ID != organizationID {
		t.Fatalf("award event did not carry its organization subject")
	}
	if _, matched := appended[0].Actor.(event.ActorSystem); !matched {
		t.Fatalf("award actor = %T, want the system actor (no user identity reaches the service)", appended[0].Actor)
	}
	if !assetsRecipientsContain(events.RecipientsAt(0), recipient) {
		t.Fatalf("collectible_awarded recipients missed the awarded member")
	}
}
