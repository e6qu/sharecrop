package wasmdemo

type HostRequest struct {
	Method string
	Path   string
	Body   string
}

type HostResponse struct {
	Status int
	Body   string
}

type HostRequestResult interface {
	hostRequestResult()
}

type HostRequestAccepted struct {
	Value HostRequest
}

type HostRequestRejected struct {
	Reason string
}

func (HostRequestAccepted) hostRequestResult() {}
func (HostRequestRejected) hostRequestResult() {}

func NewHostRequest(method string, path string, body string) HostRequestResult {
	result := NewRequest(method, path, body)
	accepted, matched := result.(RequestAccepted)
	if !matched {
		return HostRequestRejected{Reason: result.(RequestRejected).Reason}
	}
	return HostRequestAccepted{Value: HostRequest{Method: accepted.Value.Method.String(), Path: accepted.Value.Path, Body: accepted.Value.Body}}
}

type HostRuntime interface {
	Storage() BrowserStorage
	Clock() HandlerClock
	Actor() HandlerActor
	InteractionIDs() InteractionIDSource
}

type HostRuntimeResult interface {
	hostRuntimeResult()
}

type HostRuntimeAccepted struct {
	Storage        BrowserStorage
	Clock          HandlerClock
	Actor          HandlerActor
	InteractionIDs InteractionIDSource
}

type HostRuntimeRejected struct {
	Reason string
}

func (HostRuntimeAccepted) hostRuntimeResult() {}
func (HostRuntimeRejected) hostRuntimeResult() {}

func ValidateHostRuntime(runtime HostRuntime) HostRuntimeResult {
	if runtime == nil {
		return HostRuntimeRejected{Reason: "host runtime is required"}
	}
	if runtime.Storage() == nil {
		return HostRuntimeRejected{Reason: "host storage adapter is required"}
	}
	if runtime.Clock() == nil {
		return HostRuntimeRejected{Reason: "host clock adapter is required"}
	}
	if runtime.Actor() == nil {
		return HostRuntimeRejected{Reason: "host actor adapter is required"}
	}
	if runtime.InteractionIDs() == nil {
		return HostRuntimeRejected{Reason: "host interaction id adapter is required"}
	}
	return HostRuntimeAccepted{
		Storage:        runtime.Storage(),
		Clock:          runtime.Clock(),
		Actor:          runtime.Actor(),
		InteractionIDs: runtime.InteractionIDs(),
	}
}
