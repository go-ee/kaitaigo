package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/iancoleman/strcase"
)

type Meta struct {
	ID            string `yaml:"id,omitempty"`
	Title         string `yaml:"title,omitempty"`
	Application   string `yaml:"application,omitempty"`
	Imports       string `yaml:"imports,omitempty"`
	Encoding      string `yaml:"encoding,omitempty"`
	Endian        string `yaml:"endian,omitempty"`
	KSVersion     string `yaml:"ks-version,omitempty"`
	KSDebug       string `yaml:"ks-debug,omitempty"`
	KSOpaqueTypes string `yaml:"ksopaquetypes,omitempty"`
	Licence       string `yaml:"licence,omitempty"`
	FileExtension string `yaml:"fileextension,omitempty"`
}

var endian = "binary.LittleEndian"

var endianess = map[string]string{
	"le": "binary.LittleEndian",
	"be": "binary.BigEndian",
}

type TypeSwitch struct {
	SwitchOn string             `yaml:"switch-on,omitempty"`
	Cases    map[string]TypeKey `yaml:"cases,omitempty"`
}

type TypeKey struct {
	Type       string
	TypeSwitch TypeSwitch
	CustomType bool
}

func (y *TypeKey) UnmarshalYAML(unmarshal func(interface{}) error) error {
	err := unmarshal(&y.Type)
	if err != nil {
		err = unmarshal(&y.TypeSwitch)
		return err
	}
	if _, ok := typeMapping[y.Type]; !ok {
		y.CustomType = true
	}
	return nil
}

func (y *TypeKey) String() string {
	if y.Type != "" {
		if val, ok := typeMapping[y.Type]; ok {
			return val
		}
		return strcase.ToCamel(y.Type)
	} else if y.TypeSwitch.SwitchOn != "" {
		return "runtime.Decoder"
	}
	return "[]byte"
}

type Contents struct {
	ContentString string
	ContentArray  []interface{}
	TypeSwitch    TypeSwitch
}

func (y *Contents) UnmarshalYAML(unmarshal func(interface{}) error) error {
	err := unmarshal(&y.ContentString)
	if err != nil {
		err := unmarshal(&y.ContentArray)
		if err != nil {
			err = unmarshal(&y.TypeSwitch)
			return err
		}
		return err
	}
	return nil
}

func (y *Contents) Len() int {
	if len(y.ContentString) != 0 {
		return len(y.ContentString)
	}
	if len(y.ContentArray) == 0 {
		return 0
	}
	switch v := y.ContentArray[0].(type) {
	case string:
		return len(v)
	default:
		return len(y.ContentArray)
	}
}

type Attribute struct {
	Category    string   `-`
	ID          string   `yaml:"id,omitempty"`
	Type        TypeKey  `yaml:"type"`
	Size        string   `yaml:"size,omitempty"`
	SizeEos     string   `yaml:"size-eos,omitempty"`
	Doc         string   `yaml:"doc,omitempty"`
	Repeat      string   `yaml:"repeat,omitempty"`
	RepeatExpr  string   `yaml:"repeat-expr,omitempty"`
	RepeatUntil string   `yaml:"repeat-until,omitempty"`
	Contents    Contents `yaml:"contents,omitempty"`
	Value       string   `yaml:"value,omitempty"`
	Pos         string   `yaml:"pos,omitempty"`
	Whence      string   `yaml:"whence,omitempty"`
	Enum        string   `yaml:"enum,omitempty"`
	If          string   `yaml:"if,omitempty"`
	Process     string   `yaml:"process,omitempty"`
	Terminator  string   `yaml:"terminator,omitempty"`
	Consume     string   `yaml:"consume,omitempty"`
	Include     string   `yaml:"include,omitempty"`
	EosError    string   `yaml:"eos-error,omitempty"`
	Pad         string   `yaml:"pad-right,omitempty"`
	// Encoding    string   `yaml:"encoding,omitempty"`
}

func (k *Attribute) Name() string {
	return strcase.ToLowerCamel(k.ID)
}

func (k *Attribute) ChildType() string {
	dataType := k.Type.String()
	if dataType == "[]byte" { // || dataType == "runtime.String" {
		if k.Value != "" {
			dataType = getType(k.Value)
		} else if k.Size != "" {
			k.Size = strings.Replace(k.Size, "%", "%%", -1)
		} else if k.Contents.Len() != 0 {
			k.Size = fmt.Sprintf("%d", k.Contents.Len())
		}
	}
	return dataType
}

func (k *Attribute) DataType() string {
	dataType := k.ChildType()
	if k.Repeat != "" {
		if !isInt(k.RepeatExpr) || k.Repeat == "eos" {
			dataType = "[]" + dataType
		} else {
			dataType = "[" + goExpr(k.RepeatExpr, "") + "]" + dataType
		}
	} else if k.Type.CustomType {
		dataType = "*" + dataType
	}
	return dataType
}

func (k *Attribute) String() string {
	doc := ""
	if k.Doc != "" {
		doc = "\n/* " + strings.TrimSpace(k.Doc) + "*/"
	}

	return k.Name() + " " + k.DataType() + "`ks:\"" + k.ID + "," + k.Category + "\"`" + doc
}

type Type struct {
	Meta      Meta                           `yaml:"meta,omitempty"`
	Types     map[string]Type                `yaml:"types,omitempty"`
	Seq       []Attribute                    `yaml:"seq,omitempty"`
	Enums     map[string]map[int]interface{} `yaml:"enums,omitempty"`
	Doc       string                         `yaml:"doc,omitempty"`
	Instances map[string]Attribute           `yaml:"instances,omitempty"`
}

func (k *Type) InitElem(attrHolder string, errHolder string, attr Attribute, dataType string, init bool) (goCode string) {
	var buffer LineBuffer

	// defer func() { buffer.WriteLine("k.Stream.SetErr(err)"); goCode = buffer.String() }()
	defer func() { goCode = buffer.String() }()

	// init and parse element
	if init && dataType != "[]byte" {

	}
	if dataType == "[]byte" && attr.Size != "" {
		//buffer.WriteLine(attrHolder + " = make([]byte, " + goExpr(attr.Size, "") + ")")
	}

	terminated := attr.Terminator != "" || attr.Type.Type == "strz"
	term := "byte(0)"
	if attr.Terminator != "" {
		term = goExpr(attr.Terminator, "")
	}
	resetPos := (attr.Size == "" && terminated) || attr.Size != ""

	if resetPos {
		// save position
		//buffer.WriteLine("pos, _ := k.Seek(0, io.SeekCurrent)")
	}

	// read data
	if isNative(dataType) {
		if strings.HasSuffix(attr.ID, "be") {
			endian = "binary.BigEndian"
		}
		if attr.SizeEos != "" {
			buffer.WriteLine(attrHolder + ", " + errHolder + " = k.ReadBytesFull()")
		} else if terminated {

			if attr.Size == "" {
				buffer.WriteLine(attrHolder + ", " + errHolder + " =k.ReadBytes(" + term + ")")
				buffer.WriteLine("if " + errHolder + " != nil && " + errHolder + " == io.EOF { " + errHolder + " = nil }")
			} else {
				// term & size
				buffer.WriteLine("_, " + errHolder + " = k.Stream.Read(" + attrHolder + ")")
			}

			// eos
			if attr.EosError == "" {
				attr.EosError = "true"
			}
			buffer.WriteLine("if " + errHolder + " != nil && (" + errHolder + " != io.EOF || " + goExpr(attr.EosError, "") + ") {")
			buffer.WriteLine("return")
			buffer.WriteLine("}")
			buffer.WriteLine(errHolder + " = nil")

		} else {
			buffer.WriteLine(fmt.Sprintf(attrHolder+", "+errHolder+" = k.%v", toReadFunc(&attr, "le")))
		}
	} else {
		if resetPos {
			buffer.WriteLine("var reader io.ReadSeeker")
			buffer.WriteLine("if reader, err = k.ReadBytesAsReader(k.Length()); err != nil {")
			buffer.WriteLine("return")
			buffer.WriteLine("}")
			buffer.WriteLine(attrHolder + ".Read(reader, lazy, k, k.Root())")
		} else {
			buffer.WriteLine(attrHolder + ".Read(k.Stream, lazy, k, k.Root())")
		}

		if attrHolder != "ret" {
			buffer.WriteLine("if " + attrHolder + ".DecodeErr != nil {")
			buffer.WriteLine(errHolder + " = " + attrHolder + ".DecodeErr")
			buffer.WriteLine("}")
		} else {
			buffer.WriteLine("if k.DecodeErr != nil {")
			buffer.WriteLine(errHolder + " = k.DecodeErr")
			buffer.WriteLine("}")
		}
	}

	// pad
	if attr.Pad != "" {
		buffer.WriteLine(attrHolder + " = bytes.TrimRight(" + attrHolder + ", string(" + goExpr(attr.Pad, "") + "))")
	}

	// term
	if terminated && attr.Size != "" {
		buffer.WriteLine("i := bytes.IndexByte(" + attrHolder + ", " + term + ")")
		buffer.WriteLine("if i != -1 {")
		buffer.WriteLine(attrHolder + " = " + attrHolder + "[:i+1]")
		buffer.WriteLine("}")
	}

	// calc new pos
	if resetPos {
		if attr.Size != "" {
			//buffer.WriteLine("pos = pos+int64(" + goExpr(attr.Size, "") + ")")
		} else if dataType == "[]byte" {
			//buffer.WriteLine("pos = pos+int64(len(elem))")
		}
	}

	// include
	// buffer.WriteLine("pos, _ := k.Seek(0, io.SeekCurrent)")
	// buffer.WriteLine("fmt.Println(pos, elem)")
	if (terminated) && attr.Include == "" { // && attr.Size == "" {
		buffer.WriteLine(attrHolder + " = " + attrHolder + "[:len(" + attrHolder + ")-1]")
	}

	// reset position
	if resetPos {
		if terminated && attr.Consume != "" {
			// consume
			buffer.WriteLine("if !" + goExpr(attr.Consume, "") + " {")
			//buffer.WriteLine("pos -= 1")
			buffer.WriteLine("}")
		}
		//buffer.WriteLine("_, err = k.Seek(pos, io.SeekStart)")
	}

	return
}

func (k *Type) CallAttr(attr Attribute) (ret string) {
	if isNative(attr.DataType()) {
		ret = "k.read" + strings.Title(attr.Name()) + "()"
	} else {
		ret = "k.read" + strings.Title(attr.Name()) + "(lazy)"
	}
	return
}

func (k *Type) InitAttr(attr Attribute, typeName string) (goCode string) {
	var buffer LineBuffer

	defer func() { goCode = buffer.String() }()

	if isNative(attr.DataType()) {
		buffer.WriteLine("func (k *" + typeName + ") read" + strings.Title(attr.Name()) + "() (ret " + attr.DataType() + ", err error){")
	} else {
		buffer.WriteLine("func (k *" + typeName + ") read" + strings.Title(attr.Name()) + "(lazy bool) (ret " + attr.DataType() + ", err error){")
	}

	var attrHolder, errHolder string
	if attr.Value == "" {
		attrHolder = "ret"
		errHolder = "err"
		//buffer.WriteLine("var elem " + attr.ChildType())
	}
	// buffer.WriteLine("elem = &" + attr.ChildType() + "{}")

	if attr.If != "" {
		buffer.WriteLine("if " + goExpr(attr.If, "") + "{")
		defer buffer.WriteLine("}") // end if
	}

	if attr.Value != "" {
		// value instance
		if attr.DataType() == "runtime.KSYDecoder" || strings.HasPrefix(attr.DataType(), "*") {
			buffer.WriteLine(attrHolder + " = " + goExpr(attr.Value, ""))
		} else {
			buffer.WriteLine(attrHolder + " = " + attr.DataType() + "(" + goExpr(attr.Value, "") + ")")
		}
		buffer.WriteLine("return")
		buffer.WriteLine("}")
		return
	}

	if attr.Pos != "" {
		// save position
		//buffer.WriteLine("pos" + attr.Name() + ", _ := k.Seek(0, io.SeekCurrent) // Cannot fail")
		whence := "io.SeekCurrent"
		whenceMap := map[string]string{
			"seek_set": "io.SeekStart",
			"seek_end": "io.SeekEnd",
			"seek_cur": "io.SeekCurrent",
		}
		if val, ok := whenceMap[attr.Whence]; ok {
			whence = val
		}
		if whence == "io.SeekCurrent" {
			//buffer.WriteLine("k.Seek(0, io.SeekStart)")
		}
		//buffer.WriteLine("_, err = k.Seek(" + goExpr(attr.Pos, "int64") + ", " + whence + ")")
		buffer.WriteLine("if " + errHolder + " != nil { return }")
		// restore position
		//defer buffer.WriteLine("k.Seek(pos" + attr.Name() + ", io.SeekStart)")
	}

	switch {
	case attr.Repeat != "":
		before := "true"
		until := ""
		fall := false
		switch attr.Repeat {
		case "expr":
			if attr.RepeatExpr == "" {
				panic("RepeatExpr is missing") // TODO: move to parsing
			}
			before = "index < int(" + goExpr(attr.RepeatExpr, "") + ")"
			fall = true
			fallthrough
		case "until":
			if !fall {
				if attr.RepeatUntil == "" {
					panic("RepeatUntil is missing") // TODO: move to parsing
				}
				until = goExprAttr(attr.RepeatUntil, "", attr.Name()+"[index]")
			}
			fallthrough
		case "eos":
			if attr.Value == "" {
				buffer.WriteLine("var elem " + attr.ChildType())
				attrHolder = "elem"
			}

			// slice
			if strings.HasPrefix(attr.DataType(), "[]") {
				//buffer.WriteLine(" ret = " + attr.DataType() + "{}")
			}

			buffer.WriteLine("for index := 0; " + before + "; index++ {")

			buffer.WriteString(k.InitElem(attrHolder, errHolder, attr, attr.ChildType(), true))

			// break on error
			buffer.WriteLine("if " + errHolder + " != nil {")
			buffer.WriteLine("if " + errHolder + " == io.EOF { " + errHolder + " = nil }")
			buffer.WriteLine("break")
			buffer.WriteLine("}")

			// add element
			if strings.HasPrefix(attr.DataType(), "[]") {
				buffer.WriteLine("ret = append(ret, " + attrHolder + ")")
			} else {
				buffer.WriteLine("k." + attr.Name() + "[index] = " + attrHolder)
			}

			// break on repeat-until
			if until != "" {
				buffer.WriteLine("if " + until + "{break}")
			}

			buffer.WriteLine("}")
			buffer.WriteLine("return")
			buffer.WriteLine("}")
			return
		}
	case attr.Type.CustomType:
		// custom struct
		// init variable
		// if attr.Size != "" {
		// 	buffer.WriteLine(attr.Name() + "pos, _ := k.Seek(0, io.SeekCurrent) // Cannot fail")
		// 	defer buffer.WriteLine("k.Seek(" + attr.Name() + "pos + " + goExpr(attr.Size, "int64") + ", io.SeekStart)")
		// }
		// buffer.WriteLine("k." + attr.Name() + " = &" + attr.DataType()[1:] + "{}")
	case attr.Type.TypeSwitch.SwitchOn != "":
		buffer.WriteLine("switch " + goExpr(attr.Type.TypeSwitch.SwitchOn, "") + " {")
		for casevalue, casetype := range attr.Type.TypeSwitch.Cases {
			if casevalue == "_" {
				buffer.WriteLine("default:")
			} else {
				buffer.WriteLine("case " + goenum(casevalue, "int64") + ":")
			}
			buffer.WriteLine(attrHolder + " = &" + casetype.String() + "{}")
		}
		buffer.WriteLine("}")
	}

	if attr.Type.CustomType && attr.Value == "" {
		buffer.WriteLine(attrHolder + " = &" + attr.ChildType() + "{}")
	}
	buffer.WriteString(k.InitElem(attrHolder, errHolder, attr, attr.DataType(), false))

	if attr.Process != "" {
		process := attr.Process
		parts := strings.SplitN(process, "(", 2)
		parameters := []string{}

		cmd := parts[0]
		if len(parts) > 1 {
			parts[1] = strings.Trim(parts[1], "()")
			for _, parameter := range strings.Split(parts[1], ",") {
				parameter = strings.TrimSpace(parameter)
				parameter = goExpr(parameter, "")
				parameters = append(parameters, parameter)
			}
		}
		parameterList := strings.Join(parameters, ", ")

		switch cmd {
		case "xor":
			list := "[]byte{byte(" + parameterList + ")}"
			if strings.Contains(parameterList, ",") || (strings.HasPrefix(parameterList, "k") && getType(parameterList) != "uint8") {
				list = "[]byte(" + parameterList + ")"
			}
			buffer.WriteLine(attrHolder + " = " + "runtime.ProcessXOR(" + attrHolder + ", " + list + ")")
		case "rol":
			buffer.WriteLine(attrHolder + " = " + "runtime.ProcessRotateLeft(" + attrHolder + ", int(" + parameterList + "))")
		case "ror":
			buffer.WriteLine(attrHolder + " = " + "runtime.ProcessRotateRight(" + attrHolder + ", int(" + parameterList + "))")
		case "zlib":
			buffer.WriteLine(attrHolder + ", err = " + "runtime.ProcessZlib(" + attrHolder + ")")
		default:
			buffer.WriteLine(attrHolder + " = " + goExpr(cmd, "")[2:len(goExpr(cmd, ""))-1] + attrHolder + ", " + parameterList + ")")
		}
	}

	if attr.Type.CustomType {
		//buffer.WriteLine("k." + attr.Name() + " = &elem")
	} else {
		//buffer.WriteLine("k." + attr.Name() + " = elem")
	}
	buffer.WriteLine("return")
	buffer.WriteLine("}")

	return
}

func (k *Type) String(typeName string, parent string, root string) string {
	var buffer LineBuffer

	if val, ok := endianess[k.Meta.Endian]; ok {
		endian = val
	}

	// print doc string
	if k.Doc != "" {
		buffer.WriteLine("/* " + strings.TrimSpace(k.Doc) + "*/")
	}

	// print type start
	buffer.WriteLine("type " + typeName + " struct {")
	buffer.WriteLine("*runtime.TypeIO")

	// print attrs and insts
	for _, attr := range k.Seq {
		attr.Category = "attribute"
		buffer.WriteLine(attr.String())
	}

	for name, inst := range k.Instances {
		inst.Category = "instance"
		inst.ID = name
		buffer.WriteLine(inst.String())
		buffer.WriteLine(strcase.ToLowerCamel(inst.ID) + "Set bool")
	}

	// print type end
	buffer.WriteLine("}")

	// parent function
	buffer.WriteLine("func (k *" + typeName + ") Parent() (*" + parent + ") {")
	buffer.WriteLine("return k.ParentBase.(*" + parent + ")")
	buffer.WriteLine("}")

	// root function
	buffer.WriteLine("func (k *" + typeName + ") Root() (*" + root + ") {")
	buffer.WriteLine("return k.RootBase.(*" + root + ")")
	buffer.WriteLine("}")

	// decode function
	buffer.WriteLine("func (k *" + typeName + ") Read(reader io.ReadSeeker, lazy bool, ancestors ...interface{}) {")
	buffer.WriteLine("if k.TypeIO = runtime.NewTypeIO(reader, k, ancestors...); k.TypeIO.DecodeErr != nil {")
	buffer.WriteLine("return")
	buffer.WriteLine("}")

	for _, attr := range k.Seq {
		buffer.WriteLine("if k." + attr.Name() + ", k.DecodeErr = " + k.CallAttr(attr) + "; k.DecodeErr != nil {")
		buffer.WriteLine("return")
		buffer.WriteLine("}")
	}
	buffer.WriteLine("return")
	buffer.WriteLine("}")

	for _, attr := range k.Seq {
		buffer.WriteLine(k.InitAttr(attr, typeName))
	}

	// create getter
	for _, attr := range k.Seq {
		buffer.WriteLine("func (k *" + typeName + ") " + strcase.ToCamel(attr.Name()) + "() (value " + attr.DataType() + ") {")
		buffer.WriteLine("return " + "" + "k." + attr.Name())
		buffer.WriteLine("}")
	}

	// create inst getter
	for name, inst := range k.Instances {
		inst.ID = name
		buffer.WriteLine("func (k *" + typeName + ") " + strcase.ToCamel(inst.Name()) + "() (value " + inst.DataType() + ") {")
		buffer.WriteLine("if !k." + inst.Name() + "Set {")
		init := k.InitAttr(inst, typeName)
		if strings.Contains(init, "err") {
			buffer.WriteLine("var err error")
		}
		buffer.WriteString(init)
		buffer.WriteLine("k." + inst.Name() + "Set = true")
		buffer.WriteLine("}")
		buffer.WriteLine("return k." + inst.Name())
		buffer.WriteLine("}")
	}

	// print subtypes (flattened)
	for name, t := range k.Types {
		typeStr := t.String(strcase.ToCamel(name), getParent(strcase.ToCamel(name)), root)
		buffer.WriteLine(typeStr)
	}

	// print enums
	for enum, values := range k.Enums {
		buffer.WriteLine("var " + strcase.ToCamel(enum) + " = struct {")
		for _, value := range values {
			enumLiteral := toEnumLiteral(value)
			buffer.WriteLine(enumLiteral.nameCamel + " " + getEnumType(enum))
		}
		buffer.WriteLine("}{")
		for x, value := range values {
			enumLiteral := toEnumLiteral(value)
			buffer.WriteLine(enumLiteral.nameCamel + ": " + strconv.Itoa(x) + ",")
		}
		buffer.WriteLine("}")
	}

	return buffer.String()
}

type EnumLiteral struct {
	name      string
	nameCamel string
	doc       string
}

func toEnumLiteral(value interface{}) (ret *EnumLiteral) {
	if m, ok := value.(map[string]string); ok {
		ret = &EnumLiteral{name: m["id"], doc: m["doc"]}
	} else {
		ret = &EnumLiteral{name: fmt.Sprintf("%v", value)}
	}
	ret.nameCamel = strcase.ToCamel(ret.name)
	return
}

func toReadFunc(attr *Attribute, defaultEndian string) (ret string) {
	t := attr.Type.Type
	switch t {
	case "str":
		if attr.Size != "" {
			ret = fmt.Sprintf("ReadBytesString(uint16(%v))", goExpr(attr.Size, ""))
		} else if attr.Contents.Len() > 0 {
			ret = fmt.Sprintf("ReadBytesString(uint16(%v))", attr.Contents.Len())
		} else {
			ret = "ReadBytesFullString()"
		}
	case "strz":
		if attr.Size != "" {
			ret = fmt.Sprintf("ReadBytesString(uint16(%v))", goExpr(attr.Size, ""))
		} else if attr.Contents.Len() > 0 {
			ret = fmt.Sprintf("ReadBytesString(uint16(%v))", attr.Contents.Len())
		} else {
			ret = "ReadBytesFullString()"
		}
	case "":
		if attr.Size != "" {
			ret = fmt.Sprintf("ReadBytes(%v)", goExpr(attr.Size, ""))
		} else if attr.Contents.Len() > 0 {
			ret = fmt.Sprintf("ReadBytes(%v)", attr.Contents.Len())
		} else {
			ret = "ReadBytesFull()"
		}
	default:
		firstCharacter := t[0:1]
		other := t[1:len(t)]
		title := strings.ToUpper(firstCharacter) + other
		suffix := ""
		if strings.Contains(title, "B1") {
			suffix = "Bool"
		}
		if strings.HasSuffix(t, "e") {
			ret = fmt.Sprintf("Read%v%v()", title, suffix)
		} else {
			ret = fmt.Sprintf("Read%v%v%v()", title, defaultEndian, suffix)
		}
	}
	return
}
