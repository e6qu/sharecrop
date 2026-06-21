package core

type VisibilityScope interface {
	visibilityScope()
}

type PublicVisibility struct{}

type UserVisibility struct {
	UserID UserID
}

type OrganizationVisibility struct {
	OrganizationID OrganizationID
}

func (PublicVisibility) visibilityScope() {}

func (UserVisibility) visibilityScope() {}

func (OrganizationVisibility) visibilityScope() {}

type VisibilityKind struct {
	value string
}

var (
	VisibilityKindPublic       = VisibilityKind{value: "public"}
	VisibilityKindUser         = VisibilityKind{value: "user"}
	VisibilityKindOrganization = VisibilityKind{value: "organization"}
)

type VisibilityKindResult interface {
	visibilityKindResult()
}

type VisibilityKindParsed struct {
	Value VisibilityKind
}

type VisibilityKindRejected struct {
	Reason DomainError
}

func (VisibilityKindParsed) visibilityKindResult() {}

func (VisibilityKindRejected) visibilityKindResult() {}

func ParseVisibilityKind(raw string) VisibilityKindResult {
	switch raw {
	case VisibilityKindPublic.value:
		return VisibilityKindParsed{Value: VisibilityKindPublic}
	case VisibilityKindUser.value:
		return VisibilityKindParsed{Value: VisibilityKindUser}
	case VisibilityKindOrganization.value:
		return VisibilityKindParsed{Value: VisibilityKindOrganization}
	default:
		return VisibilityKindRejected{
			Reason: NewDomainError(ErrorCodeInvalidEnum, "unknown visibility kind"),
		}
	}
}

func (kind VisibilityKind) String() string {
	return kind.value
}
