package taskbridge

import (
	"testing"

	"github.com/e6qu/sharecrop/internal/attachment"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/e6qu/sharecrop/internal/task/tasktest"
)

func newUserID(t *testing.T) core.UserID {
	t.Helper()
	created, matched := core.NewUserID().(core.UserIDCreated)
	if !matched {
		t.Fatalf("user id rejected")
	}
	return created.Value
}

func creditAmount(t *testing.T, value int64) task.CreditRewardAmount {
	t.Helper()
	accepted, matched := task.NewCreditRewardAmount(value).(task.CreditRewardAmountAccepted)
	if !matched {
		t.Fatalf("credit reward amount rejected")
	}
	return accepted.Value
}

func collectibleCount(t *testing.T, value int) task.CollectibleRewardCount {
	t.Helper()
	accepted, matched := task.NewCollectibleRewardCount(value).(task.CollectibleRewardCountAccepted)
	if !matched {
		t.Fatalf("collectible reward count rejected")
	}
	return accepted.Value
}

// sampleTask populates every union arm and value type worth stressing: an
// organization-team owner, a bundle reward, an existing-series placement, a JSON
// payload, an organization-team visibility, and an attachment.
func sampleTask(t *testing.T) task.Task {
	t.Helper()
	position, matched := task.NewSeriesPosition(3).(task.SeriesPositionAccepted)
	if !matched {
		t.Fatalf("series position rejected")
	}
	payload, matched := task.NewPayloadSource(`{"k":"v"}`).(task.PayloadSourceAccepted)
	if !matched {
		t.Fatalf("payload source rejected")
	}
	schema, matched := task.NewResponseSchemaSource(`{"type":"object"}`).(task.ResponseSchemaSourceAccepted)
	if !matched {
		t.Fatalf("response schema rejected")
	}
	title, matched := task.NewTitle("Review the PR").(task.TitleAccepted)
	if !matched {
		t.Fatalf("title rejected")
	}
	description, matched := task.NewDescription("Please review carefully.").(task.DescriptionAccepted)
	if !matched {
		t.Fatalf("description rejected")
	}
	reference, matched := task.NewReferenceURL("https://example.com/pr/1").(task.ReferenceURLAccepted)
	if !matched {
		t.Fatalf("reference rejected")
	}
	ttl, matched := task.NewReservationTTL(72).(task.ReservationTTLAccepted)
	if !matched {
		t.Fatalf("reservation ttl rejected")
	}
	att, matched := attachment.NewStoredAttachment("proof.txt", "text/plain", []byte("hello")).(attachment.AttachmentAccepted)
	if !matched {
		t.Fatalf("attachment rejected")
	}
	orgID, matched := core.NewOrganizationID().(core.OrganizationIDCreated)
	if !matched {
		t.Fatalf("organization id rejected")
	}
	teamID, matched := core.NewTeamID().(core.TeamIDCreated)
	if !matched {
		t.Fatalf("team id rejected")
	}
	return task.Task{
		ID:             tasktest.NewTaskID(t),
		Owner:          task.OrganizationTeamOwner{OrganizationID: orgID.Value, TeamID: teamID.Value},
		Title:          title.Value,
		Description:    description.Value,
		Type:           task.TaskTypeCodeReview,
		Reference:      reference.Value,
		Reward:         task.BundleRewardSpec{Credit: creditAmount(t, 40), Collectible: collectibleCount(t, 2)},
		Participation:  task.ParticipationPolicyReservationRequired,
		AssigneeScope:  task.AssigneeScopeOrganizationTeam,
		ReservationTTL: ttl.Value,
		State:          task.StateOpen,
		Visibility:     task.OrganizationTeamVisibility{OrganizationID: orgID.Value, TeamID: teamID.Value},
		Placement:      task.ExistingSeriesPlacement{SeriesID: tasktest.NewSeriesID(t), Position: position.Value},
		ResponseSchema: schema.Value,
		Payload:        task.JSONDataPayload{Source: payload.Value},
		Attachments:    []attachment.Attachment{att.Value},
		CreatedBy:      newUserID(t),
	}
}

func TestTaskRoundTrip(t *testing.T) {
	original := sampleTask(t)
	restored, err := decodeTask(encodeTask(original))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if diff := tasktest.TaskDiff(restored, original); diff != "" {
		t.Errorf("task mismatch: %s", diff)
	}
}

func TestCreateCommandRoundTrip(t *testing.T) {
	value := sampleTask(t)
	command := task.CreateCommand{
		Actor:          auth.UserSubject{ID: value.CreatedBy},
		Owner:          value.Owner,
		Title:          tasktest.Title(t, "Review the bridge"),
		Description:    tasktest.Description(t, "Check the bridge end to end."),
		Type:           value.Type,
		Reference:      value.Reference,
		Reward:         value.Reward,
		Participation:  value.Participation,
		AssigneeScope:  value.AssigneeScope,
		ReservationTTL: value.ReservationTTL,
		Visibility:     value.Visibility,
		Placement:      value.Placement,
		ResponseSchema: tasktest.SchemaSource(t, `{"kind":"freeform"}`),
		Payload:        value.Payload,
		Attachments:    value.Attachments,
	}
	restored, err := decodeCreateCommand(encodeCreateCommand(command))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if restored.Actor.ID != command.Actor.ID {
		t.Errorf("actor id did not round-trip: %s != %s", restored.Actor.ID, command.Actor.ID)
	}
	if restored.Title.String() != command.Title.String() {
		t.Errorf("title did not round-trip")
	}
	if rewardKindKey(restored.Reward) != rewardKindKey(command.Reward) {
		t.Errorf("reward did not round-trip")
	}
}

func TestSeriesRoundTrip(t *testing.T) {
	title, matched := task.NewSeriesTitle("Bridge series").(task.SeriesTitleAccepted)
	if !matched {
		t.Fatalf("series title rejected")
	}
	description, matched := task.NewSeriesDescription("A series.").(task.SeriesDescriptionAccepted)
	if !matched {
		t.Fatalf("series description rejected")
	}
	original := task.Series{
		ID:          tasktest.NewSeriesID(t),
		Owner:       task.UserOwner{UserID: newUserID(t)},
		Title:       title.Value,
		Description: description.Value,
		State:       task.SeriesStatePublished,
		CreatedBy:   newUserID(t),
	}
	restored, err := decodeSeries(encodeSeries(original))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if diff := tasktest.SeriesDiff(restored, original); diff != "" {
		t.Errorf("series mismatch: %s", diff)
	}
}

func TestReservationRoundTrip(t *testing.T) {
	orgID, matched := core.NewOrganizationID().(core.OrganizationIDCreated)
	if !matched {
		t.Fatalf("organization id rejected")
	}
	teamID, matched := core.NewTeamID().(core.TeamIDCreated)
	if !matched {
		t.Fatalf("team id rejected")
	}
	original := task.Reservation{
		ID:          tasktest.NewReservationID(t),
		TaskID:      tasktest.NewTaskID(t),
		Assignee:    task.OrganizationTeamAssignee{OrganizationID: orgID.Value, TeamID: teamID.Value},
		State:       task.ReservationStateActive,
		RequestedBy: newUserID(t),
	}
	restored, err := decodeReservation(encodeReservation(original))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if diff := tasktest.ReservationDiff(restored, original); diff != "" {
		t.Errorf("reservation mismatch: %s", diff)
	}
}

func TestCommentRoundTrip(t *testing.T) {
	commentID, matched := core.NewTaskCommentID().(core.TaskCommentIDCreated)
	if !matched {
		t.Fatalf("task comment id rejected")
	}
	original := task.TaskComment{
		ID:       commentID.Value,
		TaskID:   tasktest.NewTaskID(t),
		AuthorID: newUserID(t),
		Body:     tasktest.CommentBody(t, "please clarify"),
	}
	restored, err := decodeTaskComment(encodeTaskComment(original))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if restored.ID != original.ID || restored.Body.String() != original.Body.String() {
		t.Errorf("task comment did not round-trip: %+v", restored)
	}
}

func TestListScopeAndFiltersRoundTrip(t *testing.T) {
	scope := task.OrganizationListScope{OrganizationID: orgIDForTest(t), UserID: newUserID(t), IncludeReserved: true}
	restoredScope, err := decodeListScope(encodeListScope(scope))
	if err != nil {
		t.Fatalf("decode scope: %v", err)
	}
	organization, matched := restoredScope.(task.OrganizationListScope)
	if !matched || !organization.IncludeReserved || organization.UserID != scope.UserID {
		t.Errorf("organization scope did not round-trip: %+v", restoredScope)
	}

	filters := task.ListFilters{
		State:         task.StateIn{Values: []task.State{task.StateOpen, task.StateClosed}},
		Participation: task.ParticipationPolicyEquals{Value: task.ParticipationPolicyOpen},
		Search:        task.SearchContains{Value: searchText(t, "audit")},
		Type:          task.TypeEquals{Value: task.TaskTypeSecurityReview},
		Sort:          task.SortRewardDesc,
	}
	restoredFilters, err := decodeListFilters(encodeListFilters(filters))
	if err != nil {
		t.Fatalf("decode filters: %v", err)
	}
	stateIn, matched := restoredFilters.State.(task.StateIn)
	if !matched || len(stateIn.Values) != 2 {
		t.Errorf("state filter did not round-trip: %+v", restoredFilters.State)
	}
	if restoredFilters.Sort.String() != "reward_desc" {
		t.Errorf("sort did not round-trip: %s", restoredFilters.Sort)
	}
}

func TestResultRoundTrips(t *testing.T) {
	value := sampleTask(t)

	created, err := decodeCreateTaskResult(encodeCreateTaskResult(task.CreateTaskStoreAccepted{Value: value}))
	if err != nil {
		t.Fatalf("decode create: %v", err)
	}
	accepted, matched := created.(task.CreateTaskStoreAccepted)
	if !matched {
		t.Fatalf("create result = %T, want accepted", created)
	}
	if diff := tasktest.TaskDiff(accepted.Value, value); diff != "" {
		t.Errorf("created task mismatch: %s", diff)
	}

	listed, err := decodeListTasksResult(encodeListTasksResult(task.ListTasksStoreAccepted{Values: []task.ListItem{{Task: value, ActiveAssignee: task.NoActiveAssignee{}}}}))
	if err != nil {
		t.Fatalf("decode list: %v", err)
	}
	items, matched := listed.(task.ListTasksStoreAccepted)
	if !matched || len(items.Values) != 1 {
		t.Fatalf("list result = %T, want one item", listed)
	}
	if diff := tasktest.ListItemDiff(items.Values[0], task.ListItem{Task: value, ActiveAssignee: task.NoActiveAssignee{}}); diff != "" {
		t.Errorf("list item mismatch: %s", diff)
	}
}

// ---- small helpers ----

func searchText(t *testing.T, raw string) task.SearchText {
	t.Helper()
	accepted, matched := task.NewSearchText(raw).(task.SearchTextAccepted)
	if !matched {
		t.Fatalf("search text rejected")
	}
	return accepted.Value
}

func orgIDForTest(t *testing.T) core.OrganizationID {
	t.Helper()
	created, matched := core.NewOrganizationID().(core.OrganizationIDCreated)
	if !matched {
		t.Fatalf("organization id rejected")
	}
	return created.Value
}

func rewardKindKey(reward task.RewardSpec) string {
	switch reward.(type) {
	case task.CreditRewardSpec:
		return "credit"
	case task.CollectibleRewardSpec:
		return "collectible"
	case task.BundleRewardSpec:
		return "bundle"
	default:
		return "none"
	}
}
