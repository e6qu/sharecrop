package db

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/internal/task"
)

func requirePageForTest(t *testing.T, limit, offset int) core.Page {
	t.Helper()
	page, ok := core.NewPage(limit, offset).(core.PageAccepted)
	if !ok {
		t.Fatalf("page rejected")
	}
	return page.Value
}

func newAuditEventIDForTest(t *testing.T) core.AuditEventID {
	t.Helper()
	created, ok := core.NewAuditEventID().(core.AuditEventIDCreated)
	if !ok {
		t.Fatalf("audit event id rejected")
	}
	return created.Value
}

// TestAuditStoreOnSQLite exercises the NamedArgs path (@limit/@offset/@action)
// and strftime-ordered listing against SQLite.
func TestAuditStoreOnSQLite(t *testing.T) {
	ctx := context.Background()
	store := AuditStore{db: NewSQLite(openSQLiteWithSchema(t))}
	actor := newUserIDForTest(t)

	funded := audit.Event{
		ID:          newAuditEventIDForTest(t),
		ActorUserID: actor,
		Action:      audit.ActionTaskFunded,
		Subject:     audit.Subject{Kind: "task", ID: "t1"},
		Metadata:    audit.EmptyMetadata(),
		CreatedAt:   time.Now().UTC().Add(-time.Minute).Round(time.Microsecond),
	}
	accepted := audit.Event{
		ID:          newAuditEventIDForTest(t),
		ActorUserID: actor,
		Action:      audit.ActionSubmissionAccepted,
		Subject:     audit.Subject{Kind: "submission", ID: "s1"},
		Metadata:    audit.EmptyMetadata(),
		CreatedAt:   time.Now().UTC().Round(time.Microsecond),
	}
	for _, event := range []audit.Event{funded, accepted} {
		if _, ok := store.Record(ctx, event).(audit.EventRecorded); !ok {
			t.Fatalf("record rejected")
		}
	}

	page := requirePageForTest(t, 50, 0)
	all, ok := store.List(ctx, audit.NoListFilters(), page).(audit.EventsListed)
	if !ok {
		t.Fatalf("list rejected")
	}
	if len(all.Values) != 2 {
		t.Fatalf("listed %d events, want 2", len(all.Values))
	}
	if all.Values[0].ID != accepted.ID {
		t.Fatalf("newest-first ordering wrong: got %s, want %s", all.Values[0].ID, accepted.ID)
	}

	filtered, ok := store.List(ctx, audit.ListFilters{
		Action:      audit.ActionEquals{Value: audit.ActionTaskFunded},
		SubjectKind: audit.AnySubjectKind{},
		SubjectID:   audit.AnySubjectID{},
	}, page).(audit.EventsListed)
	if !ok {
		t.Fatalf("filtered list rejected")
	}
	if len(filtered.Values) != 1 || filtered.Values[0].ID != funded.ID {
		t.Fatalf("action filter wrong: %d events", len(filtered.Values))
	}

	got, ok := store.Get(ctx, funded.ID).(audit.EventFound)
	if !ok {
		t.Fatalf("get rejected")
	}
	if !got.Value.CreatedAt.Equal(funded.CreatedAt) {
		t.Fatalf("get timestamp = %s, want %s", got.Value.CreatedAt, funded.CreatedAt)
	}
}

// TestAttachmentEncodeOnSQLite exercises the registered encode(x,'base64')
// function together with jsonb_agg/jsonb_build_object ordering — the submission
// and task attachment read path.
func TestAttachmentEncodeOnSQLite(t *testing.T) {
	ctx := context.Background()
	sqlHandle := openSQLiteWithSchema(t)
	handle := NewSQLite(sqlHandle)

	content := []byte("hello attachment body")
	if _, err := sqlHandle.ExecContext(ctx, `insert into submission_attachments (submission_id, attachment_index, filename, content_type, content) values ('s1', 0, 'note.txt', 'text/plain', ?)`, content); err != nil {
		t.Fatalf("seed attachment: %v", err)
	}

	var aggregated string
	err := handle.QueryRow(ctx, `
		select coalesce(jsonb_agg(
			jsonb_build_object(
				'name', submission_attachments.filename,
				'content', encode(submission_attachments.content, 'base64')
			)
			order by submission_attachments.attachment_index
		), '[]'::jsonb)::text
		from submission_attachments
		where submission_attachments.submission_id = $1
	`, "s1").Scan(&aggregated)
	if err != nil {
		t.Fatalf("attachment aggregation: %v", err)
	}

	var attachments []struct {
		Name    string `json:"name"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(aggregated), &attachments); err != nil {
		t.Fatalf("unmarshal aggregated %q: %v", aggregated, err)
	}
	if len(attachments) != 1 || attachments[0].Name != "note.txt" {
		t.Fatalf("attachments = %+v, want one note.txt", attachments)
	}
	decoded, err := base64.StdEncoding.DecodeString(attachments[0].Content)
	if err != nil {
		t.Fatalf("content is not base64: %v", err)
	}
	if string(decoded) != string(content) {
		t.Fatalf("decoded content = %q, want %q", decoded, content)
	}
}

// TestArrayAggOnSQLite exercises the array_agg translation (→ json_group_array
// with a null FILTER) and the StringArray Scanner parsing the JSON result.
func TestArrayAggOnSQLite(t *testing.T) {
	ctx := context.Background()
	sqlHandle := openSQLiteWithSchema(t)
	handle := NewSQLite(sqlHandle)

	if _, err := sqlHandle.ExecContext(ctx, `insert into agent_credential_scopes (credential_id, scope) values ('c1','tasks_read'),('c1','submissions_review')`); err != nil {
		t.Fatalf("seed scopes: %v", err)
	}

	var scopes StringArray
	err := handle.QueryRow(ctx, `
		select coalesce(array_remove(array_agg(agent_credential_scopes.scope), null), '{}')::text
		from agent_credential_scopes
		where credential_id = $1
	`, "c1").Scan(&scopes)
	if err != nil {
		t.Fatalf("array_agg query: %v", err)
	}
	if len(scopes) != 2 {
		t.Fatalf("scopes = %v, want 2 elements", scopes)
	}
	found := map[string]bool{}
	for _, scope := range scopes {
		found[scope] = true
	}
	if !found["tasks_read"] || !found["submissions_review"] {
		t.Fatalf("scopes = %v, want tasks_read + submissions_review", scopes)
	}

	// Empty aggregation must yield an empty slice, not an error.
	var empty StringArray
	if err := handle.QueryRow(ctx, `
		select coalesce(array_remove(array_agg(agent_credential_scopes.scope), null), '{}')::text
		from agent_credential_scopes
		where credential_id = $1
	`, "missing").Scan(&empty); err != nil {
		t.Fatalf("empty array_agg query: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("empty aggregation = %v, want []", empty)
	}
}

// TestSavedQueueViewUpsertOnSQLite exercises ON CONFLICT DO UPDATE + excluded on
// SQLite: a second upsert for the same (user, scope, name) updates in place.
func TestSavedQueueViewUpsertOnSQLite(t *testing.T) {
	ctx := context.Background()
	store := SavedQueueViewStore{db: NewSQLite(openSQLiteWithSchema(t))}
	user := newUserIDForTest(t)

	view := httpserver.SavedQueueView{
		UserID:      user,
		Scope:       "team_work",
		Name:        "mine",
		Query:       "state:open",
		StateFilter: "open",
		TypeFilter:  "",
		Sort:        "newest",
	}
	if _, ok := store.Upsert(ctx, view).(httpserver.SavedQueueViewSaved); !ok {
		t.Fatalf("first upsert rejected")
	}
	view.Query = "state:closed"
	if _, ok := store.Upsert(ctx, view).(httpserver.SavedQueueViewSaved); !ok {
		t.Fatalf("second upsert (on conflict) rejected")
	}

	listed, ok := store.List(ctx, user, "team_work", core.DefaultPage()).(httpserver.SavedQueueViewsListed)
	if !ok {
		t.Fatalf("list rejected")
	}
	if len(listed.Values) != 1 {
		t.Fatalf("listed %d views, want 1 (upsert must update, not insert)", len(listed.Values))
	}
	if listed.Values[0].Query != "state:closed" {
		t.Fatalf("query = %q, want updated to state:closed", listed.Values[0].Query)
	}
}

// TestCreateTaskWithCollectibleEscrowOnSQLite proves task creation with reward
// collectibles is all-or-nothing on the demo (SQLite) engine, which shares the
// single TaskStore implementation: escrowing a collectible the creator does
// not own rejects the create and leaves no task row behind, while an owned
// collectible is escrowed inside the same create transaction.
func TestCreateTaskWithCollectibleEscrowOnSQLite(t *testing.T) {
	ctx := context.Background()
	sqlHandle := openSQLiteWithSchema(t)
	handle := NewSQLite(sqlHandle)
	store := NewTaskStoreFromHandle(handle)

	creator := newUserIDForTest(t)
	stranger := newUserIDForTest(t)

	seedCollectible := func(t *testing.T, owner core.UserID) core.CollectibleID {
		t.Helper()
		created, matched := core.NewCollectibleID().(core.CollectibleIDCreated)
		if !matched {
			t.Fatalf("collectible id rejected")
		}
		if _, err := sqlHandle.ExecContext(ctx, `
			insert into collectibles (id, name, kind, state, transfer_policy, owner_user_id, owner_kind, art)
			values (?, 'Golden Hoe', 'badge', 'minted', 'transferable_between_users', ?, 'user', '')
		`, created.Value.String(), owner.String()); err != nil {
			t.Fatalf("seed collectible: %v", err)
		}
		return created.Value
	}

	countValue := func(t *testing.T, query string, argument string) int {
		t.Helper()
		var count int
		if err := handle.QueryRow(ctx, query, argument).Scan(&count); err != nil {
			t.Fatalf("count query: %v", err)
		}
		return count
	}

	newCollectibleCreateCommand := func(t *testing.T, collectibleID core.CollectibleID) task.CreateCommand {
		t.Helper()
		reference, referenceMatched := task.NewReferenceURL("").(task.ReferenceURLAccepted)
		if !referenceMatched {
			t.Fatalf("reference rejected")
		}
		title, titleMatched := task.NewTitle("Escrow at create").(task.TitleAccepted)
		if !titleMatched {
			t.Fatalf("title rejected")
		}
		description, descriptionMatched := task.NewDescription("Escrow a reward collectible inside the create transaction.").(task.DescriptionAccepted)
		if !descriptionMatched {
			t.Fatalf("description rejected")
		}
		schema, schemaMatched := task.NewResponseSchemaSource(`{"kind":"freeform"}`).(task.ResponseSchemaSourceAccepted)
		if !schemaMatched {
			t.Fatalf("schema rejected")
		}
		count, countMatched := task.NewCollectibleRewardCount(1).(task.CollectibleRewardCountAccepted)
		if !countMatched {
			t.Fatalf("collectible reward count rejected")
		}
		return task.CreateCommand{
			Actor:              auth.UserSubject{ID: creator},
			Owner:              task.UserOwner{UserID: creator},
			Title:              title.Value,
			Description:        description.Value,
			Type:               task.TaskTypeGeneral,
			Reference:          reference.Value,
			Reward:             task.CollectibleRewardSpec{Count: count.Value},
			Participation:      task.ParticipationPolicyOpen,
			AssigneeScope:      task.AssigneeScopeUser,
			ReservationTTL:     task.DefaultReservationTTL(),
			Visibility:         task.PublicVisibility{},
			Placement:          task.StandalonePlacement{},
			ResponseSchema:     schema.Value,
			Payload:            task.NoDataPayload{},
			FundCollectibleIDs: []core.CollectibleID{collectibleID},
		}
	}

	newTaskID := func(t *testing.T) core.TaskID {
		t.Helper()
		created, matched := core.NewTaskID().(core.TaskIDCreated)
		if !matched {
			t.Fatalf("task id rejected")
		}
		return created.Value
	}
	newSeriesID := func(t *testing.T) core.TaskSeriesID {
		t.Helper()
		created, matched := core.NewTaskSeriesID().(core.TaskSeriesIDCreated)
		if !matched {
			t.Fatalf("series id rejected")
		}
		return created.Value
	}

	t.Run("a collectible the creator does not own rolls the whole create back", func(t *testing.T) {
		unowned := seedCollectible(t, stranger)
		taskID := newTaskID(t)
		result := store.CreateTask(ctx, newSeriesID(t), taskID, newCollectibleCreateCommand(t, unowned))
		if _, rejected := result.(task.CreateTaskStoreRejected); !rejected {
			t.Fatalf("create task result = %T, want rejected", result)
		}
		if got := countValue(t, "select count(*) from tasks where id = $1", taskID.String()); got != 0 {
			t.Fatalf("task rows after rejected create = %d, want 0", got)
		}
		if got := countValue(t, "select count(*) from task_fund_collectibles where task_id = $1", taskID.String()); got != 0 {
			t.Fatalf("escrow rows after rejected create = %d, want 0", got)
		}
		var state string
		if err := handle.QueryRow(ctx, "select state from collectibles where id = $1", unowned.String()).Scan(&state); err != nil {
			t.Fatalf("read collectible state: %v", err)
		}
		if state != "minted" {
			t.Fatalf("collectible state after rejected create = %q, want minted", state)
		}
	})

	t.Run("an owned collectible is escrowed inside the create transaction", func(t *testing.T) {
		owned := seedCollectible(t, creator)
		taskID := newTaskID(t)
		result := store.CreateTask(ctx, newSeriesID(t), taskID, newCollectibleCreateCommand(t, owned))
		if _, accepted := result.(task.CreateTaskStoreAccepted); !accepted {
			t.Fatalf("create task result = %#v, want accepted", result)
		}
		if got := countValue(t, "select count(*) from tasks where id = $1", taskID.String()); got != 1 {
			t.Fatalf("task rows after create = %d, want 1", got)
		}
		if got := countValue(t, "select count(*) from task_fund_collectibles where task_id = $1", taskID.String()); got != 1 {
			t.Fatalf("escrow rows after create = %d, want 1", got)
		}
		var state string
		if err := handle.QueryRow(ctx, "select state from collectibles where id = $1", owned.String()).Scan(&state); err != nil {
			t.Fatalf("read collectible state: %v", err)
		}
		if state != "escrowed" {
			t.Fatalf("collectible state after create = %q, want escrowed", state)
		}
	})
}

// TestTaskCommentsPaginationOnSQLite pins the bounded comment listing: pages
// are newest-first and limit/offset windows do not overlap.
func TestTaskCommentsPaginationOnSQLite(t *testing.T) {
	ctx := context.Background()
	sqlHandle := openSQLiteWithSchema(t)
	store := NewTaskStoreFromHandle(NewSQLite(sqlHandle))

	taskID, matched := core.NewTaskID().(core.TaskIDCreated)
	if !matched {
		t.Fatalf("task id rejected")
	}
	author := newUserIDForTest(t)
	commentIDs := make([]string, 0, 3)
	for index := 0; index < 3; index++ {
		created, idMatched := core.NewTaskCommentID().(core.TaskCommentIDCreated)
		if !idMatched {
			t.Fatalf("comment id rejected")
		}
		createdAt := time.Date(2026, 1, 2, 3, index, 0, 0, time.UTC).Format(time.RFC3339Nano)
		if _, err := sqlHandle.ExecContext(ctx, `
			insert into task_comments (id, task_id, author_user_id, body, created_at)
			values (?, ?, ?, ?, ?)
		`, created.Value.String(), taskID.Value.String(), author.String(), "comment", createdAt); err != nil {
			t.Fatalf("seed comment: %v", err)
		}
		commentIDs = append(commentIDs, created.Value.String())
	}

	firstPage, ok := store.ListTaskComments(ctx, taskID.Value, requirePageForTest(t, 2, 0)).(task.ListTaskCommentsStoreAccepted)
	if !ok {
		t.Fatalf("first page rejected")
	}
	if len(firstPage.Values) != 2 {
		t.Fatalf("first page = %d comments, want 2", len(firstPage.Values))
	}
	if firstPage.Values[0].ID.String() != commentIDs[2] || firstPage.Values[1].ID.String() != commentIDs[1] {
		t.Fatalf("first page is not newest-first: %s, %s", firstPage.Values[0].ID, firstPage.Values[1].ID)
	}

	secondPage, ok := store.ListTaskComments(ctx, taskID.Value, requirePageForTest(t, 2, 2)).(task.ListTaskCommentsStoreAccepted)
	if !ok {
		t.Fatalf("second page rejected")
	}
	if len(secondPage.Values) != 1 || secondPage.Values[0].ID.String() != commentIDs[0] {
		t.Fatalf("second page = %#v, want the single oldest comment", secondPage.Values)
	}
}
