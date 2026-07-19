// Package openapi generates a machine-readable OpenAPI document from the
// route registrations in internal/http/server.go, instead of hand-authoring
// one that can drift from the actual mux table.
package openapi

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
)

// Route describes one HTTP route registered on the production mux.
// RequestType and ResponseType are the Go DTO struct names (keys into
// Extracted.Structs) that the handler decodes/writes, resolved on a
// best-effort basis; either is empty when the handler does not match one of
// the standard decode/write patterns.
type Route struct {
	Method           string
	Path             string
	OperationID      string
	RequiresAuth     bool
	RequestMediaType string
	RequestType      string
	ResponseType     string
}

type ExtractResult interface {
	extractResult()
}

type Extracted struct {
	Routes  []Route
	Structs map[string]StructShape
}

type ExtractionRejected struct {
	Reason string
}

func (Extracted) extractResult() {}

func (ExtractionRejected) extractResult() {}

// authGateways lists the Server methods that resolve and verify a caller's
// identity from the request (internal/http/server.go, admin_config.go,
// org_credits.go, agent_mcp.go). Each of these itself calls
// requireUserSubject, the single place that parses the Authorization
// header. A handler requires caller identity if it can reach one of these
// through the local call graph, directly or through a shared helper such as
// changeTaskState.
var authGateways = []string{
	"requireUserSubject",
	"requireWorkerSubject",
	"requireAdminSubject",
	"requireOrganizationBilling",
	"verifyAgent",
}

// Extract parses the given internal/http Go source files (path -> content,
// non-test files only) and returns every route registered on the mux in
// server.go, each annotated with whether its handler requires caller
// identity.
func Extract(sources map[string][]byte) ExtractResult {
	fset := token.NewFileSet()
	files := make(map[string]*ast.File, len(sources))
	for name, content := range sources {
		file, err := parser.ParseFile(fset, name, content, 0)
		if err != nil {
			return ExtractionRejected{Reason: "parse " + name + " failed: " + err.Error()}
		}
		files[name] = file
	}

	authGatedFuncs := collectAuthGatedFuncs(files)
	formBodyFuncs := collectFormBodyFuncs(files)
	requestTypeByFunc, responseTypeByFunc := resolveDTOTypes(files)
	structs := collectStructShapes(files)

	routesByKey := map[string]Route{}
	for _, file := range files {
		for _, route := range routesInFile(file) {
			key := route.Method + " " + route.Path
			if _, duplicate := routesByKey[key]; duplicate {
				return ExtractionRejected{Reason: "duplicate route registration for " + key}
			}
			route.RequiresAuth = authGatedFuncs[route.OperationID]
			if formBodyFuncs[route.OperationID] {
				route.RequestMediaType = "application/x-www-form-urlencoded"
			}
			route.RequestType = requestTypeByFunc[route.OperationID]
			route.ResponseType = responseTypeByFunc[route.OperationID]
			routesByKey[key] = route
		}
	}
	if len(routesByKey) == 0 {
		return ExtractionRejected{Reason: "no mux routes were found"}
	}

	routes := make([]Route, 0, len(routesByKey))
	for _, route := range routesByKey {
		routes = append(routes, route)
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Path != routes[j].Path {
			return routes[i].Path < routes[j].Path
		}
		return routes[i].Method < routes[j].Method
	})
	return Extracted{Routes: routes, Structs: structs}
}

func collectFormBodyFuncs(files map[string]*ast.File) map[string]bool {
	result := map[string]bool{}
	for _, file := range files {
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if ok && (selector.Sel.Name == "ParseForm" || selector.Sel.Name == "FormValue" || selector.Sel.Name == "PostFormValue") {
					result[function.Name.Name] = true
					return false
				}
				return true
			})
		}
	}
	return result
}

func routesInFile(file *ast.File) []Route {
	var routes []Route
	ast.Inspect(file, func(node ast.Node) bool {
		call, isCall := node.(*ast.CallExpr)
		if !isCall {
			return true
		}
		selector, isSelector := call.Fun.(*ast.SelectorExpr)
		if !isSelector {
			return true
		}
		if selector.Sel.Name != "HandleFunc" && selector.Sel.Name != "Handle" {
			return true
		}
		receiver, isIdent := selector.X.(*ast.Ident)
		if !isIdent || receiver.Name != "mux" || len(call.Args) != 2 {
			return true
		}
		pattern, isString := call.Args[0].(*ast.BasicLit)
		if !isString || pattern.Kind != token.STRING {
			return true
		}
		rawPattern := strings.Trim(pattern.Value, "\"")
		method, path, hasMethod := strings.Cut(rawPattern, " ")
		if !hasMethod {
			return true
		}
		routes = append(routes, Route{
			Method:      method,
			Path:        path,
			OperationID: handlerOperationID(call.Args[1]),
		})
		return true
	})
	return routes
}

func handlerOperationID(arg ast.Expr) string {
	switch value := arg.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.SelectorExpr:
		return value.Sel.Name
	case *ast.CallExpr:
		return handlerOperationID(value.Fun)
	default:
		return "handler"
	}
}

// collectAuthGatedFuncs builds a local call graph (function name -> the
// names of functions it calls, by identifier or selector, regardless of
// receiver) and returns the set of function names that can reach one of
// authGateways transitively. A textual/substring search over each function
// body would miss handlers that delegate to a shared helper (for example
// openTask/cancelTask both call changeTaskState, which is the one that
// actually calls requireUserSubject), so this walks the graph instead.
func collectAuthGatedFuncs(files map[string]*ast.File) map[string]bool {
	calls := map[string][]string{}
	for _, file := range files {
		for _, decl := range file.Decls {
			funcDecl, isFunc := decl.(*ast.FuncDecl)
			if !isFunc || funcDecl.Body == nil {
				continue
			}
			calls[funcDecl.Name.Name] = calledNames(funcDecl.Body)
		}
	}

	gated := map[string]bool{}
	for name := range calls {
		if reachesAuthGateway(name, calls, map[string]bool{}) {
			gated[name] = true
		}
	}
	return gated
}

func calledNames(body *ast.BlockStmt) []string {
	var names []string
	ast.Inspect(body, func(node ast.Node) bool {
		call, isCall := node.(*ast.CallExpr)
		if !isCall {
			return true
		}
		switch fn := call.Fun.(type) {
		case *ast.Ident:
			names = append(names, fn.Name)
		case *ast.SelectorExpr:
			names = append(names, fn.Sel.Name)
		}
		return true
	})
	return names
}

func reachesAuthGateway(name string, calls map[string][]string, visited map[string]bool) bool {
	if visited[name] {
		return false
	}
	visited[name] = true
	for _, gateway := range authGateways {
		if name == gateway {
			return true
		}
	}
	for _, callee := range calls[name] {
		if reachesAuthGateway(callee, calls, visited) {
			return true
		}
	}
	return false
}
