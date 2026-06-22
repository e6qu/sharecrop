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
	toolSubmitResponse      = "sharecrop.submit_response"
	toolGetSubmissionStatus = "sharecrop.get_submission_status"
	toolListTaskSubmissions = "sharecrop.list_task_submissions"
	toolAcceptSubmission    = "sharecrop.accept_submission"
	toolRequestChanges      = "sharecrop.request_submission_changes"
	toolRejectSubmission    = "sharecrop.reject_submission"
	toolListTaskSeries      = "sharecrop.list_task_series"
	toolGetTaskSeries       = "sharecrop.get_task_series"
)

type toolDefinition struct {
	Name        string
	Description string
	Scope       agent.Scope
	InputSchema json.RawMessage
}

func toolDefinitions() []toolDefinition {
	return []toolDefinition{
		{
			Name:        toolListTasks,
			Description: "List tasks the agent is permitted to see. Scope is \"public\" or \"user\".",
			Scope:       agent.ScopeTasksRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"scope":{"type":"string","enum":["public","user"]}},"required":["scope"]}`),
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
			Description: "Create a user-owned task. Visibility is \"user\" or \"public\". Reward kind is \"none\", \"credit\", \"collectible\", or \"bundle\".",
			Scope:       agent.ScopeTasksWrite,
			InputSchema: json.RawMessage(`{"type":"object","additionalProperties":false,"properties":{"title":{"type":"string"},"description":{"type":"string"},"response_schema_json":{"type":"string"},"visibility":{"type":"string","enum":["user","public"]},"reward_kind":{"type":"string","enum":["none","credit","collectible","bundle"]},"reward_credit_amount":{"type":"integer","minimum":1}},"required":["title","description","response_schema_json","visibility","reward_kind"]}`),
		},
		{
			Name:        toolSubmitResponse,
			Description: "Submit a response to a task as the agent's user.",
			Scope:       agent.ScopeSubmissionsWrite,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string"},"response_json":{"type":"string"}},"required":["task_id","response_json"]}`),
		},
		{
			Name:        toolGetSubmissionStatus,
			Description: "Get the redacted status of a submission by its receipt token.",
			Scope:       agent.ScopeSubmissionsRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"receipt_token":{"type":"string"}},"required":["receipt_token"]}`),
		},
		{
			Name:        toolListTaskSubmissions,
			Description: "List submissions for a task owned by the agent's user.",
			Scope:       agent.ScopeSubmissionsRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string"}},"required":["task_id"]}`),
		},
		{
			Name:        toolAcceptSubmission,
			Description: "Accept a submission for a task owned by the agent's user, paying the escrowed reward when present. Optional payout_amount pays part of the credit escrow, and optional tip_amount pays extra credits from the requester balance.",
			Scope:       agent.ScopeSubmissionsReview,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string"},"submission_id":{"type":"string"},"idempotency_key":{"type":"string"},"payout_amount":{"type":"integer","minimum":1},"tip_amount":{"type":"integer","minimum":1}},"required":["task_id","submission_id","idempotency_key"]}`),
		},
		{
			Name:        toolRequestChanges,
			Description: "Request changes for submitted work, keeping the task reserved for the same implementor.",
			Scope:       agent.ScopeSubmissionsReview,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string"},"submission_id":{"type":"string"},"review_note":{"type":"string"}},"required":["task_id","submission_id","review_note"]}`),
		},
		{
			Name:        toolRejectSubmission,
			Description: "Reject submitted work with required notes. Optional partial_credit_amount pays part of held credit escrow, optional tip_amount pays extra credits from requester balance, and ban_implementor prevents the worker from doing the same task again.",
			Scope:       agent.ScopeSubmissionsReview,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string"},"submission_id":{"type":"string"},"idempotency_key":{"type":"string"},"review_note":{"type":"string"},"partial_credit_amount":{"type":"integer","minimum":1},"tip_amount":{"type":"integer","minimum":1},"ban_implementor":{"type":"boolean"}},"required":["task_id","submission_id","idempotency_key","review_note"]}`),
		},
		{
			Name:        toolListTaskSeries,
			Description: "List the task series the agent's user owns.",
			Scope:       agent.ScopeTasksRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		},
		{
			Name:        toolGetTaskSeries,
			Description: "Get a task series and its ordered tasks.",
			Scope:       agent.ScopeTasksRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"series_id":{"type":"string"}},"required":["series_id"]}`),
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
