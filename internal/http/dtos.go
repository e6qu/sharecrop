package httpserver

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	SubjectKind string `json:"subject_kind"`
	SubjectID   string `json:"subject_id"`
	AccessToken string `json:"access_token"`
}

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
	Reward             taskRewardRequest        `json:"reward"`
	Participation      taskParticipationRequest `json:"participation"`
	Visibility         taskVisibilityRequest    `json:"visibility"`
	Placement          taskPlacementRequest     `json:"placement"`
	ResponseSchemaJSON string                   `json:"response_schema_json"`
	Payload            taskPayloadRequest       `json:"payload"`
}

type taskRewardRequest struct {
	Kind         string `json:"kind"`
	CreditAmount int64  `json:"credit_amount"`
}

type taskParticipationRequest struct {
	Policy                 string `json:"policy"`
	AssigneeScope          string `json:"assignee_scope"`
	ReservationExpiryHours int    `json:"reservation_expiry_hours"`
}

type submissionRequest struct {
	ResponseJSON string `json:"response_json"`
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

type taskResponse struct {
	ID                     string `json:"id"`
	OwnerKind              string `json:"owner_kind"`
	OwnerID                string `json:"owner_id"`
	Title                  string `json:"title"`
	Description            string `json:"description"`
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

type submissionResponse struct {
	ID               string                              `json:"id"`
	TaskID           string                              `json:"task_id"`
	SubmitterID      string                              `json:"submitter_id"`
	State            string                              `json:"state"`
	ResponseJSON     string                              `json:"response_json"`
	ReviewNote       string                              `json:"review_note"`
	ValidationErrors []submissionValidationErrorResponse `json:"validation_errors"`
}

type submissionsResponse struct {
	Submissions []submissionResponse `json:"submissions"`
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
