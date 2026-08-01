package mcp

import (
	"encoding/json"

	"github.com/e6qu/sharecrop/internal/agent"
)

const (
	toolListTasks           = "sharecrop.list_tasks"
	toolGetTask             = "sharecrop.get_task"
	toolGetTaskSchema       = "sharecrop.get_task_schema"
	toolCreateTask          = "sharecrop.create_task"
	toolOpenTask            = "sharecrop.open_task"
	toolFundTask            = "sharecrop.fund_task"
	toolSubmitResponse      = "sharecrop.submit_response"
	toolGetSubmissionStatus = "sharecrop.get_submission_status"
	toolGetSubmission       = "sharecrop.get_submission"
	toolListTaskSubmissions = "sharecrop.list_task_submissions"
	toolAcceptSubmission    = "sharecrop.accept_submission"
	toolRequestChanges      = "sharecrop.request_submission_changes"
	toolRejectSubmission    = "sharecrop.reject_submission"
	toolListTaskSeries      = "sharecrop.list_task_series"
	toolGetTaskSeries       = "sharecrop.get_task_series"
	toolCreateSeries        = "sharecrop.create_series"
	toolAddTaskToSeries     = "sharecrop.add_task_to_series"
	toolRemoveSeriesTask    = "sharecrop.remove_task_from_series"
	toolPublishSeries       = "sharecrop.publish_series"
	toolUnpublishSeries     = "sharecrop.unpublish_series"
	toolCloseSeries         = "sharecrop.close_series"
	toolReopenSeries        = "sharecrop.reopen_series"
	toolAddSeriesComment    = "sharecrop.add_series_comment"
	toolListSeriesComments  = "sharecrop.list_series_comments"
	toolAddTaskComment      = "sharecrop.add_task_comment"
	toolListTaskComments    = "sharecrop.list_task_comments"

	toolAddSubmissionComment   = "sharecrop.add_submission_comment"
	toolListSubmissionComments = "sharecrop.list_submission_comments"
	toolUnpublishTask          = "sharecrop.unpublish_task"
	toolReserveTask            = "sharecrop.reserve_task"
	toolListReservations       = "sharecrop.list_task_reservations"
	toolCancelReservation      = "sharecrop.cancel_task_reservation"

	toolCancelTask    = "sharecrop.cancel_task"
	toolRefundTask    = "sharecrop.refund_task"
	toolUpdateSeries  = "sharecrop.update_series"
	toolReorderSeries = "sharecrop.reorder_series"
)

type toolDefinition struct {
	Name        string
	Description string
	Scope       agent.Scope
	InputSchema json.RawMessage
}

func toolDefinitions() []toolDefinition {
	definitions := coreToolDefinitions()
	definitions = append(definitions, orgToolDefinitions()...)
	definitions = append(definitions, collectiblesToolDefinitions()...)
	definitions = append(definitions, notificationsToolDefinitions()...)
	definitions = append(definitions, eventsToolDefinitions()...)
	definitions = append(definitions, creditsToolDefinitions()...)
	definitions = append(definitions, webhooksToolDefinitions()...)
	definitions = append(definitions, usersToolDefinitions()...)
	definitions = append(definitions, adminToolDefinitions()...)
	return definitions
}

func coreToolDefinitions() []toolDefinition {
	return []toolDefinition{
		{
			Name:        toolListTasks,
			Description: "List tasks the agent is permitted to see. Scope is \"public\" (open tasks to work) or \"user\" (the agent's own tasks). Optional filters mirror the REST task listing: repeated states, participation_policy, query (title/id search), task_type, created_after (RFC3339; only tasks created strictly after it), funded (reward_funded, reward_unfunded, or no_credit_reward), sort, and limit/offset paging. state is a deprecated single-state alias for states. Rows carry creator_display_name, holder_display_name (the active reservation holder, when user-assigned), funded, and pending_review_count (submissions awaiting review, populated only on the caller's own tasks). The result carries next_offset (0 on the last page) and total (rows matching the filter, ignoring paging).",
			Scope:       agent.ScopeTasksRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"scope":{"type":"string","enum":["public","user"]},"state":{"type":"string","enum":["draft","open","closed","cancelled","expired"]},"states":{"type":"array","items":{"type":"string","enum":["draft","open","closed","cancelled","expired"]}},"participation_policy":{"type":"string","enum":["open","reservation_required"]},"query":{"type":"string"},"task_type":{"type":"string","enum":["general","code_review","security_review","product_review","ui_ux_review","qa_testing"]},"created_after":{"type":"string"},"funded":{"type":"string","enum":["reward_funded","reward_unfunded","no_credit_reward"]},"sort":{"type":"string","enum":["newest","oldest","title_asc","title_desc","reward_desc","reward_asc"]},"limit":{"type":"integer","minimum":1},"offset":{"type":"integer","minimum":0}},"required":["scope"]}`),
		},
		{
			Name:        toolGetTask,
			Description: "Get a single task the agent is permitted to see, including its response schema and payload.",
			Scope:       agent.ScopeTasksRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string"}},"required":["task_id"]}`),
		},
		{
			Name:        toolGetTaskSchema,
			Description: "Get the response schema JSON for a task.",
			Scope:       agent.ScopeTasksRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string"}},"required":["task_id"]}`),
		},
		{
			Name:        toolCreateTask,
			Description: "Create a task in draft state, with full REST parity. owner (default: the agent's user) may be a user, team, organization, or organization_team. visibility_kind is REQUIRED and states who can see the task: \"public\" appears in the shared marketplace for anyone to find and work; \"user\" is private to you and the named visibility_user_id (default: your own user, so nobody else can see it); \"team\", \"organization\", and \"organization_team\" limit it to the group named by the matching visibility_* id. Reward kind is \"none\", \"credit\", \"collectible\" (requires reward_collectible_ids, escrowed at creation), or \"bundle\" (credit amount plus optional reward_collectible_ids). participation_policy (default \"open\"), assignee_scope (default \"user\"), and reservation_expiry_hours (1-720, default 48) control how workers claim it. series_id/series_position place the task in an existing series. payload_json attaches machine-readable task data; attachments carry files as data URLs; expires_at (RFC3339, in the future) makes an open task expire automatically. After creating, call fund_task (for credit or bundle rewards) and then open_task so others can pick it up.",
			Scope:       agent.ScopeTasksWrite,
			InputSchema: json.RawMessage(`{"type":"object","additionalProperties":false,"properties":{"title":{"type":"string"},"description":{"type":"string"},"response_schema_json":{"type":"string"},"owner":{"type":"object","additionalProperties":false,"properties":{"kind":{"type":"string","enum":["user","team","organization","organization_team"]},"user_id":{"type":"string"},"team_id":{"type":"string"},"organization_id":{"type":"string"}}},"visibility_kind":{"type":"string","enum":["public","user","team","organization","organization_team"]},"visibility_user_id":{"type":"string"},"visibility_team_id":{"type":"string"},"visibility_organization_id":{"type":"string"},"reward_kind":{"type":"string","enum":["none","credit","collectible","bundle"]},"reward_credit_amount":{"type":"integer","minimum":1},"reward_collectible_ids":{"type":"array","items":{"type":"string"}},"participation_policy":{"type":"string","enum":["open","reservation_required"]},"assignee_scope":{"type":"string","enum":["user","team","organization_team"]},"reservation_expiry_hours":{"type":"integer","minimum":1,"maximum":720},"task_type":{"type":"string","enum":["general","code_review","security_review","product_review","ui_ux_review","qa_testing"]},"reference_url":{"type":"string"},"series_id":{"type":"string"},"series_position":{"type":"integer","minimum":1},"payload_json":{"type":"string"},"attachments":{"type":"array","items":{"type":"object","additionalProperties":false,"properties":{"name":{"type":"string"},"content_type":{"type":"string"},"data_url":{"type":"string"}},"required":["name","content_type","data_url"]}},"expires_at":{"type":"string"}},"required":["title","description","response_schema_json","reward_kind","visibility_kind"]}`),
		},
		{
			Name:        toolOpenTask,
			Description: "Open a draft task the agent's user owns so it becomes discoverable and workable. A credit or bundle reward task must be funded first.",
			Scope:       agent.ScopeTasksWrite,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string"}},"required":["task_id"]}`),
		},
		{
			Name:        toolFundTask,
			Description: "Escrow credits from the agent's user onto a task they own, so an accepted submission can be paid. amount is in credit base units; idempotency_key makes a retried fund safe.",
			Scope:       agent.ScopeTasksWrite,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string"},"amount":{"type":"integer","minimum":1},"idempotency_key":{"type":"string"}},"required":["task_id","amount","idempotency_key"]}`),
		},
		{
			Name:        toolSubmitResponse,
			Description: "Submit a response to a task as the agent's user. response_json must match the task's response schema; a schema-invalid response is stored with state \"invalid\" and the validation_errors array (path + message) is returned so the errors can be fixed and the response resubmitted immediately (an active reservation is kept). Optional attachments carry files as data URLs, mirroring the REST submission body.",
			Scope:       agent.ScopeSubmissionsWrite,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string"},"response_json":{"type":"string"},"attachments":{"type":"array","items":{"type":"object","additionalProperties":false,"properties":{"name":{"type":"string"},"content_type":{"type":"string"},"data_url":{"type":"string"}},"required":["name","content_type","data_url"]}}},"required":["task_id","response_json"]}`),
		},
		{
			Name:        toolGetSubmissionStatus,
			Description: "Get the redacted status of a submission by its receipt token.",
			Scope:       agent.ScopeSubmissionsRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"receipt_token":{"type":"string"}},"required":["receipt_token"]}`),
		},
		{
			Name:        toolGetSubmission,
			Description: "Read one submission's full content for review: response_json, attachments, validation_errors, state, review_note, submitter_id, submitter_display_name, and created_at. Available to the submission's submitter and to the task's owner/reviewer (the same authorization as accept_submission); an organization credential reads submissions on its own organization's tasks.",
			Scope:       agent.ScopeSubmissionsRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"submission_id":{"type":"string"}},"required":["submission_id"]}`),
		},
		{
			Name:        toolListTaskSubmissions,
			Description: "List submissions for a task the caller reviews: a personal agent credential lists tasks its user owns, and an organization credential lists its own organization's tasks. Rows are summaries (id, task_id, submitter_id, submitter_display_name, state, created_at); call get_submission for a submission's full content.",
			Scope:       agent.ScopeSubmissionsRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string"},"limit":{"type":"integer","minimum":1},"offset":{"type":"integer","minimum":0}},"required":["task_id"]}`),
		},
		{
			Name:        toolAcceptSubmission,
			Description: "Accept a submission for a task the caller reviews, paying the task's allocated reward when present. Optional payout_amount pays part of the allocated credits, optional tip_amount pays extra credits from the requester balance, and optional tip_collectible_id gifts one of the requester's collectibles as a tip. An organization credential accepts on its own organization's tasks, but cannot pass the tip fields (tips move personal value).",
			Scope:       agent.ScopeSubmissionsReview,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string"},"submission_id":{"type":"string"},"idempotency_key":{"type":"string"},"payout_amount":{"type":"integer","minimum":1},"tip_amount":{"type":"integer","minimum":1},"tip_collectible_id":{"type":"string"}},"required":["task_id","submission_id","idempotency_key"]}`),
		},
		{
			Name:        toolRequestChanges,
			Description: "Request changes for submitted work, keeping the task reserved for the same implementor. An organization credential requests changes on its own organization's tasks.",
			Scope:       agent.ScopeSubmissionsReview,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string"},"submission_id":{"type":"string"},"idempotency_key":{"type":"string"},"review_note":{"type":"string"}},"required":["task_id","submission_id","idempotency_key","review_note"]}`),
		},
		{
			Name:        toolRejectSubmission,
			Description: "Reject submitted work with required notes. Optional partial_credit_amount pays part of the task's allocated credits, optional tip_amount pays extra credits from requester balance, and ban_selection=ban_implementor prevents the worker from doing the same task again. An organization credential rejects on its own organization's tasks, but cannot pass the tip or ban fields (those act as a person).",
			Scope:       agent.ScopeSubmissionsReview,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string"},"submission_id":{"type":"string"},"idempotency_key":{"type":"string"},"review_note":{"type":"string"},"partial_credit_amount":{"type":"integer","minimum":1},"tip_amount":{"type":"integer","minimum":1},"ban_selection":{"type":"string","enum":["none","ban_implementor"]}},"required":["task_id","submission_id","idempotency_key","review_note"]}`),
		},
		{
			Name:        toolListTaskSeries,
			Description: "List the task series the agent's user owns. Optional limit/offset page the listing.",
			Scope:       agent.ScopeTasksRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"limit":{"type":"integer","minimum":1},"offset":{"type":"integer","minimum":0}}}`),
		},
		{
			Name:        toolGetTaskSeries,
			Description: "Get a task series and its ordered tasks.",
			Scope:       agent.ScopeTasksRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"series_id":{"type":"string"}},"required":["series_id"]}`),
		},
		{
			Name:        toolCreateSeries,
			Description: "Create a draft task series the agent's user owns. Add tasks to it as work proceeds, then publish it. Returns the series and its tasks.",
			Scope:       agent.ScopeTasksWrite,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"title":{"type":"string"},"description":{"type":"string"}},"required":["title"]}`),
		},
		{
			Name:        toolAddTaskToSeries,
			Description: "Append a task the agent's user created to the end of one of its series. Useful for multi-round feedback cycles where new tasks are added over time.",
			Scope:       agent.ScopeTasksWrite,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"series_id":{"type":"string"},"task_id":{"type":"string"}},"required":["series_id","task_id"]}`),
		},
		{
			Name:        toolRemoveSeriesTask,
			Description: "Remove a task from one of the agent's series (the task itself is not deleted).",
			Scope:       agent.ScopeTasksWrite,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"series_id":{"type":"string"},"task_id":{"type":"string"}},"required":["series_id","task_id"]}`),
		},
		{
			Name:        toolPublishSeries,
			Description: "Publish a draft series so it becomes visible to others.",
			Scope:       agent.ScopeTasksWrite,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"series_id":{"type":"string"}},"required":["series_id"]}`),
		},
		{
			Name:        toolUnpublishSeries,
			Description: "Move a published series back to draft so it is private again.",
			Scope:       agent.ScopeTasksWrite,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"series_id":{"type":"string"}},"required":["series_id"]}`),
		},
		{
			Name:        toolCloseSeries,
			Description: "Close a series, retiring it.",
			Scope:       agent.ScopeTasksWrite,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"series_id":{"type":"string"}},"required":["series_id"]}`),
		},
		{
			Name:        toolReopenSeries,
			Description: "Reopen a closed series back to draft.",
			Scope:       agent.ScopeTasksWrite,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"series_id":{"type":"string"}},"required":["series_id"]}`),
		},
		{
			Name:        toolAddSeriesComment,
			Description: "Post a comment on a series the agent can view, for clarifying questions and feedback between rounds.",
			Scope:       agent.ScopeTasksWrite,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"series_id":{"type":"string"},"body":{"type":"string"}},"required":["series_id","body"]}`),
		},
		{
			Name:        toolListSeriesComments,
			Description: "List the comment thread on a series the agent can view.",
			Scope:       agent.ScopeTasksRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"series_id":{"type":"string"},"limit":{"type":"integer","minimum":1},"offset":{"type":"integer","minimum":0}},"required":["series_id"]}`),
		},
		{
			Name:        toolAddTaskComment,
			Description: "Post a comment on a task the agent can view, for clarifying questions on a detailed task such as a code review.",
			Scope:       agent.ScopeTasksWrite,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string"},"body":{"type":"string"}},"required":["task_id","body"]}`),
		},
		{
			Name:        toolListTaskComments,
			Description: "List the comment thread on a task the agent can view.",
			Scope:       agent.ScopeTasksRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string"},"limit":{"type":"integer","minimum":1},"offset":{"type":"integer","minimum":0}},"required":["task_id"]}`),
		},
		{
			Name:        toolAddSubmissionComment,
			Description: "Post a comment on a submission the agent authored or owns the task for, to discuss the submission while it is under review.",
			Scope:       agent.ScopeSubmissionsWrite,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"submission_id":{"type":"string"},"body":{"type":"string"}},"required":["submission_id","body"]}`),
		},
		{
			Name:        toolListSubmissionComments,
			Description: "List the comment thread on a submission the agent authored or owns the task for.",
			Scope:       agent.ScopeSubmissionsRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"submission_id":{"type":"string"},"limit":{"type":"integer","minimum":1},"offset":{"type":"integer","minimum":0}},"required":["submission_id"]}`),
		},
		{
			Name:        toolUnpublishTask,
			Description: "Move an open task the agent's user owns back to draft so it is no longer discoverable or workable.",
			Scope:       agent.ScopeTasksWrite,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string"}},"required":["task_id"]}`),
		},
		{
			Name:        toolReserveTask,
			Description: "Reserve a task; the reservation is active immediately (there is no approval step). Omit assignee_kind to reserve as the agent's user, or pass organization_team with organization_id and team_id to reserve for an organization team the user belongs to.",
			Scope:       agent.ScopeSubmissionsWrite,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string"},"assignee_kind":{"type":"string","enum":["user","organization_team"]},"organization_id":{"type":"string"},"team_id":{"type":"string"}},"required":["task_id"]}`),
		},
		{
			Name:        toolListReservations,
			Description: "List a task's reservations for its owner. Rows carry holder_display_name (the requesting user's display name).",
			Scope:       agent.ScopeSubmissionsRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string"},"limit":{"type":"integer","minimum":1},"offset":{"type":"integer","minimum":0}},"required":["task_id"]}`),
		},
		{
			Name:        toolCancelReservation,
			Description: "Cancel an active reservation for a task owned by the agent's user.",
			Scope:       agent.ScopeSubmissionsReview,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string"},"reservation_id":{"type":"string"}},"required":["task_id","reservation_id"]}`),
		},
		{
			Name:        toolCancelTask,
			Description: "Cancel a task the agent's user or organization owns, ending it without publishing further.",
			Scope:       agent.ScopeTasksWrite,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string"}},"required":["task_id"]}`),
		},
		{
			Name:        toolRefundTask,
			Description: "Refund a task's allocated credits back to the agent's user. idempotency_key makes a retried refund safe.",
			Scope:       agent.ScopeTasksWrite,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string"},"idempotency_key":{"type":"string"}},"required":["task_id","idempotency_key"]}`),
		},
		{
			Name:        toolUpdateSeries,
			Description: "Update a task series' title and description. Only the series' creator may do this.",
			Scope:       agent.ScopeTasksWrite,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"series_id":{"type":"string"},"title":{"type":"string"},"description":{"type":"string"}},"required":["series_id","title","description"]}`),
		},
		{
			Name:        toolReorderSeries,
			Description: "Reorder the tasks within a series. task_ids must list every task currently in the series, in the desired order.",
			Scope:       agent.ScopeTasksWrite,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"series_id":{"type":"string"},"task_ids":{"type":"array","items":{"type":"string"}}},"required":["series_id","task_ids"]}`),
		},
	}
}

type toolListEntry struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

type toolListResult struct {
	Tools []toolListEntry `json:"tools"`
}

// ToolNames returns the registered tool names for documentation and tests.
func ToolNames() []string {
	definitions := toolDefinitions()
	names := make([]string, 0, len(definitions))
	for index := range definitions {
		names = append(names, definitions[index].Name)
	}
	return names
}
