package db

import (
	"context"
	"errors"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthStore struct {
	db Beginner
}

func NewAuthStore(pool *pgxpool.Pool) AuthStore {
	return NewAuthStoreFromHandle(NewPGX(pool))
}

func NewAuthStoreFromHandle(handle Beginner) AuthStore {
	return AuthStore{db: handle}
}

func (store AuthStore) CreateUserCredential(ctx context.Context, id core.UserID, email auth.EmailAddress, passwordHash auth.PasswordHash) auth.StoreUserResult {
	tx, err := store.db.Begin(ctx)
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

func (store AuthStore) FindOrCreateExternalIdentity(ctx context.Context, identity auth.ExternalIdentity, email auth.EmailAddress) auth.ExternalIdentityResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return auth.ExternalIdentityRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin external identity transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var rawID string
	err = tx.QueryRow(ctx, "select user_id::text from external_identities where issuer = $1 and subject = $2", identity.Issuer, identity.Subject).Scan(&rawID)
	if errors.Is(err, ErrNoRows) {
		created := core.NewUserID()
		value, ok := created.(core.UserIDCreated)
		if !ok {
			return auth.ExternalIdentityRejected{Reason: created.(core.UserIDRejected).Reason}
		}
		createdUser, err := tx.Exec(ctx, "insert into users (id, email) values ($1, $2) on conflict (email) do nothing", value.Value.String(), email.String())
		if err != nil {
			return auth.ExternalIdentityRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "create external user failed")}
		}
		if createdUser != 1 {
			return auth.ExternalIdentityRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "email address is already associated with another account")}
		}
		rawID = value.Value.String()
		_, err = tx.Exec(ctx, "insert into external_identities (issuer, subject, user_id) values ($1, $2, $3)", identity.Issuer, identity.Subject, rawID)
		if err != nil {
			return auth.ExternalIdentityRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "link external identity failed")}
		}
	} else if err != nil {
		return auth.ExternalIdentityRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "load external identity failed")}
	}
	var status string
	if err := tx.QueryRow(ctx, "select status from users where id = $1", rawID).Scan(&status); err != nil {
		return auth.ExternalIdentityRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "load external user status failed")}
	}
	if status != "active" {
		return auth.ExternalIdentityRejected{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "external user is not active")}
	}
	id := core.ParseUserID(rawID)
	parsed, ok := id.(core.UserIDCreated)
	if !ok {
		return auth.ExternalIdentityRejected{Reason: id.(core.UserIDRejected).Reason}
	}
	if err := tx.Commit(ctx); err != nil {
		return auth.ExternalIdentityRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit external identity failed")}
	}
	return auth.ExternalIdentityFound{UserID: parsed.Value}
}

func (store AuthStore) FindCredentialByEmail(ctx context.Context, email auth.EmailAddress) auth.CredentialLookupResult {
	row := store.db.QueryRow(ctx, `
		select users.id::text, users.email, password_credentials.password_hash, users.status
		from users
		join password_credentials on password_credentials.user_id = users.id
		where users.email = $1
	`, email.String())
	return scanCredential(row)
}

func (store AuthStore) FindCredentialByUserID(ctx context.Context, userID core.UserID) auth.CredentialLookupResult {
	row := store.db.QueryRow(ctx, `
		select users.id::text, users.email, password_credentials.password_hash, users.status
		from users
		join password_credentials on password_credentials.user_id = users.id
		where users.id = $1
	`, userID.String())
	return scanCredential(row)
}

func (store AuthStore) ListUsers(ctx context.Context, query string, page core.Page) auth.UserDirectoryResult {
	limit := page.Limit()
	offset := page.Offset()
	// Escape LIKE metacharacters in the caller's query so a value full of
	// '%'/'_' is matched literally (as the browser-store substring match
	// does) rather than expanding into an expensive wildcard scan.
	rows, err := store.db.Query(ctx, `
		select id::text, email, status
		from users
		where status = 'active'
		and ($1 = '' or email ilike '%' || replace(replace(replace($1, '\', '\\'), '%', '\%'), '_', '\_') || '%' or id::text = $1)
		order by email asc
		limit $2 offset $3
	`, query, limit, offset)
	if err != nil {
		return auth.UserDirectoryRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list users failed")}
	}
	defer rows.Close()

	values := make([]auth.UserDirectoryEntry, 0)
	for rows.Next() {
		var rawID string
		var rawEmail string
		var status string
		if err := rows.Scan(&rawID, &rawEmail, &status); err != nil {
			return auth.UserDirectoryRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan user directory failed")}
		}
		idResult := core.ParseUserID(rawID)
		id, idMatched := idResult.(core.UserIDCreated)
		if !idMatched {
			return auth.UserDirectoryRejected{Reason: idResult.(core.UserIDRejected).Reason}
		}
		emailResult := auth.NewEmailAddress(rawEmail)
		email, emailMatched := emailResult.(auth.EmailAddressAccepted)
		if !emailMatched {
			return auth.UserDirectoryRejected{Reason: emailResult.(auth.EmailAddressRejected).Reason}
		}
		values = append(values, auth.UserDirectoryEntry{ID: id.Value, Email: email.Value, Status: status})
	}
	if rows.Err() != nil {
		return auth.UserDirectoryRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read user directory failed")}
	}
	return auth.UsersListed{Values: values}
}

func scanCredential(row Row) auth.CredentialLookupResult {
	var rawUserID string
	var rawEmail string
	var rawPasswordHash string
	var rawStatus string
	if err := row.Scan(&rawUserID, &rawEmail, &rawPasswordHash, &rawStatus); err != nil {
		if errors.Is(err, ErrNoRows) {
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
			Status:       rawStatus,
		},
	}
}

func (store AuthStore) UpdateUserEmail(ctx context.Context, userID core.UserID, email auth.EmailAddress) auth.AccountMutationResult {
	tag, err := store.db.Exec(ctx, "update users set email = $2, email_verified_at = null where id = $1 and status = 'active'", userID.String(), email.String())
	if err != nil {
		if isUniqueViolation(err) {
			return auth.AccountMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "email address is already registered")}
		}
		return auth.AccountMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "update user email failed")}
	}
	if tag == 0 {
		return auth.AccountMutationRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "account was not found")}
	}
	return auth.AccountMutationAccepted{}
}

func (store AuthStore) UpdatePassword(ctx context.Context, userID core.UserID, passwordHash auth.PasswordHash) auth.AccountMutationResult {
	tag, err := store.db.Exec(ctx, "update password_credentials set password_hash = $2 where user_id = $1", userID.String(), passwordHash.String())
	if err != nil {
		return auth.AccountMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "update password failed")}
	}
	if tag == 0 {
		return auth.AccountMutationRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "account was not found")}
	}
	if _, err := store.db.Exec(ctx, "update refresh_tokens set status = 'revoked' where user_id = $1 and status = 'active'", userID.String()); err != nil {
		return auth.AccountMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "revoke account sessions failed")}
	}
	return auth.AccountMutationAccepted{}
}

func (store AuthStore) DeactivateUser(ctx context.Context, userID core.UserID) auth.AccountMutationResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return auth.AccountMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin deactivate account failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// A user still owning tasks that hold allocated credits or held collectibles
	// cannot deactivate: those funds would be stranded. Refunds are easy and
	// auto-granted, so this stays a light guard that asks them to settle first.
	var holdsFundedRewards bool
	if err := tx.QueryRow(ctx, `
		select exists(
			select 1 from tasks
			where tasks.created_by_user_id = $1
			and (
				exists(select 1 from task_funds where task_funds.task_id = tasks.id)
				or exists(select 1 from task_fund_collectibles where task_fund_collectibles.task_id = tasks.id)
			)
		)
	`, userID.String()).Scan(&holdsFundedRewards); err != nil {
		return auth.AccountMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "check held task rewards failed")}
	}
	if holdsFundedRewards {
		return auth.AccountMutationRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "refund or close your funded tasks before deactivating")}
	}
	tombstoneEmail := "deactivated+" + userID.String() + "@sharecrop.invalid"
	tag, err := tx.Exec(ctx, "update users set status = 'deactivated', email = $2, email_verified_at = null where id = $1 and status = 'active'", userID.String(), tombstoneEmail)
	if err != nil {
		return auth.AccountMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "deactivate account failed")}
	}
	if tag == 0 {
		return auth.AccountMutationRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "account was not found")}
	}
	if _, err := tx.Exec(ctx, "update refresh_tokens set status = 'revoked' where user_id = $1 and status = 'active'", userID.String()); err != nil {
		return auth.AccountMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "revoke account sessions failed")}
	}
	if _, err := tx.Exec(ctx, "update account_tokens set status = 'revoked' where user_id = $1 and status = 'active'", userID.String()); err != nil {
		return auth.AccountMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "revoke account tokens failed")}
	}
	if _, err := tx.Exec(ctx, "delete from password_credentials where user_id = $1", userID.String()); err != nil {
		return auth.AccountMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "delete account password credential failed")}
	}
	if err := tx.Commit(ctx); err != nil {
		return auth.AccountMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit deactivate account failed")}
	}
	return auth.AccountMutationAccepted{}
}

func (store AuthStore) CreateGuestSubject(ctx context.Context, id core.GuestID) auth.StoreGuestResult {
	_, err := store.db.Exec(ctx, "insert into guest_subjects (id) values ($1)", id.String())
	if err != nil {
		return auth.StoreGuestRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "insert guest subject failed")}
	}

	return auth.StoreGuestAccepted{}
}

// RevokeRefreshFamily revokes every active token in the family of the presented
// token (logout). An unknown token matches no family and revokes nothing.
func (store AuthStore) RevokeRefreshFamily(ctx context.Context, hash auth.RefreshTokenHash) auth.RevokeRefreshFamilyResult {
	_, err := store.db.Exec(ctx, `
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

	_, err := store.db.Exec(ctx, `
		insert into refresh_tokens (id, family_id, token_hash, subject_kind, user_id, guest_id, status, expires_at)
		values ($1, $2, $3, $4, nullif($5, '')::uuid, nullif($6, '')::uuid, 'active', $7)
	`, record.ID.String(), record.FamilyID.String(), record.Hash.String(), subjectKind, userID, guestID, record.ExpiresAt)
	if err != nil {
		return auth.StoreRefreshTokenRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "insert refresh token failed")}
	}

	return auth.StoreRefreshTokenAccepted{}
}

func (store AuthStore) ValidateRefreshToken(ctx context.Context, hash auth.RefreshTokenHash, now time.Time) auth.ValidateRefreshTokenResult {
	var active int
	err := store.db.QueryRow(ctx, `
		select 1 from refresh_tokens
		where token_hash = $1 and status = 'active' and expires_at > $2
	`, hash.String(), now).Scan(&active)
	if errors.Is(err, ErrNoRows) {
		return auth.RefreshTokenInactive{}
	}
	if err != nil {
		return auth.ValidateRefreshTokenRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "validate refresh token failed")}
	}
	return auth.RefreshTokenActive{}
}

func (store AuthStore) StoreAccountToken(ctx context.Context, userID core.UserID, kind auth.AccountTokenKind, token auth.AccountToken) auth.AccountTokenStoreResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return auth.AccountTokenStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin account token transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "update account_tokens set status = 'revoked' where user_id = $1 and kind = $2 and status = 'active'", userID.String(), kind.String()); err != nil {
		return auth.AccountTokenStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "revoke previous account tokens failed")}
	}
	_, err = tx.Exec(ctx, `
		insert into account_tokens (id, user_id, token_hash, kind, status, expires_at)
		values ($1, $2, $3, $4, 'active', $5)
	`, token.ID.String(), userID.String(), token.Hash.String(), kind.String(), token.ExpiresAt)
	if err != nil {
		return auth.AccountTokenStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "store account token failed")}
	}
	if err := tx.Commit(ctx); err != nil {
		return auth.AccountTokenStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit account token transaction failed")}
	}
	return auth.AccountTokenStored{}
}

func (store AuthStore) ConsumeAccountToken(ctx context.Context, kind auth.AccountTokenKind, hash auth.AccountTokenHash, now time.Time) auth.AccountTokenConsumeResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return auth.AccountTokenConsumeRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin account token consume failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var rawUserID string
	var expiresAt time.Time
	scanErr := tx.QueryRow(ctx, `
		select user_id::text, expires_at
		from account_tokens
		where token_hash = $1 and kind = $2 and status = 'active'
		for update
	`, hash.String(), kind.String()).Scan(&rawUserID, &expiresAt)
	if errors.Is(scanErr, ErrNoRows) {
		return auth.AccountTokenNotConsumed{}
	}
	if scanErr != nil {
		return auth.AccountTokenConsumeRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "account token lookup failed")}
	}
	if !expiresAt.After(now) {
		return auth.AccountTokenNotConsumed{}
	}
	if _, err := tx.Exec(ctx, "update account_tokens set status = 'consumed', consumed_at = $3 where token_hash = $1 and kind = $2", hash.String(), kind.String(), now); err != nil {
		return auth.AccountTokenConsumeRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "consume account token failed")}
	}
	if kind == auth.AccountTokenKindEmailVerification {
		if _, err := tx.Exec(ctx, "update users set email_verified_at = $2 where id = $1", rawUserID, now); err != nil {
			return auth.AccountTokenConsumeRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "mark email verified failed")}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return auth.AccountTokenConsumeRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit account token consume failed")}
	}
	userResult := core.ParseUserID(rawUserID)
	userCreated, userMatched := userResult.(core.UserIDCreated)
	if !userMatched {
		return auth.AccountTokenConsumeRejected{Reason: userResult.(core.UserIDRejected).Reason}
	}
	return auth.AccountTokenConsumed{UserID: userCreated.Value}
}

func (store AuthStore) ConsumeRefreshToken(ctx context.Context, hash auth.RefreshTokenHash, consumedAt time.Time) auth.ConsumeRefreshTokenResult {
	tx, err := store.db.Begin(ctx)
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
	if errors.Is(scanErr, ErrNoRows) {
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
func insertSignupGrant(ctx context.Context, tx Tx, userID core.UserID) signupGrantResult {
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
func insertOrganizationCreditGrant(ctx context.Context, tx Tx, organizationID core.OrganizationID) signupGrantResult {
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
