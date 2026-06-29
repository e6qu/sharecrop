package mcp

import (
	"context"
	"encoding/json"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
)

// Services is the set of domain operations the MCP adapter exposes as tools.
type Services interface {
	ListTasks(context.Context, auth.UserSubject, task.ListScope, task.ListFilters) task.ListResult
	GetTask(context.Context, auth.UserSubject, core.TaskID) task.GetResult
	CreateTask(context.Context, task.CreateCommand) task.CreateResult
	OpenTask(context.Context, auth.UserSubject, core.TaskID) task.ChangeStateResult
	FundTask(context.Context, core.UserID, core.TaskID, ledger.CreditAmount, ledger.IdempotencyKey) ledger.FundResult
	SubmitResponse(context.Context, submission.SubmitCommand) submission.SubmitResult
	GetSubmissionStatus(context.Context, submission.ReceiptTokenPlain) submission.ReceiptStatusResult
	ListTaskSubmissions(context.Context, auth.UserSubject, core.TaskID) submission.ListResult
	AcceptSubmission(context.Context, core.UserID, core.TaskID, core.SubmissionID, ledger.IdempotencyKey) ledger.AcceptResult
	ReviewAcceptSubmission(context.Context, core.UserID, core.TaskID, core.SubmissionID, ledger.IdempotencyKey, ledger.CreditReviewSelection, ledger.TipSelection, ledger.CollectibleTipSelection) ledger.AcceptResult
	RequestChanges(context.Context, core.UserID, core.TaskID, core.SubmissionID, submission.ReviewNote) ledger.RequestChangesResult
	RejectSubmission(context.Context, core.UserID, core.TaskID, core.SubmissionID, ledger.IdempotencyKey, submission.ReviewNote, ledger.CreditReviewSelection, ledger.TipSelection, ledger.BanSelection) ledger.RejectResult
	ListSeries(context.Context, auth.UserSubject) task.ListSeriesResult
	GetSeries(context.Context, auth.UserSubject, core.TaskSeriesID) task.GetSeriesResult
	CreateSeries(context.Context, auth.UserSubject, task.SeriesTitle, task.SeriesDescription) task.SeriesMutationResult
	ChangeSeriesState(context.Context, auth.UserSubject, core.TaskSeriesID, task.SeriesStateTransition) task.SeriesMutationResult
	AddTaskToSeries(context.Context, auth.UserSubject, core.TaskSeriesID, core.TaskID) task.SeriesMutationResult
	RemoveTaskFromSeries(context.Context, auth.UserSubject, core.TaskSeriesID, core.TaskID) task.SeriesMutationResult
	AddSeriesComment(context.Context, auth.UserSubject, core.TaskSeriesID, task.CommentBody) task.SeriesCommentResult
	ListSeriesComments(context.Context, auth.UserSubject, core.TaskSeriesID) task.SeriesCommentsResult
	AddTaskComment(context.Context, auth.UserSubject, core.TaskID, task.CommentBody) task.TaskCommentResult
	ListTaskComments(context.Context, auth.UserSubject, core.TaskID) task.TaskCommentsResult
	AddSubmissionComment(context.Context, auth.UserSubject, core.SubmissionID, task.CommentBody) submission.SubmissionCommentResult
	ListSubmissionComments(context.Context, auth.UserSubject, core.SubmissionID) submission.SubmissionCommentsResult
	UnpublishTask(context.Context, auth.UserSubject, core.TaskID) task.ChangeStateResult
	ReserveTask(context.Context, auth.UserSubject, core.TaskID) task.ReservationResult
	ReserveTaskForOrganizationTeam(context.Context, auth.UserSubject, core.TaskID, core.OrganizationID, core.TeamID) task.ReservationResult
	ListReservations(context.Context, auth.UserSubject, core.TaskID) task.ReservationsListResult
	ApproveReservation(context.Context, auth.UserSubject, core.TaskID, core.TaskReservationID) task.ReservationStateChangeResult
	DeclineReservation(context.Context, auth.UserSubject, core.TaskID, core.TaskReservationID) task.ReservationStateChangeResult
	CancelReservation(context.Context, auth.UserSubject, core.TaskID, core.TaskReservationID) task.ReservationStateChangeResult
}

type Server struct {
	services Services
}

func NewServer(services Services) Server {
	return Server{services: services}
}

// Handle dispatches a single JSON-RPC request for an authenticated agent.
func (server Server) Handle(ctx context.Context, subject auth.UserSubject, scopes agent.ScopeSet, request Request) Response {
	if request.JSONRPC != jsonRPCVersion {
		return errorResponse(request.ID, codeInvalidRequest, "jsonrpc version must be 2.0")
	}

	switch request.Method {
	case "initialize":
		return server.handleInitialize(request)
	case "ping":
		return successResponse(request.ID, json.RawMessage(`{}`))
	case "tools/list":
		return server.handleToolsList(request)
	case "tools/call":
		return server.handleToolsCall(ctx, subject, scopes, request)
	default:
		return errorResponse(request.ID, codeMethodNotFound, "unknown method: "+request.Method)
	}
}

func (server Server) handleInitialize(request Request) Response {
	result := initializeResult{
		ProtocolVersion: protocolVersion,
		Capabilities:    capabilities{Tools: toolsCapability{}},
		ServerInfo:      serverInfo{Name: serverName, Version: serverVersion},
	}
	return marshalResult(request.ID, result)
}

func (server Server) handleToolsList(request Request) Response {
	definitions := toolDefinitions()
	entries := make([]toolListEntry, 0, len(definitions))
	for index := range definitions {
		entries = append(entries, toolListEntry{
			Name:        definitions[index].Name,
			Description: definitions[index].Description,
			InputSchema: definitions[index].InputSchema,
		})
	}
	return marshalResult(request.ID, toolListResult{Tools: entries})
}

func (server Server) handleToolsCall(ctx context.Context, subject auth.UserSubject, scopes agent.ScopeSet, request Request) Response {
	var params toolCallParams
	if err := json.Unmarshal(request.Params, &params); err != nil {
		return errorResponse(request.ID, codeInvalidParams, "tools/call params are invalid")
	}

	definition, found := findTool(params.Name)
	if !found {
		return errorResponse(request.ID, codeInvalidParams, "unknown tool: "+params.Name)
	}
	if _, granted := scopes.Allows(definition.Scope).(agent.ScopeGranted); !granted {
		return errorResponse(request.ID, codeScopeDenied, "agent credential is missing the "+definition.Scope.String()+" scope")
	}

	outcome := server.dispatchTool(ctx, subject, definition.Name, params.Arguments)
	switch typed := outcome.(type) {
	case toolSucceeded:
		return marshalResult(request.ID, toolCallResult{Content: []contentItem{{Type: "text", Text: string(typed.payload)}}})
	case toolFailed:
		return marshalResult(request.ID, toolCallResult{Content: []contentItem{{Type: "text", Text: typed.message}}, IsError: true})
	case toolProtocolError:
		return errorResponse(request.ID, typed.code, typed.message)
	default:
		return errorResponse(request.ID, codeInternalError, "tool produced no result")
	}
}

func (server Server) dispatchTool(ctx context.Context, subject auth.UserSubject, name string, arguments json.RawMessage) toolResult {
	switch name {
	case toolListTasks:
		return server.callListTasks(ctx, subject, arguments)
	case toolGetTask:
		return server.callGetTask(ctx, subject, arguments)
	case toolGetTaskSchema:
		return server.callGetTaskSchema(ctx, subject, arguments)
	case toolCreateTask:
		return server.callCreateTask(ctx, subject, arguments)
	case toolOpenTask:
		return server.callOpenTask(ctx, subject, arguments)
	case toolFundTask:
		return server.callFundTask(ctx, subject, arguments)
	case toolSubmitResponse:
		return server.callSubmitResponse(ctx, subject, arguments)
	case toolGetSubmissionStatus:
		return server.callGetSubmissionStatus(ctx, arguments)
	case toolListTaskSubmissions:
		return server.callListTaskSubmissions(ctx, subject, arguments)
	case toolAcceptSubmission:
		return server.callAcceptSubmission(ctx, subject, arguments)
	case toolRequestChanges:
		return server.callRequestChanges(ctx, subject, arguments)
	case toolRejectSubmission:
		return server.callRejectSubmission(ctx, subject, arguments)
	case toolListTaskSeries:
		return server.callListTaskSeries(ctx, subject)
	case toolGetTaskSeries:
		return server.callGetTaskSeries(ctx, subject, arguments)
	case toolCreateSeries:
		return server.callCreateSeries(ctx, subject, arguments)
	case toolAddTaskToSeries:
		return server.callAddTaskToSeries(ctx, subject, arguments)
	case toolRemoveSeriesTask:
		return server.callRemoveTaskFromSeries(ctx, subject, arguments)
	case toolPublishSeries:
		return server.callChangeSeriesState(ctx, subject, arguments, task.PublishSeriesState)
	case toolUnpublishSeries:
		return server.callChangeSeriesState(ctx, subject, arguments, task.UnpublishSeriesState)
	case toolCloseSeries:
		return server.callChangeSeriesState(ctx, subject, arguments, task.CloseSeriesState)
	case toolReopenSeries:
		return server.callChangeSeriesState(ctx, subject, arguments, task.ReopenSeriesState)
	case toolAddSeriesComment:
		return server.callAddSeriesComment(ctx, subject, arguments)
	case toolListSeriesComments:
		return server.callListSeriesComments(ctx, subject, arguments)
	case toolAddTaskComment:
		return server.callAddTaskComment(ctx, subject, arguments)
	case toolListTaskComments:
		return server.callListTaskComments(ctx, subject, arguments)
	case toolAddSubmissionComment:
		return server.callAddSubmissionComment(ctx, subject, arguments)
	case toolListSubmissionComments:
		return server.callListSubmissionComments(ctx, subject, arguments)
	case toolUnpublishTask:
		return server.callUnpublishTask(ctx, subject, arguments)
	case toolReserveTask:
		return server.callReserveTask(ctx, subject, arguments)
	case toolListReservations:
		return server.callListReservations(ctx, subject, arguments)
	case toolApproveReservation:
		return server.callChangeReservation(ctx, subject, arguments, server.services.ApproveReservation)
	case toolDeclineReservation:
		return server.callChangeReservation(ctx, subject, arguments, server.services.DeclineReservation)
	case toolCancelReservation:
		return server.callChangeReservation(ctx, subject, arguments, server.services.CancelReservation)
	default:
		return toolProtocolError{code: codeInvalidParams, message: "unknown tool: " + name}
	}
}

func findTool(name string) (toolDefinition, bool) {
	for _, definition := range toolDefinitions() {
		if definition.Name == name {
			return definition, true
		}
	}
	return toolDefinition{}, false
}

func marshalResult(id json.RawMessage, value resultValue) Response {
	encoded, err := json.Marshal(value)
	if err != nil {
		return errorResponse(id, codeInternalError, "failed to encode result")
	}
	return successResponse(id, encoded)
}

type resultValue interface {
	resultValue()
}

func (initializeResult) resultValue() {}

func (toolListResult) resultValue() {}

func (toolCallResult) resultValue() {}

type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type initializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    capabilities `json:"capabilities"`
	ServerInfo      serverInfo   `json:"serverInfo"`
}

type capabilities struct {
	Tools toolsCapability `json:"tools"`
}

type toolsCapability struct{}

type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type contentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type toolCallResult struct {
	Content []contentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type toolResult interface {
	toolResult()
}

type toolSucceeded struct {
	payload json.RawMessage
}

type toolFailed struct {
	message string
}

type toolProtocolError struct {
	code    int
	message string
}

func (toolSucceeded) toolResult() {}

func (toolFailed) toolResult() {}

func (toolProtocolError) toolResult() {}
