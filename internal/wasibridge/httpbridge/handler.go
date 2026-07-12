//go:build !wasip1

package httpbridge

import (
	"net/http"

	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
)

// Handler turns a caller into an http.Handler: each request is serialized, run
// through a guest instance (which may make store calls back over the same unit
// of work), and its serialized response written out. The caller is an
// rpc.Caller, so the same handler drives either a fresh instance per request
// (*rpc.Host, used by tests) or a reused instance from a pool (*rpc.Pool, used
// by the production host) with no drift in how a request is bridged.
func Handler(caller rpc.Caller) http.Handler {
	return bridgeHandler{caller: caller}
}

type bridgeHandler struct {
	caller rpc.Caller
}

func (h bridgeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestBytes, err := EncodeRequest(r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	responseBytes, err := h.caller.Call(r.Context(), "http.handle", requestBytes)
	if err != nil {
		http.Error(w, "bridge error: "+err.Error(), http.StatusBadGateway)
		return
	}
	if err := WriteResponse(w, responseBytes); err != nil {
		http.Error(w, "write response: "+err.Error(), http.StatusInternalServerError)
	}
}
