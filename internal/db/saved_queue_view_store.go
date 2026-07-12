package db

import (
	"context"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/core/id"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SavedQueueViewStore struct {
	db Querier
}

func NewSavedQueueViewStore(pool *pgxpool.Pool) SavedQueueViewStore {
	return NewSavedQueueViewStoreFromHandle(NewPGX(pool))
}

func NewSavedQueueViewStoreFromHandle(handle Beginner) SavedQueueViewStore {
	return SavedQueueViewStore{db: handle}
}

func (store SavedQueueViewStore) List(ctx context.Context, userID core.UserID, scope string) httpserver.SavedQueueViewsListResult {
	query := `
		select id::text, user_id::text, scope, name, query_text, state_filter, type_filter, sort_order
		from saved_queue_views
		where user_id = $1
	`
	if scope != "" {
		query += " and scope = $2"
	}
	query += " order by updated_at desc, name"
	var rows Rows
	var err error
	if scope == "" {
		rows, err = store.db.Query(ctx, query, userID.String())
	} else {
		rows, err = store.db.Query(ctx, query, userID.String(), scope)
	}
	if err != nil {
		return httpserver.SavedQueueViewsListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list saved queue views failed")}
	}
	defer rows.Close()

	views := make([]httpserver.SavedQueueView, 0)
	for rows.Next() {
		var view httpserver.SavedQueueView
		var rawUserID string
		if err := rows.Scan(&view.ID, &rawUserID, &view.Scope, &view.Name, &view.Query, &view.StateFilter, &view.TypeFilter, &view.Sort); err != nil {
			return httpserver.SavedQueueViewsListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan saved queue view failed")}
		}
		parsed := core.ParseUserID(rawUserID)
		created, matched := parsed.(core.UserIDCreated)
		if !matched {
			return httpserver.SavedQueueViewsListRejected{Reason: parsed.(core.UserIDRejected).Reason}
		}
		view.UserID = created.Value
		views = append(views, view)
	}
	if err := rows.Err(); err != nil {
		return httpserver.SavedQueueViewsListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read saved queue views failed")}
	}
	return httpserver.SavedQueueViewsListed{Values: views}
}

func (store SavedQueueViewStore) Upsert(ctx context.Context, view httpserver.SavedQueueView) httpserver.SavedQueueViewMutationResult {
	viewIDResult := id.New()
	viewID, viewIDMatched := viewIDResult.(id.IDCreated)
	if !viewIDMatched {
		return httpserver.SavedQueueViewSaveRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidID, viewIDResult.(id.IDRejected).Description)}
	}
	row := store.db.QueryRow(ctx, `
		insert into saved_queue_views (id, user_id, scope, name, query_text, state_filter, type_filter, sort_order)
		values ($1, $2, $3, $4, $5, $6, $7, $8)
		on conflict (user_id, scope, name) do update
		set query_text = excluded.query_text,
			state_filter = excluded.state_filter,
			type_filter = excluded.type_filter,
			sort_order = excluded.sort_order,
			updated_at = now()
		returning id::text
	`, viewID.Value.String(), view.UserID.String(), view.Scope, view.Name, view.Query, view.StateFilter, view.TypeFilter, view.Sort)
	if err := row.Scan(&view.ID); err != nil {
		return httpserver.SavedQueueViewSaveRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "save queue view failed")}
	}
	return httpserver.SavedQueueViewSaved{Value: view}
}
