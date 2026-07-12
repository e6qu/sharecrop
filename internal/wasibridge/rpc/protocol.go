// Package rpc is the generic method-keyed transport for the WASI hosting
// bridge. It carries each store call as (method name, JSON args) -> (JSON
// result), so a generated bridge can drive every method on a store over the
// same channel, and it carries each unit of work (a store method for the store
// guest, an HTTP request for the app guest) the same way.
//
// The channel is plain WASI stdin/stdout. On stdout the guest writes "call"
// frames (a store operation the host must service) and, when a unit of work is
// done, a "result" frame. On stdin the host writes "work" frames (the next unit
// of work), "reply" frames (the answer to a store call), and a "shutdown" frame
// (no more work - exit). The guest loops over work frames until shutdown, so one
// instance can serve many units of work in sequence (see rpc.Pool); the host
// side keeps the stream strictly request/response, driven for each instance by a
// single goroutine, which is the concurrency-safe shape from the Phase 1
// findings.
package rpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/e6qu/sharecrop/internal/wasibridge/wire"
)

// guestFrame is what the guest writes to the host. A "call" carries a store
// method invocation the host must service; a "result" carries the guest's final
// answer for the current unit of work (Result on success, Error when the guest's
// own handler failed). Args and Result are raw JSON so the transport never has
// to know a payload's shape.
type guestFrame struct {
	Kind   string          `json:"kind"`
	Method string          `json:"method,omitempty"`
	Args   json.RawMessage `json:"args,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

// hostFrame is what the host writes to the guest. A "work" starts the next unit
// of work; a "reply" answers a store call (Result on success, Error for a
// transport-level failure - a domain rejection is carried inside Result); a
// "shutdown" tells the guest's loop to exit.
type hostFrame struct {
	Kind   string          `json:"kind"`
	Method string          `json:"method,omitempty"`
	Args   json.RawMessage `json:"args,omitempty"`
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
	if err := writeGuestFrame(out, guestFrame{Kind: "call", Method: method, Args: rawOrNull(args)}); err != nil {
		return nil, err
	}
	reply, err := readHostFrame(in)
	if err != nil {
		return nil, err
	}
	if reply.Kind != "reply" {
		return nil, fmt.Errorf("expected a reply frame, got %q", reply.Kind)
	}
	if reply.Error != "" {
		return nil, errors.New(reply.Error)
	}
	return reply.Result, nil
}

// Serve runs the guest's unit-of-work loop: read a "work" frame, hand its
// (method, args) to handle, write the result (or the handler's error) back as a
// "result" frame, and repeat until the host sends "shutdown" or closes the
// channel. One instance therefore serves many units of work in
// sequence, which is what lets the host pool and reuse instances.
func Serve(handle func(method string, args []byte) ([]byte, error)) error {
	for {
		frame, err := readHostFrame(os.Stdin)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		switch frame.Kind {
		case "shutdown":
			return nil
		case "work":
			result, handleErr := handle(frame.Method, frame.Args)
			out := guestFrame{Kind: "result"}
			if handleErr != nil {
				out.Error = handleErr.Error()
			} else {
				out.Result = rawOrNull(result)
			}
			if err := writeGuestFrame(os.Stdout, out); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unexpected host frame %q", frame.Kind)
		}
	}
}

func writeGuestFrame(w io.Writer, frame guestFrame) error {
	payload, err := json.Marshal(frame)
	if err != nil {
		return fmt.Errorf("marshal guest frame: %w", err)
	}
	return wire.WriteFrame(w, payload)
}

func readGuestFrame(r io.Reader) (guestFrame, error) {
	var frame guestFrame
	payload, err := wire.ReadFrame(r)
	if err != nil {
		return frame, err
	}
	if err := json.Unmarshal(payload, &frame); err != nil {
		return frame, fmt.Errorf("unmarshal guest frame: %w", err)
	}
	return frame, nil
}

func writeHostFrame(w io.Writer, frame hostFrame) error {
	payload, err := json.Marshal(frame)
	if err != nil {
		return fmt.Errorf("marshal host frame: %w", err)
	}
	return wire.WriteFrame(w, payload)
}

func readHostFrame(r io.Reader) (hostFrame, error) {
	var frame hostFrame
	payload, err := wire.ReadFrame(r)
	if err != nil {
		return frame, err
	}
	if err := json.Unmarshal(payload, &frame); err != nil {
		return frame, fmt.Errorf("unmarshal host frame: %w", err)
	}
	return frame, nil
}
