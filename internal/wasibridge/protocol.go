// Package wasibridge is the Phase 2 spike for running Sharecrop's backend as a
// GOOS=wasip1 WASM guest hosted by a native process. The guest cannot open a
// Postgres connection (TCP networking is a stub under wasip1), so every storage
// call it makes is serialized and sent to the host, which services it against
// the real internal/db store and sends the result back.
//
// This package proves the pattern for exactly one store method
// (auth.AuthStore.FindCredentialByEmail): the smallest real read, chosen so the
// spike exercises the found, not-found, and rejected (DomainError) paths and
// verifies the DomainError shape survives the serialization boundary.
//
// The bridge deliberately contains NO business logic. The host side parses the
// request, calls the real store method, and serializes the result — nothing
// more. Keeping decisions out of the bridge is what lets a future phase
// generate it from the store interfaces without the generated code drifting
// from the hand-written server.
//
// Wire framing is a 4-byte big-endian length prefix followed by a JSON payload,
// carried over the guest's stdin/stdout. The guest writes request frames to
// stdout and reads response frames from stdin; the host does the mirror. Every
// unit of work runs in a fresh guest instance driven by a single goroutine, so
// the stream is strictly request/response with no interleaving to disambiguate.
package wasibridge

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
)

// maxFrameBytes bounds a single frame so a corrupt or hostile length prefix
// cannot make the reader allocate without limit. Credential frames are a few
// hundred bytes; 1 MiB is generous headroom.
const maxFrameBytes = 1 << 20

// errorCodesByString reverses ErrorCode.String so a DomainError serialized as
// its code string can be rebuilt with the canonical core.ErrorCode value rather
// than a stringly-typed stand-in. Building it here (not in internal/core) keeps
// the mapping next to the serialization that needs it.
var errorCodesByString = func() map[string]core.ErrorCode {
	codes := []core.ErrorCode{
		core.ErrorCodeInvalidID,
		core.ErrorCodeInvalidEnum,
		core.ErrorCodeInvalidState,
		core.ErrorCodeInvalidArgument,
		core.ErrorCodeNotFound,
		core.ErrorCodePermissionDenied,
		core.ErrorCodeConflict,
	}
	byString := make(map[string]core.ErrorCode, len(codes))
	for _, code := range codes {
		byString[code.String()] = code
	}
	return byString
}()

// guestFrame is everything the guest sends to the host over stdout. Exactly one
// field is populated, selected by Kind: a "store_call" asks the host to service
// a storage operation, and a "result" reports the guest's final answer for the
// unit of work (used by the spike to prove the value round-tripped intact).
type guestFrame struct {
	Kind   string          `json:"kind"`
	Call   *lookupRequest  `json:"call,omitempty"`
	Result *credentialWire `json:"result,omitempty"`
}

// hostFrame is the host's reply to a store_call.
type hostFrame struct {
	Response *credentialWire `json:"response,omitempty"`
}

// lookupRequest identifies a single store method invocation. Store and Method
// are carried explicitly so the host dispatches by name — the shape a generated
// multi-method bridge would use — even though the spike wires only one.
type lookupRequest struct {
	Store  string `json:"store"`
	Method string `json:"method"`
	Email  string `json:"email"`
}

// credentialWire is the serialized form of auth.CredentialLookupResult. Variant
// selects which of the sealed-union cases is represented: "found" populates the
// record fields, "missing" carries nothing, and "rejected" carries the
// DomainError's code and description so the exact error shape can be rebuilt.
type credentialWire struct {
	Variant string `json:"variant"`

	UserID       string `json:"user_id,omitempty"`
	Email        string `json:"email,omitempty"`
	PasswordHash string `json:"password_hash,omitempty"`
	Status       string `json:"status,omitempty"`

	ErrorCode        string `json:"error_code,omitempty"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// writeGuestFrame and writeHostFrame serialize a frame and write it with a
// length prefix. They are thin typed wrappers over writeFramePayload so no call
// site has to hand-marshal, and so the transport carries only concrete types.
func writeGuestFrame(w io.Writer, frame guestFrame) error {
	payload, err := json.Marshal(frame)
	if err != nil {
		return fmt.Errorf("marshal guest frame: %w", err)
	}
	return writeFramePayload(w, payload)
}

func writeHostFrame(w io.Writer, frame hostFrame) error {
	payload, err := json.Marshal(frame)
	if err != nil {
		return fmt.Errorf("marshal host frame: %w", err)
	}
	return writeFramePayload(w, payload)
}

func writeFramePayload(w io.Writer, payload []byte) error {
	if len(payload) > maxFrameBytes {
		return fmt.Errorf("frame of %d bytes exceeds limit %d", len(payload), maxFrameBytes)
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(payload)))
	if _, err := w.Write(header[:]); err != nil {
		return fmt.Errorf("write frame header: %w", err)
	}
	if _, err := w.Write(payload); err != nil {
		return fmt.Errorf("write frame payload: %w", err)
	}
	return nil
}

// readGuestFrame and readHostFrame read one length-prefixed frame and decode it
// into the concrete frame type. They return io.EOF only when the stream is
// cleanly closed on a frame boundary, which the host uses to detect the guest
// exiting.
func readGuestFrame(r io.Reader) (guestFrame, error) {
	var frame guestFrame
	payload, err := readFramePayload(r)
	if err != nil {
		return frame, err
	}
	if err := json.Unmarshal(payload, &frame); err != nil {
		return frame, fmt.Errorf("unmarshal guest frame: %w", err)
	}
	return frame, nil
}

func readHostFrame(r io.Reader) (hostFrame, error) {
	var frame hostFrame
	payload, err := readFramePayload(r)
	if err != nil {
		return frame, err
	}
	if err := json.Unmarshal(payload, &frame); err != nil {
		return frame, fmt.Errorf("unmarshal host frame: %w", err)
	}
	return frame, nil
}

func readFramePayload(r io.Reader) ([]byte, error) {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		if err == io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("read frame header: %w", err)
		}
		return nil, err
	}
	length := binary.BigEndian.Uint32(header[:])
	if length > maxFrameBytes {
		return nil, fmt.Errorf("frame of %d bytes exceeds limit %d", length, maxFrameBytes)
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, fmt.Errorf("read frame payload: %w", err)
	}
	return payload, nil
}

// credentialToWire serializes a store result. Used on the host to encode the
// real internal/db result, and on the guest to re-encode the reconstructed
// result so the spike can prove it survived the trip.
func credentialToWire(result auth.CredentialLookupResult) credentialWire {
	switch typed := result.(type) {
	case auth.CredentialFound:
		return credentialWire{
			Variant:      "found",
			UserID:       typed.Record.UserID.String(),
			Email:        typed.Record.Email.String(),
			PasswordHash: typed.Record.PasswordHash.String(),
			Status:       typed.Record.Status,
		}
	case auth.CredentialMissing:
		return credentialWire{Variant: "missing"}
	case auth.CredentialLookupRejected:
		return credentialWire{
			Variant:          "rejected",
			ErrorCode:        typed.Reason.Code().String(),
			ErrorDescription: typed.Reason.Description(),
		}
	default:
		return credentialWire{
			Variant:          "rejected",
			ErrorCode:        core.ErrorCodeInvalidState.String(),
			ErrorDescription: fmt.Sprintf("unknown credential result %T", result),
		}
	}
}

// credentialFromWire reconstructs a store result from its wire form. A "found"
// frame whose fields fail to parse, or a "rejected" frame with an unrecognized
// code, degrades to a CredentialLookupRejected carrying an invalid_state error
// rather than silently losing information.
func credentialFromWire(wire credentialWire) auth.CredentialLookupResult {
	switch wire.Variant {
	case "found":
		userID, ok := core.ParseUserID(wire.UserID).(core.UserIDCreated)
		if !ok {
			return rejected(core.ErrorCodeInvalidID, "bridge: invalid user id in credential frame")
		}
		email, ok := auth.NewEmailAddress(wire.Email).(auth.EmailAddressAccepted)
		if !ok {
			return rejected(core.ErrorCodeInvalidArgument, "bridge: invalid email in credential frame")
		}
		hash, ok := auth.ParsePasswordHash(wire.PasswordHash).(auth.PasswordHashCreated)
		if !ok {
			return rejected(core.ErrorCodeInvalidArgument, "bridge: invalid password hash in credential frame")
		}
		return auth.CredentialFound{Record: auth.CredentialRecord{
			UserID:       userID.Value,
			Email:        email.Value,
			PasswordHash: hash.Value,
			Status:       wire.Status,
		}}
	case "missing":
		return auth.CredentialMissing{}
	case "rejected":
		code, ok := errorCodesByString[wire.ErrorCode]
		if !ok {
			return rejected(core.ErrorCodeInvalidState, "bridge: unknown error code "+wire.ErrorCode)
		}
		return auth.CredentialLookupRejected{Reason: core.NewDomainError(code, wire.ErrorDescription)}
	default:
		return rejected(core.ErrorCodeInvalidState, "bridge: unknown credential variant "+wire.Variant)
	}
}

func rejected(code core.ErrorCode, description string) auth.CredentialLookupResult {
	return auth.CredentialLookupRejected{Reason: core.NewDomainError(code, description)}
}
