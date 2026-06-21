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
const accessTokenTTL = 15 * time.Minute
const refreshTokenTTL = 30 * 24 * time.Hour

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

type AccessToken struct {
	value string
}

func (token AccessToken) String() string {
	return token.value
}

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

type Subject interface {
	subject()
}

type UserSubject struct {
	ID core.UserID
}

type GuestSubject struct {
	ID core.GuestID
}

func (UserSubject) subject() {}

func (GuestSubject) subject() {}

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
