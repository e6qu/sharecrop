package httpserver

import (
	"encoding/json"
	"testing"
)

// These fixtures pin the wire JSON shape of API responses so accidental
// field renames or shape changes are caught before they reach clients and
// the generated Elm contracts drift.

func TestAuthResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(authResponse{SubjectKind: "user", SubjectID: "subject-1", AccessToken: "token-1", Role: "member"})
	assertWireShape(t, encoded, err, `{"subject_kind":"user","subject_id":"subject-1","access_token":"token-1","role":"member"}`)
}

func TestAuthRequestWireShape(t *testing.T) {
	encoded, err := json.Marshal(authRequest{Email: "person@example.com", Password: "correct horse battery staple"})
	assertWireShape(t, encoded, err, `{"email":"person@example.com","password":"correct horse battery staple"}`)
}

func TestAccountTokenResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(accountTokenResponse{Token: "account-token-1"})
	assertWireShape(t, encoded, err, `{"token":"account-token-1"}`)
}

func TestAccountTokenSentResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(accountTokenSentResponse{Status: "sent"})
	assertWireShape(t, encoded, err, `{"status":"sent"}`)
}

func TestAccountTokenRequestWireShape(t *testing.T) {
	encoded, err := json.Marshal(accountTokenRequest{Token: "account-token-1"})
	assertWireShape(t, encoded, err, `{"token":"account-token-1"}`)
}

func TestPasswordResetRequestWireShape(t *testing.T) {
	encoded, err := json.Marshal(passwordResetRequest{Email: "person@example.com"})
	assertWireShape(t, encoded, err, `{"email":"person@example.com"}`)
}

func TestPasswordResetConfirmRequestWireShape(t *testing.T) {
	encoded, err := json.Marshal(passwordResetConfirmRequest{Token: "reset-token-1", Password: "changed horse battery staple"})
	assertWireShape(t, encoded, err, `{"token":"reset-token-1","password":"changed horse battery staple"}`)
}

func TestPasswordChangeRequestWireShape(t *testing.T) {
	encoded, err := json.Marshal(passwordChangeRequest{CurrentPassword: "old password", NewPassword: "new password"})
	assertWireShape(t, encoded, err, `{"current_password":"old password","new_password":"new password"}`)
}

func TestAccountProfileRequestWireShape(t *testing.T) {
	encoded, err := json.Marshal(accountProfileRequest{Email: "new@example.com"})
	assertWireShape(t, encoded, err, `{"email":"new@example.com"}`)
}

func TestPrivacyRequestWireShape(t *testing.T) {
	encoded, err := json.Marshal(privacyRequest{Kind: "data_export"})
	assertWireShape(t, encoded, err, `{"kind":"data_export"}`)
}

func TestPrivacyRequestResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(privacyRequestResponse{ID: "privacy-1", Kind: "data_export", Status: "queued", RequestedBy: "user-1", ExportJSON: "", ResolutionNote: ""})
	assertWireShape(t, encoded, err, `{"id":"privacy-1","kind":"data_export","status":"queued","requested_by":"user-1","export_json":"","resolution_note":""}`)
}

func TestPrivacyRequestsResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(privacyRequestsResponse{Requests: []privacyRequestResponse{{ID: "privacy-1", Kind: "sensitive_field_deletion", Status: "resolved", RequestedBy: "user-1", ExportJSON: "", ResolutionNote: "done"}}})
	assertWireShape(t, encoded, err, `{"requests":[{"id":"privacy-1","kind":"sensitive_field_deletion","status":"resolved","requested_by":"user-1","export_json":"","resolution_note":"done"}]}`)
}

func TestUsersResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(usersResponse{Users: []userDirectoryEntryResponse{{ID: "user-1", Email: "person@example.com", Status: "active"}}})
	assertWireShape(t, encoded, err, `{"users":[{"id":"user-1","email":"person@example.com","status":"active"}]}`)
}

func TestHealthResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(healthResponse{Status: "ok"})
	assertWireShape(t, encoded, err, `{"status":"ok"}`)
}

func TestErrorResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(errorResponse{Error: "request body is invalid"})
	assertWireShape(t, encoded, err, `{"error":"request body is invalid"}`)
}

func TestEmptyResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(emptyResponse{Status: "password_changed"})
	assertWireShape(t, encoded, err, `{"status":"password_changed"}`)
}

func TestBalanceResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(balanceResponse{Amount: 100})
	assertWireShape(t, encoded, err, `{"amount":100}`)
}

func TestLedgerEntryResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(ledgerEntryResponse{ID: "entry-1", Kind: "signup_grant", Amount: 100, TaskID: ""})
	assertWireShape(t, encoded, err, `{"id":"entry-1","kind":"signup_grant","amount":100,"task_id":""}`)
}

func TestLedgerListResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(ledgerListResponse{Entries: []ledgerEntryResponse{{ID: "entry-1", Kind: "task_payout", Amount: 25, TaskID: "task-1"}}})
	assertWireShape(t, encoded, err, `{"entries":[{"id":"entry-1","kind":"task_payout","amount":25,"task_id":"task-1"}]}`)
}

func TestTaskEscrowResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(taskEscrowResponse{TaskID: "task-1", Amount: 40, State: "held"})
	assertWireShape(t, encoded, err, `{"task_id":"task-1","amount":40,"state":"held"}`)
}

func TestAcceptSubmissionResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(acceptSubmissionResponse{TaskID: "task-1", SubmissionID: "submission-1", PayoutKind: "bundle", PayoutAmount: 25, WorkerUserID: "user-1", CollectibleIDs: []string{"collectible-1", "collectible-2"}, TipAmount: 5})
	assertWireShape(t, encoded, err, `{"task_id":"task-1","submission_id":"submission-1","payout_kind":"bundle","payout_amount":25,"worker_user_id":"user-1","collectible_ids":["collectible-1","collectible-2"],"tip_amount":5}`)
}

func TestReviewSubmissionResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(reviewSubmissionResponse{TaskID: "task-1", SubmissionID: "submission-1", State: "rejected", ReviewNote: "Not current.", PayoutKind: "credit", PayoutAmount: 10, WorkerUserID: "user-1", TipAmount: 2})
	assertWireShape(t, encoded, err, `{"task_id":"task-1","submission_id":"submission-1","state":"rejected","review_note":"Not current.","payout_kind":"credit","payout_amount":10,"worker_user_id":"user-1","tip_amount":2}`)
}

func TestReservationResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(reservationResponse{ID: "reservation-1", TaskID: "task-1", AssigneeKind: "user", AssigneeID: "user-1", State: "active", RequestedBy: "user-1"})
	assertWireShape(t, encoded, err, `{"id":"reservation-1","task_id":"task-1","assignee_kind":"user","assignee_id":"user-1","state":"active","requested_by":"user-1"}`)
}

func TestReservationsResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(reservationsResponse{Reservations: []reservationResponse{{ID: "reservation-1", TaskID: "task-1", AssigneeKind: "user", AssigneeID: "user-1", State: "requested", RequestedBy: "user-1"}}})
	assertWireShape(t, encoded, err, `{"reservations":[{"id":"reservation-1","task_id":"task-1","assignee_kind":"user","assignee_id":"user-1","state":"requested","requested_by":"user-1"}]}`)
}

func TestTeamResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(teamResponse{ID: "team-1", OwnerKind: "user", OrganizationID: "", OwnerUserID: "user-1", Name: "Survey crew", CreatedBy: "user-1"})
	assertWireShape(t, encoded, err, `{"id":"team-1","owner_kind":"user","organization_id":"","owner_user_id":"user-1","name":"Survey crew","created_by":"user-1"}`)
}

func TestOrganizationResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(organizationResponse{ID: "org-1", Name: "Lattice Field Co", CreatedBy: "user-1"})
	assertWireShape(t, encoded, err, `{"id":"org-1","name":"Lattice Field Co","created_by":"user-1"}`)
}

func TestOrganizationsResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(organizationsResponse{Organizations: []organizationResponse{{ID: "org-1", Name: "Lattice Field Co", CreatedBy: "user-1"}}})
	assertWireShape(t, encoded, err, `{"organizations":[{"id":"org-1","name":"Lattice Field Co","created_by":"user-1"}]}`)
}

func TestOrganizationRequestWireShape(t *testing.T) {
	encoded, err := json.Marshal(organizationRequest{Name: "Lattice Field Co"})
	assertWireShape(t, encoded, err, `{"name":"Lattice Field Co"}`)
}

func TestProvisionMemberRequestWireShape(t *testing.T) {
	encoded, err := json.Marshal(provisionMemberRequest{Email: "member@example.com", Roles: []string{"member", "reviewer"}})
	assertWireShape(t, encoded, err, `{"email":"member@example.com","roles":["member","reviewer"]}`)
}

func TestUpdateMemberRolesRequestWireShape(t *testing.T) {
	encoded, err := json.Marshal(updateMemberRolesRequest{Roles: []string{"owner"}})
	assertWireShape(t, encoded, err, `{"roles":["owner"]}`)
}

func TestOrganizationMemberResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(organizationMemberResponse{ID: "member-1", OrganizationID: "org-1", UserID: "user-1", Status: "active", Roles: []string{"owner"}})
	assertWireShape(t, encoded, err, `{"id":"member-1","organization_id":"org-1","user_id":"user-1","status":"active","roles":["owner"]}`)
}

func TestOrganizationMembersResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(organizationMembersResponse{Members: []organizationMemberResponse{{ID: "member-1", OrganizationID: "org-1", UserID: "user-1", Status: "active", Roles: []string{"owner", "reviewer"}}}})
	assertWireShape(t, encoded, err, `{"members":[{"id":"member-1","organization_id":"org-1","user_id":"user-1","status":"active","roles":["owner","reviewer"]}]}`)
}

func TestTeamRequestWireShape(t *testing.T) {
	encoded, err := json.Marshal(teamRequest{Name: "Survey crew"})
	assertWireShape(t, encoded, err, `{"name":"Survey crew"}`)
}

func TestTeamMemberRequestWireShape(t *testing.T) {
	encoded, err := json.Marshal(teamMemberRequest{Email: "member@example.com"})
	assertWireShape(t, encoded, err, `{"email":"member@example.com"}`)
}

func TestTeamDetailResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(teamDetailResponse{Team: teamResponse{ID: "team-1", OwnerKind: "organization", OrganizationID: "org-1", OwnerUserID: "", Name: "Survey crew", CreatedBy: "user-1"}, Members: []string{"user-1", "user-2"}})
	assertWireShape(t, encoded, err, `{"team":{"id":"team-1","owner_kind":"organization","organization_id":"org-1","owner_user_id":"","name":"Survey crew","created_by":"user-1"},"members":["user-1","user-2"]}`)
}

func TestTeamsResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(teamsResponse{Teams: []teamResponse{{ID: "team-1", OwnerKind: "user", OrganizationID: "", OwnerUserID: "user-1", Name: "Field hands", CreatedBy: "user-1"}}})
	assertWireShape(t, encoded, err, `{"teams":[{"id":"team-1","owner_kind":"user","organization_id":"","owner_user_id":"user-1","name":"Field hands","created_by":"user-1"}]}`)
}

func TestTaskCapabilityTokenResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(taskCapabilityTokenResponse{ID: "cap-1", TaskID: "task-1", State: "active", Token: "secret-token"})
	assertWireShape(t, encoded, err, `{"id":"cap-1","task_id":"task-1","state":"active","token":"secret-token"}`)
}

func TestTaskResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(taskResponse{ID: "task-1", OwnerKind: "user", OwnerID: "user-1", Title: "Label receipts", Description: "Extract totals.", TaskType: "general", ReferenceURL: "https://example.com/pr/1", RewardKind: "credit", RewardCreditAmount: 25, RewardCollectibleCount: 1, ParticipationPolicy: "reservation_required", AssigneeScope: "user", ReservationExpiryHours: 48, State: "open", VisibilityKind: "public", VisibilityID: "", SeriesKind: "standalone", SeriesID: "", SeriesPosition: 0, ResponseSchemaJSON: `{"kind":"freeform"}`, PayloadKind: "json", PayloadJSON: `{"batch":"A"}`, CreatedBy: "user-1", AvailabilityKind: "available", ViewerAction: "reserve", ReviewerAction: "none"})
	assertWireShape(t, encoded, err, `{"id":"task-1","owner_kind":"user","owner_id":"user-1","title":"Label receipts","description":"Extract totals.","task_type":"general","reference_url":"https://example.com/pr/1","reward_kind":"credit","reward_credit_amount":25,"reward_collectible_count":1,"participation_policy":"reservation_required","assignee_scope":"user","reservation_expiry_hours":48,"state":"open","visibility_kind":"public","visibility_id":"","series_kind":"standalone","series_id":"","series_position":0,"response_schema_json":"{\"kind\":\"freeform\"}","payload_kind":"json","payload_json":"{\"batch\":\"A\"}","created_by":"user-1","availability_kind":"available","viewer_action":"reserve","reviewer_action":"none"}`)
}

func TestTasksResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(tasksResponse{Tasks: []taskListItemResponse{{ID: "task-1", OwnerKind: "user", Title: "Label receipts", RewardKind: "none", RewardCreditAmount: 0, RewardCollectibleCount: 0, ParticipationPolicy: "open", AssigneeScope: "user", ReservationExpiryHours: 48, State: "open", VisibilityKind: "public", AvailabilityKind: "available", ViewerAction: "submit", ReviewerAction: "none", CreatedBy: "user-1", ActiveAssigneeKind: "", ActiveAssigneeID: ""}}})
	assertWireShape(t, encoded, err, `{"tasks":[{"id":"task-1","owner_kind":"user","title":"Label receipts","reward_kind":"none","reward_credit_amount":0,"reward_collectible_count":0,"participation_policy":"open","assignee_scope":"user","reservation_expiry_hours":48,"state":"open","visibility_kind":"public","availability_kind":"available","viewer_action":"submit","reviewer_action":"none","created_by":"user-1","active_assignee_kind":"","active_assignee_id":""}]}`)
}

func TestTaskRequestWireShape(t *testing.T) {
	encoded, err := json.Marshal(taskRequest{
		Owner:        taskOwnerRequest{Kind: "organization", OrganizationID: "org-1"},
		Title:        "Label receipts",
		Description:  "Extract the receipt totals.",
		TaskType:     "qa_testing",
		ReferenceURL: "https://example.com/pr/1",
		Reward: taskRewardRequest{
			Kind:           "bundle",
			CreditAmount:   25,
			CollectibleIDs: []string{"collectible-1"},
		},
		Participation:      taskParticipationRequest{Policy: "approval_required", AssigneeScope: "organization_team", ReservationExpiryHours: 72},
		Visibility:         taskVisibilityRequest{Kind: "organization", OrganizationID: "org-1"},
		Placement:          taskPlacementRequest{Kind: "existing_series", SeriesID: "series-1", SeriesPosition: 2},
		ResponseSchemaJSON: `{"kind":"freeform"}`,
		Payload:            taskPayloadRequest{Kind: "json", JSON: `{"batch":"A"}`},
	})
	assertWireShape(t, encoded, err, `{"owner":{"kind":"organization","user_id":"","team_id":"","organization_id":"org-1"},"title":"Label receipts","description":"Extract the receipt totals.","task_type":"qa_testing","reference_url":"https://example.com/pr/1","reward":{"kind":"bundle","credit_amount":25,"collectible_ids":["collectible-1"]},"participation":{"policy":"approval_required","assignee_scope":"organization_team","reservation_expiry_hours":72},"visibility":{"kind":"organization","user_id":"","team_id":"","organization_id":"org-1"},"placement":{"kind":"existing_series","series_id":"series-1","series_title":"","series_position":2},"response_schema_json":"{\"kind\":\"freeform\"}","payload":{"kind":"json","json":"{\"batch\":\"A\"}"}}`)
}

func TestFundingRequestWireShape(t *testing.T) {
	encoded, err := json.Marshal(fundingRequest{Amount: 50, IdempotencyKey: "funding-key-1", OrganizationID: "org-1"})
	assertWireShape(t, encoded, err, `{"amount":50,"idempotency_key":"funding-key-1","organization_id":"org-1"}`)
}

func TestIdempotentRequestWireShape(t *testing.T) {
	encoded, err := json.Marshal(idempotentRequest{IdempotencyKey: "key-1"})
	assertWireShape(t, encoded, err, `{"idempotency_key":"key-1"}`)
}

func TestReservationRequestWireShape(t *testing.T) {
	encoded, err := json.Marshal(reservationRequest{AssigneeKind: "organization_team", OrganizationID: "org-1", TeamID: "team-1"})
	assertWireShape(t, encoded, err, `{"assignee_kind":"organization_team","organization_id":"org-1","team_id":"team-1"}`)
}

func TestSubmissionRequestWireShape(t *testing.T) {
	encoded, err := json.Marshal(submissionRequest{ResponseJSON: `{"answer":42}`})
	assertWireShape(t, encoded, err, `{"response_json":"{\"answer\":42}"}`)
}

func TestAcceptSubmissionRequestWireShape(t *testing.T) {
	encoded, err := json.Marshal(acceptSubmissionRequest{IdempotencyKey: "accept-key-1", PayoutAmount: 20, TipAmount: 5, TipCollectibleID: "collectible-1"})
	assertWireShape(t, encoded, err, `{"idempotency_key":"accept-key-1","payout_amount":20,"tip_amount":5,"tip_collectible_id":"collectible-1"}`)
}

func TestRequestChangesRequestWireShape(t *testing.T) {
	encoded, err := json.Marshal(requestChangesRequest{ReviewNote: "Please include totals."})
	assertWireShape(t, encoded, err, `{"review_note":"Please include totals."}`)
}

func TestRejectSubmissionRequestWireShape(t *testing.T) {
	encoded, err := json.Marshal(rejectSubmissionRequest{IdempotencyKey: "reject-key-1", ReviewNote: "Invalid response.", PartialCreditAmount: 3, TipAmount: 1, BanImplementor: true})
	assertWireShape(t, encoded, err, `{"idempotency_key":"reject-key-1","review_note":"Invalid response.","partial_credit_amount":3,"tip_amount":1,"ban_implementor":true}`)
}

func TestSubmissionCreatedResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(submissionCreatedResponse{Submission: submissionResponse{ID: "submission-1", TaskID: "task-1", SubmitterID: "user-1", State: "submitted", ResponseJSON: "{}", ReviewNote: "", ValidationErrors: []submissionValidationErrorResponse{}, SensitiveFields: []submissionSensitiveFieldResponse{}}, ReceiptToken: "receipt-1"})
	assertWireShape(t, encoded, err, `{"submission":{"id":"submission-1","task_id":"task-1","submitter_id":"user-1","state":"submitted","response_json":"{}","review_note":"","validation_errors":[],"sensitive_fields":[]},"receipt_token":"receipt-1"}`)
}

func TestSubmissionValidationErrorResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(submissionValidationErrorResponse{Path: "email", Message: "is required"})
	assertWireShape(t, encoded, err, `{"path":"email","message":"is required"}`)
}

func TestSubmissionResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(submissionResponse{ID: "submission-1", TaskID: "task-1", SubmitterID: "user-1", State: "changes_requested", ResponseJSON: "{}", ReviewNote: "Use the current API.", ValidationErrors: []submissionValidationErrorResponse{}, SensitiveFields: []submissionSensitiveFieldResponse{{Path: "email", Category: "pii", Retention: "delete_on_request", Redaction: "replace"}}})
	assertWireShape(t, encoded, err, `{"id":"submission-1","task_id":"task-1","submitter_id":"user-1","state":"changes_requested","response_json":"{}","review_note":"Use the current API.","validation_errors":[],"sensitive_fields":[{"path":"email","category":"pii","retention":"delete_on_request","redaction":"replace"}]}`)
}

func TestSubmissionsResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(submissionsResponse{Submissions: []submissionResponse{{ID: "submission-1", TaskID: "task-1", SubmitterID: "user-1", State: "submitted", ResponseJSON: "{}", ReviewNote: "", ValidationErrors: []submissionValidationErrorResponse{}, SensitiveFields: []submissionSensitiveFieldResponse{}}}})
	assertWireShape(t, encoded, err, `{"submissions":[{"id":"submission-1","task_id":"task-1","submitter_id":"user-1","state":"submitted","response_json":"{}","review_note":"","validation_errors":[],"sensitive_fields":[]}]}`)
}

func TestSubmissionCommentsResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(submissionCommentsResponse{Comments: []submissionCommentResponse{{ID: "comment-1", SubmissionID: "submission-1", AuthorUserID: "user-1", Body: "Please revise.", CreatedAt: "2026-06-29T00:00:00Z"}}})
	assertWireShape(t, encoded, err, `{"comments":[{"id":"comment-1","submission_id":"submission-1","author_user_id":"user-1","body":"Please revise.","created_at":"2026-06-29T00:00:00Z"}]}`)
}

func TestSubmissionCommentRequestWireShape(t *testing.T) {
	encoded, err := json.Marshal(submissionCommentRequest{Body: "Please revise."})
	assertWireShape(t, encoded, err, `{"body":"Please revise."}`)
}

func TestTaskCommentsResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(taskCommentsResponse{Comments: []taskCommentResponse{{ID: "comment-1", TaskID: "task-1", AuthorUserID: "user-1", Body: "Looks ready.", CreatedAt: "2026-06-29T00:00:00Z"}}})
	assertWireShape(t, encoded, err, `{"comments":[{"id":"comment-1","task_id":"task-1","author_user_id":"user-1","body":"Looks ready.","created_at":"2026-06-29T00:00:00Z"}]}`)
}

func TestCreateSeriesRequestWireShape(t *testing.T) {
	encoded, err := json.Marshal(createSeriesRequest{Title: "Release checks", Description: "Grouped QA work."})
	assertWireShape(t, encoded, err, `{"title":"Release checks","description":"Grouped QA work."}`)
}

func TestTaskSeriesListResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(taskSeriesListResponse{Series: []taskSeriesResponse{{ID: "series-1", OwnerKind: "user", Title: "Release checks", Description: "Grouped QA work.", State: "published", CreatedBy: "user-1"}}})
	assertWireShape(t, encoded, err, `{"series":[{"id":"series-1","owner_kind":"user","title":"Release checks","description":"Grouped QA work.","state":"published","created_by":"user-1"}]}`)
}

func TestAddTaskToSeriesRequestWireShape(t *testing.T) {
	encoded, err := json.Marshal(addTaskToSeriesRequest{TaskID: "task-1"})
	assertWireShape(t, encoded, err, `{"task_id":"task-1"}`)
}

func TestReorderSeriesRequestWireShape(t *testing.T) {
	encoded, err := json.Marshal(reorderSeriesRequest{TaskIDs: []string{"task-2", "task-1"}})
	assertWireShape(t, encoded, err, `{"task_ids":["task-2","task-1"]}`)
}

func TestSeriesCommentRequestWireShape(t *testing.T) {
	encoded, err := json.Marshal(seriesCommentRequest{Body: "Ship after review."})
	assertWireShape(t, encoded, err, `{"body":"Ship after review."}`)
}

func TestTaskSeriesDetailResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(taskSeriesDetailResponse{
		Series:   taskSeriesResponse{ID: "series-1", OwnerKind: "user", Title: "Release checks", Description: "Grouped QA work.", State: "published", CreatedBy: "user-1"},
		Tasks:    []taskResponse{{ID: "task-1", OwnerKind: "user", OwnerID: "user-1", Title: "Label receipts", Description: "Extract totals.", TaskType: "general", ReferenceURL: "", RewardKind: "none", RewardCreditAmount: 0, RewardCollectibleCount: 0, ParticipationPolicy: "open", AssigneeScope: "user", ReservationExpiryHours: 48, State: "open", VisibilityKind: "public", VisibilityID: "", SeriesKind: "existing_series", SeriesID: "series-1", SeriesPosition: 1, ResponseSchemaJSON: `{"kind":"freeform"}`, PayloadKind: "none", PayloadJSON: "", CreatedBy: "user-1", AvailabilityKind: "available", ViewerAction: "submit", ReviewerAction: "none"}},
		Comments: []seriesCommentResponse{{ID: "comment-1", SeriesID: "series-1", AuthorUserID: "user-1", Body: "Ready.", CreatedAt: "2026-06-29T00:00:00Z"}},
	})
	assertWireShape(t, encoded, err, `{"series":{"id":"series-1","owner_kind":"user","title":"Release checks","description":"Grouped QA work.","state":"published","created_by":"user-1"},"tasks":[{"id":"task-1","owner_kind":"user","owner_id":"user-1","title":"Label receipts","description":"Extract totals.","task_type":"general","reference_url":"","reward_kind":"none","reward_credit_amount":0,"reward_collectible_count":0,"participation_policy":"open","assignee_scope":"user","reservation_expiry_hours":48,"state":"open","visibility_kind":"public","visibility_id":"","series_kind":"existing_series","series_id":"series-1","series_position":1,"response_schema_json":"{\"kind\":\"freeform\"}","payload_kind":"none","payload_json":"","created_by":"user-1","availability_kind":"available","viewer_action":"submit","reviewer_action":"none"}],"comments":[{"id":"comment-1","series_id":"series-1","author_user_id":"user-1","body":"Ready.","created_at":"2026-06-29T00:00:00Z"}]}`)
}

func TestMintCollectibleRequestWireShape(t *testing.T) {
	encoded, err := json.Marshal(mintCollectibleRequest{Name: "Harvest Star", Kind: "badge", TransferPolicy: "transferable_between_users", Art: "harvest-star"})
	assertWireShape(t, encoded, err, `{"name":"Harvest Star","kind":"badge","transfer_policy":"transferable_between_users","art":"harvest-star"}`)
}

func TestCollectibleResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(collectibleResponse{ID: "collectible-1", Name: "Harvest Star", Kind: "badge", State: "minted", TransferPolicy: "transferable_between_users", OwnerID: "user-1", OwnerKind: "user", OrganizationID: "org-1", Art: "harvest-star"})
	assertWireShape(t, encoded, err, `{"id":"collectible-1","name":"Harvest Star","kind":"badge","state":"minted","transfer_policy":"transferable_between_users","owner_id":"user-1","owner_kind":"user","organization_id":"org-1","art":"harvest-star"}`)
}

func TestCollectiblesResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(collectiblesResponse{Collectibles: []collectibleResponse{{ID: "collectible-1", Name: "Harvest Star", Kind: "badge", State: "minted", TransferPolicy: "transferable_between_users", OwnerID: "user-1", OwnerKind: "user", OrganizationID: "", Art: "harvest-star"}}})
	assertWireShape(t, encoded, err, `{"collectibles":[{"id":"collectible-1","name":"Harvest Star","kind":"badge","state":"minted","transfer_policy":"transferable_between_users","owner_id":"user-1","owner_kind":"user","organization_id":"","art":"harvest-star"}]}`)
}

func TestCollectibleRewardRequestWireShape(t *testing.T) {
	encoded, err := json.Marshal(collectibleRewardRequest{CollectibleID: "collectible-1"})
	assertWireShape(t, encoded, err, `{"collectible_id":"collectible-1"}`)
}

func TestAwardCollectibleRequestWireShape(t *testing.T) {
	encoded, err := json.Marshal(awardCollectibleRequest{Slug: "harvest-star", RecipientKind: "user", RecipientID: "user-1", OrganizationID: "org-1"})
	assertWireShape(t, encoded, err, `{"slug":"harvest-star","recipient_kind":"user","recipient_id":"user-1","organization_id":"org-1"}`)
}

func TestTransferCollectibleRequestWireShape(t *testing.T) {
	encoded, err := json.Marshal(transferCollectibleRequest{RecipientID: "user-2"})
	assertWireShape(t, encoded, err, `{"recipient_id":"user-2"}`)
}

func TestCollectibleCatalogResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(collectibleCatalogResponse{Entries: []catalogEntryResponse{{Slug: "harvest-star", Name: "Harvest Star", Kind: "badge", TransferPolicy: "transferable_between_users", Art: "harvest-star"}}})
	assertWireShape(t, encoded, err, `{"entries":[{"slug":"harvest-star","name":"Harvest Star","kind":"badge","transfer_policy":"transferable_between_users","art":"harvest-star"}]}`)
}

func TestAgentCredentialResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(agentCredentialResponse{ID: "cred-1", Label: "Local agent", Scopes: []string{"tasks_read", "submissions_write"}, State: "active"})
	assertWireShape(t, encoded, err, `{"id":"cred-1","label":"Local agent","scopes":["tasks_read","submissions_write"],"state":"active"}`)
}

func TestAgentCredentialRequestWireShape(t *testing.T) {
	encoded, err := json.Marshal(agentCredentialRequest{Label: "Local agent", Scopes: []string{"tasks_read", "submissions_write"}})
	assertWireShape(t, encoded, err, `{"label":"Local agent","scopes":["tasks_read","submissions_write"]}`)
}

func TestAgentCredentialCreatedResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(agentCredentialCreatedResponse{Credential: agentCredentialResponse{ID: "cred-1", Label: "Local agent", Scopes: []string{"tasks_read"}, State: "active"}, Secret: "agent-secret"})
	assertWireShape(t, encoded, err, `{"credential":{"id":"cred-1","label":"Local agent","scopes":["tasks_read"],"state":"active"},"secret":"agent-secret"}`)
}

func TestAgentCredentialsResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(agentCredentialsResponse{Credentials: []agentCredentialResponse{{ID: "cred-1", Label: "Local agent", Scopes: []string{"tasks_read"}, State: "active"}}})
	assertWireShape(t, encoded, err, `{"credentials":[{"id":"cred-1","label":"Local agent","scopes":["tasks_read"],"state":"active"}]}`)
}

func TestUserProfileResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(userProfileResponse{ID: "user-1", Tasks: []taskListItemResponse{{ID: "task-1", OwnerKind: "user", Title: "Label receipts", RewardKind: "none", RewardCreditAmount: 0, RewardCollectibleCount: 0, ParticipationPolicy: "open", AssigneeScope: "user", ReservationExpiryHours: 48, State: "open", VisibilityKind: "public", AvailabilityKind: "available", ViewerAction: "submit", ReviewerAction: "none", CreatedBy: "user-1", ActiveAssigneeKind: "", ActiveAssigneeID: ""}}})
	assertWireShape(t, encoded, err, `{"id":"user-1","tasks":[{"id":"task-1","owner_kind":"user","title":"Label receipts","reward_kind":"none","reward_credit_amount":0,"reward_collectible_count":0,"participation_policy":"open","assignee_scope":"user","reservation_expiry_hours":48,"state":"open","visibility_kind":"public","availability_kind":"available","viewer_action":"submit","reviewer_action":"none","created_by":"user-1","active_assignee_kind":"","active_assignee_id":""}]}`)
}

func TestOperationsResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(operationsResponse{Status: "ok", AccountTokenDelivery: "log", MCPStorage: "process_memory", RateLimitStorage: "process_memory", ActiveMCPSessions: 2, ActiveIPRateBuckets: 3, ActiveSubjectRateBuckets: 4, SecureCookies: "enabled"})
	assertWireShape(t, encoded, err, `{"status":"ok","account_token_delivery":"log","mcp_storage":"process_memory","rate_limit_storage":"process_memory","active_mcp_sessions":2,"active_ip_rate_buckets":3,"active_subject_rate_buckets":4,"secure_cookies":"enabled"}`)
}

func TestAuditEventsResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(auditEventsResponse{Events: []auditEventResponse{{
		ID:           "event-1",
		ActorUserID:  "user-1",
		Action:       "submission_accepted",
		SubjectKind:  "submission",
		SubjectID:    "submission-1",
		MetadataJSON: "{}",
		CreatedAt:    "2026-06-29T00:00:00Z",
	}}})
	assertWireShape(t, encoded, err, `{"events":[{"id":"event-1","actor_user_id":"user-1","action":"submission_accepted","subject_kind":"submission","subject_id":"submission-1","metadata_json":"{}","created_at":"2026-06-29T00:00:00Z"}]}`)
}

func TestNotificationResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(notificationResponse{
		ID:              "notification-1",
		RecipientUserID: "user-2",
		ActorUserID:     "user-1",
		Kind:            "submission_created",
		SubjectKind:     "submission",
		SubjectID:       "submission-1",
		State:           "unread",
		MetadataJSON:    `{"task_id":"task-1"}`,
		CreatedAt:       "2026-06-29T00:00:00Z",
	})
	assertWireShape(t, encoded, err, `{"id":"notification-1","recipient_user_id":"user-2","actor_user_id":"user-1","kind":"submission_created","subject_kind":"submission","subject_id":"submission-1","state":"unread","metadata_json":"{\"task_id\":\"task-1\"}","created_at":"2026-06-29T00:00:00Z"}`)
}

func TestNotificationsResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(notificationsResponse{Notifications: []notificationResponse{{ID: "notification-1", RecipientUserID: "user-2", ActorUserID: "user-1", Kind: "submission_commented", SubjectKind: "submission", SubjectID: "submission-1", State: "unread", MetadataJSON: `{"task_id":"task-1"}`, CreatedAt: "2026-06-29T00:00:00Z"}}})
	assertWireShape(t, encoded, err, `{"notifications":[{"id":"notification-1","recipient_user_id":"user-2","actor_user_id":"user-1","kind":"submission_commented","subject_kind":"submission","subject_id":"submission-1","state":"unread","metadata_json":"{\"task_id\":\"task-1\"}","created_at":"2026-06-29T00:00:00Z"}]}`)
}

func TestSavedQueueViewResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(savedQueueViewResponse{ID: "saved-view-1", Scope: "team_work", Name: "Ready work", Query: "review", StateFilter: "ready", TypeFilter: "code_review", Sort: "title_asc"})
	assertWireShape(t, encoded, err, `{"id":"saved-view-1","scope":"team_work","name":"Ready work","query":"review","state_filter":"ready","type_filter":"code_review","sort":"title_asc"}`)
}

func TestSavedQueueViewsResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(savedQueueViewsResponse{Views: []savedQueueViewResponse{{ID: "saved-view-1", Scope: "organization_tasks", Name: "Open org", Query: "field", StateFilter: "open", TypeFilter: "", Sort: "newest"}}})
	assertWireShape(t, encoded, err, `{"views":[{"id":"saved-view-1","scope":"organization_tasks","name":"Open org","query":"field","state_filter":"open","type_filter":"","sort":"newest"}]}`)
}

func TestTaskListItemResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(taskListItemResponse{
		ID:                     "task-1",
		OwnerKind:              "user",
		Title:                  "Label receipts",
		RewardKind:             "credit",
		RewardCreditAmount:     25,
		RewardCollectibleCount: 0,
		ParticipationPolicy:    "reservation_required",
		AssigneeScope:          "user",
		ReservationExpiryHours: 48,
		State:                  "open",
		VisibilityKind:         "public",
		AvailabilityKind:       "reserved",
		ViewerAction:           "wait",
		ReviewerAction:         "none",
		CreatedBy:              "user-1",
		ActiveAssigneeKind:     "user",
		ActiveAssigneeID:       "user-2",
	})
	assertWireShape(t, encoded, err, `{"id":"task-1","owner_kind":"user","title":"Label receipts","reward_kind":"credit","reward_credit_amount":25,"reward_collectible_count":0,"participation_policy":"reservation_required","assignee_scope":"user","reservation_expiry_hours":48,"state":"open","visibility_kind":"public","availability_kind":"reserved","viewer_action":"wait","reviewer_action":"none","created_by":"user-1","active_assignee_kind":"user","active_assignee_id":"user-2"}`)
}

func TestOrganizationTeamTaskListItemResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(taskListItemResponse{
		ID:                     "task-2",
		OwnerKind:              "organization",
		Title:                  "Review org queue",
		RewardKind:             "none",
		RewardCreditAmount:     0,
		RewardCollectibleCount: 0,
		ParticipationPolicy:    "open",
		AssigneeScope:          "organization_team",
		ReservationExpiryHours: 48,
		State:                  "open",
		VisibilityKind:         "organization_team",
		AvailabilityKind:       "available",
		ViewerAction:           "submit",
		ReviewerAction:         "none",
		CreatedBy:              "user-1",
		ActiveAssigneeKind:     "",
		ActiveAssigneeID:       "",
	})
	assertWireShape(t, encoded, err, `{"id":"task-2","owner_kind":"organization","title":"Review org queue","reward_kind":"none","reward_credit_amount":0,"reward_collectible_count":0,"participation_policy":"open","assignee_scope":"organization_team","reservation_expiry_hours":48,"state":"open","visibility_kind":"organization_team","availability_kind":"available","viewer_action":"submit","reviewer_action":"none","created_by":"user-1","active_assignee_kind":"","active_assignee_id":""}`)
}

func assertWireShape(t *testing.T, got []byte, err error, want string) {
	t.Helper()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(got) != want {
		t.Fatalf("wire shape =\n  %s\nwant\n  %s", string(got), want)
	}
}
