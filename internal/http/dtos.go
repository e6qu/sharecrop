package httpserver

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	SubjectKind string `json:"subject_kind"`
	SubjectID   string `json:"subject_id"`
	AccessToken string `json:"access_token"`
	Role        string `json:"role"`
}

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
	Requests []privacyRequestResponse `json:"requests"`
}

func (privacyRequestsResponse) writableResponse() {}

type privacyResolveRequest struct {
	ResolutionNote string `json:"resolution_note"`
}

type savedQueueViewRequest struct {
	Scope       string `json:"scope"`
	Name        string `json:"name"`
	Query       string `json:"query"`
	StateFilter string `json:"state_filter"`
	TypeFilter  string `json:"type_filter"`
	Sort        string `json:"sort"`
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
	Views []savedQueueViewResponse `json:"views"`
}

func (savedQueueViewsResponse) writableResponse() {}

type errorResponse struct {
	Error string `json:"error"`
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

type taskOwnerRequest struct {
	Kind           string `json:"kind"`
	UserID         string `json:"user_id"`
	TeamID         string `json:"team_id"`
	OrganizationID string `json:"organization_id"`
}

type taskVisibilityRequest struct {
	Kind           string `json:"kind"`
	UserID         string `json:"user_id"`
	TeamID         string `json:"team_id"`
	OrganizationID string `json:"organization_id"`
}

type taskPlacementRequest struct {
	Kind           string `json:"kind"`
	SeriesID       string `json:"series_id"`
	SeriesTitle    string `json:"series_title"`
	SeriesPosition int    `json:"series_position"`
}

type taskPayloadRequest struct {
	Kind string `json:"kind"`
	JSON string `json:"json"`
}

type taskRequest struct {
	Owner              taskOwnerRequest         `json:"owner"`
	Title              string                   `json:"title"`
	Description        string                   `json:"description"`
	TaskType           string                   `json:"task_type"`
	ReferenceURL       string                   `json:"reference_url"`
	Reward             taskRewardRequest        `json:"reward"`
	Participation      taskParticipationRequest `json:"participation"`
	Visibility         taskVisibilityRequest    `json:"visibility"`
	Placement          taskPlacementRequest     `json:"placement"`
	ResponseSchemaJSON string                   `json:"response_schema_json"`
	Payload            taskPayloadRequest       `json:"payload"`
}

type taskRewardRequest struct {
	Kind           string   `json:"kind"`
	CreditAmount   int64    `json:"credit_amount"`
	CollectibleIDs []string `json:"collectible_ids"`
}

type taskParticipationRequest struct {
	Policy                 string `json:"policy"`
	AssigneeScope          string `json:"assignee_scope"`
	ReservationExpiryHours int    `json:"reservation_expiry_hours"`
}

type submissionRequest struct {
	ResponseJSON string `json:"response_json"`
}

type reservationRequest struct {
	AssigneeKind   string `json:"assignee_kind"`
	OrganizationID string `json:"organization_id"`
	TeamID         string `json:"team_id"`
}

type organizationResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedBy string `json:"created_by"`
}

type organizationsResponse struct {
	Organizations []organizationResponse `json:"organizations"`
}

type organizationMemberResponse struct {
	ID             string   `json:"id"`
	OrganizationID string   `json:"organization_id"`
	UserID         string   `json:"user_id"`
	Status         string   `json:"status"`
	Roles          []string `json:"roles"`
}

type organizationMembersResponse struct {
	Members []organizationMemberResponse `json:"members"`
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
	Teams []teamResponse `json:"teams"`
}

type userDirectoryEntryResponse struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Status string `json:"status"`
}

type usersResponse struct {
	Users []userDirectoryEntryResponse `json:"users"`
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
	ParticipationPolicy    string `json:"participation_policy"`
	AssigneeScope          string `json:"assignee_scope"`
	ReservationExpiryHours int    `json:"reservation_expiry_hours"`
	State                  string `json:"state"`
	VisibilityKind         string `json:"visibility_kind"`
	VisibilityID           string `json:"visibility_id"`
	SeriesKind             string `json:"series_kind"`
	SeriesID               string `json:"series_id"`
	SeriesPosition         int    `json:"series_position"`
	ResponseSchemaJSON     string `json:"response_schema_json"`
	PayloadKind            string `json:"payload_kind"`
	PayloadJSON            string `json:"payload_json"`
	CreatedBy              string `json:"created_by"`
	AvailabilityKind       string `json:"availability_kind"`
	ViewerAction           string `json:"viewer_action"`
	ReviewerAction         string `json:"reviewer_action"`
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
}

type tasksResponse struct {
	Tasks []taskListItemResponse `json:"tasks"`
}

type taskCapabilityTokenResponse struct {
	ID     string `json:"id"`
	TaskID string `json:"task_id"`
	State  string `json:"state"`
	Token  string `json:"token"`
}

type reservationResponse struct {
	ID           string `json:"id"`
	TaskID       string `json:"task_id"`
	AssigneeKind string `json:"assignee_kind"`
	AssigneeID   string `json:"assignee_id"`
	State        string `json:"state"`
	RequestedBy  string `json:"requested_by"`
}

type reservationsResponse struct {
	Reservations []reservationResponse `json:"reservations"`
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
	ID               string                              `json:"id"`
	TaskID           string                              `json:"task_id"`
	SubmitterID      string                              `json:"submitter_id"`
	State            string                              `json:"state"`
	ResponseJSON     string                              `json:"response_json"`
	ReviewNote       string                              `json:"review_note"`
	ValidationErrors []submissionValidationErrorResponse `json:"validation_errors"`
	SensitiveFields  []submissionSensitiveFieldResponse  `json:"sensitive_fields"`
}

type submissionsResponse struct {
	Submissions []submissionResponse `json:"submissions"`
}

type notificationResponse struct {
	ID              string `json:"id"`
	RecipientUserID string `json:"recipient_user_id"`
	ActorUserID     string `json:"actor_user_id"`
	Kind            string `json:"kind"`
	SubjectKind     string `json:"subject_kind"`
	SubjectID       string `json:"subject_id"`
	State           string `json:"state"`
	MetadataJSON    string `json:"metadata_json"`
	CreatedAt       string `json:"created_at"`
}

type notificationsResponse struct {
	Notifications []notificationResponse `json:"notifications"`
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
	OrganizationID string `json:"organization_id"`
}

type idempotentRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
}

type acceptSubmissionRequest struct {
	IdempotencyKey   string `json:"idempotency_key"`
	PayoutAmount     int64  `json:"payout_amount"`
	TipAmount        int64  `json:"tip_amount"`
	TipCollectibleID string `json:"tip_collectible_id"`
}

type requestChangesRequest struct {
	ReviewNote string `json:"review_note"`
}

type rejectSubmissionRequest struct {
	IdempotencyKey      string `json:"idempotency_key"`
	ReviewNote          string `json:"review_note"`
	PartialCreditAmount int64  `json:"partial_credit_amount"`
	TipAmount           int64  `json:"tip_amount"`
	BanImplementor      bool   `json:"ban_implementor"`
}

type writableResponse interface {
	writableResponse()
}

type balanceResponse struct {
	Amount int64 `json:"amount"`
}

type ledgerEntryResponse struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Amount int64  `json:"amount"`
	TaskID string `json:"task_id"`
}

type ledgerListResponse struct {
	Entries []ledgerEntryResponse `json:"entries"`
}

type taskEscrowResponse struct {
	TaskID string `json:"task_id"`
	Amount int64  `json:"amount"`
	State  string `json:"state"`
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
