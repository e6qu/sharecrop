// Package tasktest holds test-support helpers for task models, shared by the
// task bridge's codec tests and the integration dual-run test so the two do not
// carry duplicate comparisons.
package tasktest

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/e6qu/sharecrop/internal/attachment"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/task"
)

// ---- id builders shared by the codec and integration tests ----

// NewTaskID mints a fresh task id or fails the test.
func NewTaskID(t testing.TB) core.TaskID {
	t.Helper()
	created, matched := core.NewTaskID().(core.TaskIDCreated)
	if !matched {
		t.Fatalf("task id rejected")
	}
	return created.Value
}

// NewSeriesID mints a fresh task series id or fails the test.
func NewSeriesID(t testing.TB) core.TaskSeriesID {
	t.Helper()
	created, matched := core.NewTaskSeriesID().(core.TaskSeriesIDCreated)
	if !matched {
		t.Fatalf("series id rejected")
	}
	return created.Value
}

// NewReservationID mints a fresh task reservation id or fails the test.
func NewReservationID(t testing.TB) core.TaskReservationID {
	t.Helper()
	created, matched := core.NewTaskReservationID().(core.TaskReservationIDCreated)
	if !matched {
		t.Fatalf("reservation id rejected")
	}
	return created.Value
}

// Title builds a validated task title or fails the test.
func Title(t testing.TB, raw string) task.Title {
	t.Helper()
	accepted, matched := task.NewTitle(raw).(task.TitleAccepted)
	if !matched {
		t.Fatalf("title rejected")
	}
	return accepted.Value
}

// Description builds a validated task description or fails the test.
func Description(t testing.TB, raw string) task.Description {
	t.Helper()
	accepted, matched := task.NewDescription(raw).(task.DescriptionAccepted)
	if !matched {
		t.Fatalf("description rejected")
	}
	return accepted.Value
}

// SchemaSource builds a validated response schema source or fails the test.
func SchemaSource(t testing.TB, raw string) task.ResponseSchemaSource {
	t.Helper()
	accepted, matched := task.NewResponseSchemaSource(raw).(task.ResponseSchemaSourceAccepted)
	if !matched {
		t.Fatalf("response schema rejected")
	}
	return accepted.Value
}

// CommentBody builds a validated comment body or fails the test.
func CommentBody(t testing.TB, raw string) task.CommentBody {
	t.Helper()
	accepted, matched := task.NewCommentBody(raw).(task.CommentBodyAccepted)
	if !matched {
		t.Fatalf("comment body rejected")
	}
	return accepted.Value
}

// TaskDiff returns a description of the first field in which two tasks differ,
// or "" if they are equal. It compares every field the store persists (scalars
// by their string/int form, unions by a canonical key), so a bridge that drops
// or garbles one is caught.
func TaskDiff(got, want task.Task) string {
	for _, field := range []struct {
		name     string
		got      string
		expected string
	}{
		{"id", got.ID.String(), want.ID.String()},
		{"owner", ownerKey(got.Owner), ownerKey(want.Owner)},
		{"title", got.Title.String(), want.Title.String()},
		{"description", got.Description.String(), want.Description.String()},
		{"type", got.Type.String(), want.Type.String()},
		{"reference", got.Reference.String(), want.Reference.String()},
		{"reward", rewardKey(got.Reward), rewardKey(want.Reward)},
		{"participation", got.Participation.String(), want.Participation.String()},
		{"assignee_scope", got.AssigneeScope.String(), want.AssigneeScope.String()},
		{"reservation_ttl", itoa(got.ReservationTTL.Hours()), itoa(want.ReservationTTL.Hours())},
		{"state", got.State.String(), want.State.String()},
		{"visibility", visibilityKey(got.Visibility), visibilityKey(want.Visibility)},
		{"placement", placementKey(got.Placement), placementKey(want.Placement)},
		{"response_schema", got.ResponseSchema.String(), want.ResponseSchema.String()},
		{"payload", payloadKey(got.Payload), payloadKey(want.Payload)},
		{"created_by", got.CreatedBy.String(), want.CreatedBy.String()},
	} {
		if field.got != field.expected {
			return fmt.Sprintf("%s: %q != %q", field.name, field.got, field.expected)
		}
	}
	return attachmentsDiff(got.Attachments, want.Attachments)
}

// SeriesDiff compares two series.
func SeriesDiff(got, want task.Series) string {
	switch {
	case got.ID != want.ID:
		return fmt.Sprintf("id: %s != %s", got.ID, want.ID)
	case ownerKey(got.Owner) != ownerKey(want.Owner):
		return fmt.Sprintf("owner: %s != %s", ownerKey(got.Owner), ownerKey(want.Owner))
	case got.Title.String() != want.Title.String():
		return fmt.Sprintf("title: %s != %s", got.Title, want.Title)
	case got.Description.String() != want.Description.String():
		return fmt.Sprintf("description: %s != %s", got.Description, want.Description)
	case got.State.String() != want.State.String():
		return fmt.Sprintf("state: %s != %s", got.State, want.State)
	case got.CreatedBy != want.CreatedBy:
		return fmt.Sprintf("created_by: %s != %s", got.CreatedBy, want.CreatedBy)
	default:
		return ""
	}
}

// ReservationDiff compares two reservations.
func ReservationDiff(got, want task.Reservation) string {
	switch {
	case got.ID != want.ID:
		return fmt.Sprintf("id: %s != %s", got.ID, want.ID)
	case got.TaskID != want.TaskID:
		return fmt.Sprintf("task_id: %s != %s", got.TaskID, want.TaskID)
	case assigneeKey(got.Assignee) != assigneeKey(want.Assignee):
		return fmt.Sprintf("assignee: %s != %s", assigneeKey(got.Assignee), assigneeKey(want.Assignee))
	case got.State.String() != want.State.String():
		return fmt.Sprintf("state: %s != %s", got.State, want.State)
	case got.RequestedBy != want.RequestedBy:
		return fmt.Sprintf("requested_by: %s != %s", got.RequestedBy, want.RequestedBy)
	default:
		return ""
	}
}

// ListItemDiff compares a task-list item (task plus active assignee).
func ListItemDiff(got, want task.ListItem) string {
	if diff := TaskDiff(got.Task, want.Task); diff != "" {
		return diff
	}
	if activeAssigneeKey(got.ActiveAssignee) != activeAssigneeKey(want.ActiveAssignee) {
		return fmt.Sprintf("active_assignee: %s != %s", activeAssigneeKey(got.ActiveAssignee), activeAssigneeKey(want.ActiveAssignee))
	}
	return ""
}

func attachmentsDiff(got, want []attachment.Attachment) string {
	if len(got) != len(want) {
		return fmt.Sprintf("attachments: length %d != %d", len(got), len(want))
	}
	for index := range want {
		if got[index].Name.String() != want[index].Name.String() {
			return fmt.Sprintf("attachment %d name: %s != %s", index, got[index].Name, want[index].Name)
		}
		if string(got[index].Content.Bytes()) != string(want[index].Content.Bytes()) {
			return fmt.Sprintf("attachment %d content differs", index)
		}
	}
	return ""
}

func ownerKey(owner task.Owner) string {
	switch typed := owner.(type) {
	case task.UserOwner:
		return "user:" + typed.UserID.String()
	case task.TeamOwner:
		return "team:" + typed.TeamID.String()
	case task.OrganizationOwner:
		return "organization:" + typed.OrganizationID.String()
	case task.OrganizationTeamOwner:
		return "organization_team:" + typed.OrganizationID.String() + ":" + typed.TeamID.String()
	default:
		return "unknown"
	}
}

func visibilityKey(visibility task.Visibility) string {
	switch typed := visibility.(type) {
	case task.PublicVisibility:
		return "public"
	case task.UserVisibility:
		return "user:" + typed.UserID.String()
	case task.TeamVisibility:
		return "team:" + typed.TeamID.String()
	case task.OrganizationVisibility:
		return "organization:" + typed.OrganizationID.String()
	case task.OrganizationTeamVisibility:
		return "organization_team:" + typed.OrganizationID.String() + ":" + typed.TeamID.String()
	default:
		return "unknown"
	}
}

func assigneeKey(assignee task.Assignee) string {
	switch typed := assignee.(type) {
	case task.UserAssignee:
		return "user:" + typed.UserID.String()
	case task.TeamAssignee:
		return "team:" + typed.TeamID.String()
	case task.OrganizationTeamAssignee:
		return "organization_team:" + typed.OrganizationID.String() + ":" + typed.TeamID.String()
	default:
		return "unknown"
	}
}

func activeAssigneeKey(assignee task.ActiveAssignee) string {
	switch typed := assignee.(type) {
	case task.NoActiveAssignee:
		return "none"
	case task.ActiveUserAssignee:
		return "user:" + typed.UserID.String()
	case task.ActiveTeamAssignee:
		return "team:" + typed.TeamID.String()
	case task.ActiveOrganizationTeamAssignee:
		return "organization_team:" + typed.OrganizationID.String() + ":" + typed.TeamID.String()
	default:
		return "unknown"
	}
}

func rewardKey(reward task.RewardSpec) string {
	switch typed := reward.(type) {
	case task.NoRewardSpec:
		return "none"
	case task.CreditRewardSpec:
		return "credit:" + itoa64(typed.Amount.Int64())
	case task.CollectibleRewardSpec:
		return "collectible:" + itoa(typed.Count.Int())
	case task.BundleRewardSpec:
		return "bundle:" + itoa64(typed.Credit.Int64()) + ":" + itoa(typed.Collectible.Int())
	default:
		return "unknown"
	}
}

func placementKey(placement task.SeriesPlacement) string {
	switch typed := placement.(type) {
	case task.StandalonePlacement:
		return "standalone"
	case task.NewSeriesPlacement:
		return "new:" + typed.Title.String() + ":" + itoa(typed.Position.Int())
	case task.ExistingSeriesPlacement:
		return "existing:" + typed.SeriesID.String() + ":" + itoa(typed.Position.Int())
	default:
		return "unknown"
	}
}

func payloadKey(payload task.DataPayload) string {
	json, matched := payload.(task.JSONDataPayload)
	if !matched {
		return "none"
	}
	return "json:" + json.Source.String()
}

func itoa(value int) string { return strconv.Itoa(value) }

func itoa64(value int64) string { return strconv.FormatInt(value, 10) }
