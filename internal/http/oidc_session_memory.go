package httpserver

import (
	"context"
	"sync"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
)

type memoryOpenIDConnectSessionStore struct {
	mu       sync.Mutex
	sessions map[string]auth.OpenIDConnectSession
	claims   map[[4]string]time.Time
}

func newMemoryOpenIDConnectSessionStore() *memoryOpenIDConnectSessionStore {
	return &memoryOpenIDConnectSessionStore{sessions: map[string]auth.OpenIDConnectSession{}, claims: map[[4]string]time.Time{}}
}

func (store *memoryOpenIDConnectSessionStore) StoreOpenIDConnectSession(_ context.Context, hash auth.RefreshTokenHash, session auth.OpenIDConnectSession) auth.StoreOpenIDConnectSessionResult {
	store.mu.Lock()
	store.sessions[hash.String()] = session
	store.mu.Unlock()
	return auth.OpenIDConnectSessionStored{}
}

func (store *memoryOpenIDConnectSessionStore) FindOpenIDConnectSession(_ context.Context, hash auth.RefreshTokenHash) auth.FindOpenIDConnectSessionResult {
	store.mu.Lock()
	defer store.mu.Unlock()
	session, found := store.sessions[hash.String()]
	if !found {
		return auth.OpenIDConnectSessionNotFound{}
	}
	return auth.OpenIDConnectSessionFound{Session: session}
}

func (store *memoryOpenIDConnectSessionStore) ApplyBackchannelLogout(_ context.Context, claim auth.OpenIDConnectLogoutClaim, now time.Time) auth.BackchannelLogoutResult {
	key := [4]string{claim.Provider, claim.Issuer, claim.ClientID, claim.JWTID}
	store.mu.Lock()
	defer store.mu.Unlock()
	for replayKey, expiry := range store.claims {
		if !expiry.After(now) {
			delete(store.claims, replayKey)
		}
	}
	if _, replayed := store.claims[key]; replayed {
		return auth.BackchannelLogoutReplay{}
	}
	store.claims[key] = claim.ExpiresAt
	for hash, session := range store.sessions {
		if session.Provider != claim.Provider || session.Issuer != claim.Issuer || session.ClientID != claim.ClientID {
			continue
		}
		if (claim.SID != "" && session.SID == claim.SID) || (claim.SID == "" && session.Subject == claim.Subject) {
			delete(store.sessions, hash)
		}
	}
	return auth.BackchannelLogoutApplied{}
}

func (store *memoryOpenIDConnectSessionStore) ApplyFrontchannelLogout(_ context.Context, claim auth.OpenIDConnectFrontchannelLogout) auth.FrontchannelLogoutResult {
	store.mu.Lock()
	defer store.mu.Unlock()
	for hash, session := range store.sessions {
		if session.Provider == claim.Provider && session.Issuer == claim.Issuer && session.ClientID == claim.ClientID && session.SID == claim.SID {
			delete(store.sessions, hash)
		}
	}
	return auth.FrontchannelLogoutApplied{}
}
