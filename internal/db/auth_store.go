package db

import (
	"context"
	"errors"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthStore struct {
	pool *pgxpool.Pool
}

func NewAuthStore(pool *pgxpool.Pool) AuthStore {
	return AuthStore{pool: pool}
}

func (store AuthStore) CreateUserCredential(ctx context.Context, id core.UserID, email auth.EmailAddress, passwordHash auth.PasswordHash) auth.StoreUserResult {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return auth.StoreUserRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "begin create user transaction failed")}
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	_, err = tx.Exec(ctx, "insert into users (id, email) values ($1, $2)", id.String(), email.String())
	if err != nil {
		if isUniqueViolation(err) {
			return auth.StoreUserRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "email address is already registered")}
		}
		return auth.StoreUserRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "insert user failed")}
	}

	_, err = tx.Exec(ctx, "insert into password_credentials (user_id, password_hash) values ($1, $2)", id.String(), passwordHash.String())
	if err != nil {
		return auth.StoreUserRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "insert password credential failed")}
	}

	grantResult := insertSignupGrant(ctx, tx, id)
	if rejected, matched := grantResult.(signupGrantRejected); matched {
		return auth.StoreUserRejected{Reason: rejected.reason}
	}

	if err := tx.Commit(ctx); err != nil {
		return auth.StoreUserRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "commit create user transaction failed")}
	}

	return auth.StoreUserAccepted{}
}

func (store AuthStore) FindCredentialByEmail(ctx context.Context, email auth.EmailAddress) auth.CredentialLookupResult {
	row := store.pool.QueryRow(ctx, `
		select users.id::text, users.email, password_credentials.password_hash
		from users
		join password_credentials on password_credentials.user_id = users.id
		where users.email = $1
	`, email.String())

	var rawUserID string
	var rawEmail string
	var rawPasswordHash string
	if err := row.Scan(&rawUserID, &rawEmail, &rawPasswordHash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.CredentialMissing{}
		}
		return auth.CredentialLookupRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "credential lookup failed")}
	}

	userResult := core.ParseUserID(rawUserID)
	userCreated, userMatched := userResult.(core.UserIDCreated)
	if !userMatched {
		rejected := userResult.(core.UserIDRejected)
		return auth.CredentialLookupRejected{Reason: rejected.Reason}
	}

	emailResult := auth.NewEmailAddress(rawEmail)
	emailAccepted, emailMatched := emailResult.(auth.EmailAddressAccepted)
	if !emailMatched {
		rejected := emailResult.(auth.EmailAddressRejected)
		return auth.CredentialLookupRejected{Reason: rejected.Reason}
	}

	hashResult := auth.ParsePasswordHash(rawPasswordHash)
	hashCreated, hashMatched := hashResult.(auth.PasswordHashCreated)
	if !hashMatched {
		rejected := hashResult.(auth.PasswordHashRejected)
		return auth.CredentialLookupRejected{Reason: rejected.Reason}
	}

	return auth.CredentialFound{
		Record: auth.CredentialRecord{
			UserID:       userCreated.Value,
			Email:        emailAccepted.Value,
			PasswordHash: hashCreated.Value,
		},
	}
}

func (store AuthStore) CreateGuestSubject(ctx context.Context, id core.GuestID) auth.StoreGuestResult {
	_, err := store.pool.Exec(ctx, "insert into guest_subjects (id) values ($1)", id.String())
	if err != nil {
		return auth.StoreGuestRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "insert guest subject failed")}
	}

	return auth.StoreGuestAccepted{}
}

// RevokeRefreshFamily revokes every active token in the family of the presented
// token (logout). An unknown token matches no family and revokes nothing.
func (store AuthStore) RevokeRefreshFamily(ctx context.Context, hash auth.RefreshTokenHash) auth.RevokeRefreshFamilyResult {
	_, err := store.pool.Exec(ctx, `
		update refresh_tokens set status = 'revoked'
		where status = 'active'
		and family_id = (select family_id from refresh_tokens where token_hash = $1)
	`, hash.String())
	if err != nil {
		return auth.RevokeRefreshFamilyRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "revoke refresh family failed")}
	}
	return auth.RefreshFamilyRevoked{}
}

func (store AuthStore) StoreRefreshToken(ctx context.Context, record auth.RefreshTokenRecord) auth.StoreRefreshTokenResult {
	subjectKind := ""
	userID := ""
	guestID := ""
	switch subject := record.Subject.(type) {
	case auth.UserSubject:
		subjectKind = "user"
		userID = subject.ID.String()
	case auth.GuestSubject:
		subjectKind = "guest"
		guestID = subject.ID.String()
	default:
		return auth.StoreRefreshTokenRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "refresh token subject is invalid")}
	}

	_, err := store.pool.Exec(ctx, `
		insert into refresh_tokens (id, family_id, token_hash, subject_kind, user_id, guest_id, status, expires_at)
		values ($1, $2, $3, $4, nullif($5, '')::uuid, nullif($6, '')::uuid, 'active', $7)
	`, record.ID.String(), record.FamilyID.String(), record.Hash.String(), subjectKind, userID, guestID, record.ExpiresAt)
	if err != nil {
		return auth.StoreRefreshTokenRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "insert refresh token failed")}
	}

	return auth.StoreRefreshTokenAccepted{}
}

func (store AuthStore) ConsumeRefreshToken(ctx context.Context, hash auth.RefreshTokenHash, consumedAt time.Time) auth.ConsumeRefreshTokenResult {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return auth.ConsumeRefreshTokenRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "begin consume refresh token failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var status string
	var rawFamilyID string
	var subjectKind string
	var rawUserID string
	var rawGuestID string
	var expiresAt time.Time
	scanErr := tx.QueryRow(ctx, `
		select status, family_id::text, subject_kind, coalesce(user_id::text, ''), coalesce(guest_id::text, ''), expires_at
		from refresh_tokens
		where token_hash = $1
		for update
	`, hash.String()).Scan(&status, &rawFamilyID, &subjectKind, &rawUserID, &rawGuestID, &expiresAt)
	if errors.Is(scanErr, pgx.ErrNoRows) {
		return auth.RefreshTokenNotConsumed{}
	}
	if scanErr != nil {
		return auth.ConsumeRefreshTokenRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "consume refresh token failed")}
	}

	// Presenting a token that was already consumed or revoked indicates reuse,
	// which can mean the token was stolen. Revoke the whole family so neither
	// the legitimate holder nor an attacker can keep rotating it.
	if status != "active" {
		if _, err := tx.Exec(ctx, "update refresh_tokens set status = 'revoked' where family_id = $1 and status = 'active'", rawFamilyID); err != nil {
			return auth.ConsumeRefreshTokenRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "revoke refresh token family failed")}
		}
		if err := tx.Commit(ctx); err != nil {
			return auth.ConsumeRefreshTokenRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "commit refresh token family revocation failed")}
		}
		return auth.RefreshTokenReuseDetected{}
	}

	if !expiresAt.After(consumedAt) {
		return auth.RefreshTokenNotConsumed{}
	}

	if _, err := tx.Exec(ctx, "update refresh_tokens set status = 'consumed', consumed_at = $2 where token_hash = $1", hash.String(), consumedAt); err != nil {
		return auth.ConsumeRefreshTokenRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "consume refresh token failed")}
	}
	if err := tx.Commit(ctx); err != nil {
		return auth.ConsumeRefreshTokenRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "commit consume refresh token failed")}
	}

	familyResult := core.ParseRefreshTokenID(rawFamilyID)
	familyCreated, familyMatched := familyResult.(core.RefreshTokenIDCreated)
	if !familyMatched {
		return auth.ConsumeRefreshTokenRejected{Reason: familyResult.(core.RefreshTokenIDRejected).Reason}
	}

	switch subjectKind {
	case "user":
		return consumeUserRefreshToken(rawUserID, familyCreated.Value)
	case "guest":
		return consumeGuestRefreshToken(rawGuestID, familyCreated.Value)
	default:
		return auth.ConsumeRefreshTokenRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "refresh token subject kind is invalid")}
	}
}

func consumeUserRefreshToken(rawUserID string, family core.RefreshTokenID) auth.ConsumeRefreshTokenResult {
	result := core.ParseUserID(rawUserID)
	created, matched := result.(core.UserIDCreated)
	if !matched {
		rejected := result.(core.UserIDRejected)
		return auth.ConsumeRefreshTokenRejected{Reason: rejected.Reason}
	}

	return auth.RefreshTokenConsumed{Subject: auth.UserSubject{ID: created.Value}, Family: family}
}

func consumeGuestRefreshToken(rawGuestID string, family core.RefreshTokenID) auth.ConsumeRefreshTokenResult {
	result := core.ParseGuestID(rawGuestID)
	created, matched := result.(core.GuestIDCreated)
	if !matched {
		rejected := result.(core.GuestIDRejected)
		return auth.ConsumeRefreshTokenRejected{Reason: rejected.Reason}
	}

	return auth.RefreshTokenConsumed{Subject: auth.GuestSubject{ID: created.Value}, Family: family}
}

type signupGrantResult interface {
	signupGrantResult()
}

type signupGrantInserted struct{}

type signupGrantRejected struct {
	reason core.DomainError
}

func (signupGrantInserted) signupGrantResult() {}

func (signupGrantRejected) signupGrantResult() {}

// insertSignupGrant creates the user's credit account and the signup grant
// ledger entry inside the user-creation transaction.
func insertSignupGrant(ctx context.Context, tx pgx.Tx, userID core.UserID) signupGrantResult {
	accountResult := core.NewCreditAccountID()
	account, accountMatched := accountResult.(core.CreditAccountIDCreated)
	if !accountMatched {
		return signupGrantRejected{reason: accountResult.(core.CreditAccountIDRejected).Reason}
	}

	entryResult := core.NewLedgerEntryID()
	entry, entryMatched := entryResult.(core.LedgerEntryIDCreated)
	if !entryMatched {
		return signupGrantRejected{reason: entryResult.(core.LedgerEntryIDRejected).Reason}
	}

	if _, err := tx.Exec(ctx, "insert into credit_accounts (id, owner_kind, user_id) values ($1, 'user', $2)", account.Value.String(), userID.String()); err != nil {
		return signupGrantRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert credit account failed")}
	}

	_, err := tx.Exec(ctx, `
		insert into ledger_entries (id, account_id, kind, amount, idempotency_key)
		values ($1, $2, 'signup_grant', $3, $4)
	`, entry.Value.String(), account.Value.String(), ledger.SignupGrantAmount().Int64(), "signup_grant:"+userID.String())
	if err != nil {
		return signupGrantRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert signup grant ledger entry failed")}
	}

	return signupGrantInserted{}
}

// insertOrganizationCreditGrant creates an organization credit account and its
// initial grant inside the organization-creation transaction.
func insertOrganizationCreditGrant(ctx context.Context, tx pgx.Tx, organizationID core.OrganizationID) signupGrantResult {
	accountResult := core.NewCreditAccountID()
	account, accountMatched := accountResult.(core.CreditAccountIDCreated)
	if !accountMatched {
		return signupGrantRejected{reason: accountResult.(core.CreditAccountIDRejected).Reason}
	}

	entryResult := core.NewLedgerEntryID()
	entry, entryMatched := entryResult.(core.LedgerEntryIDCreated)
	if !entryMatched {
		return signupGrantRejected{reason: entryResult.(core.LedgerEntryIDRejected).Reason}
	}

	if _, err := tx.Exec(ctx, "insert into credit_accounts (id, owner_kind, organization_id) values ($1, 'organization', $2)", account.Value.String(), organizationID.String()); err != nil {
		return signupGrantRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert organization credit account failed")}
	}

	_, err := tx.Exec(ctx, `
		insert into ledger_entries (id, account_id, kind, amount, idempotency_key)
		values ($1, $2, 'signup_grant', $3, $4)
	`, entry.Value.String(), account.Value.String(), ledger.SignupGrantAmount().Int64(), "org_grant:"+organizationID.String())
	if err != nil {
		return signupGrantRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert organization credit grant failed")}
	}

	return signupGrantInserted{}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}

	return false
}
