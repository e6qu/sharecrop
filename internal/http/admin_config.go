package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/core"
)

type PlatformAdminRecord struct {
	UserID    core.UserID
	Source    string
	CreatedAt time.Time
}

type PlatformAdminService interface {
	IsAdmin(context.Context, core.UserID) PlatformAdminCheckResult
	List(context.Context, core.Page) PlatformAdminListResult
	Grant(context.Context, core.UserID, core.UserID) PlatformAdminMutationResult
	Revoke(context.Context, core.UserID) PlatformAdminMutationResult
}

type PlatformAdminCheckResult interface{ platformAdminCheckResult() }
type PlatformAdminAllowed struct{}
type PlatformAdminDenied struct{ Reason core.DomainError }

func (PlatformAdminAllowed) platformAdminCheckResult() {}
func (PlatformAdminDenied) platformAdminCheckResult()  {}

type PlatformAdminListResult interface{ platformAdminListResult() }
type PlatformAdminsListed struct{ Values []PlatformAdminRecord }
type PlatformAdminListRejected struct{ Reason core.DomainError }

func (PlatformAdminsListed) platformAdminListResult()      {}
func (PlatformAdminListRejected) platformAdminListResult() {}

type PlatformAdminMutationResult interface{ platformAdminMutationResult() }
type PlatformAdminSaved struct{ Value PlatformAdminRecord }
type PlatformAdminMutationRejected struct{ Reason core.DomainError }

func (PlatformAdminSaved) platformAdminMutationResult()            {}
func (PlatformAdminMutationRejected) platformAdminMutationResult() {}

type memoryPlatformAdminService struct {
	mu        sync.Mutex
	bootstrap map[string]PlatformAdminRecord
	granted   map[string]PlatformAdminRecord
}

func newMemoryPlatformAdminService(bootstrap map[string]bool) *memoryPlatformAdminService {
	records := map[string]PlatformAdminRecord{}
	for rawID := range bootstrap {
		parsed := core.ParseUserID(rawID)
		created, matched := parsed.(core.UserIDCreated)
		if matched {
			records[rawID] = PlatformAdminRecord{UserID: created.Value, Source: "bootstrap", CreatedAt: time.Now().UTC()}
		}
	}
	return &memoryPlatformAdminService{bootstrap: records, granted: map[string]PlatformAdminRecord{}}
}

func (service *memoryPlatformAdminService) IsAdmin(_ context.Context, userID core.UserID) PlatformAdminCheckResult {
	service.mu.Lock()
	defer service.mu.Unlock()
	if _, ok := service.bootstrap[userID.String()]; ok {
		return PlatformAdminAllowed{}
	}
	if _, ok := service.granted[userID.String()]; ok {
		return PlatformAdminAllowed{}
	}
	return PlatformAdminDenied{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "platform admin access is required")}
}

func (service *memoryPlatformAdminService) List(_ context.Context, page core.Page) PlatformAdminListResult {
	service.mu.Lock()
	defer service.mu.Unlock()
	values := make([]PlatformAdminRecord, 0, len(service.bootstrap)+len(service.granted))
	for _, record := range service.bootstrap {
		values = append(values, record)
	}
	for id, record := range service.granted {
		if _, bootstrap := service.bootstrap[id]; !bootstrap {
			values = append(values, record)
		}
	}
	sort.Slice(values, func(left int, right int) bool {
		return values[left].CreatedAt.After(values[right].CreatedAt)
	})
	return PlatformAdminsListed{Values: platformAdminPage(values, page)}
}

func (service *memoryPlatformAdminService) Grant(_ context.Context, userID core.UserID, _ core.UserID) PlatformAdminMutationResult {
	service.mu.Lock()
	defer service.mu.Unlock()
	if record, ok := service.bootstrap[userID.String()]; ok {
		return PlatformAdminSaved{Value: record}
	}
	record := PlatformAdminRecord{UserID: userID, Source: "granted", CreatedAt: time.Now().UTC()}
	service.granted[userID.String()] = record
	return PlatformAdminSaved{Value: record}
}

func (service *memoryPlatformAdminService) Revoke(_ context.Context, userID core.UserID) PlatformAdminMutationResult {
	service.mu.Lock()
	defer service.mu.Unlock()
	if _, ok := service.bootstrap[userID.String()]; ok {
		return PlatformAdminMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "bootstrap platform admins cannot be revoked")}
	}
	record, ok := service.granted[userID.String()]
	if !ok {
		return PlatformAdminMutationRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "platform admin was not found")}
	}
	delete(service.granted, userID.String())
	return PlatformAdminSaved{Value: record}
}

func platformAdminPage(values []PlatformAdminRecord, page core.Page) []PlatformAdminRecord {
	start := page.Offset()
	if start > len(values) {
		start = len(values)
	}
	end := start + page.Limit()
	if end > len(values) {
		end = len(values)
	}
	return values[start:end]
}

func (server Server) requireAdminSubject(w http.ResponseWriter, r *http.Request) (authSubject userSubjectAccepted, ok bool) {
	actorResult := server.requireUserSubject(r)
	actor, matched := actorResult.(userSubjectAccepted)
	if !matched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return userSubjectAccepted{}, false
	}
	check := server.platformAdmins.IsAdmin(r.Context(), actor.subject.ID)
	if _, allowed := check.(PlatformAdminAllowed); !allowed {
		writeDomainError(w, check.(PlatformAdminDenied).Reason)
		return userSubjectAccepted{}, false
	}
	return actor, true
}

func (server Server) isPlatformAdmin(ctx context.Context, userID core.UserID) bool {
	_, allowed := server.platformAdmins.IsAdmin(ctx, userID).(PlatformAdminAllowed)
	return allowed
}

func (server Server) listPlatformAdmins(w http.ResponseWriter, r *http.Request) {
	if _, ok := server.requireAdminSubject(w, r); !ok {
		return
	}
	page, pageOK := parsePageOrReject(w, r)
	if !pageOK {
		return
	}
	result := server.platformAdmins.List(r.Context(), page)
	listed, matched := result.(PlatformAdminsListed)
	if !matched {
		writeDomainError(w, result.(PlatformAdminListRejected).Reason)
		return
	}
	response := platformAdminsResponse{Admins: make([]platformAdminResponse, 0, len(listed.Values))}
	for _, record := range listed.Values {
		response.Admins = append(response.Admins, platformAdminToResponse(record))
	}
	writeJSON(w, http.StatusOK, response)
}

func (server Server) grantPlatformAdmin(w http.ResponseWriter, r *http.Request) {
	actor, ok := server.requireAdminSubject(w, r)
	if !ok {
		return
	}
	var request platformAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}
	userIDResult := core.ParseUserID(request.UserID)
	userID, matched := userIDResult.(core.UserIDCreated)
	if !matched {
		writeDomainError(w, userIDResult.(core.UserIDRejected).Reason)
		return
	}
	result := server.platformAdmins.Grant(r.Context(), userID.Value, actor.subject.ID)
	saved, savedMatched := result.(PlatformAdminSaved)
	if !savedMatched {
		writeDomainError(w, result.(PlatformAdminMutationRejected).Reason)
		return
	}
	metadataResult := encodeJSONMetadata(map[string]string{"source": saved.Value.Source})
	metadata, metadataMatched := metadataResult.(jsonMetadataEncoded)
	if !metadataMatched {
		writeDomainError(w, metadataResult.(jsonMetadataRejected).reason)
		return
	}
	if !server.recordAudit(w, r.Context(), actor.subject.ID, audit.ActionFromString("platform_admin_granted"), audit.Subject{Kind: "user", ID: saved.Value.UserID.String()}, audit.Metadata{JSON: metadata.value}) {
		return
	}
	writeJSON(w, http.StatusCreated, platformAdminToResponse(saved.Value))
}

func (server Server) revokePlatformAdmin(w http.ResponseWriter, r *http.Request) {
	actor, ok := server.requireAdminSubject(w, r)
	if !ok {
		return
	}
	userIDResult := core.ParseUserID(r.PathValue("user_id"))
	userID, matched := userIDResult.(core.UserIDCreated)
	if !matched {
		writeDomainError(w, userIDResult.(core.UserIDRejected).Reason)
		return
	}
	result := server.platformAdmins.Revoke(r.Context(), userID.Value)
	saved, savedMatched := result.(PlatformAdminSaved)
	if !savedMatched {
		writeDomainError(w, result.(PlatformAdminMutationRejected).Reason)
		return
	}
	metadataResult := encodeJSONMetadata(map[string]string{"source": saved.Value.Source})
	metadata, metadataMatched := metadataResult.(jsonMetadataEncoded)
	if !metadataMatched {
		writeDomainError(w, metadataResult.(jsonMetadataRejected).reason)
		return
	}
	if !server.recordAudit(w, r.Context(), actor.subject.ID, audit.ActionFromString("platform_admin_revoked"), audit.Subject{Kind: "user", ID: saved.Value.UserID.String()}, audit.Metadata{JSON: metadata.value}) {
		return
	}
	writeJSON(w, http.StatusOK, platformAdminToResponse(saved.Value))
}

func platformAdminToResponse(record PlatformAdminRecord) platformAdminResponse {
	return platformAdminResponse{UserID: record.UserID.String(), Source: record.Source, CreatedAt: record.CreatedAt.UTC().Format(time.RFC3339Nano)}
}

type jsonMetadataResult interface{ jsonMetadataResult() }
type jsonMetadataEncoded struct{ value string }
type jsonMetadataRejected struct{ reason core.DomainError }

func (jsonMetadataEncoded) jsonMetadataResult()  {}
func (jsonMetadataRejected) jsonMetadataResult() {}

func encodeJSONMetadata(value map[string]string) jsonMetadataResult {
	encoded, err := json.Marshal(value)
	if err != nil {
		return jsonMetadataRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "metadata encoding failed")}
	}
	return jsonMetadataEncoded{value: string(encoded)}
}
