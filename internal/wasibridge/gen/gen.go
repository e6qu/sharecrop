// Package gen generates the WASI store bridge (the Dispatch host router and the
// GuestStore client) from a domain Store interface, so the per-method plumbing
// can never silently drift from the interface it serves. It introspects the
// interface's methods via go/ast and emits code that calls the hand-written
// per-type codecs in the bridge package; a type used by a method but missing
// from the codec registry is a generation error, not a silent gap.
//
// This is Phase 3 of the WASI hosting effort and targets exactly one store
// (internal/audit). The registries below name the codecs the audit types need;
// broadening to more stores means extending them (and the hand-written codecs)
// for those stores' types.
package gen

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
)

// argCodec names how one method argument type crosses the bridge: the envelope
// field it occupies, its wire type, and the encode/decode functions in the
// bridge package. Decode always returns (T, error) for the audit types.
type argCodec struct {
	field    string
	goType   string
	wireType string
	encodeFn string
	decodeFn string
}

// resultCodec names how one method result type crosses the bridge.
type resultCodec struct {
	goType       string
	wireType     string
	encodeFn     string
	decodeFn     string
	rejectedType string
}

// argCodecs and resultCodecs are the per-type registry for the audit store. A
// method argument or result type absent here fails generation loudly.
var argCodecs = map[string]argCodec{
	"core.AuditEventID": {field: "ID", goType: "core.AuditEventID", wireType: "string", encodeFn: "encodeAuditEventID", decodeFn: "decodeAuditEventID"},
	"audit.Event":       {field: "Event", goType: "audit.Event", wireType: "eventWire", encodeFn: "encodeEvent", decodeFn: "decodeEvent"},
	"audit.ListFilters": {field: "Filters", goType: "audit.ListFilters", wireType: "listFiltersWire", encodeFn: "encodeListFilters", decodeFn: "decodeListFilters"},
	"core.Page":         {field: "Page", goType: "core.Page", wireType: "pageWire", encodeFn: "encodePage", decodeFn: "decodePage"},
}

var resultCodecs = map[string]resultCodec{
	"audit.RecordResult": {goType: "audit.RecordResult", wireType: "recordResultWire", encodeFn: "encodeRecordResult", decodeFn: "decodeRecordResult", rejectedType: "audit.RecordRejected"},
	"audit.GetResult":    {goType: "audit.GetResult", wireType: "getResultWire", encodeFn: "encodeGetResult", decodeFn: "decodeGetResult", rejectedType: "audit.GetRejected"},
	"audit.ListResult":   {goType: "audit.ListResult", wireType: "listResultWire", encodeFn: "encodeListResult", decodeFn: "decodeListResult", rejectedType: "audit.ListRejected"},
}

type method struct {
	name   string
	args   []argCodec
	result resultCodec
}

// Generate parses the given package sources (path -> content), extracts the
// named interface's methods, and returns the formatted bridge source. It fails
// if the interface is not found or a method uses an unregistered type.
func Generate(sources map[string][]byte, interfaceName string) (string, error) {
	methods, err := extractMethods(sources, interfaceName)
	if err != nil {
		return "", err
	}
	return emit(methods)
}

func extractMethods(sources map[string][]byte, interfaceName string) ([]method, error) {
	fset := token.NewFileSet()

	var iface *ast.InterfaceType
	var packageName string
	// Sort paths for deterministic parsing order; the interface lives in one
	// file, but keep the walk stable regardless.
	for _, path := range sortedKeys(sources) {
		file, err := parser.ParseFile(fset, path, sources[path], 0)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		packageName = file.Name.Name
		ast.Inspect(file, func(node ast.Node) bool {
			spec, matched := node.(*ast.TypeSpec)
			if !matched || spec.Name.Name != interfaceName {
				return true
			}
			if typed, isInterface := spec.Type.(*ast.InterfaceType); isInterface {
				iface = typed
			}
			return true
		})
	}
	if iface == nil {
		return nil, fmt.Errorf("interface %q not found", interfaceName)
	}

	// Types local to the interface's own package appear unqualified in the AST
	// (e.g. "Event", not "audit.Event"); qualify them so registry lookups match.
	qualify := func(typeName string) string {
		if strings.Contains(typeName, ".") {
			return typeName
		}
		return packageName + "." + typeName
	}

	methods := make([]method, 0, len(iface.Methods.List))
	for _, field := range iface.Methods.List {
		if len(field.Names) != 1 {
			continue
		}
		name := field.Names[0].Name
		funcType, isFunc := field.Type.(*ast.FuncType)
		if !isFunc {
			return nil, fmt.Errorf("member %q is not a method", name)
		}

		args := make([]argCodec, 0)
		for _, param := range funcType.Params.List {
			paramType := qualify(typeString(param.Type))
			if paramType == "context.Context" {
				continue
			}
			codec, known := argCodecs[paramType]
			if !known {
				return nil, fmt.Errorf("method %s: no codec registered for argument type %q", name, paramType)
			}
			args = append(args, codec)
		}

		if funcType.Results == nil || len(funcType.Results.List) != 1 {
			return nil, fmt.Errorf("method %s: expected exactly one result", name)
		}
		resultType := qualify(typeString(funcType.Results.List[0].Type))
		result, known := resultCodecs[resultType]
		if !known {
			return nil, fmt.Errorf("method %s: no codec registered for result type %q", name, resultType)
		}

		methods = append(methods, method{name: name, args: args, result: result})
	}
	return methods, nil
}

// typeString renders a type expression as source text (e.g. "core.AuditEventID").
func typeString(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.SelectorExpr:
		return typeString(typed.X) + "." + typed.Sel.Name
	case *ast.StarExpr:
		return "*" + typeString(typed.X)
	case *ast.ArrayType:
		return "[]" + typeString(typed.Elt)
	default:
		return fmt.Sprintf("<unsupported %T>", expr)
	}
}

func constName(method string) string { return "method" + method }

func argsType(method string) string {
	return strings.ToLower(method[:1]) + method[1:] + "Args"
}

func paramName(field string) string { return "arg" + field }

func emit(methods []method) (string, error) {
	var b strings.Builder
	b.WriteString(`// Code generated by "sharecrop generate wasi-bridge"; DO NOT EDIT.

package auditbridge

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/core"
)

// Method names namespace each audit.Store method on the wire.
const (
`)
	for _, m := range methods {
		fmt.Fprintf(&b, "\t%s = %q\n", constName(m.name), "audit."+m.name)
	}
	b.WriteString(")\n\n")

	// Argument envelopes.
	for _, m := range methods {
		fmt.Fprintf(&b, "type %s struct {\n", argsType(m.name))
		for _, arg := range m.args {
			fmt.Fprintf(&b, "\t%s %s `json:%q`\n", arg.field, arg.wireType, strings.ToLower(arg.field))
		}
		b.WriteString("}\n\n")
	}

	emitDispatch(&b, methods)
	emitGuestStore(&b, methods)

	formatted, err := format.Source([]byte(b.String()))
	if err != nil {
		return "", fmt.Errorf("format generated bridge: %w", err)
	}
	return string(formatted), nil
}

func emitDispatch(b *strings.Builder, methods []method) {
	b.WriteString(`// Dispatch services one store call against store: decode the arguments, call the
// real method, encode the result. Every branch is exactly that - no business
// logic lives here.
func Dispatch(ctx context.Context, store audit.Store, method string, args []byte) ([]byte, error) {
	switch method {
`)
	for _, m := range methods {
		fmt.Fprintf(b, "\tcase %s:\n", constName(m.name))
		fmt.Fprintf(b, "\t\tvar decoded %s\n", argsType(m.name))
		b.WriteString("\t\tif err := json.Unmarshal(args, &decoded); err != nil {\n")
		fmt.Fprintf(b, "\t\t\treturn nil, fmt.Errorf(%q, err)\n", "audit bridge: decode "+m.name+" args: %w")
		b.WriteString("\t\t}\n")
		callArgs := []string{"ctx"}
		for _, arg := range m.args {
			fmt.Fprintf(b, "\t\t%s, err := %s(decoded.%s)\n", paramName(arg.field), arg.decodeFn, arg.field)
			b.WriteString("\t\tif err != nil {\n\t\t\treturn nil, err\n\t\t}\n")
			callArgs = append(callArgs, paramName(arg.field))
		}
		fmt.Fprintf(b, "\t\treturn json.Marshal(%s(store.%s(%s)))\n", m.result.encodeFn, m.name, strings.Join(callArgs, ", "))
	}
	b.WriteString(`	default:
		return nil, fmt.Errorf("audit bridge: unknown method %q", method)
	}
}

`)
}

func emitGuestStore(b *strings.Builder, methods []method) {
	b.WriteString(`// Invoker sends a store call to the host and returns the serialized result. The
// guest supplies rpc.Invoke; a test can supply an in-process stand-in.
type Invoker func(method string, args []byte) ([]byte, error)

// GuestStore implements audit.Store by forwarding each call over an Invoker to
// the host, which services it against the real store. Context is not carried
// across the bridge; the host uses its own context for the real call.
type GuestStore struct {
	invoke Invoker
}

// NewGuestStore builds a GuestStore over the given invoker.
func NewGuestStore(invoke Invoker) GuestStore {
	return GuestStore{invoke: invoke}
}

`)
	for _, m := range methods {
		params := make([]string, 0, len(m.args))
		fields := make([]string, 0, len(m.args))
		for _, arg := range m.args {
			params = append(params, paramName(arg.field)+" "+arg.goType)
			fields = append(fields, arg.field+": "+arg.encodeFn+"("+paramName(arg.field)+")")
		}
		signature := "ctx context.Context"
		if len(params) > 0 {
			signature += ", " + strings.Join(params, ", ")
		}
		reject := m.result.rejectedType + "{Reason: guestError(err)}"

		fmt.Fprintf(b, "func (g GuestStore) %s(%s) %s {\n", m.name, signature, m.result.goType)
		fmt.Fprintf(b, "\targs, err := json.Marshal(%s{%s})\n", argsType(m.name), strings.Join(fields, ", "))
		fmt.Fprintf(b, "\tif err != nil {\n\t\treturn %s\n\t}\n", reject)
		fmt.Fprintf(b, "\traw, err := g.invoke(%s, args)\n", constName(m.name))
		fmt.Fprintf(b, "\tif err != nil {\n\t\treturn %s\n\t}\n", reject)
		fmt.Fprintf(b, "\tvar wire %s\n", m.result.wireType)
		fmt.Fprintf(b, "\tif err := json.Unmarshal(raw, &wire); err != nil {\n\t\treturn %s\n\t}\n", reject)
		fmt.Fprintf(b, "\tresult, err := %s(wire)\n", m.result.decodeFn)
		fmt.Fprintf(b, "\tif err != nil {\n\t\treturn %s\n\t}\n", reject)
		b.WriteString("\treturn result\n}\n\n")
	}

	b.WriteString(`// guestError wraps a transport/serialization failure as a domain rejection so a
// guest-side call always returns a well-formed result.
func guestError(err error) core.DomainError {
	return core.NewDomainError(core.ErrorCodeInvalidState, "audit bridge: "+err.Error())
}

// GuestStore must satisfy the real Store interface - if a method is added to
// audit.Store and the bridge is not regenerated, this fails to compile.
var _ audit.Store = GuestStore{}
`)
}

func sortedKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	// insertion sort keeps this dependency-free and deterministic
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j-1] > keys[j]; j-- {
			keys[j-1], keys[j] = keys[j], keys[j-1]
		}
	}
	return keys
}
