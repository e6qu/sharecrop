//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestTaskSeriesListAndFind(t *testing.T) {
	pool := newPool(t)
	owner := createUser(t, pool, "series-owner")
	seriesID := newTaskSeriesID(t)
	insertSeries(t, pool, seriesID, owner)
	insertSeriesTask(t, pool, seriesID, owner, 1)
	insertSeriesTask(t, pool, seriesID, owner, 2)

	store := db.NewTaskStore(pool)

	listed, matched := store.ListSeries(context.Background(), owner, core.DefaultPage()).(task.ListSeriesStoreAccepted)
	if !matched {
		t.Fatalf("list series rejected")
	}
	found := false
	for index := range listed.Values {
		if listed.Values[index].ID == seriesID {
			found = true
		}
	}
	if !found {
		t.Fatalf("created series not found in list")
	}

	detail, detailMatched := store.FindSeries(context.Background(), seriesID).(task.FindSeriesStoreAccepted)
	if !detailMatched {
		t.Fatalf("find series rejected")
	}
	if detail.Value.Series.ID != seriesID {
		t.Fatalf("found series id mismatch")
	}
	if len(detail.Value.Tasks) != 2 {
		t.Fatalf("series task count = %d, want 2", len(detail.Value.Tasks))
	}
}

func newTaskSeriesID(t *testing.T) core.TaskSeriesID {
	t.Helper()
	created, matched := core.NewTaskSeriesID().(core.TaskSeriesIDCreated)
	if !matched {
		t.Fatalf("task series id rejected")
	}
	return created.Value
}

func insertSeries(t *testing.T, pool *pgxpool.Pool, seriesID core.TaskSeriesID, owner core.UserID) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		insert into task_series (id, owner_kind, user_id, title, created_by_user_id)
		values ($1, 'user', $2, 'Integration series', $2)
	`, seriesID.String(), owner.String())
	if err != nil {
		t.Fatalf("insert series: %v", err)
	}
}

func insertSeriesTask(t *testing.T, pool *pgxpool.Pool, seriesID core.TaskSeriesID, owner core.UserID, position int) {
	t.Helper()
	taskID := newTaskID(t)
	_, err := pool.Exec(context.Background(), `
		insert into tasks (id, series_id, series_position, owner_kind, user_id, title, description, state, response_schema_json, data_payload_kind, created_by_user_id)
		values ($1, $2, $3, 'user', $4, 'Series task', 'A task in a series', 'draft', '{}'::jsonb, 'none', $4)
	`, taskID.String(), seriesID.String(), position, owner.String())
	if err != nil {
		t.Fatalf("insert series task: %v", err)
	}
	_, err = pool.Exec(context.Background(), `
		insert into task_visibility_scopes (task_id, visibility_kind, scope_key, user_id)
		values ($1, 'user', $2, $3)
	`, taskID.String(), owner.String(), owner.String())
	if err != nil {
		t.Fatalf("insert task visibility: %v", err)
	}
}
