package openapi

import (
	"go/ast"
	"sort"
	"strings"
)

// Status derivation: each operation's response status codes are read from the
// handler source instead of a hand-maintained table, so they cannot drift
// from what the handlers actually write.
//
// A success status is an `http.Status…` constant in the 2xx/3xx range passed
// as an argument to one of the response writers (writeJSON, a write<Foo>Response
// wrapper, ResponseWriter.WriteHeader, or http.Redirect). An error status is a
// 4xx `http.Status…` constant passed to writeError. Handlers that can reach
// writeDomainError produce the statusForError mapping's client-error statuses
// (400, 403, 404, 409); those are added wholesale because the domain error's
// code is a runtime value. Both sets propagate transitively through the local
// call graph, exactly like the auth-gateway detection, so a handler that
// delegates to a shared helper (changeTaskState, accountTokenDelivery.write)
// still reports the helper's statuses.

// successStatusCodes maps the http.Status… identifiers that count as success
// (or redirect) response codes.
var successStatusCodes = map[string]int{
	"StatusOK":                200,
	"StatusCreated":           201,
	"StatusAccepted":          202,
	"StatusNoContent":         204,
	"StatusMovedPermanently":  301,
	"StatusFound":             302,
	"StatusSeeOther":          303,
	"StatusTemporaryRedirect": 307,
	"StatusPermanentRedirect": 308,
}

// clientErrorStatusCodes maps the http.Status… identifiers that count as
// client-error response codes worth listing per operation. Server-side 5xx
// statuses stay under the uniform `default` error response.
var clientErrorStatusCodes = map[string]int{
	"StatusBadRequest":            400,
	"StatusUnauthorized":          401,
	"StatusForbidden":             403,
	"StatusNotFound":              404,
	"StatusNotAcceptable":         406,
	"StatusConflict":              409,
	"StatusRequestEntityTooLarge": 413,
	"StatusTooManyRequests":       429,
}

// domainErrorStatuses are the client-error statuses statusForError can map a
// domain rejection to (internal/http/server.go); 401 and 429 are reachable in
// principle but arrive through requireUserSubject/rate limiting in practice,
// so they are derived from those paths instead of being claimed everywhere.
var domainErrorStatuses = []int{400, 403, 404, 409}

// successWriterNames are the call targets whose status argument sets the
// response status directly.
var successWriterNames = map[string]bool{
	"writeJSON":   true,
	"WriteHeader": true,
	"Redirect":    true,
}

const errorWriterName = "writeError"

const domainErrorWriterName = "writeDomainError"

// functionStatuses is one function's directly written statuses plus whether
// it calls writeDomainError itself.
type functionStatuses struct {
	success         []int
	clientError     []int
	writesDomainErr bool
	writesErrorBody bool
}

// resolveResponseStatuses walks every function and returns, per function
// name, the transitively reachable success statuses, client-error statuses,
// and whether an error body can be written at all.
func resolveResponseStatuses(files map[string]*ast.File) (successByFunc map[string][]int, errorsByFunc map[string][]int, emitsErrorsByFunc map[string]bool) {
	direct := map[string]functionStatuses{}
	calls := map[string][]string{}
	for _, file := range files {
		for _, decl := range file.Decls {
			funcDecl, isFunc := decl.(*ast.FuncDecl)
			if !isFunc || funcDecl.Body == nil {
				continue
			}
			direct[funcDecl.Name.Name] = directStatuses(funcDecl.Body)
			calls[funcDecl.Name.Name] = calledNames(funcDecl.Body)
		}
	}

	successByFunc = map[string][]int{}
	errorsByFunc = map[string][]int{}
	emitsErrorsByFunc = map[string]bool{}
	for name := range direct {
		successSet := map[int]bool{}
		errorSet := map[int]bool{}
		emitsErrors := false
		collectStatuses(name, direct, calls, map[string]bool{}, successSet, errorSet, &emitsErrors)
		successByFunc[name] = sortedStatuses(successSet)
		errorsByFunc[name] = sortedStatuses(errorSet)
		emitsErrorsByFunc[name] = emitsErrors
	}
	return successByFunc, errorsByFunc, emitsErrorsByFunc
}

// collectStatuses folds the transitive closure of one function's statuses
// into the given sets.
func collectStatuses(name string, direct map[string]functionStatuses, calls map[string][]string, visited map[string]bool, successSet map[int]bool, errorSet map[int]bool, emitsErrors *bool) {
	if visited[name] {
		return
	}
	visited[name] = true
	statuses := direct[name]
	for _, status := range statuses.success {
		successSet[status] = true
	}
	for _, status := range statuses.clientError {
		errorSet[status] = true
	}
	if statuses.writesErrorBody {
		*emitsErrors = true
	}
	if statuses.writesDomainErr {
		*emitsErrors = true
		for _, status := range domainErrorStatuses {
			errorSet[status] = true
		}
	}
	for _, callee := range calls[name] {
		collectStatuses(callee, direct, calls, visited, successSet, errorSet, emitsErrors)
	}
}

// directStatuses reads the status constants one function body writes itself.
func directStatuses(body *ast.BlockStmt) functionStatuses {
	var statuses functionStatuses
	ast.Inspect(body, func(node ast.Node) bool {
		call, isCall := node.(*ast.CallExpr)
		if !isCall {
			return true
		}
		callee := calleeName(call)
		switch {
		case callee == errorWriterName:
			statuses.writesErrorBody = true
			for _, argument := range call.Args {
				if status, matched := httpStatusConstant(argument, clientErrorStatusCodes); matched {
					statuses.clientError = append(statuses.clientError, status)
				}
			}
		case callee == domainErrorWriterName:
			statuses.writesDomainErr = true
		case successWriterNames[callee] || (strings.HasPrefix(callee, "write") && callee != domainErrorWriterName):
			for _, argument := range call.Args {
				if status, matched := httpStatusConstant(argument, successStatusCodes); matched {
					statuses.success = append(statuses.success, status)
				}
			}
		}
		return true
	})
	return statuses
}

func calleeName(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return fn.Name
	case *ast.SelectorExpr:
		return fn.Sel.Name
	default:
		return ""
	}
}

// httpStatusConstant matches an `http.Status…` selector argument against the
// given identifier table.
func httpStatusConstant(expression ast.Expr, table map[string]int) (int, bool) {
	selector, isSelector := expression.(*ast.SelectorExpr)
	if !isSelector {
		return 0, false
	}
	packageName, isIdent := selector.X.(*ast.Ident)
	if !isIdent || packageName.Name != "http" {
		return 0, false
	}
	status, matched := table[selector.Sel.Name]
	return status, matched
}

// withStatus inserts a status into an ascending status list when absent.
func withStatus(statuses []int, status int) []int {
	for _, existing := range statuses {
		if existing == status {
			return statuses
		}
	}
	merged := append(append([]int{}, statuses...), status)
	sort.Ints(merged)
	return merged
}

func sortedStatuses(set map[int]bool) []int {
	statuses := make([]int, 0, len(set))
	for status := range set {
		statuses = append(statuses, status)
	}
	sort.Ints(statuses)
	return statuses
}
