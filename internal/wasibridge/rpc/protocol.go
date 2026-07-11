// Package rpc is the generic method-keyed transport for the WASI hosting
// bridge. Where the Phase 2 spike wired one store method by hand, this carries
// each store call as (method name, JSON args) -> (JSON result), so a generated
// bridge can drive every method on a store over the same channel.
//
// The guest writes a "call" frame to its stdout for each store operation and
// reads the host's reply from its stdin; when its unit of work is done it writes
// a "result" frame. The host services calls against the real store and captures
// the result. As in Phase 2, each unit of work runs in one fresh guest instance
// driven by a single goroutine, so the stream is strictly request/response.
package rpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/e6qu/sharecrop/internal/wasibridge/wire"
)

// callFrame is what the guest writes to the host. A "call" carries a store
// method invocation the host must service; a "result" carries the guest's final
// answer for its unit of work. Args and Result are raw JSON so the transport
// never has to know a payload's shape.
type callFrame struct {
	Kind   string          `json:"kind"`
	Method string          `json:"method,omitempty"`
	Args   json.RawMessage `json:"args,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
}

// replyFrame is the host's answer to a call: the serialized result, or a
// transport-level error string when the host could not service the call at all
// (a domain rejection is carried inside Result, not here).
type replyFrame struct {
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

// rawOrNull returns valid JSON for a payload, substituting null for an empty
// slice so an omitted argument or result never produces a malformed frame.
func rawOrNull(payload []byte) json.RawMessage {
	if len(payload) == 0 {
		return json.RawMessage("null")
	}
	return json.RawMessage(payload)
}

// Invoke is the guest-side store call: it sends one method invocation to the
// host and returns the host's serialized result. A generated guest client calls
// this for every store method.
func Invoke(method string, args []byte) ([]byte, error) {
	return invoke(os.Stdout, os.Stdin, method, args)
}

func invoke(out io.Writer, in io.Reader, method string, args []byte) ([]byte, error) {
	if err := writeCallFrame(out, callFrame{Kind: "call", Method: method, Args: rawOrNull(args)}); err != nil {
		return nil, err
	}
	reply, err := readReplyFrame(in)
	if err != nil {
		return nil, err
	}
	if reply.Error != "" {
		return nil, errors.New(reply.Error)
	}
	return reply.Result, nil
}

// ReportResult writes the guest's final result for its unit of work, which the
// host captures as the return value of the call it kicked off.
func ReportResult(result []byte) error {
	return writeCallFrame(os.Stdout, callFrame{Kind: "result", Result: rawOrNull(result)})
}

// UnitOfWork reads the (method, args) the host handed this guest instance as
// program arguments when it instantiated the module.
func UnitOfWork() (string, []byte, error) {
	if len(os.Args) < 3 {
		return "", nil, fmt.Errorf("guest expected method and args arguments, got %d", len(os.Args)-1)
	}
	return os.Args[1], []byte(os.Args[2]), nil
}

func writeCallFrame(w io.Writer, frame callFrame) error {
	payload, err := json.Marshal(frame)
	if err != nil {
		return fmt.Errorf("marshal call frame: %w", err)
	}
	return wire.WriteFrame(w, payload)
}

func readCallFrame(r io.Reader) (callFrame, error) {
	var frame callFrame
	payload, err := wire.ReadFrame(r)
	if err != nil {
		return frame, err
	}
	if err := json.Unmarshal(payload, &frame); err != nil {
		return frame, fmt.Errorf("unmarshal call frame: %w", err)
	}
	return frame, nil
}

func writeReplyFrame(w io.Writer, frame replyFrame) error {
	payload, err := json.Marshal(frame)
	if err != nil {
		return fmt.Errorf("marshal reply frame: %w", err)
	}
	return wire.WriteFrame(w, payload)
}

func readReplyFrame(r io.Reader) (replyFrame, error) {
	var frame replyFrame
	payload, err := wire.ReadFrame(r)
	if err != nil {
		return frame, err
	}
	if err := json.Unmarshal(payload, &frame); err != nil {
		return frame, fmt.Errorf("unmarshal reply frame: %w", err)
	}
	return frame, nil
}
