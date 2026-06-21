package contracts

type ModuleName struct {
	value string
}

type ElmTypeName struct {
	value string
}

type ElmValueName struct {
	value string
}

type JSONFieldName struct {
	value string
}

func NewModuleName(value string) ModuleName {
	return ModuleName{value: value}
}

func NewElmTypeName(value string) ElmTypeName {
	return ElmTypeName{value: value}
}

func NewElmValueName(value string) ElmValueName {
	return ElmValueName{value: value}
}

func NewJSONFieldName(value string) JSONFieldName {
	return JSONFieldName{value: value}
}

func (name ModuleName) String() string {
	return name.value
}

func (name ElmTypeName) String() string {
	return name.value
}

func (name ElmValueName) String() string {
	return name.value
}

func (name JSONFieldName) String() string {
	return name.value
}

type TypeRef interface {
	typeRef()
}

type StringRef struct{}

type IntRef struct{}

type NamedRef struct {
	Name ElmTypeName
}

type ListRef struct {
	Element TypeRef
}

func (StringRef) typeRef() {}

func (IntRef) typeRef() {}

func (NamedRef) typeRef() {}

func (ListRef) typeRef() {}

type Field struct {
	Name     ElmValueName
	JSONName JSONFieldName
	Type     TypeRef
}

type Variant struct {
	Name ElmTypeName
	Tag  string
}

type Definition interface {
	definition()
}

type Alias struct {
	Name ElmTypeName
	Type TypeRef
}

type Enum struct {
	Name     ElmTypeName
	Variants []Variant
}

type Product struct {
	Name   ElmTypeName
	Fields []Field
}

func (Alias) definition() {}

func (Enum) definition() {}

func (Product) definition() {}

type Module struct {
	Name        ModuleName
	Definitions []Definition
}
