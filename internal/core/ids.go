package core

import "github.com/e6qu/sharecrop/internal/core/id"

type UserID struct {
	value id.ID
}

type TaskID struct {
	value id.ID
}

type OrganizationID struct {
	value id.ID
}

type GuestID struct {
	value id.ID
}

type RefreshTokenID struct {
	value id.ID
}

type UserIDResult interface {
	userIDResult()
}

type UserIDCreated struct {
	Value UserID
}

type UserIDRejected struct {
	Reason DomainError
}

func (UserIDCreated) userIDResult() {}

func (UserIDRejected) userIDResult() {}

func NewUserID() UserIDResult {
	return userIDFromIDResult(id.New())
}

func ParseUserID(raw string) UserIDResult {
	return userIDFromIDResult(id.Parse(raw))
}

func (id UserID) String() string {
	return id.value.String()
}

func userIDFromIDResult(result id.IDResult) UserIDResult {
	switch typed := result.(type) {
	case id.IDCreated:
		return UserIDCreated{Value: UserID{value: typed.Value}}
	case id.IDRejected:
		return UserIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, typed.Description)}
	default:
		return UserIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, "unknown id result")}
	}
}

type TaskIDResult interface {
	taskIDResult()
}

type TaskIDCreated struct {
	Value TaskID
}

type TaskIDRejected struct {
	Reason DomainError
}

func (TaskIDCreated) taskIDResult() {}

func (TaskIDRejected) taskIDResult() {}

func NewTaskID() TaskIDResult {
	return taskIDFromIDResult(id.New())
}

func ParseTaskID(raw string) TaskIDResult {
	return taskIDFromIDResult(id.Parse(raw))
}

func (id TaskID) String() string {
	return id.value.String()
}

func taskIDFromIDResult(result id.IDResult) TaskIDResult {
	switch typed := result.(type) {
	case id.IDCreated:
		return TaskIDCreated{Value: TaskID{value: typed.Value}}
	case id.IDRejected:
		return TaskIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, typed.Description)}
	default:
		return TaskIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, "unknown id result")}
	}
}

type OrganizationIDResult interface {
	organizationIDResult()
}

type OrganizationIDCreated struct {
	Value OrganizationID
}

type OrganizationIDRejected struct {
	Reason DomainError
}

func (OrganizationIDCreated) organizationIDResult() {}

func (OrganizationIDRejected) organizationIDResult() {}

func NewOrganizationID() OrganizationIDResult {
	return organizationIDFromIDResult(id.New())
}

func ParseOrganizationID(raw string) OrganizationIDResult {
	return organizationIDFromIDResult(id.Parse(raw))
}

func (id OrganizationID) String() string {
	return id.value.String()
}

func organizationIDFromIDResult(result id.IDResult) OrganizationIDResult {
	switch typed := result.(type) {
	case id.IDCreated:
		return OrganizationIDCreated{Value: OrganizationID{value: typed.Value}}
	case id.IDRejected:
		return OrganizationIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, typed.Description)}
	default:
		return OrganizationIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, "unknown id result")}
	}
}

type GuestIDResult interface {
	guestIDResult()
}

type GuestIDCreated struct {
	Value GuestID
}

type GuestIDRejected struct {
	Reason DomainError
}

func (GuestIDCreated) guestIDResult() {}

func (GuestIDRejected) guestIDResult() {}

func NewGuestID() GuestIDResult {
	return guestIDFromIDResult(id.New())
}

func ParseGuestID(raw string) GuestIDResult {
	return guestIDFromIDResult(id.Parse(raw))
}

func (id GuestID) String() string {
	return id.value.String()
}

func guestIDFromIDResult(result id.IDResult) GuestIDResult {
	switch typed := result.(type) {
	case id.IDCreated:
		return GuestIDCreated{Value: GuestID{value: typed.Value}}
	case id.IDRejected:
		return GuestIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, typed.Description)}
	default:
		return GuestIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, "unknown id result")}
	}
}

type RefreshTokenIDResult interface {
	refreshTokenIDResult()
}

type RefreshTokenIDCreated struct {
	Value RefreshTokenID
}

type RefreshTokenIDRejected struct {
	Reason DomainError
}

func (RefreshTokenIDCreated) refreshTokenIDResult() {}

func (RefreshTokenIDRejected) refreshTokenIDResult() {}

func NewRefreshTokenID() RefreshTokenIDResult {
	return refreshTokenIDFromIDResult(id.New())
}

func ParseRefreshTokenID(raw string) RefreshTokenIDResult {
	return refreshTokenIDFromIDResult(id.Parse(raw))
}

func (id RefreshTokenID) String() string {
	return id.value.String()
}

func refreshTokenIDFromIDResult(result id.IDResult) RefreshTokenIDResult {
	switch typed := result.(type) {
	case id.IDCreated:
		return RefreshTokenIDCreated{Value: RefreshTokenID{value: typed.Value}}
	case id.IDRejected:
		return RefreshTokenIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, typed.Description)}
	default:
		return RefreshTokenIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, "unknown id result")}
	}
}
