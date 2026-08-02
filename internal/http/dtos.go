package httpserver

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	// DisplayName is optional: absent or empty derives the display name from
	// the email's local part.
	DisplayName string `json:"display_name,omitempty"`
}

type authResponse struct {
	SubjectKind string `json:"subject_kind"`
	SubjectID   string `json:"subject_id"`
	AccessToken string `json:"access_token"`
	Role        string `json:"role"`
	// Username is the identity the provider asserted, shown in the application
	// shell. Empty for a session that did not come from the provider.
	Username string `json:"username"`
	// DisplayName is the signed-in user's display name. Empty for guest
	// sessions, which have no profile.
	DisplayName string `json:"display_name"`
	// EmailVerificationState is "unverified" or "verified" for user sessions
	// (a fresh registration is always unverified until the email-verification
	// confirm lands the signup grant). Empty for guest sessions, which have
	// no email.
	EmailVerificationState string `json:"email_verification_state"`
}

// accountProfileResponse is the signed-in user's own profile.
type accountProfileResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	// EmailVerificationState is "unverified" or "verified"; the signup credit
	// grant lands when the account first becomes verified.
	EmailVerificationState string `json:"email_verification_state"`
}

func (accountProfileResponse) writableResponse() {}

type displayNameRequest struct {
	DisplayName string `json:"display_name"`
}

type logoutResponse struct {
	LogoutURL string `json:"logout_url"`
}

func (logoutResponse) writableResponse() {}

type accountTokenResponse struct {
	Token string `json:"token"`
}

func (accountTokenResponse) writableResponse() {}

type accountTokenSentResponse struct {
	Status string `json:"status"`
}

func (accountTokenSentResponse) writableResponse() {}

type accountTokenRequest struct {
	Token string `json:"token"`
}

type passwordResetRequest struct {
	Email string `json:"email"`
}

type passwordResetConfirmRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

type passwordChangeRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type accountProfileRequest struct {
	Email string `json:"email"`
}

type privacyRequest struct {
	Kind string `json:"kind"`
}

type privacyRequestResponse struct {
	ID                 string `json:"id"`
	Kind               string `json:"kind"`
	Status             string `json:"status"`
	RequestedBy        string `json:"requested_by"`
	ExportJSON         string `json:"export_json"`
	ResolutionNote     string `json:"resolution_note"`
	CreatedAt          string `json:"created_at"`
	ResolvedAt         string `json:"resolved_at"`
	RedactedFieldCount int    `json:"redacted_field_count"`
}

func (privacyRequestResponse) writableResponse() {}

type privacyRequestsResponse struct {
	Requests   []privacyRequestResponse `json:"requests"`
	NextOffset int                      `json:"next_offset"`
}

func (privacyRequestsResponse) writableResponse() {}

type privacyResolveRequest struct {
	// ResolutionNote is optional; a resolution may carry no note.
	ResolutionNote string `json:"resolution_note,omitempty"`
}

type privacyRetentionRunResponse struct {
	RedactedFieldCount int `json:"redacted_field_count"`
}

func (privacyRetentionRunResponse) writableResponse() {}

type moderationReportRequest struct {
	SubjectKind string `json:"subject_kind"`
	SubjectID   string `json:"subject_id"`
	Reason      string `json:"reason"`
	Details     string `json:"details,omitempty"`
}

type moderationReportResponse struct {
	ID             string `json:"id"`
	SubjectKind    string `json:"subject_kind"`
	SubjectID      string `json:"subject_id"`
	SubjectHref    string `json:"subject_href"`
	Reason         string `json:"reason"`
	Details        string `json:"details"`
	ReporterUserID string `json:"reporter_user_id"`
	State          string `json:"state"`
	ResolutionNote string `json:"resolution_note"`
	UpdatedBy      string `json:"updated_by"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

func (moderationReportResponse) writableResponse() {}

type moderationReportsResponse struct {
	Reports    []moderationReportResponse `json:"reports"`
	NextOffset int                        `json:"next_offset"`
	// Total counts every row matching the filter, ignoring limit/offset.
	Total int64 `json:"total"`
}

func (moderationReportsResponse) writableResponse() {}

type moderationTriageRequest struct {
	State          string `json:"state"`
	ResolutionNote string `json:"resolution_note,omitempty"`
}

type platformAdminRequest struct {
	UserID string `json:"user_id"`
}

type platformAdminResponse struct {
	UserID    string `json:"user_id"`
	Source    string `json:"source"`
	CreatedAt string `json:"created_at"`
}

func (platformAdminResponse) writableResponse() {}

type platformAdminsResponse struct {
	Admins     []platformAdminResponse `json:"admins"`
	NextOffset int                     `json:"next_offset"`
}

func (platformAdminsResponse) writableResponse() {}

type savedQueueViewRequest struct {
	Scope string `json:"scope"`
	Name  string `json:"name"`
	// The filter fields are optional; an absent value stores an empty
	// filter.
	Query       string `json:"query,omitempty"`
	StateFilter string `json:"state_filter,omitempty"`
	TypeFilter  string `json:"type_filter,omitempty"`
	Sort        string `json:"sort,omitempty"`
}

type savedQueueViewResponse struct {
	ID          string `json:"id"`
	Scope       string `json:"scope"`
	Name        string `json:"name"`
	Query       string `json:"query"`
	StateFilter string `json:"state_filter"`
	TypeFilter  string `json:"type_filter"`
	Sort        string `json:"sort"`
}

func (savedQueueViewResponse) writableResponse() {}

type savedQueueViewsResponse struct {
	Views      []savedQueueViewResponse `json:"views"`
	NextOffset int                      `json:"next_offset"`
}

func (savedQueueViewsResponse) writableResponse() {}

type errorResponse struct {
	Error string `json:"error"`
	// Code is the machine-readable error code (core.ErrorCode wire value):
	// one of invalid_id, invalid_enum, invalid_state, invalid_argument,
	// not_found, permission_denied, conflict, unauthenticated, rate_limited,
	// budget_exceeded, or unavailable.
	Code string `json:"code"`
}

type organizationRequest struct {
	Name string `json:"name"`
}

type provisionMemberRequest struct {
	Email string   `json:"email"`
	Roles []string `json:"roles"`
}

type updateMemberRolesRequest struct {
	Roles []string `json:"roles"`
}

type teamRequest struct {
	Name string `json:"name"`
}

// taskOwnerRequest is a discriminated union: kind selects which id field is
// required (user_id for user, team_id for team/organization_team,
// organization_id for organization/organization_team).
type taskOwnerRequest struct {
	Kind           string `json:"kind"`
	UserID         string `json:"user_id,omitempty"`
	TeamID         string `json:"team_id,omitempty"`
	OrganizationID string `json:"organization_id,omitempty"`
}

// taskVisibilityRequest is a discriminated union like taskOwnerRequest; kind
// "default" derives the visibility from the owner without an id field.
type taskVisibilityRequest struct {
	Kind           string `json:"kind"`
	UserID         string `json:"user_id,omitempty"`
	TeamID         string `json:"team_id,omitempty"`
	OrganizationID string `json:"organization_id,omitempty"`
}

// taskPlacementRequest is a discriminated union: kind standalone needs no
// other field; new_series requires series_title and series_position;
// existing_series requires series_id and series_position.
type taskPlacementRequest struct {
	Kind           string `json:"kind"`
	SeriesID       string `json:"series_id,omitempty"`
	SeriesTitle    string `json:"series_title,omitempty"`
	SeriesPosition int    `json:"series_position,omitempty"`
}

// taskPayloadRequest is a discriminated union: kind none needs no payload;
// kind json requires the json field.
type taskPayloadRequest struct {
	Kind string `json:"kind"`
	JSON string `json:"json,omitempty"`
}

type attachmentRequest struct {
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	DataURL     string `json:"data_url"`
}

type taskRequest struct {
	Owner       taskOwnerRequest `json:"owner"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	// TaskType is optional; absent means general.
	TaskType string `json:"task_type,omitempty"`
	// ReferenceURL is optional.
	ReferenceURL string            `json:"reference_url,omitempty"`
	Reward       taskRewardRequest `json:"reward"`
	// Participation is optional: every sub-field has a served default (open
	// policy, user assignee scope, the default reservation TTL).
	Participation      taskParticipationRequest `json:"participation,omitempty"`
	Visibility         taskVisibilityRequest    `json:"visibility"`
	Placement          taskPlacementRequest     `json:"placement"`
	ResponseSchemaJSON string                   `json:"response_schema_json"`
	Payload            taskPayloadRequest       `json:"payload"`
	Attachments        []attachmentRequest      `json:"attachments,omitempty"`
	// ExpiresAt is an optional RFC3339 instant after which an open task
	// expires. Empty or absent means no expiration.
	ExpiresAt string `json:"expires_at,omitempty"`
}

// taskRewardRequest is a discriminated union: kind credit/bundle requires a
// positive credit_amount; kind collectible requires collectible_ids (a
// bundle may omit them and declares a single-collectible reward).
type taskRewardRequest struct {
	Kind           string   `json:"kind"`
	CreditAmount   int64    `json:"credit_amount,omitempty"`
	CollectibleIDs []string `json:"collectible_ids,omitempty"`
}

type taskParticipationRequest struct {
	Policy                 string `json:"policy,omitempty"`
	AssigneeScope          string `json:"assignee_scope,omitempty"`
	ReservationExpiryHours int    `json:"reservation_expiry_hours,omitempty"`
}

type submissionRequest struct {
	ResponseJSON string              `json:"response_json"`
	Attachments  []attachmentRequest `json:"attachments,omitempty"`
}

// reservationRequest is optional in its entirety (an empty body reserves for
// the acting user). assignee_kind organization_team requires organization_id
// and team_id; assignee_kind team requires team_id.
type reservationRequest struct {
	AssigneeKind   string `json:"assignee_kind,omitempty"`
	OrganizationID string `json:"organization_id,omitempty"`
	TeamID         string `json:"team_id,omitempty"`
}

type organizationResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedBy string `json:"created_by"`
}

type organizationsResponse struct {
	Organizations []organizationResponse `json:"organizations"`
	NextOffset    int                    `json:"next_offset"`
}

type organizationMemberResponse struct {
	ID             string   `json:"id"`
	OrganizationID string   `json:"organization_id"`
	UserID         string   `json:"user_id"`
	Status         string   `json:"status"`
	Roles          []string `json:"roles"`
}

type organizationMembersResponse struct {
	Members    []organizationMemberResponse `json:"members"`
	NextOffset int                          `json:"next_offset"`
}

type teamResponse struct {
	ID             string `json:"id"`
	OwnerKind      string `json:"owner_kind"`
	OrganizationID string `json:"organization_id"`
	OwnerUserID    string `json:"owner_user_id"`
	Name           string `json:"name"`
	CreatedBy      string `json:"created_by"`
}

type teamsResponse struct {
	Teams      []teamResponse `json:"teams"`
	NextOffset int            `json:"next_offset"`
}

type userDirectoryEntryResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
}

type usersResponse struct {
	Users      []userDirectoryEntryResponse `json:"users"`
	NextOffset int                          `json:"next_offset"`
}

type taskResponse struct {
	ID                     string `json:"id"`
	OwnerKind              string `json:"owner_kind"`
	OwnerID                string `json:"owner_id"`
	Title                  string `json:"title"`
	Description            string `json:"description"`
	TaskType               string `json:"task_type"`
	ReferenceURL           string `json:"reference_url"`
	RewardKind             string `json:"reward_kind"`
	RewardCreditAmount     int64  `json:"reward_credit_amount"`
	RewardCollectibleCount int    `json:"reward_collectible_count"`
	// AllocatedCredits / AllocatedCollectibleIDs report the reward actually
	// held (allocated) for this task right now - zero/empty until it is
	// funded, and zero/empty again after it is awarded or refunded. Distinct
	// from the Reward* fields, which are the *declared* reward. Collectibles
	// are non-fungible, so the held collectibles are listed individually by
	// id rather than counted. Populated only on the task detail response;
	// list items leave them zero/empty.
	AllocatedCredits        int64                `json:"allocated_credits"`
	AllocatedCollectibleIDs []string             `json:"allocated_collectible_ids"`
	ParticipationPolicy     string               `json:"participation_policy"`
	AssigneeScope           string               `json:"assignee_scope"`
	ReservationExpiryHours  int                  `json:"reservation_expiry_hours"`
	State                   string               `json:"state"`
	VisibilityKind          string               `json:"visibility_kind"`
	VisibilityID            string               `json:"visibility_id"`
	SeriesKind              string               `json:"series_kind"`
	SeriesID                string               `json:"series_id"`
	SeriesPosition          int                  `json:"series_position"`
	ResponseSchemaJSON      string               `json:"response_schema_json"`
	PayloadKind             string               `json:"payload_kind"`
	PayloadJSON             string               `json:"payload_json"`
	Attachments             []attachmentResponse `json:"attachments"`
	CreatedBy               string               `json:"created_by"`
	AvailabilityKind        string               `json:"availability_kind"`
	ViewerAction            string               `json:"viewer_action"`
	ReviewerAction          string               `json:"reviewer_action"`
	// ExpiresAt is the task's expiration instant in RFC3339, or empty when
	// the task has no expiration policy.
	ExpiresAt string `json:"expires_at"`
	// CreatorDisplayName names the user who created the task. Like the
	// Allocated* fields it is resolved on the task detail read path; create
	// and state-change responses leave it empty.
	CreatorDisplayName string `json:"creator_display_name"`
}

type attachmentResponse struct {
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	SizeBytes   int    `json:"size_bytes"`
	DataURL     string `json:"data_url"`
}

type taskListItemResponse struct {
	ID                     string `json:"id"`
	OwnerKind              string `json:"owner_kind"`
	Title                  string `json:"title"`
	RewardKind             string `json:"reward_kind"`
	RewardCreditAmount     int64  `json:"reward_credit_amount"`
	RewardCollectibleCount int    `json:"reward_collectible_count"`
	ParticipationPolicy    string `json:"participation_policy"`
	AssigneeScope          string `json:"assignee_scope"`
	ReservationExpiryHours int    `json:"reservation_expiry_hours"`
	State                  string `json:"state"`
	VisibilityKind         string `json:"visibility_kind"`
	AvailabilityKind       string `json:"availability_kind"`
	ViewerAction           string `json:"viewer_action"`
	ReviewerAction         string `json:"reviewer_action"`
	CreatedBy              string `json:"created_by"`
	ActiveAssigneeKind     string `json:"active_assignee_kind"`
	ActiveAssigneeID       string `json:"active_assignee_id"`
	// CreatorDisplayName names the user who created the task, on every
	// listing.
	CreatorDisplayName string `json:"creator_display_name"`
	// HolderDisplayName names the user holding the active reservation when
	// one exists and the reservation is user-assigned; empty otherwise.
	HolderDisplayName string `json:"holder_display_name"`
	// Funded reports whether the task's declared credit reward is currently
	// escrowed: reward_funded, reward_unfunded, or no_credit_reward for
	// tasks that declare no credit reward.
	Funded string `json:"funded"`
	// PendingReviewCount is the number of submissions still awaiting review
	// (state "submitted"). It is populated only for tasks the caller created
	// and is 0 on every other row, so a listing never leaks another
	// requester's review queue.
	PendingReviewCount int64 `json:"pending_review_count"`
}

type tasksResponse struct {
	Tasks      []taskListItemResponse `json:"tasks"`
	NextOffset int                    `json:"next_offset"`
	// Total counts every row matching the filter, ignoring limit/offset.
	Total int64 `json:"total"`
}

type reservationResponse struct {
	ID           string `json:"id"`
	TaskID       string `json:"task_id"`
	AssigneeKind string `json:"assignee_kind"`
	AssigneeID   string `json:"assignee_id"`
	State        string `json:"state"`
	RequestedBy  string `json:"requested_by"`
	// IssuedWorkerCredential is a one-time plaintext secret for a new
	// task-scoped agent credential, present only immediately after this
	// reservation was created or approved into an active state — never
	// re-shown afterward, matching the one-shot-reveal convention used for
	// every other credential-minting response in this codebase.
	IssuedWorkerCredential string `json:"issued_worker_credential"`
	// HolderDisplayName names the user who requested the reservation.
	HolderDisplayName string `json:"holder_display_name"`
}

type reservationsResponse struct {
	Reservations []reservationResponse `json:"reservations"`
	NextOffset   int                   `json:"next_offset"`
}

type submissionValidationErrorResponse struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type submissionSensitiveFieldResponse struct {
	Path       string `json:"path"`
	Category   string `json:"category"`
	Retention  string `json:"retention"`
	Redaction  string `json:"redaction"`
	State      string `json:"state"`
	RedactedAt string `json:"redacted_at"`
}

type submissionResponse struct {
	ID          string `json:"id"`
	TaskID      string `json:"task_id"`
	SubmitterID string `json:"submitter_id"`
	// SubmitterDisplayName names the user who submitted the work.
	SubmitterDisplayName string                              `json:"submitter_display_name"`
	State                string                              `json:"state"`
	ResponseJSON         string                              `json:"response_json"`
	ReviewNote           string                              `json:"review_note"`
	Attachments          []attachmentResponse                `json:"attachments"`
	ValidationErrors     []submissionValidationErrorResponse `json:"validation_errors"`
	SensitiveFields      []submissionSensitiveFieldResponse  `json:"sensitive_fields"`
}

type submissionsResponse struct {
	Submissions []submissionResponse `json:"submissions"`
	NextOffset  int                  `json:"next_offset"`
	// Total counts every row matching the filter, ignoring limit/offset.
	Total int64 `json:"total"`
}

type notificationResponse struct {
	ID              string `json:"id"`
	RecipientUserID string `json:"recipient_user_id"`
	ActorUserID     string `json:"actor_user_id"`
	// ActorDisplayName names the acting user; empty for system-actor
	// notifications.
	ActorDisplayName string `json:"actor_display_name"`
	Kind             string `json:"kind"`
	SubjectKind      string `json:"subject_kind"`
	SubjectID        string `json:"subject_id"`
	// SubjectTitle is the subject task's title when the subject is a task;
	// empty for other subject kinds.
	SubjectTitle string `json:"subject_title"`
	State        string `json:"state"`
	MetadataJSON string `json:"metadata_json"`
	CreatedAt    string `json:"created_at"`
}

type notificationsResponse struct {
	Notifications []notificationResponse `json:"notifications"`
	NextOffset    int                    `json:"next_offset"`
	// Total counts every row matching the filter, ignoring limit/offset.
	Total int64 `json:"total"`
}

type notificationUnreadCountResponse struct {
	UnreadCount int64 `json:"unread_count"`
}

type submissionCreatedResponse struct {
	Submission   submissionResponse `json:"submission"`
	ReceiptToken string             `json:"receipt_token"`
}

type emptyResponse struct {
	Status string `json:"status"`
}

type fundingRequest struct {
	Amount         int64  `json:"amount"`
	IdempotencyKey string `json:"idempotency_key"`
	// OrganizationID is optional: present, it funds from the organization's
	// balance (the caller needs its billing permission); absent, from the
	// caller's own.
	OrganizationID string `json:"organization_id,omitempty"`
}

type acceptSubmissionRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	// PayoutAmount is optional: absent (or 0) pays the full allocated
	// reward; a positive value pays that partial amount.
	PayoutAmount int64 `json:"payout_amount,omitempty"`
	// TipAmount and TipCollectibleID are optional extras from the
	// requester's own balance and collection.
	TipAmount        int64  `json:"tip_amount,omitempty"`
	TipCollectibleID string `json:"tip_collectible_id,omitempty"`
}

type requestChangesRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	ReviewNote     string `json:"review_note"`
}

type rejectSubmissionRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	ReviewNote     string `json:"review_note"`
	// PartialCreditAmount is optional: absent (or 0) pays nothing.
	PartialCreditAmount int64 `json:"partial_credit_amount,omitempty"`
	TipAmount           int64 `json:"tip_amount,omitempty"`
	// BanSelection is the BanSelection contract enum: "none" or
	// "ban_implementor". Absent or empty means "none".
	BanSelection string `json:"ban_selection,omitempty"`
}

type writableResponse interface {
	writableResponse()
}

type balanceResponse struct {
	SpendableCredits int64 `json:"spendable_credits"`
	AllocatedCredits int64 `json:"allocated_credits"`
}

type ledgerEntryResponse struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Amount int64  `json:"amount"`
	TaskID string `json:"task_id"`
	// Note is the entry's stored note (for example the required explanation
	// on a platform-admin credit grant); empty for entry kinds that record
	// no note.
	Note string `json:"note"`
}

type ledgerListResponse struct {
	Entries    []ledgerEntryResponse `json:"entries"`
	NextOffset int                   `json:"next_offset"`
	// Total counts every row matching the filter, ignoring limit/offset.
	Total int64 `json:"total"`
}

type taskFundResponse struct {
	TaskID       string `json:"task_id"`
	CreditAmount int64  `json:"credit_amount"`
}

type acceptSubmissionResponse struct {
	TaskID         string   `json:"task_id"`
	SubmissionID   string   `json:"submission_id"`
	PayoutKind     string   `json:"payout_kind"`
	PayoutAmount   int64    `json:"payout_amount"`
	WorkerUserID   string   `json:"worker_user_id"`
	CollectibleIDs []string `json:"collectible_ids"`
	TipAmount      int64    `json:"tip_amount"`
}

// creditGrantRequest is the platform-admin manual credit grant: target_kind
// is "user" or "organization", target_id the matching account owner id,
// amount a positive credit amount, note the required explanation, and
// idempotency_key the caller's replay guard.
type creditGrantRequest struct {
	TargetKind     string `json:"target_kind"`
	TargetID       string `json:"target_id"`
	Amount         int64  `json:"amount"`
	Note           string `json:"note"`
	IdempotencyKey string `json:"idempotency_key"`
}

type creditGrantResponse struct {
	EntryID string `json:"entry_id"`
	Amount  int64  `json:"amount"`
}

func (creditGrantResponse) writableResponse() {}

// creditTransferRequest is a peer credit send. source_kind is "self" (the
// caller's own balance) or "organization" (source_organization_id names the
// paying organization; the caller needs its billing permission). target_kind
// is "user" or "organization" and target_id the matching account owner id.
// note is an optional message recorded on both ledger rows; idempotency_key
// is the caller's replay guard.
type creditTransferRequest struct {
	SourceKind           string `json:"source_kind"`
	SourceOrganizationID string `json:"source_organization_id,omitempty"`
	TargetKind           string `json:"target_kind"`
	TargetID             string `json:"target_id"`
	Amount               int64  `json:"amount"`
	Note                 string `json:"note,omitempty"`
	IdempotencyKey       string `json:"idempotency_key"`
}

// creditTransferResponse reports the sender-side ledger entry of a completed
// peer credit send (a replayed idempotency key returns the original entry).
type creditTransferResponse struct {
	EntryID string `json:"entry_id"`
	Amount  int64  `json:"amount"`
}

func (creditTransferResponse) writableResponse() {}

type reviewSubmissionResponse struct {
	TaskID       string `json:"task_id"`
	SubmissionID string `json:"submission_id"`
	State        string `json:"state"`
	ReviewNote   string `json:"review_note"`
	PayoutKind   string `json:"payout_kind"`
	PayoutAmount int64  `json:"payout_amount"`
	WorkerUserID string `json:"worker_user_id"`
	TipAmount    int64  `json:"tip_amount"`
}
