package openapi

import (
	"go/ast"
	"reflect"
	"strconv"
	"strings"
)

// FieldKind classifies a DTO struct field for schema generation. Unknown
// covers anything this package does not confidently classify (maps,
// interfaces, imported types, pointers to unknown types): those fields are
// emitted as an unconstrained schema rather than a guessed type.
type FieldKind int

const (
	FieldUnknown FieldKind = iota
	FieldString
	FieldInteger
	FieldNumber
	FieldBoolean
	FieldArray
	FieldStruct
)

// FieldShape describes one JSON-tagged field of a DTO struct.
type FieldShape struct {
	JSONName string
	Required bool
	Kind     FieldKind
	// StructName is set when Kind is FieldStruct (the field's own type) or
	// when Kind is FieldArray and ElemKind is FieldStruct (the element
	// type).
	StructName string
	ElemKind   FieldKind
}

// StructShape describes the JSON wire shape of one Go DTO struct.
type StructShape struct {
	Fields []FieldShape
}

// collectStructShapes finds every top-level `type X struct { ... }`
// declaration across the given files and records its JSON field shape. Only
// exported-to-JSON fields (those with a `json:"..."` tag and no `json:"-"`)
// are recorded.
func collectStructShapes(files map[string]*ast.File) map[string]StructShape {
	shapes := map[string]StructShape{}
	for _, file := range files {
		for _, decl := range file.Decls {
			genDecl, isGenDecl := decl.(*ast.GenDecl)
			if !isGenDecl {
				continue
			}
			for _, spec := range genDecl.Specs {
				typeSpec, isTypeSpec := spec.(*ast.TypeSpec)
				if !isTypeSpec {
					continue
				}
				structType, isStruct := typeSpec.Type.(*ast.StructType)
				if !isStruct {
					continue
				}
				shapes[typeSpec.Name.Name] = structShapeFromAST(structType)
			}
		}
	}
	return shapes
}

func structShapeFromAST(structType *ast.StructType) StructShape {
	shape := StructShape{}
	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 || field.Tag == nil {
			continue
		}
		jsonName, required, skip := parseJSONTag(field.Tag.Value, field.Names[0].Name)
		if skip {
			continue
		}
		kind, structName, elemKind := fieldKindFromExpr(field.Type)
		shape.Fields = append(shape.Fields, FieldShape{
			JSONName:   jsonName,
			Required:   required,
			Kind:       kind,
			StructName: structName,
			ElemKind:   elemKind,
		})
	}
	return shape
}

// parseJSONTag reads the `json:"..."` struct tag, returning the wire field
// name, whether the field is required (no `,omitempty`), and whether the
// field is excluded from JSON entirely (`json:"-"`).
func parseJSONTag(rawTag string, goName string) (jsonName string, required bool, skip bool) {
	unquoted, err := strconv.Unquote(rawTag)
	if err != nil {
		return goName, true, false
	}
	tagValue, ok := reflect.StructTag(unquoted).Lookup("json")
	if !ok {
		return goName, true, false
	}
	parts := strings.Split(tagValue, ",")
	name := parts[0]
	if name == "-" && len(parts) == 1 {
		return "", false, true
	}
	if name == "" {
		name = goName
	}
	omitempty := false
	for _, option := range parts[1:] {
		if option == "omitempty" {
			omitempty = true
		}
	}
	return name, !omitempty, false
}

func fieldKindFromExpr(expr ast.Expr) (kind FieldKind, structName string, elemKind FieldKind) {
	switch value := expr.(type) {
	case *ast.Ident:
		return primitiveKind(value.Name), value.Name, FieldUnknown
	case *ast.StarExpr:
		return fieldKindFromExpr(value.X)
	case *ast.ArrayType:
		elem, elemStruct, _ := fieldKindFromExpr(value.Elt)
		if elem == FieldStruct {
			return FieldArray, elemStruct, FieldStruct
		}
		return FieldArray, "", elem
	default:
		return FieldUnknown, "", FieldUnknown
	}
}

func primitiveKind(name string) FieldKind {
	switch name {
	case "string":
		return FieldString
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		return FieldInteger
	case "float32", "float64":
		return FieldNumber
	case "bool":
		return FieldBoolean
	default:
		// An unrecognized named type is treated as a struct reference; the
		// caller looks it up in the struct registry and falls back to an
		// unconstrained schema if it isn't one.
		return FieldStruct
	}
}
