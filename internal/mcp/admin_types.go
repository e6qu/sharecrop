package mcp

import (
	"time"

	"github.com/e6qu/sharecrop/internal/core"
)

// Platform admin, moderation, and privacy have no standalone domain package
// (they live entirely inside internal/http as in-memory services), so these
// types mirror just the fields MCP tools need rather than importing
// internal/http directly, which would create an import cycle (internal/http
// already depends on internal/mcp for mcp.Server).

type PlatformAdminRecord struct {
	UserID    core.UserID
	Source    string
	CreatedAt time.Time
}

type PlatformAdminListResult interface{ platformAdminListResult() }
type PlatformAdminsListed struct{ Values []PlatformAdminRecord }
type PlatformAdminListRejected struct{ Reason core.DomainError }

func (PlatformAdminsListed) platformAdminListResult()      {}
func (PlatformAdminListRejected) platformAdminListResult() {}

type PlatformAdminMutationResult interface{ platformAdminMutationResult() }
type PlatformAdminSaved struct{ Value PlatformAdminRecord }
type PlatformAdminMutationRejected struct{ Reason core.DomainError }

func (PlatformAdminSaved) platformAdminMutationResult()            {}
func (PlatformAdminMutationRejected) platformAdminMutationResult() {}

type ModerationReport struct {
	ID             string
	SubjectKind    string
	SubjectID      string
	Reason         string
	Details        string
	ReporterUserID string
	State          string
	ResolutionNote string
	UpdatedBy      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ModerationReportResult interface{ moderationReportResult() }
type ModerationReportSaved struct{ Value ModerationReport }
type ModerationReportRejected struct{ Reason core.DomainError }

func (ModerationReportSaved) moderationReportResult()    {}
func (ModerationReportRejected) moderationReportResult() {}

type ModerationReportsListResult interface{ moderationReportsListResult() }
type ModerationReportsListed struct{ Values []ModerationReport }
type ModerationReportsListRejected struct{ Reason core.DomainError }

func (ModerationReportsListed) moderationReportsListResult()       {}
func (ModerationReportsListRejected) moderationReportsListResult() {}

type PrivacyRequestRecord struct {
	ID                 string
	RequestedBy        core.UserID
	Kind               string
	State              string
	ExportJSON         string
	ResolutionNote     string
	CreatedAt          time.Time
	ResolvedAt         time.Time
	RedactedFieldCount int
}

type PrivacyRequestResult interface{ privacyRequestResult() }
type PrivacyRequestSaved struct{ Value PrivacyRequestRecord }
type PrivacyRequestRejected struct{ Reason core.DomainError }

func (PrivacyRequestSaved) privacyRequestResult()    {}
func (PrivacyRequestRejected) privacyRequestResult() {}

type PrivacyRequestsListResult interface{ privacyRequestsListResult() }
type PrivacyRequestsListed struct{ Values []PrivacyRequestRecord }
type PrivacyRequestsListRejected struct{ Reason core.DomainError }

func (PrivacyRequestsListed) privacyRequestsListResult()       {}
func (PrivacyRequestsListRejected) privacyRequestsListResult() {}

type PrivacyRetentionResult interface{ privacyRetentionResult() }
type PrivacyRetentionRun struct{ RedactedFieldCount int }
type PrivacyRetentionRejected struct{ Reason core.DomainError }

func (PrivacyRetentionRun) privacyRetentionResult()      {}
func (PrivacyRetentionRejected) privacyRetentionResult() {}
