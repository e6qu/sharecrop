package openapi

import (
	"go/ast"
	"go/token"
)

// resolveDTOTypes returns, for every function declared across the given
// files, the DTO struct type name it decodes as a request body, when there
// is one, and the DTO struct type name it writes as a response body, when
// there is one. Both maps are best-effort: a function that does not match
// one of the standard
// decode/write patterns used throughout internal/http, directly or through
// a chain of helpers it calls, is simply absent, and callers fall back to
// an unconstrained schema rather than guessing.
//
// The patterns recognized, checked on the function itself and then walked
// transitively through the local call graph (many handlers, such as
// openTask/cancelTask, delegate to a single shared helper that does the
// actual decode/write):
//   - request: `var request <Type>` followed by
//     `json.NewDecoder(r.Body).Decode(&request)`.
//   - response: a dedicated `write<Foo>Response(w http.ResponseWriter, status
//     int, response <Type>)` wrapper, or a direct `writeJSON(w, status,
//     value)` call where `value` is a composite literal, a call to a local
//     single-return-value converter function, or a local variable assigned
//     from either of those.
func resolveDTOTypes(files map[string]*ast.File) (requestTypeByFunc, responseTypeByFunc map[string]string) {
	funcBodies := map[string]*ast.BlockStmt{}
	funcReturnType := map[string]string{}
	writeWrapperResponseType := map[string]string{}

	for _, file := range files {
		for _, decl := range file.Decls {
			funcDecl, isFunc := decl.(*ast.FuncDecl)
			if !isFunc || funcDecl.Body == nil {
				continue
			}
			funcBodies[funcDecl.Name.Name] = funcDecl.Body
			if returnType, ok := singleNamedReturnType(funcDecl); ok {
				funcReturnType[funcDecl.Name.Name] = returnType
			}
			if responseType, ok := writeWrapperParamType(funcDecl); ok {
				writeWrapperResponseType[funcDecl.Name.Name] = responseType
			}
		}
	}

	directRequestType := map[string]string{}
	directResponseType := map[string]string{}
	for name, body := range funcBodies {
		if requestType, ok := decodedRequestType(body, funcReturnType); ok {
			directRequestType[name] = requestType
		}
		if responseType, ok := writtenResponseType(body, writeWrapperResponseType, funcReturnType); ok {
			directResponseType[name] = responseType
		}
	}

	calls := map[string][]string{}
	for name, body := range funcBodies {
		calls[name] = calledNames(body)
	}

	requestTypeByFunc = map[string]string{}
	responseTypeByFunc = map[string]string{}
	for name := range funcBodies {
		if requestType, ok := transitiveLookup(name, calls, directRequestType); ok {
			requestTypeByFunc[name] = requestType
		}
		if responseType, ok := transitiveLookup(name, calls, directResponseType); ok {
			responseTypeByFunc[name] = responseType
		}
	}
	return requestTypeByFunc, responseTypeByFunc
}

// transitiveLookup returns direct[name] if present, otherwise searches the
// functions name calls (and what they call, and so on) for the first one
// present in direct, in call order. visited (implicit, one per top-level
// call) prevents infinite recursion on mutually recursive helpers.
func transitiveLookup(name string, calls map[string][]string, direct map[string]string) (string, bool) {
	return transitiveLookupVisiting(name, calls, direct, map[string]bool{})
}

func transitiveLookupVisiting(name string, calls map[string][]string, direct map[string]string, visited map[string]bool) (string, bool) {
	if visited[name] {
		return "", false
	}
	visited[name] = true
	if value, ok := direct[name]; ok {
		return value, true
	}
	for _, callee := range calls[name] {
		if value, ok := transitiveLookupVisiting(callee, calls, direct, visited); ok {
			return value, true
		}
	}
	return "", false
}

func singleNamedReturnType(funcDecl *ast.FuncDecl) (string, bool) {
	if funcDecl.Type.Results == nil || len(funcDecl.Type.Results.List) != 1 {
		return "", false
	}
	ident, isIdent := funcDecl.Type.Results.List[0].Type.(*ast.Ident)
	if !isIdent || isGoBuiltinPrimitive(ident.Name) {
		return "", false
	}
	return ident.Name, true
}

// writeWrapperParamType matches the write<Foo>Response(w http.ResponseWriter,
// status int, response <Type>) pattern. It excludes builtin-primitive third
// parameters so generic helpers with the same shape but no DTO payload (for
// example writeError(w, status, message string)) are not mistaken for a
// response-type wrapper, and excludes writeJSON itself: its third parameter
// is the writableResponse marker interface, not a concrete DTO, and callers
// of writeJSON are resolved from the actual argument at each call site
// instead.
func writeWrapperParamType(funcDecl *ast.FuncDecl) (string, bool) {
	if funcDecl.Name.Name == "writeJSON" || funcDecl.Type.Params == nil || len(funcDecl.Type.Params.List) != 3 {
		return "", false
	}
	params := funcDecl.Type.Params.List
	if !isHTTPResponseWriterType(params[0].Type) || !isIdentType(params[1].Type, "int") {
		return "", false
	}
	responseIdent, isIdent := params[2].Type.(*ast.Ident)
	if !isIdent || isGoBuiltinPrimitive(responseIdent.Name) {
		return "", false
	}
	return responseIdent.Name, true
}

func isGoBuiltinPrimitive(name string) bool {
	switch name {
	case "string", "bool",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "byte", "rune", "error":
		return true
	default:
		return false
	}
}

func isHTTPResponseWriterType(expr ast.Expr) bool {
	selector, isSelector := expr.(*ast.SelectorExpr)
	if !isSelector {
		return false
	}
	x, isIdent := selector.X.(*ast.Ident)
	return isIdent && x.Name == "http" && selector.Sel.Name == "ResponseWriter"
}

func isIdentType(expr ast.Expr, name string) bool {
	ident, isIdent := expr.(*ast.Ident)
	return isIdent && ident.Name == name
}

// decodedRequestType finds a local `<name> <Type>` declaration or assignment
// in body whose variable is later passed to a `.Decode(&<name>)` call, and
// returns <Type>.
func decodedRequestType(body *ast.BlockStmt, returnTypes map[string]string) (string, bool) {
	declaredTypes := localAssignedTypes(body, returnTypes)

	var found string
	ast.Inspect(body, func(node ast.Node) bool {
		if found != "" {
			return false
		}
		call, isCall := node.(*ast.CallExpr)
		if !isCall {
			return true
		}
		selector, isSelector := call.Fun.(*ast.SelectorExpr)
		if !isSelector || selector.Sel.Name != "Decode" || len(call.Args) != 1 {
			return true
		}
		unary, isUnary := call.Args[0].(*ast.UnaryExpr)
		if !isUnary || unary.Op != token.AND {
			return true
		}
		ident, isIdent := unary.X.(*ast.Ident)
		if !isIdent {
			return true
		}
		if typeName, ok := declaredTypes[ident.Name]; ok {
			found = typeName
		}
		return true
	})
	return found, found != ""
}

// writtenResponseType finds the DTO type body writes as its HTTP response,
// either through a dedicated write wrapper or a direct writeJSON call.
func writtenResponseType(body *ast.BlockStmt, wrapperTypes, returnTypes map[string]string) (string, bool) {
	declaredTypes := localAssignedTypes(body, returnTypes)

	var found string
	ast.Inspect(body, func(node ast.Node) bool {
		if found != "" {
			return false
		}
		call, isCall := node.(*ast.CallExpr)
		if !isCall {
			return true
		}
		funcName, hasName := calledFuncName(call.Fun)
		if !hasName {
			return true
		}
		if wrapperType, ok := wrapperTypes[funcName]; ok {
			found = wrapperType
			return false
		}
		if funcName != "writeJSON" || len(call.Args) != 3 {
			return true
		}
		if responseType, ok := resolveValueType(call.Args[2], declaredTypes, returnTypes); ok {
			found = responseType
		}
		return true
	})
	return found, found != ""
}

// localAssignedTypes walks body in source order, recording the resolved DTO
// type of every `name := ...` or `var name Type` local so that a later
// `writeJSON(w, status, name)` can look the type back up.
func localAssignedTypes(body *ast.BlockStmt, returnTypes map[string]string) map[string]string {
	types := map[string]string{}
	ast.Inspect(body, func(node ast.Node) bool {
		switch stmt := node.(type) {
		case *ast.AssignStmt:
			if stmt.Tok != token.DEFINE || len(stmt.Lhs) != 1 || len(stmt.Rhs) != 1 {
				return true
			}
			ident, isIdent := stmt.Lhs[0].(*ast.Ident)
			if !isIdent {
				return true
			}
			if typeName, ok := resolveValueType(stmt.Rhs[0], types, returnTypes); ok {
				types[ident.Name] = typeName
			}
		case *ast.DeclStmt:
			genDecl, isGenDecl := stmt.Decl.(*ast.GenDecl)
			if !isGenDecl || genDecl.Tok != token.VAR {
				return true
			}
			for _, spec := range genDecl.Specs {
				valueSpec, isValueSpec := spec.(*ast.ValueSpec)
				if !isValueSpec {
					continue
				}
				typeIdent, isIdent := valueSpec.Type.(*ast.Ident)
				if !isIdent {
					continue
				}
				for _, name := range valueSpec.Names {
					types[name.Name] = typeIdent.Name
				}
			}
		}
		return true
	})
	return types
}

func resolveValueType(expr ast.Expr, declaredTypes, returnTypes map[string]string) (string, bool) {
	switch value := expr.(type) {
	case *ast.CompositeLit:
		if ident, isIdent := value.Type.(*ast.Ident); isIdent {
			return ident.Name, true
		}
	case *ast.CallExpr:
		if funcName, ok := calledFuncName(value.Fun); ok {
			if returnType, ok := returnTypes[funcName]; ok {
				return returnType, true
			}
		}
	case *ast.Ident:
		if typeName, ok := declaredTypes[value.Name]; ok {
			return typeName, true
		}
	}
	return "", false
}

// calledFuncName returns the function or method name being called by a
// call expression's Fun, whether it is a bare identifier (writeJSON(...))
// or a method selector (server.writeAuthResponse(...)); the receiver is
// irrelevant since the resolution tables are keyed by declared function
// name regardless of receiver.
func calledFuncName(fun ast.Expr) (string, bool) {
	switch value := fun.(type) {
	case *ast.Ident:
		return value.Name, true
	case *ast.SelectorExpr:
		return value.Sel.Name, true
	default:
		return "", false
	}
}
