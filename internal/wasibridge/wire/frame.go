// Package wire is the length-prefixed framing shared by every part of the WASI
// hosting bridge: a 4-byte big-endian length followed by that many payload
// bytes. It carries JSON payloads over the guest's stdin/stdout, but knows
// nothing about their contents — callers marshal and unmarshal concrete types
// on either side. Keeping the framing in one place means the Phase 2 auth spike
// and the Phase 3 generic RPC transport cannot drift in how they frame bytes.
package wire

import (
	"encoding/binary"
	"fmt"
	"io"
)

// MaxFrameBytes bounds a single frame so a corrupt or hostile length prefix
// cannot make the reader allocate without limit. It must comfortably exceed the
// largest real payload the bridge carries: an HTTP request or response is one
// frame, its body is base64-encoded inside the frame's JSON (~4/3 inflation),
// and the app accepts request bodies up to 2 MiB (internal/http maxRequestBody)
// with attachments up to 500 KiB each (internal/attachment MaxBytes, up to
// MaxCount per item). A 1 MiB limit sat *below* the 2 MiB request limit, so a
// perfectly valid multi-attachment task - which the native mux accepts - failed
// under the guest with a 502. 16 MiB covers a max request and a single item with
// its attachments with wide headroom while still bounding allocation. (Callers
// paginate list responses, so an unbounded aggregate is not framed at once.)
const MaxFrameBytes = 16 << 20

// WriteFrame writes payload with a 4-byte big-endian length prefix.
func WriteFrame(w io.Writer, payload []byte) error {
	if len(payload) > MaxFrameBytes {
		return fmt.Errorf("frame of %d bytes exceeds limit %d", len(payload), MaxFrameBytes)
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

// ReadFrame reads one length-prefixed frame. It returns io.EOF only when the
// stream is cleanly closed on a frame boundary, which readers use to detect the
// guest exiting; a truncated frame returns a wrapped ErrUnexpectedEOF instead.
func ReadFrame(r io.Reader) ([]byte, error) {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		if err == io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("read frame header: %w", err)
		}
		return nil, err
	}
	length := binary.BigEndian.Uint32(header[:])
	if length > MaxFrameBytes {
		return nil, fmt.Errorf("frame of %d bytes exceeds limit %d", length, MaxFrameBytes)
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, fmt.Errorf("read frame payload: %w", err)
	}
	return payload, nil
}
