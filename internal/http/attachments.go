package httpserver

import (
	"github.com/e6qu/sharecrop/internal/attachment"
)

type attachmentsRequestResult interface {
	attachmentsRequestResult()
}

type attachmentsRequestAccepted struct {
	values []attachment.Attachment
}

type attachmentsRequestRejected struct {
	reason string
}

func (attachmentsRequestAccepted) attachmentsRequestResult() {}
func (attachmentsRequestRejected) attachmentsRequestResult() {}

func attachmentsFromRequest(values []attachmentRequest) attachmentsRequestResult {
	if len(values) > attachment.MaxCount {
		return attachmentsRequestRejected{reason: "too many attachments"}
	}
	attachments := make([]attachment.Attachment, 0, len(values))
	for index := range values {
		result := attachment.NewAttachment(values[index].Name, values[index].ContentType, values[index].DataURL)
		accepted, matched := result.(attachment.AttachmentAccepted)
		if !matched {
			return attachmentsRequestRejected{reason: result.(attachment.AttachmentRejected).Reason.Description()}
		}
		attachments = append(attachments, accepted.Value)
	}
	return attachmentsRequestAccepted{values: attachments}
}

func attachmentsToResponse(values []attachment.Attachment) []attachmentResponse {
	responses := make([]attachmentResponse, 0, len(values))
	for index := range values {
		value := values[index]
		responses = append(responses, attachmentResponse{
			Name:        value.Name.String(),
			ContentType: value.ContentType.String(),
			SizeBytes:   value.Content.SizeBytes(),
			DataURL:     value.DataURL(),
		})
	}
	return responses
}
