package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/mcp"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
)

// mcpServices adapts the HTTP server's domain services to the MCP tool surface.
type mcpServices struct {
	taskService       TaskService
	submissionService SubmissionService
	ledgerService     LedgerService
}

func (services mcpServices) ListTasks(ctx context.Context, subject auth.UserSubject, scope task.ListScope, filters task.ListFilters) task.ListResult {
	return services.taskService.List(ctx, subject, scope, filters, core.DefaultPage())
}

func (services mcpServices) GetTask(ctx context.Context, subject auth.UserSubject, taskID core.TaskID) task.GetResult {
	return services.taskService.Get(ctx, subject, taskID)
}

func (services mcpServices) CreateTask(ctx context.Context, command task.CreateCommand) task.CreateResult {
	return services.taskService.Create(ctx, command)
}

func (services mcpServices) OpenTask(ctx context.Context, subject auth.UserSubject, taskID core.TaskID) task.ChangeStateResult {
	return services.taskService.Open(ctx, subject, taskID)
}

func (services mcpServices) FundTask(ctx context.Context, funder core.UserID, taskID core.TaskID, amount ledger.CreditAmount, key ledger.IdempotencyKey) ledger.FundResult {
	return services.ledgerService.FundTask(ctx, funder, taskID, amount, key)
}

func (services mcpServices) SubmitResponse(ctx context.Context, command submission.SubmitCommand) submission.SubmitResult {
	return services.submissionService.Submit(ctx, command)
}

func (services mcpServices) GetSubmissionStatus(ctx context.Context, token submission.ReceiptTokenPlain) submission.ReceiptStatusResult {
	return services.submissionService.FindByReceipt(ctx, token)
}

func (services mcpServices) ListTaskSubmissions(ctx context.Context, subject auth.UserSubject, taskID core.TaskID) submission.ListResult {
	return services.submissionService.ListForTask(ctx, subject, taskID, core.DefaultPage())
}

func (services mcpServices) AcceptSubmission(ctx context.Context, requester core.UserID, taskID core.TaskID, submissionID core.SubmissionID, key ledger.IdempotencyKey) ledger.AcceptResult {
	return services.ledgerService.AcceptSubmission(ctx, requester, taskID, submissionID, key)
}

func (services mcpServices) ReviewAcceptSubmission(ctx context.Context, requester core.UserID, taskID core.TaskID, submissionID core.SubmissionID, key ledger.IdempotencyKey, creditSelection ledger.CreditReviewSelection, tipSelection ledger.TipSelection) ledger.AcceptResult {
	return services.ledgerService.ReviewAcceptSubmission(ctx, requester, taskID, submissionID, key, creditSelection, tipSelection)
}

func (services mcpServices) RequestChanges(ctx context.Context, requester core.UserID, taskID core.TaskID, submissionID core.SubmissionID, note submission.ReviewNote) ledger.RequestChangesResult {
	return services.ledgerService.RequestChanges(ctx, requester, taskID, submissionID, note)
}

func (services mcpServices) RejectSubmission(ctx context.Context, requester core.UserID, taskID core.TaskID, submissionID core.SubmissionID, key ledger.IdempotencyKey, note submission.ReviewNote, creditSelection ledger.CreditReviewSelection, tipSelection ledger.TipSelection, banSelection ledger.BanSelection) ledger.RejectResult {
	return services.ledgerService.RejectSubmission(ctx, requester, taskID, submissionID, key, note, creditSelection, tipSelection, banSelection)
}

func (services mcpServices) ListSeries(ctx context.Context, subject auth.UserSubject) task.ListSeriesResult {
	return services.taskService.ListSeries(ctx, subject, core.DefaultPage())
}

func (services mcpServices) GetSeries(ctx context.Context, subject auth.UserSubject, seriesID core.TaskSeriesID) task.GetSeriesResult {
	return services.taskService.GetSeries(ctx, subject, seriesID)
}

func (services mcpServices) CreateSeries(ctx context.Context, subject auth.UserSubject, title task.SeriesTitle, description task.SeriesDescription) task.SeriesMutationResult {
	return services.taskService.CreateSeries(ctx, subject, title, description)
}

func (services mcpServices) ChangeSeriesState(ctx context.Context, subject auth.UserSubject, seriesID core.TaskSeriesID, transition task.SeriesStateTransition) task.SeriesMutationResult {
	return services.taskService.ChangeSeriesState(ctx, subject, seriesID, transition)
}

func (services mcpServices) AddTaskToSeries(ctx context.Context, subject auth.UserSubject, seriesID core.TaskSeriesID, taskID core.TaskID) task.SeriesMutationResult {
	return services.taskService.AddTaskToSeries(ctx, subject, seriesID, taskID)
}

func (services mcpServices) RemoveTaskFromSeries(ctx context.Context, subject auth.UserSubject, seriesID core.TaskSeriesID, taskID core.TaskID) task.SeriesMutationResult {
	return services.taskService.RemoveTaskFromSeries(ctx, subject, seriesID, taskID)
}

func (services mcpServices) AddSeriesComment(ctx context.Context, subject auth.UserSubject, seriesID core.TaskSeriesID, body task.CommentBody) task.SeriesCommentResult {
	return services.taskService.AddSeriesComment(ctx, subject, seriesID, body)
}

func (services mcpServices) ListSeriesComments(ctx context.Context, subject auth.UserSubject, seriesID core.TaskSeriesID) task.SeriesCommentsResult {
	return services.taskService.ListSeriesComments(ctx, subject, seriesID)
}

func (services mcpServices) UnpublishTask(ctx context.Context, subject auth.UserSubject, taskID core.TaskID) task.ChangeStateResult {
	return services.taskService.Unpublish(ctx, subject, taskID)
}

func (services mcpServices) ReserveTask(ctx context.Context, subject auth.UserSubject, taskID core.TaskID) task.ReservationResult {
	return services.taskService.Reserve(ctx, subject, taskID)
}

func (services mcpServices) ListReservations(ctx context.Context, subject auth.UserSubject, taskID core.TaskID) task.ReservationsListResult {
	return services.taskService.ListReservations(ctx, subject, taskID)
}

func (services mcpServices) ApproveReservation(ctx context.Context, subject auth.UserSubject, taskID core.TaskID, reservationID core.TaskReservationID) task.ReservationStateChangeResult {
	return services.taskService.ApproveReservation(ctx, subject, taskID, reservationID)
}

func (services mcpServices) DeclineReservation(ctx context.Context, subject auth.UserSubject, taskID core.TaskID, reservationID core.TaskReservationID) task.ReservationStateChangeResult {
	return services.taskService.DeclineReservation(ctx, subject, taskID, reservationID)
}

func (services mcpServices) CancelReservation(ctx context.Context, subject auth.UserSubject, taskID core.TaskID, reservationID core.TaskReservationID) task.ReservationStateChangeResult {
	return services.taskService.CancelReservation(ctx, subject, taskID, reservationID)
}

type agentCredentialRequest struct {
	Label  string   `json:"label"`
	Scopes []string `json:"scopes"`
}

type agentCredentialResponse struct {
	ID     string   `json:"id"`
	Label  string   `json:"label"`
	Scopes []string `json:"scopes"`
	State  string   `json:"state"`
}

type agentCredentialCreatedResponse struct {
	Credential agentCredentialResponse `json:"credential"`
	Secret     string                  `json:"secret"`
}

type agentCredentialsResponse struct {
	Credentials []agentCredentialResponse `json:"credentials"`
}

func (agentCredentialResponse) writableResponse() {}

func (agentCredentialCreatedResponse) writableResponse() {}

func (agentCredentialsResponse) writableResponse() {}

func (server Server) getTask(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}

	taskIDResult := parseTaskPathValue(r)
	taskIDAccepted, taskIDMatched := taskIDResult.(taskIDAccepted)
	if !taskIDMatched {
		writeError(w, http.StatusBadRequest, taskIDResult.(taskIDRejected).reason)
		return
	}

	result := server.taskService.Get(r.Context(), actor.subject, taskIDAccepted.value)
	got, matched := result.(task.TaskGot)
	if !matched {
		writeError(w, http.StatusForbidden, result.(task.GetRejected).Reason.Description())
		return
	}

	writeTaskResponse(w, http.StatusOK, taskToResponse(got.Value))
}

func (server Server) createAgentCredential(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}

	var request agentCredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}

	labelResult := agent.NewLabel(request.Label)
	label, labelMatched := labelResult.(agent.LabelAccepted)
	if !labelMatched {
		writeError(w, http.StatusBadRequest, labelResult.(agent.LabelRejected).Reason.Description())
		return
	}

	scopesResult := parseAgentScopes(request.Scopes)
	scopes, scopesMatched := scopesResult.(agentScopesAccepted)
	if !scopesMatched {
		writeError(w, http.StatusBadRequest, scopesResult.(agentScopesRejected).reason)
		return
	}

	result := server.agentService.Create(r.Context(), actor.subject.ID, label.Value, scopes.value)
	created, matched := result.(agent.CredentialCreated)
	if !matched {
		writeError(w, http.StatusBadRequest, result.(agent.CreateRejected).Reason.Description())
		return
	}

	writeJSON(w, http.StatusCreated, agentCredentialCreatedResponse{
		Credential: credentialToResponse(created.Value),
		Secret:     created.Secret.String(),
	})
}

func (server Server) listAgentCredentials(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}

	result := server.agentService.List(r.Context(), actor.subject.ID, parsePage(r))
	listed, matched := result.(agent.CredentialsListed)
	if !matched {
		writeError(w, http.StatusBadRequest, result.(agent.ListRejected).Reason.Description())
		return
	}

	response := agentCredentialsResponse{Credentials: make([]agentCredentialResponse, 0, len(listed.Values))}
	for index := range listed.Values {
		response.Credentials = append(response.Credentials, credentialToResponse(listed.Values[index]))
	}
	writeJSON(w, http.StatusOK, response)
}

func (server Server) revokeAgentCredential(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}

	credentialIDResult := core.ParseAgentCredentialID(r.PathValue("credential_id"))
	credentialID, credentialMatched := credentialIDResult.(core.AgentCredentialIDCreated)
	if !credentialMatched {
		writeError(w, http.StatusBadRequest, credentialIDResult.(core.AgentCredentialIDRejected).Reason.Description())
		return
	}

	result := server.agentService.Revoke(r.Context(), actor.subject.ID, credentialID.Value)
	revoked, matched := result.(agent.CredentialRevoked)
	if !matched {
		writeError(w, http.StatusBadRequest, result.(agent.RevokeRejected).Reason.Description())
		return
	}

	writeJSON(w, http.StatusOK, credentialToResponse(revoked.Value))
}

func (server Server) mcpEndpoint(w http.ResponseWriter, r *http.Request) {
	if !originAllowed(r) {
		writeError(w, http.StatusForbidden, "origin is not allowed")
		return
	}
	if !mcpAcceptAllowed(r.Header.Get("Accept")) {
		writeError(w, http.StatusNotAcceptable, "MCP endpoint requires an Accept header allowing application/json")
		return
	}
	if !mcpProtocolVersionAllowed(r.Header.Get("MCP-Protocol-Version")) {
		writeError(w, http.StatusBadRequest, "MCP protocol version is unsupported")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxMCPBodyBytes))
	if err != nil {
		writeError(w, http.StatusBadRequest, "request body could not be read")
		return
	}
	requestInfo := classifyMCPBody(body)
	if requestInfo.invalid {
		writeError(w, http.StatusBadRequest, "request body is not valid JSON-RPC")
		return
	}

	verifyResult := server.verifyAgent(r)
	verified, verifiedMatched := verifyResult.(agent.CredentialVerified)
	if !verifiedMatched {
		writeError(w, http.StatusUnauthorized, verifyResult.(agent.VerifyRejected).Reason.Description())
		return
	}

	if !server.subjectRateLimiter.allow(verified.Subject.ID.String()) {
		writeError(w, http.StatusTooManyRequests, "too many MCP requests; slow down and retry")
		return
	}

	sessionID := r.Header.Get(mcpSessionHeader)
	if requestInfo.initializes {
		if sessionID != "" {
			writeError(w, http.StatusBadRequest, "initialize requests must not include an MCP session id")
			return
		}
	} else if sessionID == "" {
		writeError(w, http.StatusBadRequest, "MCP session id is required")
		return
	} else if !server.mcpSessions.existsForSubject(sessionID, verified.Subject.ID.String()) {
		writeError(w, http.StatusNotFound, "MCP session was not found")
		return
	}

	result := server.mcpServer.HandleRaw(r.Context(), verified.Subject, verified.Credential.Scopes, body)
	if result.SessionID != "" {
		if !server.mcpSessions.create(result.SessionID, verified.Subject.ID.String()) {
			writeError(w, http.StatusTooManyRequests, "too many active MCP sessions for this agent")
			return
		}
		w.Header().Set(mcpSessionHeader, result.SessionID)
	} else if requestInfo.initializes {
		generatedSessionID := newMCPHTTPSessionID()
		if generatedSessionID == "" {
			writeError(w, http.StatusInternalServerError, "MCP session could not be created")
			return
		}
		if !server.mcpSessions.create(generatedSessionID, verified.Subject.ID.String()) {
			writeError(w, http.StatusTooManyRequests, "too many active MCP sessions for this agent")
			return
		}
		w.Header().Set(mcpSessionHeader, generatedSessionID)
	}
	if !result.HasResponse {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	if sessionID != "" {
		_, _ = server.mcpSessions.appendEvent(sessionID, result.Payload)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(result.Payload)
}

func mcpAcceptAllowed(raw string) bool {
	if raw == "" {
		return true
	}
	for _, value := range strings.Split(raw, ",") {
		mediaType := strings.TrimSpace(strings.Split(value, ";")[0])
		if mediaType == "application/json" || mediaType == "*/*" {
			return true
		}
	}
	return false
}

func mcpProtocolVersionAllowed(raw string) bool {
	return raw == "" || raw == mcp.ProtocolVersion()
}

func (server Server) mcpStream(w http.ResponseWriter, r *http.Request) {
	if !originAllowed(r) {
		writeError(w, http.StatusForbidden, "origin is not allowed")
		return
	}
	if !mcpStreamAcceptAllowed(r.Header.Get("Accept")) {
		writeError(w, http.StatusNotAcceptable, "MCP stream requires an Accept header allowing text/event-stream")
		return
	}
	if !mcpProtocolVersionAllowed(r.Header.Get("MCP-Protocol-Version")) {
		writeError(w, http.StatusBadRequest, "MCP protocol version is unsupported")
		return
	}
	verifyResult := server.verifyAgent(r)
	verified, verifiedMatched := verifyResult.(agent.CredentialVerified)
	if !verifiedMatched {
		writeError(w, http.StatusUnauthorized, verifyResult.(agent.VerifyRejected).Reason.Description())
		return
	}
	sessionID := r.Header.Get(mcpSessionHeader)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "MCP session id is required")
		return
	}
	if !server.mcpSessions.existsForSubject(sessionID, verified.Subject.ID.String()) {
		writeError(w, http.StatusNotFound, "MCP session was not found")
		return
	}
	events, liveEvents, cancel, ok := server.mcpSessions.replayAndSubscribe(sessionID, r.Header.Get(mcpLastEventIDHeader))
	if !ok {
		writeError(w, http.StatusNotFound, "MCP session was not found")
		return
	}
	defer cancel()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	if len(events) == 0 {
		_, _ = w.Write([]byte(": sharecrop mcp stream ready\n\n"))
	}
	for index := range events {
		writeSSEEvent(w, events[index])
	}
	if flusher, matched := w.(http.Flusher); matched {
		flusher.Flush()
	}
	for {
		select {
		case event, open := <-liveEvents:
			if !open {
				return
			}
			writeSSEEvent(w, event)
			if flusher, matched := w.(http.Flusher); matched {
				flusher.Flush()
			}
		case <-r.Context().Done():
			return
		}
	}
}

func (server Server) mcpDeleteSession(w http.ResponseWriter, r *http.Request) {
	if !originAllowed(r) {
		writeError(w, http.StatusForbidden, "origin is not allowed")
		return
	}
	verifyResult := server.verifyAgent(r)
	verified, verifiedMatched := verifyResult.(agent.CredentialVerified)
	if !verifiedMatched {
		writeError(w, http.StatusUnauthorized, verifyResult.(agent.VerifyRejected).Reason.Description())
		return
	}
	sessionID := r.Header.Get(mcpSessionHeader)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "MCP session id is required")
		return
	}
	if !server.mcpSessions.existsForSubject(sessionID, verified.Subject.ID.String()) {
		writeError(w, http.StatusNotFound, "MCP session was not found")
		return
	}
	if !server.mcpSessions.terminate(sessionID) {
		writeError(w, http.StatusNotFound, "MCP session was not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

const maxMCPBodyBytes = 1 << 20

type mcpBodyInfo struct {
	initializes bool
	invalid     bool
}

func classifyMCPBody(body []byte) mcpBodyInfo {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return mcpBodyInfo{invalid: true}
	}
	if trimmed[0] == '[' {
		var requests []mcp.Request
		if err := json.Unmarshal(trimmed, &requests); err != nil {
			return mcpBodyInfo{invalid: true}
		}
		for index := range requests {
			if len(requests[index].ID) > 0 && requests[index].Method != "initialize" && !mcpRawClientResponse(requests[index]) {
				return mcpBodyInfo{}
			}
		}
		for index := range requests {
			if requests[index].Method == "initialize" {
				return mcpBodyInfo{initializes: true}
			}
		}
		return mcpBodyInfo{}
	}
	var request mcp.Request
	if err := json.Unmarshal(trimmed, &request); err != nil {
		return mcpBodyInfo{invalid: true}
	}
	return mcpBodyInfo{initializes: request.Method == "initialize"}
}

func mcpRawClientResponse(request mcp.Request) bool {
	return request.Method == "" && len(request.ID) > 0
}

// originAllowed implements the MCP DNS-rebinding protection: requests without
// an Origin header (non-browser agents) are allowed; a browser Origin must
// match the server's own host.
func originAllowed(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return parsed.Host == r.Host
}

func mcpStreamAcceptAllowed(raw string) bool {
	if raw == "" {
		return false
	}
	for _, value := range strings.Split(raw, ",") {
		mediaType := strings.TrimSpace(strings.Split(value, ";")[0])
		if mediaType == "text/event-stream" || mediaType == "*/*" {
			return true
		}
	}
	return false
}

func (server Server) verifyAgent(r *http.Request) agent.VerifyResult {
	rawHeader := r.Header.Get("Authorization")
	rawToken, matched := strings.CutPrefix(rawHeader, "Bearer ")
	if !matched {
		return agent.VerifyRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "agent credential is required")}
	}
	secretResult := agent.ParseSecretPlain(rawToken)
	secret, secretMatched := secretResult.(agent.SecretPlainAccepted)
	if !secretMatched {
		return agent.VerifyRejected{Reason: secretResult.(agent.SecretPlainRejected).Reason}
	}
	return server.agentService.Verify(r.Context(), secret.Value)
}

func credentialToResponse(value agent.Credential) agentCredentialResponse {
	scopes := value.Scopes.Values()
	rawScopes := make([]string, 0, len(scopes))
	for index := range scopes {
		rawScopes = append(rawScopes, scopes[index].String())
	}
	return agentCredentialResponse{
		ID:     value.ID.String(),
		Label:  value.Label.String(),
		Scopes: rawScopes,
		State:  value.State.String(),
	}
}

type agentScopesResult interface {
	agentScopesResult()
}

type agentScopesAccepted struct {
	value agent.ScopeSet
}

type agentScopesRejected struct {
	reason string
}

func (agentScopesAccepted) agentScopesResult() {}

func (agentScopesRejected) agentScopesResult() {}

func parseAgentScopes(raw []string) agentScopesResult {
	scopes := make([]agent.Scope, 0, len(raw))
	for _, rawScope := range raw {
		scopeResult := agent.ParseScope(rawScope)
		scope, matched := scopeResult.(agent.ScopeAccepted)
		if !matched {
			return agentScopesRejected{reason: scopeResult.(agent.ScopeRejected).Reason.Description()}
		}
		scopes = append(scopes, scope.Value)
	}
	set := agent.NewScopeSet(scopes)
	if set.IsEmpty() {
		return agentScopesRejected{reason: "at least one agent scope is required"}
	}
	return agentScopesAccepted{value: set}
}
