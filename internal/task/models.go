package task

import "github.com/e6qu/sharecrop/internal/core"

type Task struct {
	ID             core.TaskID
	Owner          Owner
	Title          Title
	Description    Description
	Reward         RewardSpec
	State          State
	Visibility     Visibility
	Placement      SeriesPlacement
	ResponseSchema ResponseSchemaSource
	Payload        DataPayload
	CreatedBy      core.UserID
}

type Series struct {
	ID        core.TaskSeriesID
	Owner     Owner
	Title     SeriesTitle
	CreatedBy core.UserID
}

type CapabilityToken struct {
	ID     core.TaskCapabilityTokenID
	TaskID core.TaskID
	State  CapabilityTokenState
}

type Owner interface {
	owner()
}

type UserOwner struct {
	UserID core.UserID
}

type TeamOwner struct {
	TeamID core.TeamID
}

type OrganizationOwner struct {
	OrganizationID core.OrganizationID
}

type OrganizationTeamOwner struct {
	OrganizationID core.OrganizationID
	TeamID         core.TeamID
}

func (UserOwner) owner() {}

func (TeamOwner) owner() {}

func (OrganizationOwner) owner() {}

func (OrganizationTeamOwner) owner() {}

type RewardSpec interface {
	rewardSpec()
}

type NoRewardSpec struct{}

type CreditRewardSpec struct {
	Amount CreditRewardAmount
}

func (NoRewardSpec) rewardSpec() {}

func (CreditRewardSpec) rewardSpec() {}

type RewardKind struct {
	value string
}

var (
	RewardKindNone   = RewardKind{value: "none"}
	RewardKindCredit = RewardKind{value: "credit"}
)

func (kind RewardKind) String() string {
	return kind.value
}

type OwnerKind struct {
	value string
}

var (
	OwnerKindUser             = OwnerKind{value: "user"}
	OwnerKindTeam             = OwnerKind{value: "team"}
	OwnerKindOrganization     = OwnerKind{value: "organization"}
	OwnerKindOrganizationTeam = OwnerKind{value: "organization_team"}
)

func (kind OwnerKind) String() string {
	return kind.value
}

type State struct {
	value string
}

var (
	StateDraft     = State{value: "draft"}
	StateOpen      = State{value: "open"}
	StateClosed    = State{value: "closed"}
	StateCancelled = State{value: "cancelled"}
	StateExpired   = State{value: "expired"}
)

type StateResult interface {
	stateResult()
}

type StateAccepted struct {
	Value State
}

type StateRejected struct {
	Reason core.DomainError
}

func (StateAccepted) stateResult() {}

func (StateRejected) stateResult() {}

func ParseState(raw string) StateResult {
	switch raw {
	case StateDraft.value:
		return StateAccepted{Value: StateDraft}
	case StateOpen.value:
		return StateAccepted{Value: StateOpen}
	case StateClosed.value:
		return StateAccepted{Value: StateClosed}
	case StateCancelled.value:
		return StateAccepted{Value: StateCancelled}
	case StateExpired.value:
		return StateAccepted{Value: StateExpired}
	default:
		return StateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "task state is invalid")}
	}
}

func (state State) String() string {
	return state.value
}

type StateTransitionResult interface {
	stateTransitionResult()
}

type StateTransitionAccepted struct {
	Value State
}

type StateTransitionRejected struct {
	Reason core.DomainError
}

func (StateTransitionAccepted) stateTransitionResult() {}

func (StateTransitionRejected) stateTransitionResult() {}

func OpenState(current State) StateTransitionResult {
	if current != StateDraft {
		return StateTransitionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "only draft tasks can be opened")}
	}
	return StateTransitionAccepted{Value: StateOpen}
}

func CancelState(current State) StateTransitionResult {
	switch current {
	case StateDraft, StateOpen:
		return StateTransitionAccepted{Value: StateCancelled}
	default:
		return StateTransitionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "task cannot be cancelled from this state")}
	}
}

type Visibility interface {
	visibility()
}

type PublicVisibility struct{}

type UserVisibility struct {
	UserID core.UserID
}

type TeamVisibility struct {
	TeamID core.TeamID
}

type OrganizationVisibility struct {
	OrganizationID core.OrganizationID
}

type OrganizationTeamVisibility struct {
	OrganizationID core.OrganizationID
	TeamID         core.TeamID
}

func (PublicVisibility) visibility() {}

func (UserVisibility) visibility() {}

func (TeamVisibility) visibility() {}

func (OrganizationVisibility) visibility() {}

func (OrganizationTeamVisibility) visibility() {}

type VisibilityKind struct {
	value string
}

var (
	VisibilityKindPublic           = VisibilityKind{value: "public"}
	VisibilityKindUser             = VisibilityKind{value: "user"}
	VisibilityKindTeam             = VisibilityKind{value: "team"}
	VisibilityKindOrganization     = VisibilityKind{value: "organization"}
	VisibilityKindOrganizationTeam = VisibilityKind{value: "organization_team"}
)

func (kind VisibilityKind) String() string {
	return kind.value
}

type SeriesPlacement interface {
	seriesPlacement()
}

type StandalonePlacement struct{}

type NewSeriesPlacement struct {
	Title    SeriesTitle
	Position SeriesPosition
}

type ExistingSeriesPlacement struct {
	SeriesID core.TaskSeriesID
	Position SeriesPosition
}

func (StandalonePlacement) seriesPlacement() {}

func (NewSeriesPlacement) seriesPlacement() {}

func (ExistingSeriesPlacement) seriesPlacement() {}

type DataPayload interface {
	dataPayload()
}

type NoDataPayload struct{}

type JSONDataPayload struct {
	Source PayloadSource
}

func (NoDataPayload) dataPayload() {}

func (JSONDataPayload) dataPayload() {}

type CapabilityTokenState struct {
	value string
}

var (
	CapabilityTokenStateActive  = CapabilityTokenState{value: "active"}
	CapabilityTokenStateRevoked = CapabilityTokenState{value: "revoked"}
)

type CapabilityTokenStateResult interface {
	capabilityTokenStateResult()
}

type CapabilityTokenStateAccepted struct {
	Value CapabilityTokenState
}

type CapabilityTokenStateRejected struct {
	Reason core.DomainError
}

func (CapabilityTokenStateAccepted) capabilityTokenStateResult() {}

func (CapabilityTokenStateRejected) capabilityTokenStateResult() {}

func ParseCapabilityTokenState(raw string) CapabilityTokenStateResult {
	switch raw {
	case CapabilityTokenStateActive.value:
		return CapabilityTokenStateAccepted{Value: CapabilityTokenStateActive}
	case CapabilityTokenStateRevoked.value:
		return CapabilityTokenStateAccepted{Value: CapabilityTokenStateRevoked}
	default:
		return CapabilityTokenStateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "task capability token state is invalid")}
	}
}

func (state CapabilityTokenState) String() string {
	return state.value
}
