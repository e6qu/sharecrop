// Package httpbridge carries a single HTTP request/response across the WASI
// boundary for Phase 4 of the hosting spike. The native host serializes an
// incoming *http.Request, runs one guest instance to handle it, and writes the
// serialized response back to its ResponseWriter; the guest deserializes the
// request, runs it through the real internal/http mux with an httptest
// recorder, and serializes the response. It is the request/response counterpart
// to the store-call transport in the rpc package, and reuses the same
// unit-of-work channel (rpc.Invoke stays available for whatever DB access the
// handler makes).
package httpbridge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
)

// Request is the wire form of an incoming HTTP request. Header is the standard
// canonicalized map; Body is the full request body (base64 in JSON).
//
// RemoteAddr and Host are carried explicitly because the guest rebuilds the
// request with httptest.NewRequest, which fills them with placeholders, and Go
// keeps neither in the header map: RemoteAddr feeds per-IP rate limiting (see
// the note in Serve) and Host feeds the MCP origin check (Origin must match the
// request host), so dropping them breaks rate limiting and rejects same-origin
// MCP requests under the guest.
type Request struct {
	Method     string      `json:"method"`
	Target     string      `json:"target"`
	RemoteAddr string      `json:"remote_addr,omitempty"`
	Host       string      `json:"host,omitempty"`
	Header     http.Header `json:"header,omitempty"`
	Body       []byte      `json:"body,omitempty"`
}

// Response is the wire form of the handler's response.
type Response struct {
	Status int         `json:"status"`
	Header http.Header `json:"header,omitempty"`
	Body   []byte      `json:"body,omitempty"`
}

// EncodeRequest serializes an incoming request (host side). Target is the
// request URI (path plus raw query) so the guest reconstructs the same URL.
func EncodeRequest(r *http.Request) ([]byte, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("read request body: %w", err)
	}
	return json.Marshal(Request{
		Method:     r.Method,
		Target:     r.URL.RequestURI(),
		RemoteAddr: r.RemoteAddr,
		Host:       r.Host,
		Header:     r.Header,
		Body:       body,
	})
}

// Serve runs one serialized request through handler and returns the serialized
// response (guest side). It uses an httptest recorder exactly as the browser
// demo does, so a real internal/http handler runs unchanged.
func Serve(handler http.Handler, requestBytes []byte) ([]byte, error) {
	var request Request
	if err := json.Unmarshal(requestBytes, &request); err != nil {
		return nil, fmt.Errorf("decode request: %w", err)
	}

	httpRequest := httptest.NewRequest(request.Method, request.Target, bytes.NewReader(request.Body))
	// httptest.NewRequest seeds a Host header; replace the whole header set with
	// the caller's so the handler sees exactly what arrived at the host.
	if request.Header != nil {
		httpRequest.Header = request.Header
	}
	// httptest.NewRequest hardcodes RemoteAddr to a placeholder; use the real
	// peer address the host observed so per-IP rate limiting keys correctly.
	if request.RemoteAddr != "" {
		httpRequest.RemoteAddr = request.RemoteAddr
	}
	// Likewise it hardcodes Host to "example.com"; use the real host so the MCP
	// origin check (Origin host must equal r.Host) accepts same-origin requests.
	if request.Host != "" {
		httpRequest.Host = request.Host
	}

	recorder := httptest.NewRecorder()
	// Hand the handler a writer that does NOT implement http.Flusher. This
	// transport buffers the whole response and returns it as one unit of work,
	// so it genuinely cannot stream: a handler that trusts a Flusher and then
	// blocks writing an open stream (the MCP SSE endpoint does exactly this)
	// would never return, wedging this guest instance forever - a few such
	// requests would exhaust the pool and hang the server. Presenting an honest
	// non-streaming writer makes such handlers fall back to a bounded response.
	handler.ServeHTTP(nonFlushingWriter{recorder: recorder}, httpRequest)
	result := recorder.Result()
	body, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("read handler response: %w", err)
	}

	return json.Marshal(Response{
		Status: result.StatusCode,
		Header: result.Header,
		Body:   body,
	})
}

// nonFlushingWriter delegates to an httptest.ResponseRecorder (preserving its
// exact capture semantics) while deliberately not promoting the recorder's
// Flush method, so it satisfies http.ResponseWriter but not http.Flusher. See
// the note in Serve for why the bridge must not look flushable.
type nonFlushingWriter struct {
	recorder *httptest.ResponseRecorder
}

func (w nonFlushingWriter) Header() http.Header { return w.recorder.Header() }

func (w nonFlushingWriter) Write(p []byte) (int, error) { return w.recorder.Write(p) }

func (w nonFlushingWriter) WriteHeader(status int) { w.recorder.WriteHeader(status) }

// WriteResponse writes a serialized response to a ResponseWriter (host side).
func WriteResponse(w http.ResponseWriter, responseBytes []byte) error {
	var response Response
	if err := json.Unmarshal(responseBytes, &response); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	for key, values := range response.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	if response.Status == 0 {
		response.Status = http.StatusOK
	}
	w.WriteHeader(response.Status)
	if _, err := w.Write(response.Body); err != nil {
		return fmt.Errorf("write response body: %w", err)
	}
	return nil
}
