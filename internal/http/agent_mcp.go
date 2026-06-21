package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
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

func (services mcpServices) ListTasks(ctx context.Context, subject auth.UserSubject, scope task.ListScope) task.ListResult {
	return services.taskService.List(ctx, subject, scope)
}

func (services mcpServices) GetTask(ctx context.Context, subject auth.UserSubject, taskID core.TaskID) task.GetResult {
	return services.taskService.Get(ctx, subject, taskID)
}

func (services mcpServices) CreateTask(ctx context.Context, command task.CreateCommand) task.CreateResult {
	return services.taskService.Create(ctx, command)
}

func (services mcpServices) SubmitResponse(ctx context.Context, command submission.SubmitCommand) submission.SubmitResult {
	return services.submissionService.Submit(ctx, command)
}

func (services mcpServices) GetSubmissionStatus(ctx context.Context, token submission.ReceiptTokenPlain) submission.ReceiptStatusResult {
	return services.submissionService.FindByReceipt(ctx, token)
}

func (services mcpServices) ListTaskSubmissions(ctx context.Context, subject auth.UserSubject, taskID core.TaskID) submission.ListResult {
	return services.submissionService.ListForTask(ctx, subject, taskID)
}

func (services mcpServices) AcceptSubmission(ctx context.Context, requester core.UserID, taskID core.TaskID, submissionID core.SubmissionID, key ledger.IdempotencyKey) ledger.AcceptResult {
	return services.ledgerService.AcceptSubmission(ctx, requester, taskID, submissionID, key)
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

	result := server.agentService.List(r.Context(), actor.subject.ID)
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
	verifyResult := server.verifyAgent(r)
	verified, verifiedMatched := verifyResult.(agent.CredentialVerified)
	if !verifiedMatched {
		writeError(w, http.StatusUnauthorized, verifyResult.(agent.VerifyRejected).Reason.Description())
		return
	}

	var request mcp.Request
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeMCPResponse(w, mcp.Response{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &mcp.RPCError{Code: -32700, Message: "request body is not valid JSON-RPC"}})
		return
	}

	if strings.HasPrefix(request.Method, "notifications/") {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	response := server.mcpServer.Handle(r.Context(), verified.Subject, verified.Credential.Scopes, request)
	writeMCPResponse(w, response)
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

func writeMCPResponse(w http.ResponseWriter, response mcp.Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
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
