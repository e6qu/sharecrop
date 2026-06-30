package attachment

import (
	"encoding/base64"
	"strings"

	"github.com/e6qu/sharecrop/internal/core"
)

const MaxBytes = 500 * 1024

type Attachment struct {
	Name        Name
	ContentType ContentType
	Content     Content
}

type Name struct {
	value string
}

type ContentType struct {
	value string
}

type Content struct {
	value []byte
}

type AttachmentResult interface {
	attachmentResult()
}

type AttachmentAccepted struct {
	Value Attachment
}

type AttachmentRejected struct {
	Reason core.DomainError
}

func (AttachmentAccepted) attachmentResult() {}
func (AttachmentRejected) attachmentResult() {}

func NewAttachment(rawName string, rawContentType string, dataURL string) AttachmentResult {
	nameResult := NewName(rawName)
	name, nameMatched := nameResult.(NameAccepted)
	if !nameMatched {
		return AttachmentRejected{Reason: nameResult.(NameRejected).Reason}
	}
	contentTypeResult := ParseContentType(rawContentType)
	contentType, contentTypeMatched := contentTypeResult.(ContentTypeAccepted)
	if !contentTypeMatched {
		return AttachmentRejected{Reason: contentTypeResult.(ContentTypeRejected).Reason}
	}
	contentResult := ParseDataURL(dataURL, contentType.Value)
	content, contentMatched := contentResult.(ContentAccepted)
	if !contentMatched {
		return AttachmentRejected{Reason: contentResult.(ContentRejected).Reason}
	}
	return AttachmentAccepted{Value: Attachment{Name: name.Value, ContentType: contentType.Value, Content: content.Value}}
}

func NewStoredAttachment(rawName string, rawContentType string, content []byte) AttachmentResult {
	nameResult := NewName(rawName)
	name, nameMatched := nameResult.(NameAccepted)
	if !nameMatched {
		return AttachmentRejected{Reason: nameResult.(NameRejected).Reason}
	}
	contentTypeResult := ParseContentType(rawContentType)
	contentType, contentTypeMatched := contentTypeResult.(ContentTypeAccepted)
	if !contentTypeMatched {
		return AttachmentRejected{Reason: contentTypeResult.(ContentTypeRejected).Reason}
	}
	contentResult := NewContent(content)
	storedContent, contentMatched := contentResult.(ContentAccepted)
	if !contentMatched {
		return AttachmentRejected{Reason: contentResult.(ContentRejected).Reason}
	}
	return AttachmentAccepted{Value: Attachment{Name: name.Value, ContentType: contentType.Value, Content: storedContent.Value}}
}

type NameResult interface {
	nameResult()
}

type NameAccepted struct {
	Value Name
}

type NameRejected struct {
	Reason core.DomainError
}

func (NameAccepted) nameResult() {}
func (NameRejected) nameResult() {}

func NewName(raw string) NameResult {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return NameRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "attachment filename is required")}
	}
	if len(trimmed) > 160 {
		return NameRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "attachment filename is too long")}
	}
	if strings.Contains(trimmed, "\x00") || strings.Contains(trimmed, "/") || strings.Contains(trimmed, "\\") {
		return NameRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "attachment filename is invalid")}
	}
	return NameAccepted{Value: Name{value: trimmed}}
}

func (name Name) String() string {
	return name.value
}

type ContentTypeResult interface {
	contentTypeResult()
}

type ContentTypeAccepted struct {
	Value ContentType
}

type ContentTypeRejected struct {
	Reason core.DomainError
}

func (ContentTypeAccepted) contentTypeResult() {}
func (ContentTypeRejected) contentTypeResult() {}

func ParseContentType(raw string) ContentTypeResult {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	switch trimmed {
	case "image/png", "image/jpeg", "image/gif", "image/webp", "text/plain", "application/json", "application/pdf":
		return ContentTypeAccepted{Value: ContentType{value: trimmed}}
	default:
		return ContentTypeRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "attachment content type is not allowed")}
	}
}

func (contentType ContentType) String() string {
	return contentType.value
}

type ContentResult interface {
	contentResult()
}

type ContentAccepted struct {
	Value Content
}

type ContentRejected struct {
	Reason core.DomainError
}

func (ContentAccepted) contentResult() {}
func (ContentRejected) contentResult() {}

func NewContent(value []byte) ContentResult {
	if len(value) == 0 {
		return ContentRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "attachment content is required")}
	}
	if len(value) > MaxBytes {
		return ContentRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "attachment content is too large")}
	}
	copied := make([]byte, len(value))
	copy(copied, value)
	return ContentAccepted{Value: Content{value: copied}}
}

func ParseDataURL(raw string, expectedContentType ContentType) ContentResult {
	value := strings.TrimSpace(raw)
	prefix := "data:" + expectedContentType.String() + ";base64,"
	if !strings.HasPrefix(value, prefix) {
		return ContentRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "attachment data URL is invalid")}
	}
	encoded := strings.TrimPrefix(value, prefix)
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return ContentRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "attachment content is invalid")}
	}
	return NewContent(decoded)
}

func (content Content) Bytes() []byte {
	copied := make([]byte, len(content.value))
	copy(copied, content.value)
	return copied
}

func (content Content) SizeBytes() int {
	return len(content.value)
}

func (attachment Attachment) DataURL() string {
	return "data:" + attachment.ContentType.String() + ";base64," + base64.StdEncoding.EncodeToString(attachment.Content.value)
}
