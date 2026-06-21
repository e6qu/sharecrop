package auth

import (
	"context"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
)

type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}

type CredentialRecord struct {
	UserID       core.UserID
	Email        EmailAddress
	PasswordHash PasswordHash
}

type RefreshTokenRecord struct {
	ID        core.RefreshTokenID
	Subject   Subject
	Hash      RefreshTokenHash
	ExpiresAt time.Time
}

type Store interface {
	CreateUserCredential(context.Context, core.UserID, EmailAddress, PasswordHash) StoreUserResult
	FindCredentialByEmail(context.Context, EmailAddress) CredentialLookupResult
	CreateGuestSubject(context.Context, core.GuestID) StoreGuestResult
	StoreRefreshToken(context.Context, RefreshTokenRecord) StoreRefreshTokenResult
	ConsumeRefreshToken(context.Context, RefreshTokenHash, time.Time) ConsumeRefreshTokenResult
}

type Service struct {
	store       Store
	tokenSecret AccessTokenSecret
	clock       Clock
}

type ServiceResult interface {
	serviceResult()
}

type ServiceCreated struct {
	Value Service
}

type ServiceRejected struct {
	Reason core.DomainError
}

func (ServiceCreated) serviceResult() {}

func (ServiceRejected) serviceResult() {}

func NewService(store Store, tokenSecret AccessTokenSecret, clock Clock) ServiceResult {
	return ServiceCreated{
		Value: Service{
			store:       store,
			tokenSecret: tokenSecret,
			clock:       clock,
		},
	}
}

type RegisterResult interface {
	registerResult()
}

type RegisterAccepted struct {
	Subject      UserSubject
	AccessToken  AccessToken
	RefreshToken RefreshTokenPlain
}

type RegisterRejected struct {
	Reason core.DomainError
}

func (RegisterAccepted) registerResult() {}

func (RegisterRejected) registerResult() {}

func (service Service) Register(ctx context.Context, email EmailAddress, password PasswordSecret) RegisterResult {
	userResult := core.NewUserID()
	userCreated, userMatched := userResult.(core.UserIDCreated)
	if !userMatched {
		rejected := userResult.(core.UserIDRejected)
		return RegisterRejected{Reason: rejected.Reason}
	}

	hashResult := HashPassword(password)
	hashCreated, hashMatched := hashResult.(PasswordHashCreated)
	if !hashMatched {
		rejected := hashResult.(PasswordHashRejected)
		return RegisterRejected{Reason: rejected.Reason}
	}

	storeResult := service.store.CreateUserCredential(ctx, userCreated.Value, email, hashCreated.Value)
	if rejected, matched := storeResult.(StoreUserRejected); matched {
		return RegisterRejected{Reason: rejected.Reason}
	}

	sessionResult := service.issueUserSession(ctx, userCreated.Value)
	sessionAccepted, sessionMatched := sessionResult.(UserSessionIssued)
	if !sessionMatched {
		rejected := sessionResult.(UserSessionRejected)
		return RegisterRejected{Reason: rejected.Reason}
	}

	return RegisterAccepted{
		Subject:      sessionAccepted.Subject,
		AccessToken:  sessionAccepted.AccessToken,
		RefreshToken: sessionAccepted.RefreshToken,
	}
}

type LoginResult interface {
	loginResult()
}

type LoginAccepted struct {
	Subject      UserSubject
	AccessToken  AccessToken
	RefreshToken RefreshTokenPlain
}

type LoginRejected struct {
	Reason core.DomainError
}

func (LoginAccepted) loginResult() {}

func (LoginRejected) loginResult() {}

func (service Service) Login(ctx context.Context, email EmailAddress, password PasswordSecret) LoginResult {
	lookupResult := service.store.FindCredentialByEmail(ctx, email)
	found, foundMatched := lookupResult.(CredentialFound)
	if !foundMatched {
		return LoginRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "credentials are invalid")}
	}

	verification := VerifyPassword(password, found.Record.PasswordHash)
	if _, matched := verification.(PasswordAccepted); !matched {
		return LoginRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "credentials are invalid")}
	}

	sessionResult := service.issueUserSession(ctx, found.Record.UserID)
	sessionAccepted, sessionMatched := sessionResult.(UserSessionIssued)
	if !sessionMatched {
		rejected := sessionResult.(UserSessionRejected)
		return LoginRejected{Reason: rejected.Reason}
	}

	return LoginAccepted{
		Subject:      sessionAccepted.Subject,
		AccessToken:  sessionAccepted.AccessToken,
		RefreshToken: sessionAccepted.RefreshToken,
	}
}

type GuestResult interface {
	guestResult()
}

type GuestAccepted struct {
	Subject      GuestSubject
	AccessToken  AccessToken
	RefreshToken RefreshTokenPlain
}

type GuestRejected struct {
	Reason core.DomainError
}

func (GuestAccepted) guestResult() {}

func (GuestRejected) guestResult() {}

func (service Service) CreateGuest(ctx context.Context) GuestResult {
	guestResult := core.NewGuestID()
	guestCreated, guestMatched := guestResult.(core.GuestIDCreated)
	if !guestMatched {
		rejected := guestResult.(core.GuestIDRejected)
		return GuestRejected{Reason: rejected.Reason}
	}

	storeResult := service.store.CreateGuestSubject(ctx, guestCreated.Value)
	if rejected, matched := storeResult.(StoreGuestRejected); matched {
		return GuestRejected{Reason: rejected.Reason}
	}

	sessionResult := service.issueGuestSession(ctx, guestCreated.Value)
	sessionAccepted, sessionMatched := sessionResult.(GuestSessionIssued)
	if !sessionMatched {
		rejected := sessionResult.(GuestSessionRejected)
		return GuestRejected{Reason: rejected.Reason}
	}

	return GuestAccepted{
		Subject:      sessionAccepted.Subject,
		AccessToken:  sessionAccepted.AccessToken,
		RefreshToken: sessionAccepted.RefreshToken,
	}
}

type RefreshResult interface {
	refreshResult()
}

type RefreshAccepted struct {
	Subject      Subject
	AccessToken  AccessToken
	RefreshToken RefreshTokenPlain
}

type RefreshRejected struct {
	Reason core.DomainError
}

func (RefreshAccepted) refreshResult() {}

func (RefreshRejected) refreshResult() {}

func (service Service) Refresh(ctx context.Context, refreshToken RefreshTokenPlain) RefreshResult {
	hash := HashRefreshToken(refreshToken)
	consumed := service.store.ConsumeRefreshToken(ctx, hash, service.clock.Now())
	accepted, matched := consumed.(RefreshTokenConsumed)
	if !matched {
		return RefreshRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "refresh token is invalid")}
	}

	switch subject := accepted.Subject.(type) {
	case UserSubject:
		userResult := service.issueUserSession(ctx, subject.ID)
		userAccepted, userMatched := userResult.(UserSessionIssued)
		if !userMatched {
			rejected := userResult.(UserSessionRejected)
			return RefreshRejected{Reason: rejected.Reason}
		}
		return RefreshAccepted{Subject: userAccepted.Subject, AccessToken: userAccepted.AccessToken, RefreshToken: userAccepted.RefreshToken}
	case GuestSubject:
		guestResult := service.issueGuestSession(ctx, subject.ID)
		guestAccepted, guestMatched := guestResult.(GuestSessionIssued)
		if !guestMatched {
			rejected := guestResult.(GuestSessionRejected)
			return RefreshRejected{Reason: rejected.Reason}
		}
		return RefreshAccepted{Subject: guestAccepted.Subject, AccessToken: guestAccepted.AccessToken, RefreshToken: guestAccepted.RefreshToken}
	default:
		return RefreshRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "refresh token subject is invalid")}
	}
}

type UserSessionResult interface {
	userSessionResult()
}

type UserSessionIssued struct {
	Subject      UserSubject
	AccessToken  AccessToken
	RefreshToken RefreshTokenPlain
}

type UserSessionRejected struct {
	Reason core.DomainError
}

func (UserSessionIssued) userSessionResult() {}

func (UserSessionRejected) userSessionResult() {}

func (service Service) issueUserSession(ctx context.Context, id core.UserID) UserSessionResult {
	subject := UserSubject{ID: id}
	accessResult := SignAccessToken(service.tokenSecret, subject, service.clock.Now())
	accessAccepted, accessMatched := accessResult.(AccessTokenAccepted)
	if !accessMatched {
		rejected := accessResult.(AccessTokenRejected)
		return UserSessionRejected{Reason: rejected.Reason}
	}

	refreshResult := service.storeNewRefreshToken(ctx, subject)
	refreshCreated, refreshMatched := refreshResult.(RefreshTokenCreated)
	if !refreshMatched {
		rejected := refreshResult.(RefreshTokenRejected)
		return UserSessionRejected{Reason: rejected.Reason}
	}

	return UserSessionIssued{Subject: subject, AccessToken: accessAccepted.Value, RefreshToken: refreshCreated.Value.Plain}
}

type GuestSessionResult interface {
	guestSessionResult()
}

type GuestSessionIssued struct {
	Subject      GuestSubject
	AccessToken  AccessToken
	RefreshToken RefreshTokenPlain
}

type GuestSessionRejected struct {
	Reason core.DomainError
}

func (GuestSessionIssued) guestSessionResult() {}

func (GuestSessionRejected) guestSessionResult() {}

func (service Service) issueGuestSession(ctx context.Context, id core.GuestID) GuestSessionResult {
	subject := GuestSubject{ID: id}
	accessResult := SignAccessToken(service.tokenSecret, subject, service.clock.Now())
	accessAccepted, accessMatched := accessResult.(AccessTokenAccepted)
	if !accessMatched {
		rejected := accessResult.(AccessTokenRejected)
		return GuestSessionRejected{Reason: rejected.Reason}
	}

	refreshResult := service.storeNewRefreshToken(ctx, subject)
	refreshCreated, refreshMatched := refreshResult.(RefreshTokenCreated)
	if !refreshMatched {
		rejected := refreshResult.(RefreshTokenRejected)
		return GuestSessionRejected{Reason: rejected.Reason}
	}

	return GuestSessionIssued{Subject: subject, AccessToken: accessAccepted.Value, RefreshToken: refreshCreated.Value.Plain}
}

func (service Service) storeNewRefreshToken(ctx context.Context, subject Subject) RefreshTokenIssueResult {
	refreshResult := NewRefreshToken(service.clock.Now())
	refreshCreated, refreshMatched := refreshResult.(RefreshTokenCreated)
	if !refreshMatched {
		return refreshResult
	}

	storeResult := service.store.StoreRefreshToken(ctx, RefreshTokenRecord{
		ID:        refreshCreated.Value.ID,
		Subject:   subject,
		Hash:      refreshCreated.Value.Hash,
		ExpiresAt: refreshCreated.Value.ExpiresAt,
	})
	if rejected, matched := storeResult.(StoreRefreshTokenRejected); matched {
		return RefreshTokenRejected{Reason: rejected.Reason}
	}

	return refreshCreated
}
