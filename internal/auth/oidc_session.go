package auth

import (
	"context"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
)

// OpenIDConnectSession retains the provider-signed coordinates required for
// RP-Initiated Logout and Back-Channel Logout. The raw ID token stays on the
// server and is never placed in the Sharecrop browser cookie.
type OpenIDConnectSession struct {
	Provider              string
	Issuer                string
	Subject               string
	SID                   string
	Username              string
	Email                 string
	Role                  string
	RawIDToken            string
	ClientID              string
	EndSessionEndpoint    string
	PostLogoutRedirectURI string
	ExpiresAt             time.Time
}

type OpenIDConnectLogoutClaim struct {
	Provider  string
	Issuer    string
	ClientID  string
	JWTID     string
	ExpiresAt time.Time
	SID       string
	Subject   string
}

type OpenIDConnectFrontchannelLogout struct {
	Provider string
	Issuer   string
	ClientID string
	SID      string
}

type StoreOpenIDConnectSessionResult interface{ storeOpenIDConnectSessionResult() }
type OpenIDConnectSessionStored struct{}
type StoreOpenIDConnectSessionRejected struct{ Reason core.DomainError }

func (OpenIDConnectSessionStored) storeOpenIDConnectSessionResult()        {}
func (StoreOpenIDConnectSessionRejected) storeOpenIDConnectSessionResult() {}

type FindOpenIDConnectSessionResult interface{ findOpenIDConnectSessionResult() }
type OpenIDConnectSessionFound struct{ Session OpenIDConnectSession }
type OpenIDConnectSessionNotFound struct{}
type FindOpenIDConnectSessionRejected struct{ Reason core.DomainError }

func (OpenIDConnectSessionFound) findOpenIDConnectSessionResult()        {}
func (OpenIDConnectSessionNotFound) findOpenIDConnectSessionResult()     {}
func (FindOpenIDConnectSessionRejected) findOpenIDConnectSessionResult() {}

type BackchannelLogoutResult interface{ backchannelLogoutResult() }
type BackchannelLogoutApplied struct{}
type BackchannelLogoutReplay struct{}
type BackchannelLogoutRejected struct{ Reason core.DomainError }

func (BackchannelLogoutApplied) backchannelLogoutResult()  {}
func (BackchannelLogoutReplay) backchannelLogoutResult()   {}
func (BackchannelLogoutRejected) backchannelLogoutResult() {}

type FrontchannelLogoutResult interface{ frontchannelLogoutResult() }
type FrontchannelLogoutApplied struct{}
type FrontchannelLogoutRejected struct{ Reason core.DomainError }

func (FrontchannelLogoutApplied) frontchannelLogoutResult()  {}
func (FrontchannelLogoutRejected) frontchannelLogoutResult() {}

type RotateOpenIDConnectSessionResult interface{ rotateOpenIDConnectSessionResult() }

// OpenIDConnectSessionRotated reports that the session is reachable by the
// replacement refresh token.
type OpenIDConnectSessionRotated struct{ Session OpenIDConnectSession }

// OpenIDConnectSessionNotRotated reports that the browser session was not
// established through the provider, which is the ordinary case for a guest.
type OpenIDConnectSessionNotRotated struct{}
type RotateOpenIDConnectSessionRejected struct{ Reason core.DomainError }

func (OpenIDConnectSessionRotated) rotateOpenIDConnectSessionResult()        {}
func (OpenIDConnectSessionNotRotated) rotateOpenIDConnectSessionResult()     {}
func (RotateOpenIDConnectSessionRejected) rotateOpenIDConnectSessionResult() {}

type OpenIDConnectSessionStore interface {
	StoreOpenIDConnectSession(context.Context, RefreshTokenHash, OpenIDConnectSession) StoreOpenIDConnectSessionResult
	FindOpenIDConnectSession(context.Context, RefreshTokenHash) FindOpenIDConnectSessionResult
	// RotateOpenIDConnectSession states the invariant that refreshing must
	// preserve: a session established through the provider stays reachable by
	// whichever refresh token currently represents the browser session.
	// Refreshing consumes the old token and issues a new one, so a store that
	// addresses sessions by individual token cannot satisfy this without being
	// told; one that addresses them by refresh-token family already does, and
	// only has to confirm it. Without the invariant the provider's end-session
	// coordinates become unreachable after a refresh and the signed-in username
	// disappears from the interface.
	RotateOpenIDConnectSession(ctx context.Context, previous, next RefreshTokenHash) RotateOpenIDConnectSessionResult
	ApplyFrontchannelLogout(context.Context, OpenIDConnectFrontchannelLogout) FrontchannelLogoutResult
	ApplyBackchannelLogout(context.Context, OpenIDConnectLogoutClaim, time.Time) BackchannelLogoutResult
}
