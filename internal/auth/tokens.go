package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
)

const refreshTokenBytes = 32
const accountTokenBytes = 32
const accessTokenTTL = 15 * time.Minute
const refreshTokenTTL = 30 * 24 * time.Hour
const emailVerificationTTL = 7 * 24 * time.Hour
const passwordResetTTL = time.Hour

type AccessTokenSecret struct {
	value string
}

type AccessTokenSecretResult interface {
	accessTokenSecretResult()
}

type AccessTokenSecretAccepted struct {
	Value AccessTokenSecret
}

type AccessTokenSecretRejected struct {
	Reason core.DomainError
}

func (AccessTokenSecretAccepted) accessTokenSecretResult() {}

func (AccessTokenSecretRejected) accessTokenSecretResult() {}

func NewAccessTokenSecret(raw string) AccessTokenSecretResult {
	if len(raw) < 32 {
		return AccessTokenSecretRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "access token secret must contain at least 32 bytes")}
	}

	return AccessTokenSecretAccepted{Value: AccessTokenSecret{value: raw}}
}

type AccessTokenVerifier struct {
	secret AccessTokenSecret
	clock  Clock
}

func NewAccessTokenVerifier(secret AccessTokenSecret, clock Clock) AccessTokenVerifier {
	return AccessTokenVerifier{secret: secret, clock: clock}
}

func (verifier AccessTokenVerifier) Verify(token AccessToken) SubjectVerifyResult {
	return VerifyAccessToken(verifier.secret, token, verifier.clock.Now())
}

type AccessToken struct {
	value string
}

func (token AccessToken) String() string {
	return token.value
}

func ParseAccessToken(raw string) AccessTokenParseResult {
	if strings.TrimSpace(raw) == "" {
		return AccessTokenParseRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "access token is required")}
	}
	return AccessTokenParsed{Value: AccessToken{value: raw}}
}

type AccessTokenParseResult interface {
	accessTokenParseResult()
}

type AccessTokenParsed struct {
	Value AccessToken
}

type AccessTokenParseRejected struct {
	Reason core.DomainError
}

func (AccessTokenParsed) accessTokenParseResult() {}

func (AccessTokenParseRejected) accessTokenParseResult() {}

type RefreshTokenPlain struct {
	value string
}

type RefreshTokenHash struct {
	value string
}

type RefreshTokenIssued struct {
	ID        core.RefreshTokenID
	Plain     RefreshTokenPlain
	Hash      RefreshTokenHash
	ExpiresAt time.Time
}

type RefreshTokenIssueResult interface {
	refreshTokenIssueResult()
}

type RefreshTokenCreated struct {
	Value RefreshTokenIssued
}

type RefreshTokenRejected struct {
	Reason core.DomainError
}

func (RefreshTokenCreated) refreshTokenIssueResult() {}

func (RefreshTokenRejected) refreshTokenIssueResult() {}

func NewRefreshToken(now time.Time) RefreshTokenIssueResult {
	idResult := core.NewRefreshTokenID()
	idCreated, idMatched := idResult.(core.RefreshTokenIDCreated)
	if !idMatched {
		rejected := idResult.(core.RefreshTokenIDRejected)
		return RefreshTokenRejected{Reason: rejected.Reason}
	}

	raw := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return RefreshTokenRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "refresh token generation failed")}
	}

	plain := base64.RawURLEncoding.EncodeToString(raw)
	hash := HashRefreshToken(RefreshTokenPlain{value: plain})

	return RefreshTokenCreated{
		Value: RefreshTokenIssued{
			ID:        idCreated.Value,
			Plain:     RefreshTokenPlain{value: plain},
			Hash:      hash,
			ExpiresAt: now.Add(refreshTokenTTL),
		},
	}
}

func ParseRefreshTokenPlain(raw string) RefreshTokenPlainResult {
	if strings.TrimSpace(raw) == "" {
		return RefreshTokenPlainRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "refresh token is required")}
	}

	return RefreshTokenPlainAccepted{Value: RefreshTokenPlain{value: raw}}
}

type RefreshTokenPlainResult interface {
	refreshTokenPlainResult()
}

type RefreshTokenPlainAccepted struct {
	Value RefreshTokenPlain
}

type RefreshTokenPlainRejected struct {
	Reason core.DomainError
}

func (RefreshTokenPlainAccepted) refreshTokenPlainResult() {}

func (RefreshTokenPlainRejected) refreshTokenPlainResult() {}

func HashRefreshToken(plain RefreshTokenPlain) RefreshTokenHash {
	sum := sha256.Sum256([]byte(plain.value))
	return RefreshTokenHash{value: hex.EncodeToString(sum[:])}
}

func (plain RefreshTokenPlain) String() string {
	return plain.value
}

func (hash RefreshTokenHash) String() string {
	return hash.value
}

// RefreshTokenHashFromString reconstructs a RefreshTokenHash from its stored
// string form (as returned by String). Storage adapters that persist and reload
// the hash - including the WASI store bridge, which carries it across the
// host/guest boundary - use this; it does not compute a hash from a plaintext
// token (that is HashRefreshToken).
func RefreshTokenHashFromString(raw string) RefreshTokenHash {
	return RefreshTokenHash{value: raw}
}

type AccountTokenKind struct {
	value string
}

var (
	AccountTokenKindEmailVerification = AccountTokenKind{value: "email_verification"}
	AccountTokenKindPasswordReset     = AccountTokenKind{value: "password_reset"}
)

func (kind AccountTokenKind) String() string {
	return kind.value
}

// AccountTokenKindFromString reconstructs an AccountTokenKind from its string
// form, for storage adapters (including the WASI store bridge) that carry it as
// a string.
func AccountTokenKindFromString(raw string) AccountTokenKind {
	return AccountTokenKind{value: raw}
}

type AccountTokenPlain struct {
	value string
}

type AccountTokenHash struct {
	value string
}

type AccountToken struct {
	ID        core.RefreshTokenID
	Plain     AccountTokenPlain
	Hash      AccountTokenHash
	ExpiresAt time.Time
}

type AccountTokenResult interface {
	accountTokenResult()
}

type AccountTokenCreated struct {
	Value AccountToken
}

type AccountTokenRejected struct {
	Reason core.DomainError
}

func (AccountTokenCreated) accountTokenResult() {}

func (AccountTokenRejected) accountTokenResult() {}

func NewAccountToken(now time.Time, kind AccountTokenKind) AccountTokenResult {
	idResult := core.NewRefreshTokenID()
	idCreated, idMatched := idResult.(core.RefreshTokenIDCreated)
	if !idMatched {
		rejected := idResult.(core.RefreshTokenIDRejected)
		return AccountTokenRejected{Reason: rejected.Reason}
	}
	raw := make([]byte, accountTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return AccountTokenRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "account token generation failed")}
	}
	plain := AccountTokenPlain{value: base64.RawURLEncoding.EncodeToString(raw)}
	ttl := emailVerificationTTL
	if kind == AccountTokenKindPasswordReset {
		ttl = passwordResetTTL
	}
	return AccountTokenCreated{Value: AccountToken{ID: idCreated.Value, Plain: plain, Hash: HashAccountToken(plain), ExpiresAt: now.Add(ttl)}}
}

type AccountTokenPlainResult interface {
	accountTokenPlainResult()
}

type AccountTokenPlainAccepted struct {
	Value AccountTokenPlain
}

type AccountTokenPlainRejected struct {
	Reason core.DomainError
}

func (AccountTokenPlainAccepted) accountTokenPlainResult() {}

func (AccountTokenPlainRejected) accountTokenPlainResult() {}

func ParseAccountTokenPlain(raw string) AccountTokenPlainResult {
	if strings.TrimSpace(raw) == "" {
		return AccountTokenPlainRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "account token is required")}
	}
	return AccountTokenPlainAccepted{Value: AccountTokenPlain{value: strings.TrimSpace(raw)}}
}

func HashAccountToken(plain AccountTokenPlain) AccountTokenHash {
	sum := sha256.Sum256([]byte(plain.value))
	return AccountTokenHash{value: hex.EncodeToString(sum[:])}
}

func (plain AccountTokenPlain) String() string {
	return plain.value
}

func (hash AccountTokenHash) String() string {
	return hash.value
}

// AccountTokenHashFromString reconstructs an AccountTokenHash from its stored
// string form, for storage adapters (including the WASI store bridge). It does
// not compute a hash from a plaintext token (that is HashAccountToken).
func AccountTokenHashFromString(raw string) AccountTokenHash {
	return AccountTokenHash{value: raw}
}

type Subject interface {
	subject()
}

type UserSubject struct {
	ID core.UserID
}

type GuestSubject struct {
	ID core.GuestID
}

// OrgSubject is an organization acting as itself via an org-wide credential
// (see internal/orgcred), rather than through an individual member. It
// carries full parity with an org-admin-role member wherever authorization
// helpers accept a Subject.
type OrgSubject struct {
	ID core.OrganizationID
}

func (UserSubject) subject() {}

func (GuestSubject) subject() {}

func (OrgSubject) subject() {}

func SignAccessToken(secret AccessTokenSecret, subject Subject, now time.Time) AccessTokenResult {
	subjectID := ""
	subjectKind := ""
	switch typed := subject.(type) {
	case UserSubject:
		subjectID = typed.ID.String()
		subjectKind = "user"
	case GuestSubject:
		subjectID = typed.ID.String()
		subjectKind = "guest"
	default:
		return AccessTokenRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "subject is invalid")}
	}

	headerJSON, err := json.Marshal(jwtHeader{Algorithm: "HS256", Type: "JWT"})
	if err != nil {
		return AccessTokenRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "access token header encoding failed")}
	}

	payloadJSON, err := json.Marshal(jwtPayload{
		Subject:     subjectID,
		SubjectKind: subjectKind,
		IssuedAt:    now.Unix(),
		ExpiresAt:   now.Add(accessTokenTTL).Unix(),
	})
	if err != nil {
		return AccessTokenRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "access token payload encoding failed")}
	}

	header := base64.RawURLEncoding.EncodeToString(headerJSON)
	payload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	unsigned := header + "." + payload
	mac := hmac.New(sha256.New, []byte(secret.value))
	_, _ = mac.Write([]byte(unsigned))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return AccessTokenAccepted{Value: AccessToken{value: unsigned + "." + signature}}
}

func VerifyAccessToken(secret AccessTokenSecret, token AccessToken, now time.Time) SubjectVerifyResult {
	parts := strings.Split(token.value, ".")
	if len(parts) != 3 {
		return SubjectVerifyRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "access token format is invalid")}
	}

	unsigned := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(secret.value))
	_, _ = mac.Write([]byte(unsigned))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return SubjectVerifyRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "access token signature is invalid")}
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return SubjectVerifyRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "access token payload is invalid")}
	}

	var payload jwtPayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return SubjectVerifyRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "access token payload is invalid")}
	}

	if payload.ExpiresAt <= now.Unix() {
		return SubjectVerifyRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "access token expired")}
	}

	switch payload.SubjectKind {
	case "user":
		userResult := core.ParseUserID(payload.Subject)
		userCreated, userMatched := userResult.(core.UserIDCreated)
		if !userMatched {
			rejected := userResult.(core.UserIDRejected)
			return SubjectVerifyRejected{Reason: rejected.Reason}
		}
		return SubjectVerified{Value: UserSubject{ID: userCreated.Value}}
	case "guest":
		guestResult := core.ParseGuestID(payload.Subject)
		guestCreated, guestMatched := guestResult.(core.GuestIDCreated)
		if !guestMatched {
			rejected := guestResult.(core.GuestIDRejected)
			return SubjectVerifyRejected{Reason: rejected.Reason}
		}
		return SubjectVerified{Value: GuestSubject{ID: guestCreated.Value}}
	default:
		return SubjectVerifyRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "access token subject kind is invalid")}
	}
}

type SubjectVerifyResult interface {
	subjectVerifyResult()
}

type SubjectVerified struct {
	Value Subject
}

type SubjectVerifyRejected struct {
	Reason core.DomainError
}

func (SubjectVerified) subjectVerifyResult() {}

func (SubjectVerifyRejected) subjectVerifyResult() {}

type jwtHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

type jwtPayload struct {
	Subject     string `json:"sub"`
	SubjectKind string `json:"sharecrop_subject_kind"`
	IssuedAt    int64  `json:"iat"`
	ExpiresAt   int64  `json:"exp"`
}

type AccessTokenResult interface {
	accessTokenResult()
}

type AccessTokenAccepted struct {
	Value AccessToken
}

type AccessTokenRejected struct {
	Reason core.DomainError
}

func (AccessTokenAccepted) accessTokenResult() {}

func (AccessTokenRejected) accessTokenResult() {}
