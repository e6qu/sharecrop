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
	Status       string
}

type RefreshTokenRecord struct {
	ID        core.RefreshTokenID
	FamilyID  core.RefreshTokenID
	Subject   Subject
	Hash      RefreshTokenHash
	ExpiresAt time.Time
}

// refreshFamilySelection decides whether a newly issued refresh token starts a
// new token family (initial login, registration, or guest creation) or
// continues an existing family (rotation during refresh).
type refreshFamilySelection interface {
	refreshFamilySelection()
}

type newRefreshFamily struct{}

type continueRefreshFamily struct {
	id core.RefreshTokenID
}

func (newRefreshFamily) refreshFamilySelection() {}

func (continueRefreshFamily) refreshFamilySelection() {}

type Store interface {
	CreateUserCredential(context.Context, core.UserID, EmailAddress, PasswordHash) StoreUserResult
	FindCredentialByEmail(context.Context, EmailAddress) CredentialLookupResult
	FindCredentialByUserID(context.Context, core.UserID) CredentialLookupResult
	ListUsers(context.Context, string, core.Page) UserDirectoryResult
	UpdateUserEmail(context.Context, core.UserID, EmailAddress) AccountMutationResult
	UpdatePassword(context.Context, core.UserID, PasswordHash) AccountMutationResult
	DeactivateUser(context.Context, core.UserID) AccountMutationResult
	CreateGuestSubject(context.Context, core.GuestID) StoreGuestResult
	StoreRefreshToken(context.Context, RefreshTokenRecord) StoreRefreshTokenResult
	ConsumeRefreshToken(context.Context, RefreshTokenHash, time.Time) ConsumeRefreshTokenResult
	RevokeRefreshFamily(context.Context, RefreshTokenHash) RevokeRefreshFamilyResult
	StoreAccountToken(context.Context, core.UserID, AccountTokenKind, AccountToken) AccountTokenStoreResult
	ConsumeAccountToken(context.Context, AccountTokenKind, AccountTokenHash, time.Time) AccountTokenConsumeResult
}

type RevokeRefreshFamilyResult interface {
	revokeRefreshFamilyResult()
}

type RefreshFamilyRevoked struct{}

type RevokeRefreshFamilyRejected struct {
	Reason core.DomainError
}

func (RefreshFamilyRevoked) revokeRefreshFamilyResult()        {}
func (RevokeRefreshFamilyRejected) revokeRefreshFamilyResult() {}

type LogoutResult interface {
	logoutResult()
}

type LogoutDone struct{}

type LogoutRejected struct {
	Reason core.DomainError
}

func (LogoutDone) logoutResult()     {}
func (LogoutRejected) logoutResult() {}

// Logout revokes the whole session family for the presented refresh token so the
// session cannot be resumed even if the token is replayed later. Revoking an
// unknown or already-revoked token is a no-op.
func (service Service) Logout(ctx context.Context, refreshToken RefreshTokenPlain) LogoutResult {
	result := service.store.RevokeRefreshFamily(ctx, HashRefreshToken(refreshToken))
	if _, done := result.(RefreshFamilyRevoked); !done {
		return LogoutRejected{Reason: result.(RevokeRefreshFamilyRejected).Reason}
	}
	return LogoutDone{}
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

	sessionResult := service.issueUserSession(ctx, userCreated.Value, newRefreshFamily{})
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
	if found.Record.Status != "active" {
		return LoginRejected{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "account is deactivated")}
	}

	verification := VerifyPassword(password, found.Record.PasswordHash)
	if _, matched := verification.(PasswordAccepted); !matched {
		return LoginRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "credentials are invalid")}
	}

	sessionResult := service.issueUserSession(ctx, found.Record.UserID, newRefreshFamily{})
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

func (service Service) ListUsers(ctx context.Context, query string, page core.Page) UserDirectoryResult {
	return service.store.ListUsers(ctx, query, page)
}

type AccountTokenIssueResult interface {
	accountTokenIssueResult()
}

type AccountTokenIssued struct {
	Token AccountTokenPlain
}

type AccountTokenIssueRejected struct {
	Reason core.DomainError
}

func (AccountTokenIssued) accountTokenIssueResult() {}

func (AccountTokenIssueRejected) accountTokenIssueResult() {}

func (service Service) RequestEmailVerification(ctx context.Context, userID core.UserID) AccountTokenIssueResult {
	return service.issueAccountToken(ctx, userID, AccountTokenKindEmailVerification)
}

func (service Service) RequestPasswordReset(ctx context.Context, email EmailAddress) AccountTokenIssueResult {
	lookup := service.store.FindCredentialByEmail(ctx, email)
	found, matched := lookup.(CredentialFound)
	if rejected, rejectedMatched := lookup.(CredentialLookupRejected); rejectedMatched {
		return AccountTokenIssueRejected{Reason: rejected.Reason}
	}
	if !matched {
		return AccountTokenIssueRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "account was not found")}
	}
	return service.issueAccountToken(ctx, found.Record.UserID, AccountTokenKindPasswordReset)
}

func (service Service) issueAccountToken(ctx context.Context, userID core.UserID, kind AccountTokenKind) AccountTokenIssueResult {
	tokenResult := NewAccountToken(service.clock.Now(), kind)
	tokenCreated, matched := tokenResult.(AccountTokenCreated)
	if !matched {
		return AccountTokenIssueRejected{Reason: tokenResult.(AccountTokenRejected).Reason}
	}
	storeResult := service.store.StoreAccountToken(ctx, userID, kind, tokenCreated.Value)
	if _, ok := storeResult.(AccountTokenStored); !ok {
		return AccountTokenIssueRejected{Reason: storeResult.(AccountTokenStoreRejected).Reason}
	}
	return AccountTokenIssued{Token: tokenCreated.Value.Plain}
}

type AccountActionResult interface {
	accountActionResult()
}

type AccountActionAccepted struct{}

type AccountActionRejected struct {
	Reason core.DomainError
}

func (AccountActionAccepted) accountActionResult() {}

func (AccountActionRejected) accountActionResult() {}

func (service Service) VerifyEmail(ctx context.Context, token AccountTokenPlain) AccountActionResult {
	consumed := service.store.ConsumeAccountToken(ctx, AccountTokenKindEmailVerification, HashAccountToken(token), service.clock.Now())
	if _, matched := consumed.(AccountTokenConsumed); !matched {
		return accountActionRejection(consumed)
	}
	return AccountActionAccepted{}
}

func (service Service) ResetPassword(ctx context.Context, token AccountTokenPlain, password PasswordSecret) AccountActionResult {
	consumed := service.store.ConsumeAccountToken(ctx, AccountTokenKindPasswordReset, HashAccountToken(token), service.clock.Now())
	tokenConsumed, matched := consumed.(AccountTokenConsumed)
	if !matched {
		return accountActionRejection(consumed)
	}
	hashResult := HashPassword(password)
	hashCreated, hashMatched := hashResult.(PasswordHashCreated)
	if !hashMatched {
		return AccountActionRejected{Reason: hashResult.(PasswordHashRejected).Reason}
	}
	update := service.store.UpdatePassword(ctx, tokenConsumed.UserID, hashCreated.Value)
	if _, accepted := update.(AccountMutationAccepted); !accepted {
		return AccountActionRejected{Reason: update.(AccountMutationRejected).Reason}
	}
	return AccountActionAccepted{}
}

func (service Service) ChangePassword(ctx context.Context, userID core.UserID, current PasswordSecret, next PasswordSecret) AccountActionResult {
	lookup := service.store.FindCredentialByUserID(ctx, userID)
	found, matched := lookup.(CredentialFound)
	if rejected, rejectedMatched := lookup.(CredentialLookupRejected); rejectedMatched {
		return AccountActionRejected{Reason: rejected.Reason}
	}
	if !matched {
		return AccountActionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "account was not found")}
	}
	if _, ok := VerifyPassword(current, found.Record.PasswordHash).(PasswordAccepted); !ok {
		return AccountActionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "current password is invalid")}
	}
	hashResult := HashPassword(next)
	hashCreated, hashMatched := hashResult.(PasswordHashCreated)
	if !hashMatched {
		return AccountActionRejected{Reason: hashResult.(PasswordHashRejected).Reason}
	}
	update := service.store.UpdatePassword(ctx, userID, hashCreated.Value)
	if _, accepted := update.(AccountMutationAccepted); !accepted {
		return AccountActionRejected{Reason: update.(AccountMutationRejected).Reason}
	}
	return AccountActionAccepted{}
}

func (service Service) UpdateProfile(ctx context.Context, userID core.UserID, email EmailAddress) AccountActionResult {
	update := service.store.UpdateUserEmail(ctx, userID, email)
	if _, accepted := update.(AccountMutationAccepted); !accepted {
		return AccountActionRejected{Reason: update.(AccountMutationRejected).Reason}
	}
	return AccountActionAccepted{}
}

func (service Service) DeactivateAccount(ctx context.Context, userID core.UserID) AccountActionResult {
	update := service.store.DeactivateUser(ctx, userID)
	if _, accepted := update.(AccountMutationAccepted); !accepted {
		return AccountActionRejected{Reason: update.(AccountMutationRejected).Reason}
	}
	return AccountActionAccepted{}
}

func accountActionRejection(consumed AccountTokenConsumeResult) AccountActionRejected {
	if rejected, matched := consumed.(AccountTokenConsumeRejected); matched {
		return AccountActionRejected{Reason: rejected.Reason}
	}
	return AccountActionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "account token is invalid")}
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

	sessionResult := service.issueGuestSession(ctx, guestCreated.Value, newRefreshFamily{})
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
	if _, reused := consumed.(RefreshTokenReuseDetected); reused {
		return RefreshRejected{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "refresh token was reused and its session family was revoked")}
	}
	accepted, matched := consumed.(RefreshTokenConsumed)
	if !matched {
		return RefreshRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "refresh token is invalid")}
	}

	family := continueRefreshFamily{id: accepted.Family}
	switch subject := accepted.Subject.(type) {
	case UserSubject:
		userResult := service.issueUserSession(ctx, subject.ID, family)
		userAccepted, userMatched := userResult.(UserSessionIssued)
		if !userMatched {
			rejected := userResult.(UserSessionRejected)
			return RefreshRejected{Reason: rejected.Reason}
		}
		return RefreshAccepted{Subject: userAccepted.Subject, AccessToken: userAccepted.AccessToken, RefreshToken: userAccepted.RefreshToken}
	case GuestSubject:
		guestResult := service.issueGuestSession(ctx, subject.ID, family)
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

func (service Service) issueUserSession(ctx context.Context, id core.UserID, family refreshFamilySelection) UserSessionResult {
	subject := UserSubject{ID: id}
	accessResult := SignAccessToken(service.tokenSecret, subject, service.clock.Now())
	accessAccepted, accessMatched := accessResult.(AccessTokenAccepted)
	if !accessMatched {
		rejected := accessResult.(AccessTokenRejected)
		return UserSessionRejected{Reason: rejected.Reason}
	}

	refreshResult := service.storeNewRefreshToken(ctx, subject, family)
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

func (service Service) issueGuestSession(ctx context.Context, id core.GuestID, family refreshFamilySelection) GuestSessionResult {
	subject := GuestSubject{ID: id}
	accessResult := SignAccessToken(service.tokenSecret, subject, service.clock.Now())
	accessAccepted, accessMatched := accessResult.(AccessTokenAccepted)
	if !accessMatched {
		rejected := accessResult.(AccessTokenRejected)
		return GuestSessionRejected{Reason: rejected.Reason}
	}

	refreshResult := service.storeNewRefreshToken(ctx, subject, family)
	refreshCreated, refreshMatched := refreshResult.(RefreshTokenCreated)
	if !refreshMatched {
		rejected := refreshResult.(RefreshTokenRejected)
		return GuestSessionRejected{Reason: rejected.Reason}
	}

	return GuestSessionIssued{Subject: subject, AccessToken: accessAccepted.Value, RefreshToken: refreshCreated.Value.Plain}
}

func (service Service) storeNewRefreshToken(ctx context.Context, subject Subject, family refreshFamilySelection) RefreshTokenIssueResult {
	refreshResult := NewRefreshToken(service.clock.Now())
	refreshCreated, refreshMatched := refreshResult.(RefreshTokenCreated)
	if !refreshMatched {
		return refreshResult
	}

	familyID := refreshCreated.Value.ID
	if continued, matched := family.(continueRefreshFamily); matched {
		familyID = continued.id
	}

	storeResult := service.store.StoreRefreshToken(ctx, RefreshTokenRecord{
		ID:        refreshCreated.Value.ID,
		FamilyID:  familyID,
		Subject:   subject,
		Hash:      refreshCreated.Value.Hash,
		ExpiresAt: refreshCreated.Value.ExpiresAt,
	})
	if rejected, matched := storeResult.(StoreRefreshTokenRejected); matched {
		return RefreshTokenRejected{Reason: rejected.Reason}
	}

	return refreshCreated
}
