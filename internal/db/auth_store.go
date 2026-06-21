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
		insert into refresh_tokens (id, token_hash, subject_kind, user_id, guest_id, status, expires_at)
		values ($1, $2, $3, nullif($4, '')::uuid, nullif($5, '')::uuid, 'active', $6)
	`, record.ID.String(), record.Hash.String(), subjectKind, userID, guestID, record.ExpiresAt)
	if err != nil {
		return auth.StoreRefreshTokenRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "insert refresh token failed")}
	}

	return auth.StoreRefreshTokenAccepted{}
}

func (store AuthStore) ConsumeRefreshToken(ctx context.Context, hash auth.RefreshTokenHash, consumedAt time.Time) auth.ConsumeRefreshTokenResult {
	row := store.pool.QueryRow(ctx, `
		update refresh_tokens
		set status = 'consumed', consumed_at = $2
		where token_hash = $1
			and status = 'active'
			and expires_at > $2
		returning subject_kind, coalesce(user_id::text, ''), coalesce(guest_id::text, '')
	`, hash.String(), consumedAt)

	var subjectKind string
	var rawUserID string
	var rawGuestID string
	if err := row.Scan(&subjectKind, &rawUserID, &rawGuestID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.RefreshTokenNotConsumed{}
		}
		return auth.ConsumeRefreshTokenRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "consume refresh token failed")}
	}

	switch subjectKind {
	case "user":
		return consumeUserRefreshToken(rawUserID)
	case "guest":
		return consumeGuestRefreshToken(rawGuestID)
	default:
		return auth.ConsumeRefreshTokenRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "refresh token subject kind is invalid")}
	}
}

func consumeUserRefreshToken(rawUserID string) auth.ConsumeRefreshTokenResult {
	result := core.ParseUserID(rawUserID)
	created, matched := result.(core.UserIDCreated)
	if !matched {
		rejected := result.(core.UserIDRejected)
		return auth.ConsumeRefreshTokenRejected{Reason: rejected.Reason}
	}

	return auth.RefreshTokenConsumed{Subject: auth.UserSubject{ID: created.Value}}
}

func consumeGuestRefreshToken(rawGuestID string) auth.ConsumeRefreshTokenResult {
	result := core.ParseGuestID(rawGuestID)
	created, matched := result.(core.GuestIDCreated)
	if !matched {
		rejected := result.(core.GuestIDRejected)
		return auth.ConsumeRefreshTokenRejected{Reason: rejected.Reason}
	}

	return auth.RefreshTokenConsumed{Subject: auth.GuestSubject{ID: created.Value}}
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

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}

	return false
}
