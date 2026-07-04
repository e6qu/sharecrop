package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/orgcred"
)

type orgCredentialRequest struct {
	Label     string   `json:"label"`
	Scopes    []string `json:"scopes"`
	ExpiresAt string   `json:"expires_at"`
}

type orgCredentialResponse struct {
	ID             string   `json:"id"`
	OrganizationID string   `json:"organization_id"`
	Label          string   `json:"label"`
	Scopes         []string `json:"scopes"`
	State          string   `json:"state"`
	ExpiresAt      string   `json:"expires_at"`
}

type orgCredentialCreatedResponse struct {
	Credential orgCredentialResponse `json:"credential"`
	Secret     string                `json:"secret"`
}

type orgCredentialsResponse struct {
	Credentials []orgCredentialResponse `json:"credentials"`
}

func (orgCredentialResponse) writableResponse() {}

func (orgCredentialCreatedResponse) writableResponse() {}

func (orgCredentialsResponse) writableResponse() {}

// requireOrgCredentialManagePermission authenticates the calling user and
// checks they hold PermissionManageMembers on the organization: minting an
// org-wide credential (which acts with full org-admin parity once issued) is
// at least as sensitive as membership management, so it's gated the same
// way — a human with that permission mints it, even though the resulting
// token then acts on its own.
func (server Server) requireOrgCredentialManagePermission(w http.ResponseWriter, r *http.Request) (core.OrganizationID, bool) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return core.OrganizationID{}, false
	}

	organizationIDResult := parseOrganizationPathValue(r)
	organizationIDAccepted, organizationIDMatched := organizationIDResult.(organizationIDAccepted)
	if !organizationIDMatched {
		writeError(w, http.StatusBadRequest, organizationIDResult.(organizationIDRejected).reason)
		return core.OrganizationID{}, false
	}

	permissionResult := server.organizationService.CheckOrganizationPermission(r.Context(), organizationIDAccepted.value, actor.subject.ID, org.PermissionManageMembers)
	if _, denied := permissionResult.(org.PermissionDenied); denied {
		writeError(w, http.StatusForbidden, "organization credential management access denied")
		return core.OrganizationID{}, false
	}

	return organizationIDAccepted.value, true
}

func (server Server) createOrgCredential(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := server.requireOrgCredentialManagePermission(w, r)
	if !ok {
		return
	}

	var request orgCredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}

	fields, ok := parseCredentialFields(w, request.Label, request.Scopes, request.ExpiresAt)
	if !ok {
		return
	}

	result := server.orgCredentialService.Create(r.Context(), organizationID, fields.label, fields.scopes, fields.expiresAt)
	created, matched := result.(orgcred.CredentialCreated)
	if !matched {
		writeError(w, http.StatusBadRequest, result.(orgcred.CreateRejected).Reason.Description())
		return
	}

	writeJSON(w, http.StatusCreated, orgCredentialCreatedResponse{
		Credential: orgCredentialToResponse(created.Value),
		Secret:     created.Secret.String(),
	})
}

func (server Server) listOrgCredentials(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := server.requireOrgCredentialManagePermission(w, r)
	if !ok {
		return
	}

	result := server.orgCredentialService.List(r.Context(), organizationID, parsePage(r))
	listed, matched := result.(orgcred.CredentialsListed)
	if !matched {
		writeError(w, http.StatusBadRequest, result.(orgcred.ListRejected).Reason.Description())
		return
	}

	response := orgCredentialsResponse{Credentials: make([]orgCredentialResponse, 0, len(listed.Values))}
	for index := range listed.Values {
		response.Credentials = append(response.Credentials, orgCredentialToResponse(listed.Values[index]))
	}
	writeJSON(w, http.StatusOK, response)
}

func (server Server) revokeOrgCredential(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := server.requireOrgCredentialManagePermission(w, r)
	if !ok {
		return
	}

	credentialIDResult := core.ParseOrgCredentialID(r.PathValue("credential_id"))
	credentialID, credentialMatched := credentialIDResult.(core.OrgCredentialIDCreated)
	if !credentialMatched {
		writeError(w, http.StatusBadRequest, credentialIDResult.(core.OrgCredentialIDRejected).Reason.Description())
		return
	}

	result := server.orgCredentialService.Revoke(r.Context(), organizationID, credentialID.Value)
	revoked, matched := result.(orgcred.CredentialRevoked)
	if !matched {
		writeError(w, http.StatusBadRequest, result.(orgcred.RevokeRejected).Reason.Description())
		return
	}

	writeJSON(w, http.StatusOK, orgCredentialToResponse(revoked.Value))
}

func orgCredentialToResponse(value orgcred.Credential) orgCredentialResponse {
	scopes := value.Scopes.Values()
	rawScopes := make([]string, 0, len(scopes))
	for index := range scopes {
		rawScopes = append(rawScopes, scopes[index].String())
	}
	var rawExpiresAt string
	if value.ExpiresAt != nil {
		rawExpiresAt = value.ExpiresAt.UTC().Format(time.RFC3339)
	}
	return orgCredentialResponse{
		ID:             value.ID.String(),
		OrganizationID: value.OrganizationID.String(),
		Label:          value.Label.String(),
		Scopes:         rawScopes,
		State:          value.State.String(),
		ExpiresAt:      rawExpiresAt,
	}
}
