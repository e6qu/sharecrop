package openapi

import "testing"

const testServerSource = `package httpserver

import "net/http"

func newMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", health)
	mux.HandleFunc("POST /api/tasks", server.createTask)
	mux.HandleFunc("GET /api/tasks/{task_id}", server.getTask)
	mux.HandleFunc("POST /api/tasks/{task_id}/open", server.openTask)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(nil)))
	return mux
}

func health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
`

const testHandlersSource = `package httpserver

import "net/http"

func (server Server) createTask(w http.ResponseWriter, r *http.Request) {
	actor, ok := server.requireUserSubject(r)
	_ = actor
	_ = ok
}

func (server Server) getTask(w http.ResponseWriter, r *http.Request) {
	// public: task detail does not require caller identity.
}

func (server Server) openTask(w http.ResponseWriter, r *http.Request) {
	server.changeTaskState(w, r)
}

func (server Server) changeTaskState(w http.ResponseWriter, r *http.Request) {
	server.requireUserSubject(r)
}
`

func TestExtractFindsRoutesAndAuthRequirement(t *testing.T) {
	result := Extract(map[string][]byte{
		"server.go":   []byte(testServerSource),
		"handlers.go": []byte(testHandlersSource),
	})
	extracted, matched := result.(Extracted)
	if !matched {
		t.Fatalf("result = %#v, want Extracted", result)
	}
	if len(extracted.Routes) != 5 {
		t.Fatalf("routes = %d, want 5: %#v", len(extracted.Routes), extracted.Routes)
	}

	byKey := map[string]Route{}
	for _, route := range extracted.Routes {
		byKey[route.Method+" "+route.Path] = route
	}

	health, ok := byKey["GET /healthz"]
	if !ok {
		t.Fatalf("missing GET /healthz route: %#v", extracted.Routes)
	}
	if health.OperationID != "health" || health.RequiresAuth {
		t.Fatalf("health route = %#v", health)
	}

	createTask, ok := byKey["POST /api/tasks"]
	if !ok {
		t.Fatalf("missing POST /api/tasks route: %#v", extracted.Routes)
	}
	if createTask.OperationID != "createTask" || !createTask.RequiresAuth {
		t.Fatalf("createTask route = %#v", createTask)
	}

	getTask, ok := byKey["GET /api/tasks/{task_id}"]
	if !ok {
		t.Fatalf("missing GET /api/tasks/{task_id} route: %#v", extracted.Routes)
	}
	if getTask.OperationID != "getTask" || getTask.RequiresAuth {
		t.Fatalf("getTask route = %#v", getTask)
	}

	openTask, ok := byKey["POST /api/tasks/{task_id}/open"]
	if !ok {
		t.Fatalf("missing POST /api/tasks/{task_id}/open route: %#v", extracted.Routes)
	}
	if openTask.OperationID != "openTask" || !openTask.RequiresAuth {
		t.Fatalf("openTask route = %#v, want RequiresAuth via the changeTaskState helper", openTask)
	}

	static, ok := byKey["GET /static/"]
	if !ok {
		t.Fatalf("missing GET /static/ route: %#v", extracted.Routes)
	}
	if static.OperationID != "StripPrefix" || static.RequiresAuth {
		t.Fatalf("static route = %#v", static)
	}
}

func TestExtractIsSortedByPathThenMethod(t *testing.T) {
	result := Extract(map[string][]byte{
		"server.go":   []byte(testServerSource),
		"handlers.go": []byte(testHandlersSource),
	})
	extracted := result.(Extracted)
	for index := 1; index < len(extracted.Routes); index++ {
		previous := extracted.Routes[index-1]
		current := extracted.Routes[index]
		if previous.Path > current.Path {
			t.Fatalf("routes not sorted by path: %#v then %#v", previous, current)
		}
		if previous.Path == current.Path && previous.Method > current.Method {
			t.Fatalf("routes not sorted by method within path: %#v then %#v", previous, current)
		}
	}
}

func TestExtractRecognizesFormEncodedHandler(t *testing.T) {
	result := Extract(map[string][]byte{"server.go": []byte(`package httpserver
import "net/http"
func routes(mux *http.ServeMux) { mux.HandleFunc("POST /logout", logout) }
func logout(w http.ResponseWriter, r *http.Request) { _ = r.ParseForm() }
`)})
	extracted, ok := result.(Extracted)
	if !ok {
		t.Fatalf("extract = %#v", result)
	}
	if len(extracted.Routes) != 1 || extracted.Routes[0].RequestMediaType != "application/x-www-form-urlencoded" {
		t.Fatalf("routes = %#v", extracted.Routes)
	}
}

func TestExtractRecognizesHandlerResponseMediaType(t *testing.T) {
	result := Extract(map[string][]byte{"server.go": []byte(`package httpserver
import "net/http"
func routes(mux *http.ServeMux) { mux.HandleFunc("GET /signed-out", signedOut) }
func signedOut(w http.ResponseWriter, _ *http.Request) { w.Header().Set("Content-Type", "text/html; charset=utf-8") }
`)})
	extracted, ok := result.(Extracted)
	if !ok {
		t.Fatalf("extract = %#v", result)
	}
	if len(extracted.Routes) != 1 || extracted.Routes[0].ResponseMediaType != "text/html" {
		t.Fatalf("routes = %#v", extracted.Routes)
	}
}

func TestExtractRejectsInvalidSource(t *testing.T) {
	result := Extract(map[string][]byte{"broken.go": []byte("not valid go")})
	if _, rejected := result.(ExtractionRejected); !rejected {
		t.Fatalf("result = %#v, want ExtractionRejected", result)
	}
}

func TestExtractRejectsWhenNoRoutesFound(t *testing.T) {
	result := Extract(map[string][]byte{"empty.go": []byte("package httpserver\n")})
	if _, rejected := result.(ExtractionRejected); !rejected {
		t.Fatalf("result = %#v, want ExtractionRejected", result)
	}
}

func TestExtractRejectsDuplicateRoutes(t *testing.T) {
	source := `package httpserver

import "net/http"

func newMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/tasks", server.listTasks)
	mux.HandleFunc("GET /api/tasks", server.listTasksAgain)
	return mux
}
`
	result := Extract(map[string][]byte{"server.go": []byte(source)})
	if _, rejected := result.(ExtractionRejected); !rejected {
		t.Fatalf("result = %#v, want ExtractionRejected", result)
	}
}
