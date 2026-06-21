package core

type LifecycleState struct {
	value string
}

var (
	LifecycleStateDraft     = LifecycleState{value: "draft"}
	LifecycleStateActive    = LifecycleState{value: "active"}
	LifecycleStateClosed    = LifecycleState{value: "closed"}
	LifecycleStateCancelled = LifecycleState{value: "cancelled"}
)

type LifecycleStateResult interface {
	lifecycleStateResult()
}

type LifecycleStateParsed struct {
	Value LifecycleState
}

type LifecycleStateRejected struct {
	Reason DomainError
}

func (LifecycleStateParsed) lifecycleStateResult() {}

func (LifecycleStateRejected) lifecycleStateResult() {}

func ParseLifecycleState(raw string) LifecycleStateResult {
	switch raw {
	case LifecycleStateDraft.value:
		return LifecycleStateParsed{Value: LifecycleStateDraft}
	case LifecycleStateActive.value:
		return LifecycleStateParsed{Value: LifecycleStateActive}
	case LifecycleStateClosed.value:
		return LifecycleStateParsed{Value: LifecycleStateClosed}
	case LifecycleStateCancelled.value:
		return LifecycleStateParsed{Value: LifecycleStateCancelled}
	default:
		return LifecycleStateRejected{
			Reason: NewDomainError(ErrorCodeInvalidState, "unknown lifecycle state"),
		}
	}
}

func (state LifecycleState) String() string {
	return state.value
}
