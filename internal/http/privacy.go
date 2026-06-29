package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/e6qu/sharecrop/internal/audit"
)

const (
	privacyKindDataExport              = "data_export"
	privacyKindSensitiveFieldDeletion  = "sensitive_field_deletion"
	privacyRequestQueuedStatus         = "queued"
	privacyRequestAuditSubjectKind     = "privacy_request"
	privacyRequestMetadataEncodingFail = "privacy request metadata could not be encoded"
)

func (server Server) createPrivacyRequest(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, matched := actorResult.(userSubjectAccepted)
	if !matched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}

	var request privacyRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}
	if !validPrivacyRequestKind(request.Kind) {
		writeError(w, http.StatusBadRequest, "privacy request kind is invalid")
		return
	}

	metadataBytes, err := json.Marshal(map[string]string{"kind": request.Kind})
	if err != nil {
		writeError(w, http.StatusInternalServerError, privacyRequestMetadataEncodingFail)
		return
	}

	if !server.recordAudit(
		w,
		r.Context(),
		actor.subject.ID,
		audit.ActionPrivacyRequestCreated,
		audit.Subject{Kind: privacyRequestAuditSubjectKind, ID: actor.subject.ID.String()},
		audit.Metadata{JSON: string(metadataBytes)},
	) {
		return
	}

	writeJSON(w, http.StatusCreated, privacyRequestResponse{
		Kind:        request.Kind,
		Status:      privacyRequestQueuedStatus,
		RequestedBy: actor.subject.ID.String(),
	})
}

func validPrivacyRequestKind(kind string) bool {
	return kind == privacyKindDataExport || kind == privacyKindSensitiveFieldDeletion
}
