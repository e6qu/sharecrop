//go:build js && wasm

package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/e6qu/sharecrop/internal/wasmdemo"
)

type wasmStatus struct {
	Name    string `json:"name"`
	Target  string `json:"target"`
	Runtime string `json:"runtime"`
}

type wasmHandleResponse struct {
	Status int    `json:"status"`
	Body   string `json:"body"`
	Error  string `json:"error"`
	Route  string `json:"route"`
}

func main() {
	js.Global().Set("sharecropWasmBackendStatus", js.FuncOf(func(js.Value, []js.Value) interface{} {
		return encodeStatus()
	}))
	js.Global().Set("sharecropHandleRequest", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		if len(args) != 3 {
			return encodeHandleResponse(wasmHandleResponse{Status: 400, Error: "method, path, and body arguments are required"})
		}
		requestResult := wasmdemo.NewRequest(args[0].String(), args[1].String(), args[2].String())
		request, requestMatched := requestResult.(wasmdemo.RequestAccepted)
		if !requestMatched {
			return encodeHandleResponse(wasmHandleResponse{Status: 400, Error: requestResult.(wasmdemo.RequestRejected).Reason})
		}
		adaptResult := wasmdemo.Adapt(request.Value)
		adapted, adaptedMatched := adaptResult.(wasmdemo.RequestAdapted)
		if !adaptedMatched {
			return encodeHandleResponse(wasmHandleResponse{Status: 404, Error: adaptResult.(wasmdemo.RequestUnsupported).Reason})
		}
		return encodeHandleResponse(wasmHandleResponse{
			Status: 501,
			Error:  "host runtime adapters are required before the Go/WASM backend can execute requests",
			Route:  adapted.Route.String(),
		})
	}))
	select {}
}

func encodeStatus() string {
	encoded, err := json.Marshal(wasmStatus{Name: "sharecrop-wasm", Target: "js/wasm", Runtime: "host-adapters-required"})
	if err != nil {
		panic("wasm status encoding failed")
	}
	return string(encoded)
}

func encodeHandleResponse(response wasmHandleResponse) string {
	encoded, err := json.Marshal(response)
	if err != nil {
		panic("wasm response encoding failed")
	}
	return string(encoded)
}
