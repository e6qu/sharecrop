//go:build !wasip1

package httpbridge

import (
	"net/http"

	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
)

// Handler turns a host into an http.Handler: each request is serialized, run
// through a fresh guest instance (which may make store calls back over the same
// unit of work), and its serialized response written out. Both the Phase 4
// health host and the app host use it, so they cannot drift in how they bridge
// a request.
func Handler(host *rpc.Host) http.Handler {
	return bridgeHandler{host: host}
}

type bridgeHandler struct {
	host *rpc.Host
}

func (h bridgeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestBytes, err := EncodeRequest(r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	responseBytes, err := h.host.Call(r.Context(), "http.handle", requestBytes)
	if err != nil {
		http.Error(w, "bridge error: "+err.Error(), http.StatusBadGateway)
		return
	}
	if err := WriteResponse(w, responseBytes); err != nil {
		http.Error(w, "write response: "+err.Error(), http.StatusInternalServerError)
	}
}
