// Package gen generates a WASI store bridge (the Dispatch host router and the
// GuestStore client) from a domain Store interface, so the per-method plumbing
// can never silently drift from the interface it serves. It introspects the
// interface's methods via go/ast and emits code that calls the hand-written
// per-type codecs in the bridge package (and the shared codecs in corewire); a
// type used by a method but missing from the codec registry is a generation
// error, not a silent gap.
//
// Each store is described by a storeSpec below. Adding a store means adding a
// spec (naming its codecs) and the hand-written codecs it references - the
// generator itself is store-agnostic.
package gen

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"sort"
	"strings"
)

// argCodec names how one method argument type crosses the bridge: the envelope
// field it occupies, its wire type, and the encode/decode functions (a bare
// name for a codec local to the bridge package, or a qualified corewire.X for a
// shared one). Decode returns (T, error).
//
// The field name is derived from the type, not the parameter (interface method
// params are usually unnamed in the AST), so a method with two arguments of the
// same type is unsupported - none of the bridged stores have one.
type argCodec struct {
	field    string
	goType   string
	wireType string
	encodeFn string
	decodeFn string
}

// resultCodec names how one method result type crosses the bridge.
type resultCodec struct {
	goType       string
	wireType     string
	encodeFn     string
	decodeFn     string
	rejectedType string
}

// storeSpec describes one bridge to generate.
type storeSpec struct {
	bridgePackage string
	domainImport  string
	domainPackage string
	interfaceName string
	wirePrefix    string
	argCodecs     map[string]argCodec
	resultCodecs  map[string]resultCodec
	// extraImports lists packages the generated signatures reference beyond the
	// domain package, core, and corewire - e.g. a method argument whose type
	// lives in another package (auth.EmailAddress). Left nil by most stores.
	extraImports []string
}

// Target names a store to (re)generate: where its interface source lives and
// where the generated bridge is written. Exported so the generate command can
// iterate every store without knowing the registry.
type Target struct {
	Key        string
	SourceDir  string
	OutputPath string
}

// Targets is the full set of stores the bridge codegen covers.
func Targets() []Target {
	return []Target{
		{Key: "audit", SourceDir: "internal/audit", OutputPath: "internal/wasibridge/auditbridge/bridge_gen.go"},
		{Key: "notification", SourceDir: "internal/notification", OutputPath: "internal/wasibridge/notificationbridge/bridge_gen.go"},
		{Key: "auth", SourceDir: "internal/auth", OutputPath: "internal/wasibridge/authbridge/bridge_gen.go"},
		{Key: "agent", SourceDir: "internal/agent", OutputPath: "internal/wasibridge/agentbridge/bridge_gen.go"},
		{Key: "orgcred", SourceDir: "internal/orgcred", OutputPath: "internal/wasibridge/orgcredbridge/bridge_gen.go"},
		{Key: "assets", SourceDir: "internal/assets", OutputPath: "internal/wasibridge/assetsbridge/bridge_gen.go"},
		{Key: "submission", SourceDir: "internal/submission", OutputPath: "internal/wasibridge/submissionbridge/bridge_gen.go"},
		{Key: "ledger", SourceDir: "internal/ledger", OutputPath: "internal/wasibridge/ledgerbridge/bridge_gen.go"},
		{Key: "org", SourceDir: "internal/org", OutputPath: "internal/wasibridge/orgbridge/bridge_gen.go"},
		{Key: "task", SourceDir: "internal/task", OutputPath: "internal/wasibridge/taskbridge/bridge_gen.go"},
		{Key: "savedqueueview", SourceDir: "internal/http", OutputPath: "internal/wasibridge/savedqueueviewbridge/bridge_gen.go"},
		{Key: "platformadmin", SourceDir: "internal/http", OutputPath: "internal/wasibridge/platformadminbridge/bridge_gen.go"},
		{Key: "moderationtriage", SourceDir: "internal/http", OutputPath: "internal/wasibridge/moderationtriagebridge/bridge_gen.go"},
		{Key: "privacy", SourceDir: "internal/http", OutputPath: "internal/wasibridge/privacybridge/bridge_gen.go"},
	}
}

func userIDArg() argCodec {
	return argCodec{field: "UserID", goType: "core.UserID", wireType: "string", encodeFn: "corewire.EncodeUserID", decodeFn: "corewire.DecodeUserID"}
}

func pageArg() argCodec {
	return argCodec{field: "Page", goType: "core.Page", wireType: "corewire.PageWire", encodeFn: "corewire.EncodePage", decodeFn: "corewire.DecodePage"}
}

var specs = map[string]storeSpec{
	"audit": {
		bridgePackage: "auditbridge",
		domainImport:  "github.com/e6qu/sharecrop/internal/audit",
		domainPackage: "audit",
		interfaceName: "Store",
		wirePrefix:    "audit",
		extraImports:  []string{"github.com/e6qu/sharecrop/internal/wasibridge/auditwire"},
		argCodecs: map[string]argCodec{
			"core.AuditEventID": {field: "ID", goType: "core.AuditEventID", wireType: "string", encodeFn: "corewire.EncodeAuditEventID", decodeFn: "corewire.DecodeAuditEventID"},
			"audit.Event":       {field: "Event", goType: "audit.Event", wireType: "auditwire.EventWire", encodeFn: "auditwire.EncodeEvent", decodeFn: "auditwire.DecodeEvent"},
			"audit.ListFilters": {field: "Filters", goType: "audit.ListFilters", wireType: "listFiltersWire", encodeFn: "encodeListFilters", decodeFn: "decodeListFilters"},
			"core.Page":         pageArg(),
		},
		resultCodecs: map[string]resultCodec{
			"audit.RecordResult": {goType: "audit.RecordResult", wireType: "recordResultWire", encodeFn: "encodeRecordResult", decodeFn: "decodeRecordResult", rejectedType: "audit.RecordRejected"},
			"audit.GetResult":    {goType: "audit.GetResult", wireType: "getResultWire", encodeFn: "encodeGetResult", decodeFn: "decodeGetResult", rejectedType: "audit.GetRejected"},
			"audit.ListResult":   {goType: "audit.ListResult", wireType: "listResultWire", encodeFn: "encodeListResult", decodeFn: "decodeListResult", rejectedType: "audit.ListRejected"},
		},
	},
	"notification": {
		bridgePackage: "notificationbridge",
		domainImport:  "github.com/e6qu/sharecrop/internal/notification",
		domainPackage: "notification",
		interfaceName: "Store",
		wirePrefix:    "notification",
		argCodecs: map[string]argCodec{
			"notification.Notification": {field: "Notification", goType: "notification.Notification", wireType: "notificationWire", encodeFn: "encodeNotification", decodeFn: "decodeNotification"},
			"core.UserID":               userIDArg(),
			"core.Page":                 pageArg(),
			"core.NotificationID":       {field: "ID", goType: "core.NotificationID", wireType: "string", encodeFn: "corewire.EncodeNotificationID", decodeFn: "corewire.DecodeNotificationID"},
		},
		resultCodecs: map[string]resultCodec{
			"notification.CreateStoreResult":   {goType: "notification.CreateStoreResult", wireType: "createResultWire", encodeFn: "encodeCreateResult", decodeFn: "decodeCreateResult", rejectedType: "notification.CreateStoreRejected"},
			"notification.ListStoreResult":     {goType: "notification.ListStoreResult", wireType: "listResultWire", encodeFn: "encodeListResult", decodeFn: "decodeListResult", rejectedType: "notification.ListStoreRejected"},
			"notification.MarkReadStoreResult": {goType: "notification.MarkReadStoreResult", wireType: "markReadResultWire", encodeFn: "encodeMarkReadResult", decodeFn: "decodeMarkReadResult", rejectedType: "notification.MarkReadStoreRejected"},
		},
	},
	"auth": {
		bridgePackage: "authbridge",
		domainImport:  "github.com/e6qu/sharecrop/internal/auth",
		domainPackage: "auth",
		interfaceName: "Store",
		wirePrefix:    "auth",
		argCodecs: map[string]argCodec{
			"core.UserID":             userIDArg(),
			"core.Page":               pageArg(),
			"core.GuestID":            {field: "GuestID", goType: "core.GuestID", wireType: "string", encodeFn: "corewire.EncodeGuestID", decodeFn: "corewire.DecodeGuestID"},
			"string":                  {field: "Query", goType: "string", wireType: "string", encodeFn: "corewire.EncodeString", decodeFn: "corewire.DecodeString"},
			"auth.ExternalIdentity":   {field: "Identity", goType: "auth.ExternalIdentity", wireType: "externalIdentityWire", encodeFn: "encodeExternalIdentity", decodeFn: "decodeExternalIdentity"},
			"time.Time":               {field: "Now", goType: "time.Time", wireType: "string", encodeFn: "corewire.EncodeTime", decodeFn: "corewire.DecodeTime"},
			"auth.EmailAddress":       {field: "Email", goType: "auth.EmailAddress", wireType: "string", encodeFn: "encodeEmail", decodeFn: "decodeEmail"},
			"auth.PasswordHash":       {field: "PasswordHash", goType: "auth.PasswordHash", wireType: "string", encodeFn: "encodePasswordHash", decodeFn: "decodePasswordHash"},
			"auth.RefreshTokenRecord": {field: "Record", goType: "auth.RefreshTokenRecord", wireType: "refreshTokenRecordWire", encodeFn: "encodeRefreshTokenRecord", decodeFn: "decodeRefreshTokenRecord"},
			"auth.RefreshTokenHash":   {field: "Hash", goType: "auth.RefreshTokenHash", wireType: "string", encodeFn: "encodeRefreshTokenHash", decodeFn: "decodeRefreshTokenHash"},
			"auth.AccountTokenKind":   {field: "Kind", goType: "auth.AccountTokenKind", wireType: "string", encodeFn: "encodeAccountTokenKind", decodeFn: "decodeAccountTokenKind"},
			"auth.AccountToken":       {field: "Token", goType: "auth.AccountToken", wireType: "accountTokenWire", encodeFn: "encodeAccountToken", decodeFn: "decodeAccountToken"},
			"auth.AccountTokenHash":   {field: "Hash", goType: "auth.AccountTokenHash", wireType: "string", encodeFn: "encodeAccountTokenHash", decodeFn: "decodeAccountTokenHash"},
		},
		resultCodecs: map[string]resultCodec{
			"auth.ExternalIdentityResult":     {goType: "auth.ExternalIdentityResult", wireType: "externalIdentityResultWire", encodeFn: "encodeExternalIdentityResult", decodeFn: "decodeExternalIdentityResult", rejectedType: "auth.ExternalIdentityRejected"},
			"auth.StoreUserResult":            {goType: "auth.StoreUserResult", wireType: "acceptedRejectedWire", encodeFn: "encodeStoreUserResult", decodeFn: "decodeStoreUserResult", rejectedType: "auth.StoreUserRejected"},
			"auth.CredentialLookupResult":     {goType: "auth.CredentialLookupResult", wireType: "credentialLookupResultWire", encodeFn: "encodeCredentialLookupResult", decodeFn: "decodeCredentialLookupResult", rejectedType: "auth.CredentialLookupRejected"},
			"auth.UserDirectoryResult":        {goType: "auth.UserDirectoryResult", wireType: "userDirectoryResultWire", encodeFn: "encodeUserDirectoryResult", decodeFn: "decodeUserDirectoryResult", rejectedType: "auth.UserDirectoryRejected"},
			"auth.AccountMutationResult":      {goType: "auth.AccountMutationResult", wireType: "acceptedRejectedWire", encodeFn: "encodeAccountMutationResult", decodeFn: "decodeAccountMutationResult", rejectedType: "auth.AccountMutationRejected"},
			"auth.StoreGuestResult":           {goType: "auth.StoreGuestResult", wireType: "acceptedRejectedWire", encodeFn: "encodeStoreGuestResult", decodeFn: "decodeStoreGuestResult", rejectedType: "auth.StoreGuestRejected"},
			"auth.StoreRefreshTokenResult":    {goType: "auth.StoreRefreshTokenResult", wireType: "acceptedRejectedWire", encodeFn: "encodeStoreRefreshTokenResult", decodeFn: "decodeStoreRefreshTokenResult", rejectedType: "auth.StoreRefreshTokenRejected"},
			"auth.ValidateRefreshTokenResult": {goType: "auth.ValidateRefreshTokenResult", wireType: "acceptedRejectedWire", encodeFn: "encodeValidateRefreshTokenResult", decodeFn: "decodeValidateRefreshTokenResult", rejectedType: "auth.ValidateRefreshTokenRejected"},
			"auth.ConsumeRefreshTokenResult":  {goType: "auth.ConsumeRefreshTokenResult", wireType: "consumeRefreshTokenResultWire", encodeFn: "encodeConsumeRefreshTokenResult", decodeFn: "decodeConsumeRefreshTokenResult", rejectedType: "auth.ConsumeRefreshTokenRejected"},
			"auth.RevokeRefreshFamilyResult":  {goType: "auth.RevokeRefreshFamilyResult", wireType: "acceptedRejectedWire", encodeFn: "encodeRevokeRefreshFamilyResult", decodeFn: "decodeRevokeRefreshFamilyResult", rejectedType: "auth.RevokeRefreshFamilyRejected"},
			"auth.AccountTokenStoreResult":    {goType: "auth.AccountTokenStoreResult", wireType: "acceptedRejectedWire", encodeFn: "encodeAccountTokenStoreResult", decodeFn: "decodeAccountTokenStoreResult", rejectedType: "auth.AccountTokenStoreRejected"},
			"auth.AccountTokenConsumeResult":  {goType: "auth.AccountTokenConsumeResult", wireType: "accountTokenConsumeResultWire", encodeFn: "encodeAccountTokenConsumeResult", decodeFn: "decodeAccountTokenConsumeResult", rejectedType: "auth.AccountTokenConsumeRejected"},
		},
	},
	"agent": {
		bridgePackage: "agentbridge",
		domainImport:  "github.com/e6qu/sharecrop/internal/agent",
		domainPackage: "agent",
		interfaceName: "Store",
		wirePrefix:    "agent",
		argCodecs: map[string]argCodec{
			"core.UserID":            userIDArg(),
			"core.Page":              pageArg(),
			"core.AgentCredentialID": {field: "ID", goType: "core.AgentCredentialID", wireType: "string", encodeFn: "corewire.EncodeAgentCredentialID", decodeFn: "corewire.DecodeAgentCredentialID"},
			"agent.Credential":       {field: "Credential", goType: "agent.Credential", wireType: "credentialWire", encodeFn: "encodeCredential", decodeFn: "decodeCredential"},
			"agent.SecretHash":       {field: "SecretHash", goType: "agent.SecretHash", wireType: "string", encodeFn: "encodeSecretHash", decodeFn: "decodeSecretHash"},
		},
		resultCodecs: map[string]resultCodec{
			"agent.CreateStoreResult": {goType: "agent.CreateStoreResult", wireType: "agentwire.CreateResultWire", encodeFn: "agentwire.EncodeCreateStoreResult", decodeFn: "agentwire.DecodeCreateStoreResult", rejectedType: "agent.CreateStoreRejected"},
			"agent.VerifyStoreResult": {goType: "agent.VerifyStoreResult", wireType: "credentialResultWire", encodeFn: "encodeVerifyResult", decodeFn: "decodeVerifyResult", rejectedType: "agent.VerifyStoreRejected"},
			"agent.ListStoreResult":   {goType: "agent.ListStoreResult", wireType: "listResultWire", encodeFn: "encodeListResult", decodeFn: "decodeListResult", rejectedType: "agent.ListStoreRejected"},
			"agent.RevokeStoreResult": {goType: "agent.RevokeStoreResult", wireType: "credentialResultWire", encodeFn: "encodeRevokeResult", decodeFn: "decodeRevokeResult", rejectedType: "agent.RevokeStoreRejected"},
		},
	},
	"orgcred": {
		bridgePackage: "orgcredbridge",
		domainImport:  "github.com/e6qu/sharecrop/internal/orgcred",
		domainPackage: "orgcred",
		interfaceName: "Store",
		wirePrefix:    "orgcred",
		argCodecs: map[string]argCodec{
			"core.OrganizationID":  {field: "OrganizationID", goType: "core.OrganizationID", wireType: "string", encodeFn: "corewire.EncodeOrganizationID", decodeFn: "corewire.DecodeOrganizationID"},
			"core.Page":            pageArg(),
			"core.OrgCredentialID": {field: "ID", goType: "core.OrgCredentialID", wireType: "string", encodeFn: "corewire.EncodeOrgCredentialID", decodeFn: "corewire.DecodeOrgCredentialID"},
			"orgcred.Credential":   {field: "Credential", goType: "orgcred.Credential", wireType: "credentialWire", encodeFn: "encodeCredential", decodeFn: "decodeCredential"},
			"orgcred.SecretHash":   {field: "SecretHash", goType: "orgcred.SecretHash", wireType: "string", encodeFn: "encodeSecretHash", decodeFn: "decodeSecretHash"},
		},
		resultCodecs: map[string]resultCodec{
			"orgcred.CreateStoreResult": {goType: "orgcred.CreateStoreResult", wireType: "agentwire.CreateResultWire", encodeFn: "agentwire.EncodeCreateStoreResult", decodeFn: "agentwire.DecodeCreateStoreResult", rejectedType: "orgcred.CreateStoreRejected"},
			"orgcred.VerifyStoreResult": {goType: "orgcred.VerifyStoreResult", wireType: "credentialResultWire", encodeFn: "encodeVerifyResult", decodeFn: "decodeVerifyResult", rejectedType: "orgcred.VerifyStoreRejected"},
			"orgcred.ListStoreResult":   {goType: "orgcred.ListStoreResult", wireType: "listResultWire", encodeFn: "encodeListResult", decodeFn: "decodeListResult", rejectedType: "orgcred.ListStoreRejected"},
			"orgcred.RevokeStoreResult": {goType: "orgcred.RevokeStoreResult", wireType: "credentialResultWire", encodeFn: "encodeRevokeResult", decodeFn: "decodeRevokeResult", rejectedType: "orgcred.RevokeStoreRejected"},
		},
	},
	"assets": {
		bridgePackage: "assetsbridge",
		domainImport:  "github.com/e6qu/sharecrop/internal/assets",
		domainPackage: "assets",
		interfaceName: "Store",
		wirePrefix:    "assets",
		argCodecs: map[string]argCodec{
			"core.UserID":                     userIDArg(),
			"core.Page":                       pageArg(),
			"core.TaskID":                     {field: "TaskID", goType: "core.TaskID", wireType: "string", encodeFn: "corewire.EncodeTaskID", decodeFn: "corewire.DecodeTaskID"},
			"string":                          {field: "Query", goType: "string", wireType: "string", encodeFn: "corewire.EncodeString", decodeFn: "corewire.DecodeString"},
			"assets.Collectible":              {field: "Collectible", goType: "assets.Collectible", wireType: "collectibleWire", encodeFn: "encodeCollectible", decodeFn: "decodeCollectible"},
			"assets.FundRewardStoreCommand":   {field: "Command", goType: "assets.FundRewardStoreCommand", wireType: "fundCommandWire", encodeFn: "encodeFundCommand", decodeFn: "decodeFundCommand"},
			"assets.RefundRewardStoreCommand": {field: "Command", goType: "assets.RefundRewardStoreCommand", wireType: "refundCommandWire", encodeFn: "encodeRefundCommand", decodeFn: "decodeRefundCommand"},
			"assets.GiftStoreCommand":         {field: "Command", goType: "assets.GiftStoreCommand", wireType: "giftCommandWire", encodeFn: "encodeGiftCommand", decodeFn: "decodeGiftCommand"},
			"assets.AwardOrganizationCollectibleStoreCommand": {field: "Command", goType: "assets.AwardOrganizationCollectibleStoreCommand", wireType: "awardCommandWire", encodeFn: "encodeAwardCommand", decodeFn: "decodeAwardCommand"},
		},
		resultCodecs: map[string]resultCodec{
			"assets.CreateStoreResult":          {goType: "assets.CreateStoreResult", wireType: "acceptedRejectedWire", encodeFn: "encodeCreateResult", decodeFn: "decodeCreateResult", rejectedType: "assets.CreateStoreRejected"},
			"assets.ListStoreResult":            {goType: "assets.ListStoreResult", wireType: "collectiblesResultWire", encodeFn: "encodeListResult", decodeFn: "decodeListResult", rejectedType: "assets.ListStoreRejected"},
			"assets.FundRewardResult":           {goType: "assets.FundRewardResult", wireType: "collectibleResultWire", encodeFn: "encodeFundRewardResult", decodeFn: "decodeFundRewardResult", rejectedType: "assets.FundRewardRejected"},
			"assets.RefundRewardResult":         {goType: "assets.RefundRewardResult", wireType: "collectiblesResultWire", encodeFn: "encodeRefundRewardResult", decodeFn: "decodeRefundRewardResult", rejectedType: "assets.RefundRewardRejected"},
			"assets.GiftResult":                 {goType: "assets.GiftResult", wireType: "collectibleResultWire", encodeFn: "encodeGiftResult", decodeFn: "decodeGiftResult", rejectedType: "assets.GiftRejected"},
			"assets.TaskHeldCollectiblesResult": {goType: "assets.TaskHeldCollectiblesResult", wireType: "taskHeldResultWire", encodeFn: "encodeTaskHeldResult", decodeFn: "decodeTaskHeldResult", rejectedType: "assets.TaskHeldCollectiblesRejected"},
		},
	},
	"submission": {
		bridgePackage: "submissionbridge",
		domainImport:  "github.com/e6qu/sharecrop/internal/submission",
		domainPackage: "submission",
		interfaceName: "Store",
		wirePrefix:    "submission",
		argCodecs: map[string]argCodec{
			"core.UserID":                   userIDArg(),
			"core.Page":                     pageArg(),
			"core.TaskID":                   {field: "TaskID", goType: "core.TaskID", wireType: "string", encodeFn: "corewire.EncodeTaskID", decodeFn: "corewire.DecodeTaskID"},
			"core.SubmissionID":             {field: "SubmissionID", goType: "core.SubmissionID", wireType: "string", encodeFn: "corewire.EncodeSubmissionID", decodeFn: "corewire.DecodeSubmissionID"},
			"core.SubmissionReceiptTokenID": {field: "ReceiptID", goType: "core.SubmissionReceiptTokenID", wireType: "string", encodeFn: "corewire.EncodeSubmissionReceiptTokenID", decodeFn: "corewire.DecodeSubmissionReceiptTokenID"},
			"submission.ReceiptTokenHash":   {field: "ReceiptHash", goType: "submission.ReceiptTokenHash", wireType: "string", encodeFn: "encodeReceiptTokenHash", decodeFn: "decodeReceiptTokenHash"},
			"submission.SubmitCommand":      {field: "Command", goType: "submission.SubmitCommand", wireType: "submitCommandWire", encodeFn: "encodeSubmitCommand", decodeFn: "decodeSubmitCommand"},
			"submission.State":              {field: "State", goType: "submission.State", wireType: "string", encodeFn: "encodeState", decodeFn: "decodeState"},
			"submission.ValidationOutcome":  {field: "Validation", goType: "submission.ValidationOutcome", wireType: "validationOutcomeWire", encodeFn: "encodeValidationOutcome", decodeFn: "decodeValidationOutcome"},
			"[]submission.SensitiveField":   {field: "SensitiveFields", goType: "[]submission.SensitiveField", wireType: "[]sensitiveFieldWire", encodeFn: "encodeSensitiveFields", decodeFn: "decodeSensitiveFields"},
			"submission.SubmissionComment":  {field: "Comment", goType: "submission.SubmissionComment", wireType: "submissionCommentWire", encodeFn: "encodeSubmissionComment", decodeFn: "decodeSubmissionComment"},
		},
		resultCodecs: map[string]resultCodec{
			"submission.CreateSubmissionStoreResult":        {goType: "submission.CreateSubmissionStoreResult", wireType: "submissionResultWire", encodeFn: "encodeCreateSubmissionResult", decodeFn: "decodeCreateSubmissionResult", rejectedType: "submission.CreateSubmissionStoreRejected"},
			"submission.FindReceiptStoreResult":             {goType: "submission.FindReceiptStoreResult", wireType: "submissionResultWire", encodeFn: "encodeFindReceiptResult", decodeFn: "decodeFindReceiptResult", rejectedType: "submission.ReceiptMissing"},
			"submission.FindSubmissionStoreResult":          {goType: "submission.FindSubmissionStoreResult", wireType: "submissionResultWire", encodeFn: "encodeFindSubmissionResult", decodeFn: "decodeFindSubmissionResult", rejectedType: "submission.FindSubmissionStoreRejected"},
			"submission.ListSubmissionsStoreResult":         {goType: "submission.ListSubmissionsStoreResult", wireType: "submissionsResultWire", encodeFn: "encodeListSubmissionsResult", decodeFn: "decodeListSubmissionsResult", rejectedType: "submission.ListSubmissionsStoreRejected"},
			"submission.CreateSubmissionCommentStoreResult": {goType: "submission.CreateSubmissionCommentStoreResult", wireType: "submissionCommentResultWire", encodeFn: "encodeCreateCommentResult", decodeFn: "decodeCreateCommentResult", rejectedType: "submission.CreateSubmissionCommentStoreRejected"},
			"submission.ListSubmissionCommentsStoreResult":  {goType: "submission.ListSubmissionCommentsStoreResult", wireType: "submissionCommentsResultWire", encodeFn: "encodeListCommentsResult", decodeFn: "decodeListCommentsResult", rejectedType: "submission.ListSubmissionCommentsStoreRejected"},
		},
	},
	"ledger": {
		bridgePackage: "ledgerbridge",
		domainImport:  "github.com/e6qu/sharecrop/internal/ledger",
		domainPackage: "ledger",
		interfaceName: "Store",
		wirePrefix:    "ledger",
		argCodecs: map[string]argCodec{
			"core.UserID":                         userIDArg(),
			"core.Page":                           pageArg(),
			"core.TaskID":                         {field: "TaskID", goType: "core.TaskID", wireType: "string", encodeFn: "corewire.EncodeTaskID", decodeFn: "corewire.DecodeTaskID"},
			"core.OrganizationID":                 {field: "OrganizationID", goType: "core.OrganizationID", wireType: "string", encodeFn: "corewire.EncodeOrganizationID", decodeFn: "corewire.DecodeOrganizationID"},
			"ledger.FundStoreCommand":             {field: "FundCommand", goType: "ledger.FundStoreCommand", wireType: "fundCommandWire", encodeFn: "encodeFundCommand", decodeFn: "decodeFundCommand"},
			"ledger.OrganizationFundStoreCommand": {field: "OrgFundCommand", goType: "ledger.OrganizationFundStoreCommand", wireType: "orgFundCommandWire", encodeFn: "encodeOrgFundCommand", decodeFn: "decodeOrgFundCommand"},
			"ledger.AcceptStoreCommand":           {field: "AcceptCommand", goType: "ledger.AcceptStoreCommand", wireType: "acceptCommandWire", encodeFn: "encodeAcceptCommand", decodeFn: "decodeAcceptCommand"},
			"ledger.RequestChangesStoreCommand":   {field: "RequestChangesCommand", goType: "ledger.RequestChangesStoreCommand", wireType: "requestChangesCommandWire", encodeFn: "encodeRequestChangesCommand", decodeFn: "decodeRequestChangesCommand"},
			"ledger.RejectStoreCommand":           {field: "RejectCommand", goType: "ledger.RejectStoreCommand", wireType: "rejectCommandWire", encodeFn: "encodeRejectCommand", decodeFn: "decodeRejectCommand"},
			"ledger.RefundStoreCommand":           {field: "RefundCommand", goType: "ledger.RefundStoreCommand", wireType: "refundCommandWire", encodeFn: "encodeRefundCommand", decodeFn: "decodeRefundCommand"},
		},
		resultCodecs: map[string]resultCodec{
			"ledger.FundResult":           {goType: "ledger.FundResult", wireType: "fundResultWire", encodeFn: "encodeFundResult", decodeFn: "decodeFundResult", rejectedType: "ledger.FundRejected"},
			"ledger.AcceptResult":         {goType: "ledger.AcceptResult", wireType: "reviewedSubmissionWire", encodeFn: "encodeAcceptResult", decodeFn: "decodeAcceptResult", rejectedType: "ledger.AcceptRejected"},
			"ledger.RequestChangesResult": {goType: "ledger.RequestChangesResult", wireType: "changesRequestedWire", encodeFn: "encodeRequestChangesResult", decodeFn: "decodeRequestChangesResult", rejectedType: "ledger.RequestChangesRejected"},
			"ledger.RejectResult":         {goType: "ledger.RejectResult", wireType: "reviewedSubmissionWire", encodeFn: "encodeRejectResult", decodeFn: "decodeRejectResult", rejectedType: "ledger.RejectRejected"},
			"ledger.RefundResult":         {goType: "ledger.RefundResult", wireType: "fundResultWire", encodeFn: "encodeRefundResult", decodeFn: "decodeRefundResult", rejectedType: "ledger.RefundRejected"},
			"ledger.TaskAllocatedResult":  {goType: "ledger.TaskAllocatedResult", wireType: "taskAllocatedWire", encodeFn: "encodeTaskAllocatedResult", decodeFn: "decodeTaskAllocatedResult", rejectedType: "ledger.TaskAllocatedRejected"},
			"ledger.BalanceResult":        {goType: "ledger.BalanceResult", wireType: "balanceWire", encodeFn: "encodeBalanceResult", decodeFn: "decodeBalanceResult", rejectedType: "ledger.BalanceRejected"},
			"ledger.ListEntriesResult":    {goType: "ledger.ListEntriesResult", wireType: "entriesWire", encodeFn: "encodeListEntriesResult", decodeFn: "decodeListEntriesResult", rejectedType: "ledger.ListEntriesRejected"},
		},
	},
	"org": {
		bridgePackage: "orgbridge",
		domainImport:  "github.com/e6qu/sharecrop/internal/org",
		domainPackage: "org",
		interfaceName: "Store",
		wirePrefix:    "org",
		extraImports:  []string{"github.com/e6qu/sharecrop/internal/auth"},
		argCodecs: map[string]argCodec{
			"core.UserID":                   userIDArg(),
			"core.Page":                     pageArg(),
			"core.OrganizationID":           {field: "OrganizationID", goType: "core.OrganizationID", wireType: "string", encodeFn: "corewire.EncodeOrganizationID", decodeFn: "corewire.DecodeOrganizationID"},
			"core.TeamID":                   {field: "TeamID", goType: "core.TeamID", wireType: "string", encodeFn: "corewire.EncodeTeamID", decodeFn: "corewire.DecodeTeamID"},
			"core.OrganizationMembershipID": {field: "MembershipID", goType: "core.OrganizationMembershipID", wireType: "string", encodeFn: "corewire.EncodeOrganizationMembershipID", decodeFn: "corewire.DecodeOrganizationMembershipID"},
			"string":                        {field: "Search", goType: "string", wireType: "string", encodeFn: "corewire.EncodeString", decodeFn: "corewire.DecodeString"},
			"org.OrganizationName":          {field: "OrganizationName", goType: "org.OrganizationName", wireType: "string", encodeFn: "encodeOrganizationName", decodeFn: "decodeOrganizationName"},
			"org.TeamName":                  {field: "TeamName", goType: "org.TeamName", wireType: "string", encodeFn: "encodeTeamName", decodeFn: "decodeTeamName"},
			"auth.EmailAddress":             {field: "Email", goType: "auth.EmailAddress", wireType: "string", encodeFn: "encodeEmail", decodeFn: "decodeEmail"},
			"[]org.Role":                    {field: "Roles", goType: "[]org.Role", wireType: "[]string", encodeFn: "encodeRoles", decodeFn: "decodeRoles"},
		},
		resultCodecs: map[string]resultCodec{
			"org.CreateOrganizationStoreResult": {goType: "org.CreateOrganizationStoreResult", wireType: "acceptedRejectedWire", encodeFn: "encodeCreateOrganizationResult", decodeFn: "decodeCreateOrganizationResult", rejectedType: "org.CreateOrganizationStoreRejected"},
			"org.ListOrganizationsResult":       {goType: "org.ListOrganizationsResult", wireType: "organizationsResultWire", encodeFn: "encodeListOrganizationsResult", decodeFn: "decodeListOrganizationsResult", rejectedType: "org.ListOrganizationsRejected"},
			"org.MemberRolesResult":             {goType: "org.MemberRolesResult", wireType: "memberRolesResultWire", encodeFn: "encodeMemberRolesResult", decodeFn: "decodeMemberRolesResult", rejectedType: "org.MemberRolesRejected"},
			"org.ListMembersResult":             {goType: "org.ListMembersResult", wireType: "membersResultWire", encodeFn: "encodeListMembersResult", decodeFn: "decodeListMembersResult", rejectedType: "org.ListMembersRejected"},
			"org.ProvisionMemberStoreResult":    {goType: "org.ProvisionMemberStoreResult", wireType: "memberResultWire", encodeFn: "encodeProvisionMemberResult", decodeFn: "decodeProvisionMemberResult", rejectedType: "org.ProvisionMemberStoreRejected"},
			"org.DeactivateMemberStoreResult":   {goType: "org.DeactivateMemberStoreResult", wireType: "acceptedRejectedWire", encodeFn: "encodeDeactivateMemberResult", decodeFn: "decodeDeactivateMemberResult", rejectedType: "org.DeactivateMemberStoreRejected"},
			"org.UpdateMemberRolesStoreResult":  {goType: "org.UpdateMemberRolesStoreResult", wireType: "memberResultWire", encodeFn: "encodeUpdateMemberRolesResult", decodeFn: "decodeUpdateMemberRolesResult", rejectedType: "org.UpdateMemberRolesStoreRejected"},
			"org.CreateTeamStoreResult":         {goType: "org.CreateTeamStoreResult", wireType: "acceptedRejectedWire", encodeFn: "encodeCreateTeamResult", decodeFn: "decodeCreateTeamResult", rejectedType: "org.CreateTeamStoreRejected"},
			"org.AddTeamMemberStoreResult":      {goType: "org.AddTeamMemberStoreResult", wireType: "acceptedRejectedWire", encodeFn: "encodeAddTeamMemberResult", decodeFn: "decodeAddTeamMemberResult", rejectedType: "org.AddTeamMemberStoreRejected"},
			"org.TeamListResult":                {goType: "org.TeamListResult", wireType: "teamsResultWire", encodeFn: "encodeTeamListResult", decodeFn: "decodeTeamListResult", rejectedType: "org.TeamListRejected"},
			"org.FindTeamResult":                {goType: "org.FindTeamResult", wireType: "findTeamResultWire", encodeFn: "encodeFindTeamResult", decodeFn: "decodeFindTeamResult", rejectedType: "org.TeamMissing"},
			"org.TeamMembersResult":             {goType: "org.TeamMembersResult", wireType: "teamMembersResultWire", encodeFn: "encodeTeamMembersResult", decodeFn: "decodeTeamMembersResult", rejectedType: "org.TeamMembersRejected"},
		},
	},
	"task": {
		bridgePackage: "taskbridge",
		domainImport:  "github.com/e6qu/sharecrop/internal/task",
		domainPackage: "task",
		interfaceName: "Store",
		wirePrefix:    "task",
		argCodecs: map[string]argCodec{
			"core.UserID":             userIDArg(),
			"core.Page":               pageArg(),
			"core.TaskID":             {field: "TaskID", goType: "core.TaskID", wireType: "string", encodeFn: "corewire.EncodeTaskID", decodeFn: "corewire.DecodeTaskID"},
			"core.TaskSeriesID":       {field: "SeriesID", goType: "core.TaskSeriesID", wireType: "string", encodeFn: "corewire.EncodeTaskSeriesID", decodeFn: "corewire.DecodeTaskSeriesID"},
			"core.TaskReservationID":  {field: "ReservationID", goType: "core.TaskReservationID", wireType: "string", encodeFn: "corewire.EncodeTaskReservationID", decodeFn: "corewire.DecodeTaskReservationID"},
			"[]core.TaskID":           {field: "TaskIDs", goType: "[]core.TaskID", wireType: "[]string", encodeFn: "encodeTaskIDs", decodeFn: "decodeTaskIDs"},
			"task.CreateCommand":      {field: "Command", goType: "task.CreateCommand", wireType: "createCommandWire", encodeFn: "encodeCreateCommand", decodeFn: "decodeCreateCommand"},
			"task.State":              {field: "State", goType: "task.State", wireType: "string", encodeFn: "encodeState", decodeFn: "decodeState"},
			"task.ListScope":          {field: "Scope", goType: "task.ListScope", wireType: "listScopeWire", encodeFn: "encodeListScope", decodeFn: "decodeListScope"},
			"task.ListFilters":        {field: "Filters", goType: "task.ListFilters", wireType: "listFiltersWire", encodeFn: "encodeListFilters", decodeFn: "decodeListFilters"},
			"task.Series":             {field: "Series", goType: "task.Series", wireType: "seriesWire", encodeFn: "encodeSeries", decodeFn: "decodeSeries"},
			"task.SeriesTitle":        {field: "SeriesTitle", goType: "task.SeriesTitle", wireType: "string", encodeFn: "encodeSeriesTitle", decodeFn: "decodeSeriesTitle"},
			"task.SeriesDescription":  {field: "SeriesDescription", goType: "task.SeriesDescription", wireType: "string", encodeFn: "encodeSeriesDescription", decodeFn: "decodeSeriesDescription"},
			"task.SeriesState":        {field: "SeriesState", goType: "task.SeriesState", wireType: "string", encodeFn: "encodeSeriesState", decodeFn: "decodeSeriesState"},
			"task.SeriesComment":      {field: "SeriesComment", goType: "task.SeriesComment", wireType: "seriesCommentWire", encodeFn: "encodeSeriesComment", decodeFn: "decodeSeriesComment"},
			"task.TaskComment":        {field: "TaskComment", goType: "task.TaskComment", wireType: "taskCommentWire", encodeFn: "encodeTaskComment", decodeFn: "decodeTaskComment"},
			"task.ReservationCommand": {field: "Command", goType: "task.ReservationCommand", wireType: "reservationCommandWire", encodeFn: "encodeReservationCommand", decodeFn: "decodeReservationCommand"},
			"task.ReservationState":   {field: "ReservationState", goType: "task.ReservationState", wireType: "string", encodeFn: "encodeReservationState", decodeFn: "decodeReservationState"},
		},
		resultCodecs: map[string]resultCodec{
			"task.CreateTaskStoreResult":             {goType: "task.CreateTaskStoreResult", wireType: "taskResultWire", encodeFn: "encodeCreateTaskResult", decodeFn: "decodeCreateTaskResult", rejectedType: "task.CreateTaskStoreRejected"},
			"task.FindTaskStoreResult":               {goType: "task.FindTaskStoreResult", wireType: "taskResultWire", encodeFn: "encodeFindTaskResult", decodeFn: "decodeFindTaskResult", rejectedType: "task.FindTaskStoreRejected"},
			"task.ChangeTaskStateStoreResult":        {goType: "task.ChangeTaskStateStoreResult", wireType: "taskResultWire", encodeFn: "encodeChangeTaskStateResult", decodeFn: "decodeChangeTaskStateResult", rejectedType: "task.ChangeTaskStateStoreRejected"},
			"task.ListTasksStoreResult":              {goType: "task.ListTasksStoreResult", wireType: "listItemsResultWire", encodeFn: "encodeListTasksResult", decodeFn: "decodeListTasksResult", rejectedType: "task.ListTasksStoreRejected"},
			"task.ListSeriesStoreResult":             {goType: "task.ListSeriesStoreResult", wireType: "seriesListResultWire", encodeFn: "encodeListSeriesResult", decodeFn: "decodeListSeriesResult", rejectedType: "task.ListSeriesStoreRejected"},
			"task.FindSeriesStoreResult":             {goType: "task.FindSeriesStoreResult", wireType: "seriesDetailResultWire", encodeFn: "encodeFindSeriesResult", decodeFn: "decodeFindSeriesResult", rejectedType: "task.FindSeriesStoreRejected"},
			"task.SeriesMutationStoreResult":         {goType: "task.SeriesMutationStoreResult", wireType: "seriesDetailResultWire", encodeFn: "encodeSeriesMutationResult", decodeFn: "decodeSeriesMutationResult", rejectedType: "task.SeriesMutationStoreRejected"},
			"task.CreateSeriesCommentStoreResult":    {goType: "task.CreateSeriesCommentStoreResult", wireType: "seriesCommentResultWire", encodeFn: "encodeCreateSeriesCommentResult", decodeFn: "decodeCreateSeriesCommentResult", rejectedType: "task.CreateSeriesCommentStoreRejected"},
			"task.ListSeriesCommentsStoreResult":     {goType: "task.ListSeriesCommentsStoreResult", wireType: "seriesCommentsResultWire", encodeFn: "encodeListSeriesCommentsResult", decodeFn: "decodeListSeriesCommentsResult", rejectedType: "task.ListSeriesCommentsStoreRejected"},
			"task.CreateTaskCommentStoreResult":      {goType: "task.CreateTaskCommentStoreResult", wireType: "taskCommentResultWire", encodeFn: "encodeCreateTaskCommentResult", decodeFn: "decodeCreateTaskCommentResult", rejectedType: "task.CreateTaskCommentStoreRejected"},
			"task.ListTaskCommentsStoreResult":       {goType: "task.ListTaskCommentsStoreResult", wireType: "taskCommentsResultWire", encodeFn: "encodeListTaskCommentsResult", decodeFn: "decodeListTaskCommentsResult", rejectedType: "task.ListTaskCommentsStoreRejected"},
			"task.CreateReservationStoreResult":      {goType: "task.CreateReservationStoreResult", wireType: "reservationResultWire", encodeFn: "encodeCreateReservationResult", decodeFn: "decodeCreateReservationResult", rejectedType: "task.CreateReservationStoreRejected"},
			"task.ChangeReservationStateStoreResult": {goType: "task.ChangeReservationStateStoreResult", wireType: "reservationResultWire", encodeFn: "encodeChangeReservationStateResult", decodeFn: "decodeChangeReservationStateResult", rejectedType: "task.ChangeReservationStateStoreRejected"},
			"task.ListReservationsStoreResult":       {goType: "task.ListReservationsStoreResult", wireType: "reservationsResultWire", encodeFn: "encodeListReservationsResult", decodeFn: "decodeListReservationsResult", rejectedType: "task.ListReservationsStoreRejected"},
			"task.SubmissionEligibilityStoreResult":  {goType: "task.SubmissionEligibilityStoreResult", wireType: "acceptedRejectedWire", encodeFn: "encodeSubmissionEligibilityResult", decodeFn: "decodeSubmissionEligibilityResult", rejectedType: "task.SubmissionEligibilityRejected"},
		},
	},
	// savedqueueview bridges an internal/http RuntimeState service (not a domain
	// Store) so the mux running in a pooled guest reaches shared Postgres state
	// instead of per-instance in-memory state. internal/http is package httpserver.
	"savedqueueview": {
		bridgePackage: "savedqueueviewbridge",
		domainImport:  "github.com/e6qu/sharecrop/internal/http",
		domainPackage: "httpserver",
		interfaceName: "SavedQueueViewService",
		wirePrefix:    "savedqueueview",
		argCodecs: map[string]argCodec{
			"core.UserID":               userIDArg(),
			"string":                    {field: "Scope", goType: "string", wireType: "string", encodeFn: "corewire.EncodeString", decodeFn: "corewire.DecodeString"},
			"httpserver.SavedQueueView": {field: "View", goType: "httpserver.SavedQueueView", wireType: "savedQueueViewWire", encodeFn: "encodeView", decodeFn: "decodeView"},
		},
		resultCodecs: map[string]resultCodec{
			"httpserver.SavedQueueViewsListResult":    {goType: "httpserver.SavedQueueViewsListResult", wireType: "viewsResultWire", encodeFn: "encodeListResult", decodeFn: "decodeListResult", rejectedType: "httpserver.SavedQueueViewsListRejected"},
			"httpserver.SavedQueueViewMutationResult": {goType: "httpserver.SavedQueueViewMutationResult", wireType: "viewResultWire", encodeFn: "encodeMutationResult", decodeFn: "decodeMutationResult", rejectedType: "httpserver.SavedQueueViewSaveRejected"},
		},
	},
	// platformadmin bridges another internal/http RuntimeState service. Grant has
	// two core.UserID arguments, so the generator suffixes the repeated field.
	"platformadmin": {
		bridgePackage: "platformadminbridge",
		domainImport:  "github.com/e6qu/sharecrop/internal/http",
		domainPackage: "httpserver",
		interfaceName: "PlatformAdminService",
		wirePrefix:    "platformadmin",
		argCodecs: map[string]argCodec{
			"core.UserID": userIDArg(),
			"core.Page":   pageArg(),
		},
		resultCodecs: map[string]resultCodec{
			"httpserver.PlatformAdminCheckResult":    {goType: "httpserver.PlatformAdminCheckResult", wireType: "checkResultWire", encodeFn: "encodeCheckResult", decodeFn: "decodeCheckResult", rejectedType: "httpserver.PlatformAdminDenied"},
			"httpserver.PlatformAdminListResult":     {goType: "httpserver.PlatformAdminListResult", wireType: "recordsResultWire", encodeFn: "encodeListResult", decodeFn: "decodeListResult", rejectedType: "httpserver.PlatformAdminListRejected"},
			"httpserver.PlatformAdminMutationResult": {goType: "httpserver.PlatformAdminMutationResult", wireType: "recordResultWire", encodeFn: "encodeMutationResult", decodeFn: "decodeMutationResult", rejectedType: "httpserver.PlatformAdminMutationRejected"},
		},
	},
	// moderationtriage bridges an internal/http RuntimeState service. RecordOpen
	// takes an audit.Event (a third package, so extraImports), and Update takes
	// two strings (state, note) which the generator disambiguates.
	"moderationtriage": {
		bridgePackage: "moderationtriagebridge",
		domainImport:  "github.com/e6qu/sharecrop/internal/http",
		domainPackage: "httpserver",
		interfaceName: "ModerationTriageService",
		wirePrefix:    "moderationtriage",
		extraImports:  []string{"github.com/e6qu/sharecrop/internal/audit", "github.com/e6qu/sharecrop/internal/wasibridge/auditwire"},
		argCodecs: map[string]argCodec{
			"core.UserID":         userIDArg(),
			"core.AuditEventID":   {field: "ReportID", goType: "core.AuditEventID", wireType: "string", encodeFn: "corewire.EncodeAuditEventID", decodeFn: "corewire.DecodeAuditEventID"},
			"[]core.AuditEventID": {field: "IDs", goType: "[]core.AuditEventID", wireType: "[]string", encodeFn: "encodeAuditEventIDs", decodeFn: "decodeAuditEventIDs"},
			"audit.Event":         {field: "Event", goType: "audit.Event", wireType: "auditwire.EventWire", encodeFn: "auditwire.EncodeEvent", decodeFn: "auditwire.DecodeEvent"},
			"string":              {field: "State", goType: "string", wireType: "string", encodeFn: "corewire.EncodeString", decodeFn: "corewire.DecodeString"},
		},
		resultCodecs: map[string]resultCodec{
			"httpserver.ModerationTriageMutationResult": {goType: "httpserver.ModerationTriageMutationResult", wireType: "recordResultWire", encodeFn: "encodeMutationResult", decodeFn: "decodeMutationResult", rejectedType: "httpserver.ModerationTriageMutationRejected"},
			"httpserver.ModerationTriageListResult":     {goType: "httpserver.ModerationTriageListResult", wireType: "recordsResultWire", encodeFn: "encodeListResult", decodeFn: "decodeListResult", rejectedType: "httpserver.ModerationTriageListRejected"},
		},
	},
	// privacy bridges the last codegen-friendly internal/http RuntimeState service.
	// RecordSensitiveFieldAccess takes a submission.Submission (a third package, so
	// extraImports), and Resolve takes two strings which the generator disambiguates.
	"privacy": {
		bridgePackage: "privacybridge",
		domainImport:  "github.com/e6qu/sharecrop/internal/http",
		domainPackage: "httpserver",
		interfaceName: "PrivacyService",
		wirePrefix:    "privacy",
		extraImports:  []string{"github.com/e6qu/sharecrop/internal/submission", "github.com/e6qu/sharecrop/internal/wasibridge/submissionbridge"},
		argCodecs: map[string]argCodec{
			"core.UserID":           userIDArg(),
			"core.Page":             pageArg(),
			"string":                {field: "Text", goType: "string", wireType: "string", encodeFn: "corewire.EncodeString", decodeFn: "corewire.DecodeString"},
			"submission.Submission": {field: "Submission", goType: "submission.Submission", wireType: "submissionbridge.SubmissionWire", encodeFn: "submissionbridge.EncodeSubmission", decodeFn: "submissionbridge.DecodeSubmission"},
		},
		resultCodecs: map[string]resultCodec{
			"httpserver.PrivacyMutationResult":  {goType: "httpserver.PrivacyMutationResult", wireType: "recordResultWire", encodeFn: "encodeMutationResult", decodeFn: "decodeMutationResult", rejectedType: "httpserver.PrivacyRequestMutationRejected"},
			"httpserver.PrivacyListResult":      {goType: "httpserver.PrivacyListResult", wireType: "recordsResultWire", encodeFn: "encodeListResult", decodeFn: "decodeListResult", rejectedType: "httpserver.PrivacyRequestListRejected"},
			"httpserver.PrivacyRetentionResult": {goType: "httpserver.PrivacyRetentionResult", wireType: "retentionResultWire", encodeFn: "encodeRetentionResult", decodeFn: "decodeRetentionResult", rejectedType: "httpserver.PrivacyRetentionRejected"},
		},
	},
}

type method struct {
	name   string
	args   []argCodec
	result resultCodec
}

// Generate parses the given package sources (path -> content) for the store
// named by key, extracts its Store interface, and returns the formatted bridge
// source. It fails if the key is unknown, the interface is not found, or a
// method uses an unregistered type.
func Generate(sources map[string][]byte, key string) (string, error) {
	spec, known := specs[key]
	if !known {
		return "", fmt.Errorf("no bridge spec for store %q", key)
	}
	methods, err := extractMethods(sources, spec)
	if err != nil {
		return "", err
	}
	return emit(spec, methods)
}

func extractMethods(sources map[string][]byte, spec storeSpec) ([]method, error) {
	fset := token.NewFileSet()

	var iface *ast.InterfaceType
	var packageName string
	for _, path := range sortedKeys(sources) {
		file, err := parser.ParseFile(fset, path, sources[path], 0)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		packageName = file.Name.Name
		ast.Inspect(file, func(node ast.Node) bool {
			typeSpec, matched := node.(*ast.TypeSpec)
			if !matched || typeSpec.Name.Name != spec.interfaceName {
				return true
			}
			if typed, isInterface := typeSpec.Type.(*ast.InterfaceType); isInterface {
				iface = typed
			}
			return true
		})
	}
	if iface == nil {
		return nil, fmt.Errorf("interface %q not found", spec.interfaceName)
	}

	// Types local to the interface's own package appear unqualified in the AST
	// (e.g. "Event", not "audit.Event"); qualify them so registry lookups match.
	// Builtins (string, int, ...) are left alone - they are not package types. A
	// slice qualifies its element type, so "[]SensitiveField" becomes
	// "[]audit.SensitiveField", not the meaningless "audit.[]SensitiveField".
	var qualify func(typeName string) string
	qualify = func(typeName string) string {
		if strings.HasPrefix(typeName, "[]") {
			return "[]" + qualify(typeName[2:])
		}
		if strings.Contains(typeName, ".") || isBuiltinType(typeName) {
			return typeName
		}
		return packageName + "." + typeName
	}

	methods := make([]method, 0, len(iface.Methods.List))
	for _, field := range iface.Methods.List {
		if len(field.Names) != 1 {
			continue
		}
		name := field.Names[0].Name
		funcType, isFunc := field.Type.(*ast.FuncType)
		if !isFunc {
			return nil, fmt.Errorf("member %q is not a method", name)
		}

		args := make([]argCodec, 0)
		usedFields := map[string]int{}
		for _, param := range funcType.Params.List {
			paramType := qualify(typeString(param.Type))
			if paramType == "context.Context" {
				continue
			}
			codec, known := spec.argCodecs[paramType]
			if !known {
				return nil, fmt.Errorf("method %s: no codec registered for argument type %q", name, paramType)
			}
			// The field name comes from the type, so a method with two arguments
			// of the same type (e.g. ListCollectiblesByOwner(string, string, ...))
			// would collide. Suffix the second and later occurrences.
			usedFields[codec.field]++
			if usedFields[codec.field] > 1 {
				codec.field = fmt.Sprintf("%s%d", codec.field, usedFields[codec.field])
			}
			args = append(args, codec)
		}

		if funcType.Results == nil || len(funcType.Results.List) != 1 {
			return nil, fmt.Errorf("method %s: expected exactly one result", name)
		}
		resultType := qualify(typeString(funcType.Results.List[0].Type))
		result, known := spec.resultCodecs[resultType]
		if !known {
			return nil, fmt.Errorf("method %s: no codec registered for result type %q", name, resultType)
		}

		methods = append(methods, method{name: name, args: args, result: result})
	}
	return methods, nil
}

// typeString renders a type expression as source text (e.g. "core.AuditEventID").
func typeString(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.SelectorExpr:
		return typeString(typed.X) + "." + typed.Sel.Name
	case *ast.StarExpr:
		return "*" + typeString(typed.X)
	case *ast.ArrayType:
		return "[]" + typeString(typed.Elt)
	default:
		return fmt.Sprintf("<unsupported %T>", expr)
	}
}

// isBuiltinType reports whether a bare type name is a Go builtin (so it must not
// be qualified with the interface's package).
func isBuiltinType(name string) bool {
	switch name {
	case "string", "bool", "byte", "rune", "error",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"float32", "float64", "complex64", "complex128":
		return true
	default:
		return false
	}
}

func constName(methodName string) string { return "method" + methodName }

func argsType(methodName string) string {
	return strings.ToLower(methodName[:1]) + methodName[1:] + "Args"
}

func paramName(field string) string { return "arg" + field }

func emit(spec storeSpec, methods []method) (string, error) {
	usesCorewire := false
	usesAgentwire := false
	usesTime := false
	note := func(refs ...string) {
		for _, ref := range refs {
			if strings.Contains(ref, "corewire.") {
				usesCorewire = true
			}
			if strings.Contains(ref, "agentwire.") {
				usesAgentwire = true
			}
		}
	}
	for _, m := range methods {
		for _, arg := range m.args {
			note(arg.encodeFn, arg.decodeFn, arg.wireType)
			if arg.goType == "time.Time" {
				usesTime = true
			}
		}
		note(m.result.encodeFn, m.result.decodeFn, m.result.wireType)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "// Code generated by \"sharecrop generate wasi-bridge\"; DO NOT EDIT.\n\npackage %s\n\n", spec.bridgePackage)
	b.WriteString("import (\n\t\"context\"\n\t\"encoding/json\"\n\t\"fmt\"\n")
	if usesTime {
		b.WriteString("\t\"time\"\n")
	}
	b.WriteString("\n")
	fmt.Fprintf(&b, "\t%q\n", spec.domainImport)
	b.WriteString("\t\"github.com/e6qu/sharecrop/internal/core\"\n")
	for _, extra := range spec.extraImports {
		fmt.Fprintf(&b, "\t%q\n", extra)
	}
	if usesAgentwire {
		b.WriteString("\t\"github.com/e6qu/sharecrop/internal/wasibridge/agentwire\"\n")
	}
	if usesCorewire {
		b.WriteString("\t\"github.com/e6qu/sharecrop/internal/wasibridge/corewire\"\n")
	}
	b.WriteString(")\n\n")

	fmt.Fprintf(&b, "// Method names namespace each %s.%s method on the wire.\nconst (\n", spec.domainPackage, spec.interfaceName)
	for _, m := range methods {
		fmt.Fprintf(&b, "\t%s = %q\n", constName(m.name), spec.wirePrefix+"."+m.name)
	}
	b.WriteString(")\n\n")

	for _, m := range methods {
		fmt.Fprintf(&b, "type %s struct {\n", argsType(m.name))
		for _, arg := range m.args {
			fmt.Fprintf(&b, "\t%s %s `json:%q`\n", arg.field, arg.wireType, strings.ToLower(arg.field))
		}
		b.WriteString("}\n\n")
	}

	emitDispatch(&b, spec, methods)
	emitGuestStore(&b, spec, methods)

	formatted, err := format.Source([]byte(b.String()))
	if err != nil {
		return "", fmt.Errorf("format generated bridge: %w", err)
	}
	return string(formatted), nil
}

func emitDispatch(b *strings.Builder, spec storeSpec, methods []method) {
	fmt.Fprintf(b, "// Dispatch services one store call against store: decode the arguments, call the\n"+
		"// real method, encode the result. Every branch is exactly that - no business\n"+
		"// logic lives here.\n"+
		"func Dispatch(ctx context.Context, store %s.%s, method string, args []byte) ([]byte, error) {\n"+
		"\tswitch method {\n", spec.domainPackage, spec.interfaceName)
	for _, m := range methods {
		fmt.Fprintf(b, "\tcase %s:\n", constName(m.name))
		fmt.Fprintf(b, "\t\tvar decoded %s\n", argsType(m.name))
		b.WriteString("\t\tif err := json.Unmarshal(args, &decoded); err != nil {\n")
		fmt.Fprintf(b, "\t\t\treturn nil, fmt.Errorf(%q, err)\n", spec.wirePrefix+" bridge: decode "+m.name+" args: %w")
		b.WriteString("\t\t}\n")
		callArgs := []string{"ctx"}
		for _, arg := range m.args {
			fmt.Fprintf(b, "\t\t%s, err := %s(decoded.%s)\n", paramName(arg.field), arg.decodeFn, arg.field)
			b.WriteString("\t\tif err != nil {\n\t\t\treturn nil, err\n\t\t}\n")
			callArgs = append(callArgs, paramName(arg.field))
		}
		fmt.Fprintf(b, "\t\treturn json.Marshal(%s(store.%s(%s)))\n", m.result.encodeFn, m.name, strings.Join(callArgs, ", "))
	}
	fmt.Fprintf(b, "\tdefault:\n\t\treturn nil, fmt.Errorf(%q, method)\n\t}\n}\n\n", spec.wirePrefix+" bridge: unknown method %q")
}

func emitGuestStore(b *strings.Builder, spec storeSpec, methods []method) {
	fmt.Fprintf(b, "// Invoker sends a store call to the host and returns the serialized result. The\n"+
		"// guest supplies rpc.Invoke; a test can supply an in-process stand-in.\n"+
		"type Invoker func(method string, args []byte) ([]byte, error)\n\n"+
		"// GuestStore implements %s.%s by forwarding each call over an Invoker to\n"+
		"// the host, which services it against the real store. Context is not carried\n"+
		"// across the bridge; the host uses its own context for the real call.\n"+
		"type GuestStore struct {\n\tinvoke Invoker\n}\n\n"+
		"// NewGuestStore builds a GuestStore over the given invoker.\n"+
		"func NewGuestStore(invoke Invoker) GuestStore {\n\treturn GuestStore{invoke: invoke}\n}\n\n",
		spec.domainPackage, spec.interfaceName)

	for _, m := range methods {
		params := make([]string, 0, len(m.args))
		fields := make([]string, 0, len(m.args))
		for _, arg := range m.args {
			params = append(params, paramName(arg.field)+" "+arg.goType)
			fields = append(fields, arg.field+": "+arg.encodeFn+"("+paramName(arg.field)+")")
		}
		signature := "ctx context.Context"
		if len(params) > 0 {
			signature += ", " + strings.Join(params, ", ")
		}
		reject := m.result.rejectedType + "{Reason: guestError(err)}"

		fmt.Fprintf(b, "func (g GuestStore) %s(%s) %s {\n", m.name, signature, m.result.goType)
		fmt.Fprintf(b, "\targs, err := json.Marshal(%s{%s})\n", argsType(m.name), strings.Join(fields, ", "))
		fmt.Fprintf(b, "\tif err != nil {\n\t\treturn %s\n\t}\n", reject)
		fmt.Fprintf(b, "\traw, err := g.invoke(%s, args)\n", constName(m.name))
		fmt.Fprintf(b, "\tif err != nil {\n\t\treturn %s\n\t}\n", reject)
		fmt.Fprintf(b, "\tvar wire %s\n", m.result.wireType)
		fmt.Fprintf(b, "\tif err := json.Unmarshal(raw, &wire); err != nil {\n\t\treturn %s\n\t}\n", reject)
		fmt.Fprintf(b, "\tresult, err := %s(wire)\n", m.result.decodeFn)
		fmt.Fprintf(b, "\tif err != nil {\n\t\treturn %s\n\t}\n", reject)
		b.WriteString("\treturn result\n}\n\n")
	}

	fmt.Fprintf(b, "// guestError wraps a transport/serialization failure as a domain rejection so a\n"+
		"// guest-side call always returns a well-formed result.\n"+
		"func guestError(err error) core.DomainError {\n"+
		"\treturn core.NewDomainError(core.ErrorCodeInvalidState, %q+err.Error())\n}\n\n",
		spec.wirePrefix+" bridge: ")

	fmt.Fprintf(b, "// GuestStore must satisfy the real Store interface - if a method is added to\n"+
		"// %s.%s and the bridge is not regenerated, this fails to compile.\n"+
		"var _ %s.%s = GuestStore{}\n", spec.domainPackage, spec.interfaceName, spec.domainPackage, spec.interfaceName)
}

func sortedKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
