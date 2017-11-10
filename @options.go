package main

import (
	"path/filepath"

	"github.com/metaleap/go-util/dev/go"
	"github.com/metaleap/go-util/fs"
)

type Cfg struct {
	In struct {
		CoreFilesDirPath string // dir path containing Some.Module.QName/corefn.json files
	}
	Out struct {
		IncludeUnusedDeps bool
		DumpAst           bool   // dumps an additional gonad.ast.json next to gonad.json
		MainDepLevel      int    // temporary option
		GoDirSrcPath      string // defaults to the first `GOPATH` found that has a `src` sub-directory
		GoNamespaceProj   string
		GoNamespaceDeps   string
	}
	CodeGen struct {
		NoAliasForEmptyInterface     bool
		VarsAsConstsWherePossible    bool
		NoTopLevelDeclParenBlocks    bool // if true, every top-level const and var stands alone, not in a var() / const() block
		ForceExplicitTypeAnnotations bool // if true, type annotations are spelled out for all vars and consts, even if not needed
		TypeSynsForNewtypes          bool // generates for every `data` with only one ctor (that is unary & non-recursive) only a type-synonym instead of a full interface+struct combo
		TypeSynsForSingletonStructs  bool // turns, where feasible, a struct declaration with a single field into a type-synonym to said field's type (eg not feasible if: struct has methods and new underlying type couldn't be method receiver)
		DataTypeAssertMethods        bool // if true, all `data` interfaces declare methods implemented by all related ctor structs, to be used instead of Go-native type-assertion case-switches
		DataAsEnumsWherePossible     bool // turns `data` types with only argument-less ctors from "1 interface + n 0-byte structs" into a single iota enum
		PtrStructMinFieldCount       int  // default 2. any struct types with fewer fields are passed/returned by value instead of by pointer (0-byte structs always are); exception being all custom DataTypeAssertMethods, if any
		Fmt                          struct {
			Reserved_Keywords    string // allows a single %s for the keyword to be escaped
			Reserved_Identifiers string // allows a single %s for the predefined-identifier to be escaped
			StructName_InstImpl  string // allows a single %s for the type-class instance name
			StructName_DataCtor  string // allows {D} and {C} for `data` name and ctor name
			FieldName_DataCtor   string // allows {I} for the 0-based field (ctor arg) index and {C} for the ctor name
			IfaceName_TypeClass  string // allows a single %s for the type-class name
			Method_ThisName      string // must be a valid identifier symbol in Golang, used for the `this` argument (aka receiver) in methods
		}
	}

	loadedFromJson bool
}

func (me *Cfg) populateDefaultsUponLoaded() {
	if me.In.CoreFilesDirPath == "" {
		me.In.CoreFilesDirPath = "output"
	}
	if me.Out.GoNamespaceProj == "" {
		panic("missing in bower.json: `Gonad{Out{GoNamespaceProj=\"...\"}}` setting (the directory path relative to either your GOPATH or the specified `Gonad{Out{GoDirSrcPath=\"...\"}}`)")
	}
	if me.Out.GoDirSrcPath == "" {
		for _, gopath := range udevgo.AllGoPaths() {
			if me.Out.GoDirSrcPath = filepath.Join(gopath, "src"); ufs.DirExists(me.Out.GoDirSrcPath) {
				break
			}
		}
	}
	if me.CodeGen.PtrStructMinFieldCount == 0 {
		me.CodeGen.PtrStructMinFieldCount = 2
	}

	fmts := &me.CodeGen.Fmt
	if fmts.StructName_InstImpl == "" {
		fmts.StructName_InstImpl = "ᛌ%s"
	}
	if fmts.IfaceName_TypeClass == "" {
		fmts.IfaceName_TypeClass = "%sᛌ"
	}
	if fmts.StructName_DataCtor == "" {
		fmts.StructName_DataCtor = "{D}۰{C}"
	}
	if fmts.FieldName_DataCtor == "" {
		fmts.FieldName_DataCtor = "{C}ˈ{I}"
	}
	if fmts.Reserved_Keywords == "" {
		fmts.Reserved_Keywords = "%sʾ"
	}
	if fmts.Reserved_Identifiers == "" {
		fmts.Reserved_Identifiers = "ʾ%s"
	}
	if fmts.Method_ThisName == "" {
		fmts.Method_ThisName = "this"
	}
}
