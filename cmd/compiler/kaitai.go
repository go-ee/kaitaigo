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
		return "runtime.KSYDecoder"
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
		doc = " // " + k.Doc
	}

	return k.Name() + " " + k.DataType() + "`ks:\"" + k.ID + "," + k.Category + "\"`" + doc
}

type Type struct {
	Meta      Meta                      `yaml:"meta,omitempty"`
	Types     map[string]Type           `yaml:"types,omitempty"`
	Seq       []Attribute               `yaml:"seq,omitempty"`
	Enums     map[string]map[int]string `yaml:"enums,omitempty"`
	Doc       string                    `yaml:"doc,omitempty"`
	Instances map[string]Attribute      `yaml:"instances,omitempty"`
}

func (k *Type) InitVar(attr Attribute, name, dataType string, init bool) string {
	var buffer LineBuffer

	// init and parse element
	if init && dataType != "[]byte" {
		buffer.WriteLine("var " + name + " " + dataType)
	}
	if dataType == "[]byte" && attr.Size != "" {
		if init {
			buffer.WriteLine(name + " := make([]byte, " + goExpr(attr.Size, "") + ")")
		} else {
			buffer.WriteLine(name + " = make([]byte, " + goExpr(attr.Size, "") + ")")
		}
	}

	if isNative(dataType) {
		if strings.HasSuffix(attr.ID, "be") {
			endian = "binary.BigEndian"
		}
		if attr.SizeEos != "" {
			buffer.WriteLine("b, err := ioutil.ReadAll(decoder)")
			buffer.WriteLine("decoder.SetErr(err)")
			buffer.WriteLine(name + " = b")
		} else if attr.Type.Type == "strz" {
			buffer.WriteLine("pos, _ := decoder.Seek(0, io.SeekCurrent)")
			buffer.WriteLine("b, err := bufio.NewReader(decoder).ReadBytes(byte(0))")
			buffer.WriteLine("if err != nil && err != io.EOF {decoder.SetErr(err); return}")
			buffer.WriteLine("_, err = decoder.Seek(pos+int64(len(b)), io.SeekStart)")
			buffer.WriteLine("if err != nil {decoder.SetErr(err); return}")
			if init {
				buffer.WriteLine(name + " := b[:len(b)-1]")
			} else {
				buffer.WriteLine(name + " = b[:len(b)-1]")
			}

		} else {
			buffer.WriteLine("decoder.SetErr(binary.Read(decoder, " + endian + ", &" + name + "))")
		}
	} else {
		buffer.WriteLine(name + ".DecodeAncestors(k, k.Root)")
	}

	return buffer.String()
}

func (k *Type) InitAttr(attr Attribute) (goCode string) {
	var buffer LineBuffer
	defer func() { goCode = buffer.String() + "\n" }()

	if attr.If != "" {
		buffer.WriteLine("if " + goExpr(attr.If, "") + "{")
		defer buffer.WriteLine("}") // end if
	}

	if attr.Value != "" {
		// value instance
		if attr.DataType() == "runtime.KSYDecoder" || strings.HasPrefix(attr.DataType(), "*") {
			buffer.WriteLine("k." + attr.Name() + " = " + goExpr(attr.Value, ""))
		} else {
			buffer.WriteLine("k." + attr.Name() + " = " + attr.DataType() + "(" + goExpr(attr.Value, "") + ")")
		}
		return
	}

	if attr.Pos != "" {
		// save position
		buffer.WriteLine("pos, _ := decoder.Seek(0, io.SeekCurrent) // Cannot fail")
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
			buffer.WriteLine("decoder.Seek(0, io.SeekStart)")
		}
		buffer.WriteLine("_, err := decoder.Seek(" + goExpr(attr.Pos, "int64") + ", " + whence + ")")
		buffer.WriteLine("if err != nil {decoder.SetErr(err); return}")
		// restore position
		defer buffer.WriteLine("decoder.Seek(pos, io.SeekStart)")
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
			before = "i < int(" + goExpr(attr.RepeatExpr, "") + ")"
			fall = true
			fallthrough
		case "until":
			if !fall {
				if attr.RepeatUntil == "" {
					panic("RepeatUntil is missing") // TODO: move to parsing
				}
				until = goExprAttr(attr.RepeatUntil, "", attr.Name()+"[i]")
			}
			fallthrough
		case "eos":
			// slice
			if strings.HasPrefix(attr.DataType(), "[]") {
				buffer.WriteLine("k." + attr.Name() + " = " + attr.DataType() + "{}")
			}

			buffer.WriteLine("for i := 0; " + before + "; i++ {")

			buffer.WriteString(k.InitVar(attr, "elem", attr.ChildType(), true))

			// break on error
			buffer.WriteLine("if decoder.Err() != nil && decoder.Err() == io.EOF { decoder.UnsetErr(); break }")
			buffer.WriteLine("if decoder.Err() != nil { break }")

			// add element
			if strings.HasPrefix(attr.DataType(), "[]") {
				buffer.WriteLine("k." + attr.Name() + " = append(k." + attr.Name() + ", elem)")
			} else {
				buffer.WriteLine("k." + attr.Name() + "[i] = elem")
			}

			// break on repeat-until
			if until != "" {
				buffer.WriteLine("if " + until + "{break}")
			}

			buffer.WriteLine("}")
			return
		}
	case attr.Type.CustomType:
		// custom struct
		// init variable
		if attr.Size != "" {
			buffer.WriteLine(attr.Name() + "pos, _ := decoder.Seek(0, io.SeekCurrent) // Cannot fail")
			defer buffer.WriteLine("decoder.Seek(" + attr.Name() + "pos + " + goExpr(attr.Size, "int64") + ", io.SeekStart)")
		}
		buffer.WriteLine("k." + attr.Name() + " = &" + attr.DataType()[1:] + "{}")
	case attr.Type.TypeSwitch.SwitchOn != "":
		buffer.WriteLine("switch " + goExpr(attr.Type.TypeSwitch.SwitchOn, "int64") + " {")
		for casevalue, casetype := range attr.Type.TypeSwitch.Cases {
			buffer.WriteLine("case " + goenum(casevalue, "int64") + ":")
			buffer.WriteLine("k." + attr.Name() + " = &" + casetype.String() + "{}")
		}
		buffer.WriteLine("}")
	}

	buffer.WriteString(k.InitVar(attr, "k."+attr.Name(), attr.DataType(), false))

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
			buffer.WriteLine("k." + attr.Name() + " = " + "runtime.ProcessXOR(k." + attr.Name() + ", " + list + ")")
		case "rol":
			buffer.WriteLine("k." + attr.Name() + " = " + "runtime.ProcessRotateLeft(k." + attr.Name() + ", int(" + parameterList + "))")
		case "ror":
			buffer.WriteLine("k." + attr.Name() + " = " + "runtime.ProcessRotateRight(k." + attr.Name() + ", int(" + parameterList + "))")
		case "zlib":
			buffer.WriteLine("k." + attr.Name() + ", err = " + "runtime.ProcessZlib(k." + attr.Name() + ")")
		default:
			buffer.WriteLine("k." + attr.Name() + " = " + goExpr(cmd, "")[2:len(goExpr(cmd, ""))-1] + "k." + attr.Name() + ", " + parameterList + ")")
		}
	}

	return
}

func (k *Type) String(typeName string, parent string, root string) string {
	var buffer LineBuffer

	// print doc string
	if k.Doc != "" {
		buffer.WriteLine("// " + strings.Replace(strings.TrimSpace(k.Doc), "\n", "\n// ", -1))
	}

	// print type start
	buffer.WriteLine("type " + typeName + " struct{")
	buffer.WriteLine("parent interface{}")
	buffer.WriteLine("Root *" + root)

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

	// decode function
	buffer.WriteLine("func (k *" + typeName + ") Decode(reader io.ReadSeeker) (err error) {")
	buffer.WriteLine("if decoder != nil && decoder.Err() != nil { return decoder.Err() }")

	if val, ok := endianess[k.Meta.Endian]; ok {
		endian = val
	}
	buffer.WriteLine("if decoder == nil { decoder = runtime.New(reader) }")
	buffer.WriteLine("k.DecodeAncestors(k, k)")
	buffer.WriteLine("return decoder.Err()")
	buffer.WriteLine("}")

	// parent function
	buffer.WriteLine("func (k *" + typeName + ") Parent() (*" + parent + ") {")
	buffer.WriteLine("return k.parent.(*" + parent + ")")
	buffer.WriteLine("}")

	// decode ancestors function
	buffer.WriteLine("func (k *" + typeName + ") DecodeAncestors(parent interface{}, root interface{}) () {")
	buffer.WriteLine("if decoder.Err() != nil { return }")
	buffer.WriteLine("k.parent = parent")
	buffer.WriteLine("k.Root = root.(*" + root + ")")

	for _, attr := range k.Seq {
		buffer.WriteString(k.InitAttr(attr))
	}
	buffer.WriteLine("}")

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
		buffer.WriteString(k.InitAttr(inst))
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
			buffer.WriteLine(strcase.ToCamel(value) + " " + getEnumType(enum))
		}
		buffer.WriteLine("}{")
		for x, value := range values {
			buffer.WriteLine(strcase.ToCamel(value) + ": " + strconv.Itoa(x) + ",")
		}
		buffer.WriteLine("}")
	}

	return buffer.String()
}
