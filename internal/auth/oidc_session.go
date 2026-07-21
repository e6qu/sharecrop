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

type OpenIDConnectSessionStore interface {
	StoreOpenIDConnectSession(context.Context, RefreshTokenHash, OpenIDConnectSession) StoreOpenIDConnectSessionResult
	FindOpenIDConnectSession(context.Context, RefreshTokenHash) FindOpenIDConnectSessionResult
	ApplyFrontchannelLogout(context.Context, OpenIDConnectFrontchannelLogout) FrontchannelLogoutResult
	ApplyBackchannelLogout(context.Context, OpenIDConnectLogoutClaim, time.Time) BackchannelLogoutResult
}
