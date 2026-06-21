package id

import (
	"github.com/google/uuid"
)

type ID struct {
	value uuid.UUID
}

type IDResult interface {
	idResult()
}

type IDCreated struct {
	Value ID
}

type IDRejected struct {
	Description string
}

func (IDCreated) idResult() {}

func (IDRejected) idResult() {}

func New() IDResult {
	value, err := uuid.NewV7()
	if err != nil {
		return IDRejected{
			Description: err.Error(),
		}
	}

	return IDCreated{Value: ID{value: value}}
}

func Parse(raw string) IDResult {
	value, err := uuid.Parse(raw)
	if err != nil {
		return IDRejected{
			Description: err.Error(),
		}
	}

	return IDCreated{Value: ID{value: value}}
}

func (id ID) String() string {
	return id.value.String()
}
