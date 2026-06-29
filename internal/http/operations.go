package httpserver

import "net/http"

type operationsResponse struct {
	Status                   string `json:"status"`
	AccountTokenDelivery     string `json:"account_token_delivery"`
	MCPStorage               string `json:"mcp_storage"`
	RateLimitStorage         string `json:"rate_limit_storage"`
	ActiveMCPSessions        int    `json:"active_mcp_sessions"`
	ActiveIPRateBuckets      int    `json:"active_ip_rate_buckets"`
	ActiveSubjectRateBuckets int    `json:"active_subject_rate_buckets"`
	SecureCookies            string `json:"secure_cookies"`
}

func (operationsResponse) writableResponse() {}

func (server Server) operationsStatus(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, matched := actorResult.(userSubjectAccepted)
	if !matched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}
	if !server.adminUserIDs[actor.subject.ID.String()] {
		writeError(w, http.StatusForbidden, "platform admin access is required")
		return
	}
	secureCookies := "enabled"
	if !server.secureCookies {
		secureCookies = "disabled"
	}
	writeJSON(w, http.StatusOK, operationsResponse{
		Status:                   "ok",
		AccountTokenDelivery:     server.accountTokens.mode,
		MCPStorage:               server.mcpSessions.storageKind(),
		RateLimitStorage:         server.ipRateLimiter.StorageKind(),
		ActiveMCPSessions:        server.mcpSessions.activeSessionCount(),
		ActiveIPRateBuckets:      server.ipRateLimiter.ActiveBuckets(),
		ActiveSubjectRateBuckets: server.subjectRateLimiter.ActiveBuckets(),
		SecureCookies:            secureCookies,
	})
}
