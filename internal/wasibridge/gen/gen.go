// Package gen generates a WASI store bridge (the Dispatch host router and the
// GuestStore client) from a domain Store interface, so the per-method plumbing
// can never silently drift from the interface it serves. It introspects the
// interface's methods via go/ast and emits code that calls the hand-written
// per-type codecs in the bridge package (and the shared codecs in corewire); a
// type used by a method but missing from the codec registry is a generation
// error, not a silent gap.
//
// Each store is described by a storeSpec below. Adding a store means adding a
// spec (naming its codecs) and the hand-written codecs it references - the
// generator itself is store-agnostic.
package gen

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"sort"
	"strings"
)

// argCodec names how one method argument type crosses the bridge: the envelope
// field it occupies, its wire type, and the encode/decode functions (a bare
// name for a codec local to the bridge package, or a qualified corewire.X for a
// shared one). Decode returns (T, error).
//
// The field name is derived from the type, not the parameter (interface method
// params are usually unnamed in the AST), so a method with two arguments of the
// same type is unsupported - none of the bridged stores have one.
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

// storeSpec describes one bridge to generate.
type storeSpec struct {
	bridgePackage string
	domainImport  string
	domainPackage string
	interfaceName string
	wirePrefix    string
	argCodecs     map[string]argCodec
	resultCodecs  map[string]resultCodec
}

// Target names a store to (re)generate: where its interface source lives and
// where the generated bridge is written. Exported so the generate command can
// iterate every store without knowing the registry.
type Target struct {
	Key        string
	SourceDir  string
	OutputPath string
}

// Targets is the full set of stores the bridge codegen covers.
func Targets() []Target {
	return []Target{
		{Key: "audit", SourceDir: "internal/audit", OutputPath: "internal/wasibridge/auditbridge/bridge_gen.go"},
		{Key: "notification", SourceDir: "internal/notification", OutputPath: "internal/wasibridge/notificationbridge/bridge_gen.go"},
	}
}

func userIDArg() argCodec {
	return argCodec{field: "UserID", goType: "core.UserID", wireType: "string", encodeFn: "corewire.EncodeUserID", decodeFn: "corewire.DecodeUserID"}
}

func pageArg() argCodec {
	return argCodec{field: "Page", goType: "core.Page", wireType: "corewire.PageWire", encodeFn: "corewire.EncodePage", decodeFn: "corewire.DecodePage"}
}

var specs = map[string]storeSpec{
	"audit": {
		bridgePackage: "auditbridge",
		domainImport:  "github.com/e6qu/sharecrop/internal/audit",
		domainPackage: "audit",
		interfaceName: "Store",
		wirePrefix:    "audit",
		argCodecs: map[string]argCodec{
			"core.AuditEventID": {field: "ID", goType: "core.AuditEventID", wireType: "string", encodeFn: "corewire.EncodeAuditEventID", decodeFn: "corewire.DecodeAuditEventID"},
			"audit.Event":       {field: "Event", goType: "audit.Event", wireType: "eventWire", encodeFn: "encodeEvent", decodeFn: "decodeEvent"},
			"audit.ListFilters": {field: "Filters", goType: "audit.ListFilters", wireType: "listFiltersWire", encodeFn: "encodeListFilters", decodeFn: "decodeListFilters"},
			"core.Page":         pageArg(),
		},
		resultCodecs: map[string]resultCodec{
			"audit.RecordResult": {goType: "audit.RecordResult", wireType: "recordResultWire", encodeFn: "encodeRecordResult", decodeFn: "decodeRecordResult", rejectedType: "audit.RecordRejected"},
			"audit.GetResult":    {goType: "audit.GetResult", wireType: "getResultWire", encodeFn: "encodeGetResult", decodeFn: "decodeGetResult", rejectedType: "audit.GetRejected"},
			"audit.ListResult":   {goType: "audit.ListResult", wireType: "listResultWire", encodeFn: "encodeListResult", decodeFn: "decodeListResult", rejectedType: "audit.ListRejected"},
		},
	},
	"notification": {
		bridgePackage: "notificationbridge",
		domainImport:  "github.com/e6qu/sharecrop/internal/notification",
		domainPackage: "notification",
		interfaceName: "Store",
		wirePrefix:    "notification",
		argCodecs: map[string]argCodec{
			"notification.Notification": {field: "Notification", goType: "notification.Notification", wireType: "notificationWire", encodeFn: "encodeNotification", decodeFn: "decodeNotification"},
			"core.UserID":               userIDArg(),
			"core.Page":                 pageArg(),
			"core.NotificationID":       {field: "ID", goType: "core.NotificationID", wireType: "string", encodeFn: "corewire.EncodeNotificationID", decodeFn: "corewire.DecodeNotificationID"},
		},
		resultCodecs: map[string]resultCodec{
			"notification.CreateStoreResult":   {goType: "notification.CreateStoreResult", wireType: "createResultWire", encodeFn: "encodeCreateResult", decodeFn: "decodeCreateResult", rejectedType: "notification.CreateStoreRejected"},
			"notification.ListStoreResult":     {goType: "notification.ListStoreResult", wireType: "listResultWire", encodeFn: "encodeListResult", decodeFn: "decodeListResult", rejectedType: "notification.ListStoreRejected"},
			"notification.MarkReadStoreResult": {goType: "notification.MarkReadStoreResult", wireType: "markReadResultWire", encodeFn: "encodeMarkReadResult", decodeFn: "decodeMarkReadResult", rejectedType: "notification.MarkReadStoreRejected"},
		},
	},
}

type method struct {
	name   string
	args   []argCodec
	result resultCodec
}

// Generate parses the given package sources (path -> content) for the store
// named by key, extracts its Store interface, and returns the formatted bridge
// source. It fails if the key is unknown, the interface is not found, or a
// method uses an unregistered type.
func Generate(sources map[string][]byte, key string) (string, error) {
	spec, known := specs[key]
	if !known {
		return "", fmt.Errorf("no bridge spec for store %q", key)
	}
	methods, err := extractMethods(sources, spec)
	if err != nil {
		return "", err
	}
	return emit(spec, methods)
}

func extractMethods(sources map[string][]byte, spec storeSpec) ([]method, error) {
	fset := token.NewFileSet()

	var iface *ast.InterfaceType
	var packageName string
	for _, path := range sortedKeys(sources) {
		file, err := parser.ParseFile(fset, path, sources[path], 0)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		packageName = file.Name.Name
		ast.Inspect(file, func(node ast.Node) bool {
			typeSpec, matched := node.(*ast.TypeSpec)
			if !matched || typeSpec.Name.Name != spec.interfaceName {
				return true
			}
			if typed, isInterface := typeSpec.Type.(*ast.InterfaceType); isInterface {
				iface = typed
			}
			return true
		})
	}
	if iface == nil {
		return nil, fmt.Errorf("interface %q not found", spec.interfaceName)
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
			codec, known := spec.argCodecs[paramType]
			if !known {
				return nil, fmt.Errorf("method %s: no codec registered for argument type %q", name, paramType)
			}
			args = append(args, codec)
		}

		if funcType.Results == nil || len(funcType.Results.List) != 1 {
			return nil, fmt.Errorf("method %s: expected exactly one result", name)
		}
		resultType := qualify(typeString(funcType.Results.List[0].Type))
		result, known := spec.resultCodecs[resultType]
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

func constName(methodName string) string { return "method" + methodName }

func argsType(methodName string) string {
	return strings.ToLower(methodName[:1]) + methodName[1:] + "Args"
}

func paramName(field string) string { return "arg" + field }

func emit(spec storeSpec, methods []method) (string, error) {
	usesCorewire := false
	for _, m := range methods {
		for _, arg := range m.args {
			if strings.Contains(arg.encodeFn, "corewire.") || strings.Contains(arg.wireType, "corewire.") {
				usesCorewire = true
			}
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "// Code generated by \"sharecrop generate wasi-bridge\"; DO NOT EDIT.\n\npackage %s\n\n", spec.bridgePackage)
	b.WriteString("import (\n\t\"context\"\n\t\"encoding/json\"\n\t\"fmt\"\n\n")
	fmt.Fprintf(&b, "\t%q\n", spec.domainImport)
	b.WriteString("\t\"github.com/e6qu/sharecrop/internal/core\"\n")
	if usesCorewire {
		b.WriteString("\t\"github.com/e6qu/sharecrop/internal/wasibridge/corewire\"\n")
	}
	b.WriteString(")\n\n")

	fmt.Fprintf(&b, "// Method names namespace each %s.%s method on the wire.\nconst (\n", spec.domainPackage, spec.interfaceName)
	for _, m := range methods {
		fmt.Fprintf(&b, "\t%s = %q\n", constName(m.name), spec.wirePrefix+"."+m.name)
	}
	b.WriteString(")\n\n")

	for _, m := range methods {
		fmt.Fprintf(&b, "type %s struct {\n", argsType(m.name))
		for _, arg := range m.args {
			fmt.Fprintf(&b, "\t%s %s `json:%q`\n", arg.field, arg.wireType, strings.ToLower(arg.field))
		}
		b.WriteString("}\n\n")
	}

	emitDispatch(&b, spec, methods)
	emitGuestStore(&b, spec, methods)

	formatted, err := format.Source([]byte(b.String()))
	if err != nil {
		return "", fmt.Errorf("format generated bridge: %w", err)
	}
	return string(formatted), nil
}

func emitDispatch(b *strings.Builder, spec storeSpec, methods []method) {
	fmt.Fprintf(b, "// Dispatch services one store call against store: decode the arguments, call the\n"+
		"// real method, encode the result. Every branch is exactly that - no business\n"+
		"// logic lives here.\n"+
		"func Dispatch(ctx context.Context, store %s.%s, method string, args []byte) ([]byte, error) {\n"+
		"\tswitch method {\n", spec.domainPackage, spec.interfaceName)
	for _, m := range methods {
		fmt.Fprintf(b, "\tcase %s:\n", constName(m.name))
		fmt.Fprintf(b, "\t\tvar decoded %s\n", argsType(m.name))
		b.WriteString("\t\tif err := json.Unmarshal(args, &decoded); err != nil {\n")
		fmt.Fprintf(b, "\t\t\treturn nil, fmt.Errorf(%q, err)\n", spec.wirePrefix+" bridge: decode "+m.name+" args: %w")
		b.WriteString("\t\t}\n")
		callArgs := []string{"ctx"}
		for _, arg := range m.args {
			fmt.Fprintf(b, "\t\t%s, err := %s(decoded.%s)\n", paramName(arg.field), arg.decodeFn, arg.field)
			b.WriteString("\t\tif err != nil {\n\t\t\treturn nil, err\n\t\t}\n")
			callArgs = append(callArgs, paramName(arg.field))
		}
		fmt.Fprintf(b, "\t\treturn json.Marshal(%s(store.%s(%s)))\n", m.result.encodeFn, m.name, strings.Join(callArgs, ", "))
	}
	fmt.Fprintf(b, "\tdefault:\n\t\treturn nil, fmt.Errorf(%q, method)\n\t}\n}\n\n", spec.wirePrefix+" bridge: unknown method %q")
}

func emitGuestStore(b *strings.Builder, spec storeSpec, methods []method) {
	fmt.Fprintf(b, "// Invoker sends a store call to the host and returns the serialized result. The\n"+
		"// guest supplies rpc.Invoke; a test can supply an in-process stand-in.\n"+
		"type Invoker func(method string, args []byte) ([]byte, error)\n\n"+
		"// GuestStore implements %s.%s by forwarding each call over an Invoker to\n"+
		"// the host, which services it against the real store. Context is not carried\n"+
		"// across the bridge; the host uses its own context for the real call.\n"+
		"type GuestStore struct {\n\tinvoke Invoker\n}\n\n"+
		"// NewGuestStore builds a GuestStore over the given invoker.\n"+
		"func NewGuestStore(invoke Invoker) GuestStore {\n\treturn GuestStore{invoke: invoke}\n}\n\n",
		spec.domainPackage, spec.interfaceName)

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

	fmt.Fprintf(b, "// guestError wraps a transport/serialization failure as a domain rejection so a\n"+
		"// guest-side call always returns a well-formed result.\n"+
		"func guestError(err error) core.DomainError {\n"+
		"\treturn core.NewDomainError(core.ErrorCodeInvalidState, %q+err.Error())\n}\n\n",
		spec.wirePrefix+" bridge: ")

	fmt.Fprintf(b, "// GuestStore must satisfy the real Store interface - if a method is added to\n"+
		"// %s.%s and the bridge is not regenerated, this fails to compile.\n"+
		"var _ %s.%s = GuestStore{}\n", spec.domainPackage, spec.interfaceName, spec.domainPackage, spec.interfaceName)
}

func sortedKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
