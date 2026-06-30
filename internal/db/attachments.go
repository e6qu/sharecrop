package db

import (
	"encoding/base64"
	"encoding/json"

	"github.com/e6qu/sharecrop/internal/attachment"
	"github.com/e6qu/sharecrop/internal/core"
)

type storedAttachmentRow struct {
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	Content     string `json:"content"`
}

type attachmentsResult interface {
	attachmentsResult()
}

type attachmentsAccepted struct {
	values []attachment.Attachment
}

type attachmentsRejected struct {
	reason core.DomainError
}

func (attachmentsAccepted) attachmentsResult() {}
func (attachmentsRejected) attachmentsResult() {}

func parseStoredAttachments(raw string) attachmentsResult {
	var rows []storedAttachmentRow
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		return attachmentsRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "decode attachments failed")}
	}
	values := make([]attachment.Attachment, 0, len(rows))
	for index := range rows {
		content, err := base64.StdEncoding.DecodeString(rows[index].Content)
		if err != nil {
			return attachmentsRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "decode attachment content failed")}
		}
		result := attachment.NewStoredAttachment(rows[index].Name, rows[index].ContentType, content)
		accepted, matched := result.(attachment.AttachmentAccepted)
		if !matched {
			return attachmentsRejected{reason: result.(attachment.AttachmentRejected).Reason}
		}
		values = append(values, accepted.Value)
	}
	return attachmentsAccepted{values: values}
}

type insertAttachmentsResult interface {
	insertAttachmentsResult()
}

type insertAttachmentsAccepted struct{}

type insertAttachmentsRejected struct {
	reason core.DomainError
}

func (insertAttachmentsAccepted) insertAttachmentsResult() {}
func (insertAttachmentsRejected) insertAttachmentsResult() {}
