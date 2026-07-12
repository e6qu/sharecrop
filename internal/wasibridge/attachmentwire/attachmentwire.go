// Package attachmentwire serializes attachment.Attachment across the WASI store
// bridge. Both the submission and task bridges carry attachments, so the codec
// lives here rather than being duplicated in each (jscpd blocks the duplicate).
// Content crosses the wire base64-encoded; the whole attachment is reconstructed
// through attachment.NewStoredAttachment, the same path the db adapter uses.
package attachmentwire

import (
	"encoding/base64"
	"fmt"

	"github.com/e6qu/sharecrop/internal/attachment"
)

// Wire is the JSON shape of one attachment.
type Wire struct {
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	Content     string `json:"content"`
}

// Encode serializes one attachment.
func Encode(value attachment.Attachment) Wire {
	return Wire{
		Name:        value.Name.String(),
		ContentType: value.ContentType.String(),
		Content:     base64.StdEncoding.EncodeToString(value.Content.Bytes()),
	}
}

// Decode reconstructs one attachment, failing if the content is not valid base64
// or the reconstructed attachment is rejected.
func Decode(wire Wire) (attachment.Attachment, error) {
	content, err := base64.StdEncoding.DecodeString(wire.Content)
	if err != nil {
		return attachment.Attachment{}, fmt.Errorf("decode attachment content: %w", err)
	}
	accepted, matched := attachment.NewStoredAttachment(wire.Name, wire.ContentType, content).(attachment.AttachmentAccepted)
	if !matched {
		return attachment.Attachment{}, fmt.Errorf("invalid attachment %q", wire.Name)
	}
	return accepted.Value, nil
}

// EncodeSlice serializes a slice of attachments.
func EncodeSlice(values []attachment.Attachment) []Wire {
	encoded := make([]Wire, 0, len(values))
	for index := range values {
		encoded = append(encoded, Encode(values[index]))
	}
	return encoded
}

// DecodeSlice reconstructs a slice of attachments.
func DecodeSlice(wires []Wire) ([]attachment.Attachment, error) {
	values := make([]attachment.Attachment, 0, len(wires))
	for index := range wires {
		value, err := Decode(wires[index])
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
