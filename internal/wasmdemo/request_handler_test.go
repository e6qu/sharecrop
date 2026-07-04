package wasmdemo

import (
	"encoding/json"
	"testing"
	"time"
)

type fixedHandlerClock struct {
	value time.Time
}

func (clock fixedHandlerClock) Now() time.Time {
	return clock.value
}

type fixedHandlerActor struct {
	userID string
}

func (actor fixedHandlerActor) UserID() string {
	return actor.userID
}

type fixedPrivacyRequestIDs struct {
	value string
}

func (ids fixedPrivacyRequestIDs) NextPrivacyRequestID() string {
	return ids.value
}

type fixedSavedQueueViewIDs struct {
	value string
}

func (ids fixedSavedQueueViewIDs) NextSavedQueueViewID() string {
	return ids.value
}

type fixedTaskIDs struct {
	value string
}

func (ids fixedTaskIDs) NextTaskID() string {
	return ids.value
}

type fixedOrganizationIDs struct {
	organizationID string
	memberIDs      []string
	teamID         string
}

func (ids *fixedOrganizationIDs) NextOrganizationID() string {
	return ids.organizationID
}

func (ids *fixedOrganizationIDs) NextOrganizationMemberID() string {
	if len(ids.memberIDs) == 0 {
		return ""
	}
	value := ids.memberIDs[0]
	ids.memberIDs = ids.memberIDs[1:]
	return value
}

func (ids *fixedOrganizationIDs) NextTeamID() string {
	return ids.teamID
}

type fixedOrganizationUserResolver struct {
	usersByEmail map[string]string
}

func (resolver fixedOrganizationUserResolver) UserIDForEmail(email string) (string, bool) {
	value, ok := resolver.usersByEmail[email]
	return value, ok
}

type fixedInteractionIDs struct {
	submissionID   string
	commentID      string
	reservationID  string
	ledgerID       string
	notificationID string
}

func (ids fixedInteractionIDs) NextSubmissionID() string {
	return ids.submissionID
}

func (ids fixedInteractionIDs) NextCommentID() string {
	return ids.commentID
}

func (ids fixedInteractionIDs) NextReservationID() string {
	return ids.reservationID
}

func (ids fixedInteractionIDs) NextLedgerEntryID() string {
	return ids.ledgerID
}

func (ids fixedInteractionIDs) NextNotificationID() string {
	if ids.notificationID != "" {
		return ids.notificationID
	}
	return "notification-1"
}

func TestPrivacyRequestHandlerCreatesAndListsThroughBrowserStorage(t *testing.T) {
	storage := newTestBrowserStorage()
	handler := NewPrivacyRequestHandler(
		storage,
		fixedHandlerClock{value: time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)},
		fixedHandlerActor{userID: "user-admin"},
		fixedPrivacyRequestIDs{value: "privacy-1"},
	)

	createResult := handler.Handle(Request{
		Method: MethodPost,
		Path:   "/api/privacy-requests",
		Body:   `{"kind":"data_export"}`,
	})
	created, createdMatched := createResult.(RequestHandled)
	if !createdMatched {
		t.Fatalf("create result = %T, want RequestHandled", createResult)
	}
	if created.Value.Status != 201 {
		t.Fatalf("create status = %d, want 201", created.Value.Status)
	}
	var createdBody StoredPrivacyRequest
	if err := json.Unmarshal([]byte(created.Value.Body), &createdBody); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if createdBody.ID != "privacy-1" || createdBody.Kind != "data_export" || createdBody.Status != "queued" || createdBody.RequestedBy != "user-admin" {
		t.Fatalf("created body = %#v", createdBody)
	}

	listResult := handler.Handle(Request{Method: MethodGet, Path: "/api/admin/privacy-requests", Body: ""})
	listed, listedMatched := listResult.(RequestHandled)
	if !listedMatched {
		t.Fatalf("list result = %T, want RequestHandled", listResult)
	}
	if listed.Value.Status != 200 {
		t.Fatalf("list status = %d, want 200", listed.Value.Status)
	}
	var listBody privacyRequestsBody
	if err := json.Unmarshal([]byte(listed.Value.Body), &listBody); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listBody.Requests) != 1 || listBody.Requests[0].ID != "privacy-1" {
		t.Fatalf("list body = %#v", listBody)
	}
}

func TestPrivacyRequestHandlerRejectsMissingIDSource(t *testing.T) {
	handler := NewPrivacyRequestHandler(
		newTestBrowserStorage(),
		fixedHandlerClock{value: time.Now().UTC()},
		fixedHandlerActor{userID: "user-admin"},
		nil,
	)
	result := handler.Handle(Request{Method: MethodPost, Path: "/api/privacy-requests", Body: `{"kind":"data_export"}`})
	rejected, matched := result.(RequestHandleRejected)
	if !matched {
		t.Fatalf("result = %T, want RequestHandleRejected", result)
	}
	if rejected.Reason != "privacy request id source is required" {
		t.Fatalf("reason = %q", rejected.Reason)
	}
}

func TestPrivacyRequestHandlerListsWithoutIDSource(t *testing.T) {
	handler := NewPrivacyRequestHandler(
		newTestBrowserStorage(),
		fixedHandlerClock{value: time.Now().UTC()},
		fixedHandlerActor{userID: "user-admin"},
		nil,
	)
	result := handler.Handle(Request{Method: MethodGet, Path: "/api/admin/privacy-requests", Body: ""})
	handled, matched := result.(RequestHandled)
	if !matched {
		t.Fatalf("result = %T, want RequestHandled", result)
	}
	if handled.Value.Status != 200 {
		t.Fatalf("status = %d, want 200", handled.Value.Status)
	}
}

func TestPrivacyRequestHandlerRejectsInvalidKind(t *testing.T) {
	handler := NewPrivacyRequestHandler(
		newTestBrowserStorage(),
		fixedHandlerClock{value: time.Now().UTC()},
		fixedHandlerActor{userID: "user-admin"},
		fixedPrivacyRequestIDs{value: "privacy-1"},
	)
	result := handler.Handle(Request{Method: MethodPost, Path: "/api/privacy-requests", Body: `{"kind":"unknown"}`})
	rejected, matched := result.(RequestHandleRejected)
	if !matched {
		t.Fatalf("result = %T, want RequestHandleRejected", result)
	}
	if rejected.Reason != "privacy request kind is invalid" {
		t.Fatalf("reason = %q", rejected.Reason)
	}
}

func TestSavedQueueViewHandlerUpsertsAndListsThroughBrowserStorage(t *testing.T) {
	storage := newTestBrowserStorage()
	handler := NewSavedQueueViewHandler(
		storage,
		fixedHandlerActor{userID: "user-admin"},
		fixedSavedQueueViewIDs{value: "saved-view-1"},
	)
	createResult := handler.Handle(Request{
		Method: MethodPost,
		Path:   "/api/saved-queue-views",
		Body:   `{"scope":"team_work","name":"Ready work","query":"review","state_filter":"ready","type_filter":"code_review","sort":"title_asc"}`,
	})
	created, createdMatched := createResult.(RequestHandled)
	if !createdMatched {
		t.Fatalf("create result = %T, want RequestHandled", createResult)
	}
	if created.Value.Status != 200 {
		t.Fatalf("create status = %d, want 200", created.Value.Status)
	}
	var createdBody StoredSavedQueueView
	if err := json.Unmarshal([]byte(created.Value.Body), &createdBody); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if createdBody.ID != "saved-view-1" || createdBody.UserID != "user-admin" || createdBody.Scope != "team_work" {
		t.Fatalf("created body = %#v", createdBody)
	}

	listResult := handler.Handle(Request{Method: MethodGet, Path: "/api/saved-queue-views?scope=team_work", Body: ""})
	listed, listedMatched := listResult.(RequestHandled)
	if !listedMatched {
		t.Fatalf("list result = %T, want RequestHandled", listResult)
	}
	var listBody savedQueueViewsBody
	if err := json.Unmarshal([]byte(listed.Value.Body), &listBody); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listBody.Views) != 1 || listBody.Views[0].Name != "Ready work" {
		t.Fatalf("list body = %#v", listBody)
	}
}

func TestSavedQueueViewHandlerRejectsMissingIDSourceForUpsert(t *testing.T) {
	handler := NewSavedQueueViewHandler(newTestBrowserStorage(), fixedHandlerActor{userID: "user-admin"}, nil)
	result := handler.Handle(Request{Method: MethodPost, Path: "/api/saved-queue-views", Body: `{"scope":"team_work","name":"Ready work"}`})
	rejected, matched := result.(RequestHandleRejected)
	if !matched {
		t.Fatalf("result = %T, want RequestHandleRejected", result)
	}
	if rejected.Reason != "saved queue view id source is required" {
		t.Fatalf("reason = %q", rejected.Reason)
	}
}

func TestTaskHandlerCreatesAndLoadsTaskWithAttachments(t *testing.T) {
	storage := newTestBrowserStorage()
	handler := NewTaskHandler(storage, fixedHandlerActor{userID: "user-1"}, fixedTaskIDs{value: "task-1"})

	createResult := handler.Handle(Request{
		Method: MethodPost,
		Path:   "/api/tasks",
		Body:   `{"owner":{"kind":"user","user_id":"user-1"},"title":"Label receipts","description":"Extract totals.","reward":{"kind":"none","credit_amount":0,"collectible_ids":[]},"participation":{"policy":"open","assignee_scope":"user","reservation_expiry_hours":48},"visibility":{"kind":"public"},"placement":{"kind":"standalone"},"response_schema_json":"{\"kind\":\"freeform\"}","payload":{"kind":"none","json":""},"task_type":"general","attachments":[{"name":"brief.txt","content_type":"text/plain","data_url":"data:text/plain;base64,aGVsbG8="}]}`,
	})
	created, createdMatched := createResult.(RequestHandled)
	if !createdMatched {
		t.Fatalf("create result = %T, want RequestHandled", createResult)
	}
	if created.Value.Status != 201 {
		t.Fatalf("create status = %d, want 201", created.Value.Status)
	}
	var createdBody taskResponseBody
	if err := json.Unmarshal([]byte(created.Value.Body), &createdBody); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if createdBody.ID != "task-1" || createdBody.Title != "Label receipts" || len(createdBody.Attachments) != 1 {
		t.Fatalf("created body = %#v", createdBody)
	}

	loadResult := handler.Handle(Request{Method: MethodGet, Path: "/api/tasks/task-1", Body: ""})
	loaded, loadedMatched := loadResult.(RequestHandled)
	if !loadedMatched {
		t.Fatalf("load result = %T, want RequestHandled", loadResult)
	}
	var loadedBody taskResponseBody
	if err := json.Unmarshal([]byte(loaded.Value.Body), &loadedBody); err != nil {
		t.Fatalf("decode load response: %v", err)
	}
	if loadedBody.Attachments[0].SizeBytes != 5 {
		t.Fatalf("loaded attachment size = %d, want 5", loadedBody.Attachments[0].SizeBytes)
	}
}

// TestTaskHandlerHandleUserWork locks in a real bug found by hand-testing the
// demo: GET /api/users/{user_id}/work matched the generic users-collection
// route before this dedicated one existed, so the browser decoded a bare
// user record where it expected {"tasks": [...]} and the profile's "Public
// work" tab failed with a JSON decode error.
func TestTaskHandlerHandleUserWork(t *testing.T) {
	storage := newTestBrowserStorage()
	handler := NewTaskHandler(storage, fixedHandlerActor{userID: "user-1"}, fixedTaskIDs{value: "task-1"})

	create := handler.Handle(Request{
		Method: MethodPost,
		Path:   "/api/tasks",
		Body:   `{"owner":{"kind":"user","user_id":"user-1"},"title":"Reserved task","description":"Do it.","reward":{"kind":"none","credit_amount":0,"collectible_ids":[]},"participation":{"policy":"open","assignee_scope":"user","reservation_expiry_hours":48},"visibility":{"kind":"public"},"placement":{"kind":"standalone"},"response_schema_json":"{\"kind\":\"freeform\"}","payload":{"kind":"none","json":""},"task_type":"general","attachments":[]}`,
	})
	if _, matched := create.(RequestHandled); !matched {
		t.Fatalf("create result = %#v, want RequestHandled", create)
	}

	activeResult := SaveReservation(storage, StoredReservation{ID: "reservation-1", TaskID: "task-1", AssigneeKind: "user", AssigneeID: "user-2", State: "active", RequestedBy: "user-1"})
	if _, matched := activeResult.(ReservationStored); !matched {
		t.Fatalf("save active reservation = %#v, want ReservationStored", activeResult)
	}
	declinedResult := SaveReservation(storage, StoredReservation{ID: "reservation-2", TaskID: "task-1", AssigneeKind: "user", AssigneeID: "user-3", State: "declined", RequestedBy: "user-1"})
	if _, matched := declinedResult.(ReservationStored); !matched {
		t.Fatalf("save declined reservation = %#v, want ReservationStored", declinedResult)
	}

	assignedResult := handler.Handle(Request{Method: MethodGet, Path: "/api/users/user-2/work", Body: ""})
	assigned, assignedMatched := assignedResult.(RequestHandled)
	if !assignedMatched {
		t.Fatalf("user-2 work result = %#v, want RequestHandled", assignedResult)
	}
	var assignedBody tasksResponseBody
	if err := json.Unmarshal([]byte(assigned.Value.Body), &assignedBody); err != nil {
		t.Fatalf("decode user-2 work response: %v", err)
	}
	if len(assignedBody.Tasks) != 1 || assignedBody.Tasks[0].ID != "task-1" {
		t.Fatalf("user-2 work tasks = %#v, want [task-1]", assignedBody.Tasks)
	}

	declinedUserResult := handler.Handle(Request{Method: MethodGet, Path: "/api/users/user-3/work", Body: ""})
	declinedUser, declinedUserMatched := declinedUserResult.(RequestHandled)
	if !declinedUserMatched {
		t.Fatalf("user-3 work result = %#v, want RequestHandled", declinedUserResult)
	}
	var declinedUserBody tasksResponseBody
	if err := json.Unmarshal([]byte(declinedUser.Value.Body), &declinedUserBody); err != nil {
		t.Fatalf("decode user-3 work response: %v", err)
	}
	if len(declinedUserBody.Tasks) != 0 {
		t.Fatalf("user-3 (declined reservation) work tasks = %#v, want none", declinedUserBody.Tasks)
	}
}

func TestInteractionHandlerCreatesCommentsReservationsSubmissionAndLedger(t *testing.T) {
	storage := newTestBrowserStorage()
	task := StoredTask{
		ID:                     "task-1",
		OwnerKind:              "user",
		OwnerID:                "user-requester",
		Title:                  "Label receipts",
		Description:            "Extract totals.",
		TaskType:               "general",
		RewardKind:             "credit",
		RewardCreditAmount:     25,
		ParticipationPolicy:    "approval_required",
		AssigneeScope:          "user",
		ReservationExpiryHours: 48,
		State:                  "draft",
		VisibilityKind:         "public",
		SeriesKind:             "standalone",
		ResponseSchemaJSON:     `{"kind":"freeform"}`,
		PayloadKind:            "none",
		CreatedBy:              "user-requester",
	}
	if _, matched := SaveTask(storage, task).(TaskStored); !matched {
		t.Fatalf("task save was rejected")
	}
	handler := NewInteractionHandler(
		storage,
		fixedHandlerClock{value: time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)},
		fixedHandlerActor{userID: "user-worker"},
		fixedInteractionIDs{submissionID: "submission-1", commentID: "comment-1", reservationID: "reservation-1", ledgerID: "ledger-1"},
	)

	commentResult := handler.Handle(Request{Method: MethodPost, Path: "/api/tasks/task-1/comments", Body: `{"body":"I can do this."}`})
	commentHandled, commentMatched := commentResult.(RequestHandled)
	if !commentMatched {
		t.Fatalf("comment result = %T, want RequestHandled", commentResult)
	}
	if commentHandled.Value.Status != 201 {
		t.Fatalf("comment status = %d, want 201", commentHandled.Value.Status)
	}
	commentsResult := handler.Handle(Request{Method: MethodGet, Path: "/api/tasks/task-1/comments", Body: ""})
	commentsHandled, commentsMatched := commentsResult.(RequestHandled)
	if !commentsMatched {
		t.Fatalf("comments result = %T, want RequestHandled", commentsResult)
	}
	var comments taskCommentsBody
	if err := json.Unmarshal([]byte(commentsHandled.Value.Body), &comments); err != nil {
		t.Fatalf("decode comments: %v", err)
	}
	if len(comments.Comments) != 1 || comments.Comments[0].Body != "I can do this." {
		t.Fatalf("comments = %#v", comments)
	}

	reservationResult := handler.Handle(Request{Method: MethodPost, Path: "/api/tasks/task-1/reservations", Body: `{"assignee_kind":"user","assignee_id":"user-worker"}`})
	reservationHandled, reservationMatched := reservationResult.(RequestHandled)
	if !reservationMatched {
		t.Fatalf("reservation result = %T, want RequestHandled", reservationResult)
	}
	var reservation StoredReservation
	if err := json.Unmarshal([]byte(reservationHandled.Value.Body), &reservation); err != nil {
		t.Fatalf("decode reservation: %v", err)
	}
	if reservation.State != "requested" {
		t.Fatalf("reservation state = %q, want requested", reservation.State)
	}
	approveResult := handler.Handle(Request{Method: MethodPost, Path: "/api/tasks/task-1/reservations/reservation-1/approve", Body: `{}`})
	approved, approvedMatched := approveResult.(RequestHandled)
	if !approvedMatched {
		t.Fatalf("approve result = %T, want RequestHandled", approveResult)
	}
	var approvedReservation StoredReservation
	if err := json.Unmarshal([]byte(approved.Value.Body), &approvedReservation); err != nil {
		t.Fatalf("decode approved reservation: %v", err)
	}
	if approvedReservation.State != "active" {
		t.Fatalf("approved state = %q, want active", approvedReservation.State)
	}

	submissionResult := handler.Handle(Request{Method: MethodPost, Path: "/api/tasks/task-1/submissions", Body: `{"response_json":"{\"answer\":\"done\"}","attachments":[{"name":"proof.txt","content_type":"text/plain","data_url":"data:text/plain;base64,ZG9uZQ=="}]}`})
	submissionHandled, submissionMatched := submissionResult.(RequestHandled)
	if !submissionMatched {
		t.Fatalf("submission result = %T, want RequestHandled", submissionResult)
	}
	if submissionHandled.Value.Status != 201 {
		t.Fatalf("submission status = %d, want 201", submissionHandled.Value.Status)
	}
	var createdSubmission submissionCreatedBody
	if err := json.Unmarshal([]byte(submissionHandled.Value.Body), &createdSubmission); err != nil {
		t.Fatalf("decode submission: %v", err)
	}
	if createdSubmission.Submission.ID != "submission-1" || len(createdSubmission.Submission.Attachments) != 1 {
		t.Fatalf("submission = %#v", createdSubmission)
	}

	submissionCommentResult := handler.Handle(Request{Method: MethodPost, Path: "/api/submissions/submission-1/comments", Body: `{"body":"Submitted proof."}`})
	if _, matched := submissionCommentResult.(RequestHandled); !matched {
		t.Fatalf("submission comment result = %T, want RequestHandled", submissionCommentResult)
	}

	acceptResult := handler.Handle(Request{Method: MethodPost, Path: "/api/tasks/task-1/submissions/submission-1/accept", Body: `{"idempotency_key":"review-1","tip_amount":5}`})
	accepted, acceptedMatched := acceptResult.(RequestHandled)
	if !acceptedMatched {
		t.Fatalf("accept result = %T, want RequestHandled", acceptResult)
	}
	var acceptBody acceptSubmissionResultBody
	if err := json.Unmarshal([]byte(accepted.Value.Body), &acceptBody); err != nil {
		t.Fatalf("decode accept: %v", err)
	}
	if acceptBody.PayoutAmount != 25 || acceptBody.TipAmount != 5 || acceptBody.WorkerUserID != "user-worker" {
		t.Fatalf("accept body = %#v", acceptBody)
	}

	balanceResult := handler.Handle(Request{Method: MethodGet, Path: "/api/credits/balance", Body: ""})
	balanceHandled, balanceMatched := balanceResult.(RequestHandled)
	if !balanceMatched {
		t.Fatalf("balance result = %T, want RequestHandled", balanceResult)
	}
	var balance balanceBody
	if err := json.Unmarshal([]byte(balanceHandled.Value.Body), &balance); err != nil {
		t.Fatalf("decode balance: %v", err)
	}
	if balance.Amount != 130 {
		t.Fatalf("balance = %d, want 130", balance.Amount)
	}
}

func TestInteractionHandlerRejectsMissingIDSourceForMutation(t *testing.T) {
	handler := NewInteractionHandler(
		newTestBrowserStorage(),
		fixedHandlerClock{value: time.Now().UTC()},
		fixedHandlerActor{userID: "user-worker"},
		nil,
	)
	result := handler.Handle(Request{Method: MethodPost, Path: "/api/tasks/task-1/comments", Body: `{"body":"hello"}`})
	rejected, matched := result.(RequestHandleRejected)
	if !matched {
		t.Fatalf("result = %T, want RequestHandleRejected", result)
	}
	if rejected.Reason != "interaction id source is required" {
		t.Fatalf("reason = %q", rejected.Reason)
	}
}

func TestTaskHandlerRejectsMissingIDSourceForCreate(t *testing.T) {
	handler := NewTaskHandler(newTestBrowserStorage(), fixedHandlerActor{userID: "user-1"}, nil)
	result := handler.Handle(Request{Method: MethodPost, Path: "/api/tasks", Body: `{}`})
	rejected, matched := result.(RequestHandleRejected)
	if !matched {
		t.Fatalf("result = %T, want RequestHandleRejected", result)
	}
	if rejected.Reason != "task id source is required" {
		t.Fatalf("reason = %q", rejected.Reason)
	}
}

func TestTaskHandlerRejectsTooManyAttachments(t *testing.T) {
	handler := NewTaskHandler(newTestBrowserStorage(), fixedHandlerActor{userID: "user-1"}, fixedTaskIDs{value: "task-1"})
	result := handler.Handle(Request{
		Method: MethodPost,
		Path:   "/api/tasks",
		Body:   `{"owner":{"kind":"user","user_id":"user-1"},"title":"Label receipts","description":"Extract totals.","reward":{"kind":"none"},"participation":{"policy":"open","assignee_scope":"user","reservation_expiry_hours":48},"visibility":{"kind":"public"},"placement":{"kind":"standalone"},"response_schema_json":"{\"kind\":\"freeform\"}","payload":{"kind":"none"},"task_type":"general","attachments":[{"name":"1.txt","content_type":"text/plain","data_url":"data:text/plain;base64,aGVsbG8="},{"name":"2.txt","content_type":"text/plain","data_url":"data:text/plain;base64,aGVsbG8="},{"name":"3.txt","content_type":"text/plain","data_url":"data:text/plain;base64,aGVsbG8="},{"name":"4.txt","content_type":"text/plain","data_url":"data:text/plain;base64,aGVsbG8="},{"name":"5.txt","content_type":"text/plain","data_url":"data:text/plain;base64,aGVsbG8="},{"name":"6.txt","content_type":"text/plain","data_url":"data:text/plain;base64,aGVsbG8="}]}`,
	})
	rejected, matched := result.(RequestHandleRejected)
	if !matched {
		t.Fatalf("result = %T, want RequestHandleRejected", result)
	}
	if rejected.Reason != "too many attachments" {
		t.Fatalf("reason = %q", rejected.Reason)
	}
}

func TestNotificationHandlerListsAndMarksReadThroughBrowserStorage(t *testing.T) {
	storage := newTestBrowserStorage()
	if _, matched := SaveNotification(storage, StoredNotification{
		ID:              "notification-1",
		RecipientUserID: "user-2",
		ActorUserID:     "user-1",
		Kind:            "submission_created",
		SubjectKind:     "submission",
		SubjectID:       "submission-1",
		State:           "unread",
		MetadataJSON:    `{"task_id":"task-1"}`,
		CreatedAt:       "2026-07-01T10:00:00Z",
	}).(NotificationStored); !matched {
		t.Fatalf("notification save was rejected")
	}

	handler := NewNotificationHandler(storage, fixedHandlerActor{userID: "user-2"})
	listResult := handler.Handle(Request{Method: MethodGet, Path: "/api/notifications?limit=1&offset=0", Body: ""})
	listed, listedMatched := listResult.(RequestHandled)
	if !listedMatched {
		t.Fatalf("list result = %T, want RequestHandled", listResult)
	}
	if listed.Value.Status != 200 {
		t.Fatalf("list status = %d, want 200", listed.Value.Status)
	}
	var listBody notificationsBody
	if err := json.Unmarshal([]byte(listed.Value.Body), &listBody); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listBody.Notifications) != 1 || listBody.Notifications[0].ID != "notification-1" {
		t.Fatalf("list body = %#v", listBody)
	}

	markResult := handler.Handle(Request{Method: MethodPost, Path: "/api/notifications/notification-1/read", Body: ""})
	marked, markedMatched := markResult.(RequestHandled)
	if !markedMatched {
		t.Fatalf("mark result = %T, want RequestHandled", markResult)
	}
	var markedBody StoredNotification
	if err := json.Unmarshal([]byte(marked.Value.Body), &markedBody); err != nil {
		t.Fatalf("decode mark response: %v", err)
	}
	if markedBody.State != "read" {
		t.Fatalf("marked state = %q, want read", markedBody.State)
	}
}

func TestNotificationHandlerRejectsInvalidPagination(t *testing.T) {
	handler := NewNotificationHandler(newTestBrowserStorage(), fixedHandlerActor{userID: "user-2"})
	result := handler.Handle(Request{Method: MethodGet, Path: "/api/notifications?limit=not-a-number", Body: ""})
	rejected, matched := result.(RequestHandleRejected)
	if !matched {
		t.Fatalf("result = %T, want RequestHandleRejected", result)
	}
	if rejected.Reason != "notification limit is invalid" {
		t.Fatalf("reason = %q", rejected.Reason)
	}
}

func TestOrganizationHandlerCreatesListsAndManagesMembers(t *testing.T) {
	storage := newTestBrowserStorage()
	ids := &fixedOrganizationIDs{
		organizationID: "org-1",
		memberIDs:      []string{"member-owner", "member-2"},
		teamID:         "team-1",
	}
	handler := NewOrganizationHandler(
		storage,
		fixedHandlerActor{userID: "user-owner"},
		ids,
		fixedOrganizationUserResolver{usersByEmail: map[string]string{"member@example.com": "user-member"}},
	)

	createOrg := handler.Handle(Request{Method: MethodPost, Path: "/api/organizations", Body: `{"name":"Field Ops"}`})
	createdOrg, createdOrgMatched := createOrg.(RequestHandled)
	if !createdOrgMatched {
		t.Fatalf("create organization = %T, want RequestHandled", createOrg)
	}
	if createdOrg.Value.Status != 201 {
		t.Fatalf("create organization status = %d, want 201", createdOrg.Value.Status)
	}
	var orgBody StoredOrganization
	if err := json.Unmarshal([]byte(createdOrg.Value.Body), &orgBody); err != nil {
		t.Fatalf("decode organization: %v", err)
	}
	if orgBody.ID != "org-1" || orgBody.CreatedBy != "user-owner" {
		t.Fatalf("organization body = %#v", orgBody)
	}

	listOrg := handler.Handle(Request{Method: MethodGet, Path: "/api/organizations?query=field&limit=1&offset=0", Body: ""})
	listedOrg, listedOrgMatched := listOrg.(RequestHandled)
	if !listedOrgMatched {
		t.Fatalf("list organizations = %T, want RequestHandled", listOrg)
	}
	var orgs organizationsBody
	if err := json.Unmarshal([]byte(listedOrg.Value.Body), &orgs); err != nil {
		t.Fatalf("decode organizations: %v", err)
	}
	if len(orgs.Organizations) != 1 || orgs.Organizations[0].ID != "org-1" {
		t.Fatalf("organizations = %#v", orgs)
	}

	createTeam := handler.Handle(Request{Method: MethodPost, Path: "/api/organizations/org-1/teams", Body: `{"name":"North crew"}`})
	createdTeam, createdTeamMatched := createTeam.(RequestHandled)
	if !createdTeamMatched {
		t.Fatalf("create team = %T, want RequestHandled", createTeam)
	}
	if createdTeam.Value.Status != 201 {
		t.Fatalf("create team status = %d, want 201", createdTeam.Value.Status)
	}
	listTeam := handler.Handle(Request{Method: MethodGet, Path: "/api/organizations/org-1/teams?query=north&limit=1&offset=0", Body: ""})
	listedTeam, listedTeamMatched := listTeam.(RequestHandled)
	if !listedTeamMatched {
		t.Fatalf("list teams = %T, want RequestHandled", listTeam)
	}
	var teams teamsBody
	if err := json.Unmarshal([]byte(listedTeam.Value.Body), &teams); err != nil {
		t.Fatalf("decode teams: %v", err)
	}
	if len(teams.Teams) != 1 || teams.Teams[0].OrganizationID != "org-1" {
		t.Fatalf("teams = %#v", teams)
	}

	// GET /api/teams/{team_id} reproduces a real bug found by hand-testing the
	// demo: the route was unclassified (RequestUnsupported, 404), so the team
	// detail page never loaded, for either standalone or organization-owned
	// teams.
	adapted, adaptedMatched := Adapt(Request{Method: MethodGet, Path: "/api/teams/team-1"}).(RequestAdapted)
	if !adaptedMatched || adapted.Route != RouteStandaloneTeams {
		t.Fatalf("adapt team detail route = %#v, want RouteStandaloneTeams", adapted)
	}
	teamDetail := handler.Handle(Request{Method: MethodGet, Path: "/api/teams/team-1", Body: ""})
	teamDetailHandled, teamDetailHandledMatched := teamDetail.(RequestHandled)
	if !teamDetailHandledMatched {
		t.Fatalf("team detail = %#v, want RequestHandled", teamDetail)
	}
	var detailBody teamDetailBody
	if err := json.Unmarshal([]byte(teamDetailHandled.Value.Body), &detailBody); err != nil {
		t.Fatalf("decode team detail: %v", err)
	}
	if detailBody.Team.ID != "team-1" || detailBody.Team.OrganizationID != "org-1" || detailBody.Members == nil {
		t.Fatalf("team detail body = %#v", detailBody)
	}

	provision := handler.Handle(Request{
		Method: MethodPost,
		Path:   "/api/organizations/org-1/members",
		Body:   `{"email":"member@example.com","roles":["member"]}`,
	})
	provisioned, provisionedMatched := provision.(RequestHandled)
	if !provisionedMatched {
		t.Fatalf("provision member = %T, want RequestHandled", provision)
	}
	var member organizationMemberBody
	if err := json.Unmarshal([]byte(provisioned.Value.Body), &member); err != nil {
		t.Fatalf("decode member: %v", err)
	}
	if member.UserID != "user-member" || member.OrganizationID != "org-1" {
		t.Fatalf("member = %#v", member)
	}

	update := handler.Handle(Request{
		Method: MethodPatch,
		Path:   "/api/organizations/org-1/members/user-member/roles",
		Body:   `{"roles":["member","reviewer"]}`,
	})
	updated, updatedMatched := update.(RequestHandled)
	if !updatedMatched {
		t.Fatalf("update roles = %T, want RequestHandled", update)
	}
	var updatedMember organizationMemberBody
	if err := json.Unmarshal([]byte(updated.Value.Body), &updatedMember); err != nil {
		t.Fatalf("decode updated member: %v", err)
	}
	if len(updatedMember.Roles) != 2 || updatedMember.Roles[1] != "reviewer" {
		t.Fatalf("updated member = %#v", updatedMember)
	}

	deactivate := handler.Handle(Request{Method: MethodPatch, Path: "/api/organizations/org-1/members/user-member/deactivate", Body: `{}`})
	deactivated, deactivatedMatched := deactivate.(RequestHandled)
	if !deactivatedMatched {
		t.Fatalf("deactivate member = %T, want RequestHandled", deactivate)
	}
	if deactivated.Value.Status != 200 {
		t.Fatalf("deactivate status = %d, want 200", deactivated.Value.Status)
	}
}

func TestOrganizationHandlerRejectsMissingResolverForProvision(t *testing.T) {
	handler := NewOrganizationHandler(
		newTestBrowserStorage(),
		fixedHandlerActor{userID: "user-owner"},
		&fixedOrganizationIDs{organizationID: "org-1", memberIDs: []string{"member-1"}},
		nil,
	)
	result := handler.Handle(Request{Method: MethodPost, Path: "/api/organizations/org-1/members", Body: `{"email":"member@example.com","roles":["member"]}`})
	rejected, matched := result.(RequestHandleRejected)
	if !matched {
		t.Fatalf("result = %T, want RequestHandleRejected", result)
	}
	if rejected.Reason != "organization user resolver is required" {
		t.Fatalf("reason = %q", rejected.Reason)
	}
}

type fixedTaskSeriesIDs struct {
	seriesID string
}

func (ids fixedTaskSeriesIDs) NextTaskSeriesID() string {
	return ids.seriesID
}

// TestTaskSeriesHandlerCreatesListsAndLoadsSeries reproduces a real bug found
// by hand-testing the demo: /api/task-series was entirely unclassified (a
// 404), so creating a task series through the browser's "Task series" page
// failed outright.
func TestTaskSeriesHandlerCreatesListsAndLoadsSeries(t *testing.T) {
	storage := newTestBrowserStorage()
	handler := NewTaskSeriesHandler(storage, fixedHandlerActor{userID: "user-owner"}, fixedTaskSeriesIDs{seriesID: "series-1"})

	adapted, adaptedMatched := Adapt(Request{Method: MethodPost, Path: "/api/task-series"}).(RequestAdapted)
	if !adaptedMatched || adapted.Route != RouteTaskSeries {
		t.Fatalf("adapt create route = %#v, want RouteTaskSeries", adapted)
	}

	create := handler.Handle(Request{Method: MethodPost, Path: "/api/task-series", Body: `{"title":"Sprint 1","description":"A batch of related tasks."}`})
	created, createdMatched := create.(RequestHandled)
	if !createdMatched {
		t.Fatalf("create result = %#v, want RequestHandled", create)
	}
	if created.Value.Status != 201 {
		t.Fatalf("create status = %d, want 201", created.Value.Status)
	}
	var createdBody taskSeriesDetailResponseBody
	if err := json.Unmarshal([]byte(created.Value.Body), &createdBody); err != nil {
		t.Fatalf("decode created series: %v", err)
	}
	if createdBody.Series.ID != "series-1" || createdBody.Series.Title != "Sprint 1" || createdBody.Series.State != "draft" || createdBody.Series.CreatedBy != "user-owner" {
		t.Fatalf("created series = %#v", createdBody)
	}

	list := handler.Handle(Request{Method: MethodGet, Path: "/api/task-series", Body: ""})
	listed, listedMatched := list.(RequestHandled)
	if !listedMatched {
		t.Fatalf("list result = %#v, want RequestHandled", list)
	}
	var listedBody taskSeriesListResponseBody
	if err := json.Unmarshal([]byte(listed.Value.Body), &listedBody); err != nil {
		t.Fatalf("decode series list: %v", err)
	}
	if len(listedBody.Series) != 1 || listedBody.Series[0].ID != "series-1" {
		t.Fatalf("series list = %#v", listedBody)
	}

	detailAdapted, detailAdaptedMatched := Adapt(Request{Method: MethodGet, Path: "/api/task-series/series-1"}).(RequestAdapted)
	if !detailAdaptedMatched || detailAdapted.Route != RouteTaskSeries {
		t.Fatalf("adapt detail route = %#v, want RouteTaskSeries", detailAdapted)
	}
	detail := handler.Handle(Request{Method: MethodGet, Path: "/api/task-series/series-1", Body: ""})
	detailHandled, detailHandledMatched := detail.(RequestHandled)
	if !detailHandledMatched {
		t.Fatalf("detail result = %#v, want RequestHandled", detail)
	}
	var detailBody taskSeriesDetailResponseBody
	if err := json.Unmarshal([]byte(detailHandled.Value.Body), &detailBody); err != nil {
		t.Fatalf("decode series detail: %v", err)
	}
	if detailBody.Series.ID != "series-1" || detailBody.Tasks == nil || detailBody.Comments == nil {
		t.Fatalf("series detail = %#v", detailBody)
	}
}
