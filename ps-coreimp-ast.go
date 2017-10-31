package main

import (
	"github.com/metaleap/go-util/str"
)

type coreImp struct { // we skip unmarshaling what isn't used for now, but DO keep these around commented-out:
	// BuiltWith  string            `json:"builtWith"`
	// ModuleName string            `json:"moduleName"`
	// ModulePath string            `json:"modulePath"`
	// Comments   []*coreImpComment `json:"comments"`
	// Foreign    []string          `json:"foreign"`
	// Exports    []string          `json:"exports"`
	Imps     [][]string     `json:"imports"`
	Body     coreImpAsts    `json:"body"`
	DeclAnns []*coreImpDecl `json:"declAnns"`
	DeclEnv  coreImpEnv     `json:"declEnv"`

	namedRequires map[string]string
	mod           *modPkg
}

func (me *coreImp) prep() {
	for _, da := range me.DeclAnns {
		da.prep()
	}
	me.DeclEnv.prep()
}

type coreImpComment struct {
	LineComment  string
	BlockComment string
}

type coreImpDecl struct {
	BindType string           `json:"bindType"`
	Ident    string           `json:"identifier"`
	Ann      *coreImpDeclAnn  `json:"annotation"`
	Expr     *coreImpDeclExpr `json:"expression"`
}

func (me *coreImpDecl) prep() {
	if me.Ann != nil {
		me.Ann.prep()
	}
	if me.Expr != nil {
		me.Expr.prep()
	}
}

type coreImpDeclAnn struct {
	SourceSpan *coreImpSourceSpan `json:"sourceSpan"`
	Type       *coreImpEnvTagType `json:"type"`
	Comments   []*coreImpComment  `json:"comments"`
	Meta       struct {
		MetaType   string   `json:"metaType"`        // IsConstructor or IsNewtype or IsTypeClassConstructor or IsForeign
		CtorType   string   `json:"constructorType"` // if MetaType=IsConstructor: SumType or ProductType
		CtorIdents []string `json:"identifiers"`     // if MetaType=IsConstructor
	} `json:"meta"`
}

func (me *coreImpDeclAnn) prep() {
	if me.Type != nil {
		me.Type.prep()
	}
}

type coreImpDeclExpr struct {
	Ann        *coreImpDeclAnn `json:"annotation"`
	ExprTag    string          `json:"type"`            // Var or Literal or Abs or App or Let or Constructor (or Accessor or ObjectUpdate or Case)
	CtorName   string          `json:"constructorName"` // if ExprTag=Constructor
	CtorType   string          `json:"typeName"`        // if ExprTag=Constructor
	CtorFields []string        `json:"fieldNames"`      // if ExprTag=Constructor
}

func (me *coreImpDeclExpr) prep() {
	if me.Ann != nil {
		me.Ann.prep()
	}
}

type coreImpSourceSpan struct {
	Name  string `json:"name"`
	Start []int  `json:"start"`
	End   []int  `json:"end"`
}

type coreImpAsts []*coreImpAst

type coreImpAst struct {
	AstSourceSpan  *coreImpSourceSpan `json:"sourceSpan"`
	AstTag         string             `json:"tag"`
	AstBody        *coreImpAst        `json:"body"`
	AstRight       *coreImpAst        `json:"rhs"`
	AstCommentDecl *coreImpAst        `json:"decl"`
	AstApplArgs    coreImpAsts        `json:"args"`
	AstOp          string             `json:"op"`
	AstFuncParams  []string           `json:"params"`
	AstFor1        *coreImpAst        `json:"for1"`
	AstFor2        *coreImpAst        `json:"for2"`
	AstThen        *coreImpAst        `json:"then"`
	AstElse        *coreImpAst        `json:"else"`

	Function             string
	StringLiteral        string
	BooleanLiteral       bool
	IntegerLiteral       int
	NumberLiteral        float64
	Block                coreImpAsts
	Var                  string
	VariableIntroduction string
	While                *coreImpAst
	App                  *coreImpAst
	Unary                *coreImpAst
	Comment              []*coreImpComment
	Binary               *coreImpAst
	ForIn                string
	For                  string
	IfElse               *coreImpAst
	ObjectLiteral        []map[string]*coreImpAst
	Return               *coreImpAst
	Throw                *coreImpAst
	ArrayLiteral         coreImpAsts
	Assignment           *coreImpAst
	Indexer              *coreImpAst
	Accessor             *coreImpAst
	InstanceOf           *coreImpAst

	parent *coreImpAst
	root   *coreImp
}

func (me *coreImpAst) ciAstForceIntoIrABlock(into *irABlock) {
	switch maybebody := me.ciAstToIrAst().(type) {
	case *irABlock:
		into.Body = maybebody.Body
		for _, a := range into.Body {
			a.Base().parent = into
		}
	default:
		into.add(maybebody)
	}
}

func (me *coreImpAst) ciAstToIrAst() (a irA) {
	istopleveldecl := (me.parent == nil)
	switch me.AstTag {
	case "StringLiteral":
		a = ªS(me.StringLiteral)
	case "BooleanLiteral":
		a = ªB(me.BooleanLiteral)
	case "NumberLiteral":
		a = ªN(me.NumberLiteral)
	case "IntegerLiteral":
		a = ªI(me.IntegerLiteral)
	case "Var":
		v := ªSymPs(me.Var, me.root.mod.irMeta.hasExport(me.Var))
		a = v
	case "Block":
		b := ªBlock()
		for _, c := range me.Block {
			b.add(c.ciAstToIrAst())
		}
		a = b
	case "While":
		f := ªFor()
		f.ForCond = me.While.ciAstToIrAst()
		f.ForCond.Base().parent = f
		me.AstBody.ciAstForceIntoIrABlock(f.ForDo)
		a = f
	case "ForIn":
		f := ªFor()
		f.ForRange = ªLet("", me.ForIn, me.AstFor1.ciAstToIrAst())
		f.ForRange.parent = f
		me.AstBody.ciAstForceIntoIrABlock(f.ForDo)
		a = f
	case "For":
		f := ªFor()
		fs := ªSymPs(me.For, me.root.mod.irMeta.hasExport(me.For))
		f.ForInit = []*irALet{ªLet("", me.For, me.AstFor1.ciAstToIrAst())}
		f.ForInit[0].parent = f
		fscmp, fsset, fsadd := *fs, *fs, *fs // quirky that we need these copies but we do
		f.ForCond = ªO2(&fscmp, "<", me.AstFor2.ciAstToIrAst())
		f.ForCond.Base().parent = f
		f.ForStep = []*irASet{ªSet(&fsset, ªO2(&fsadd, "+", ªI(1)))}
		f.ForStep[0].parent = f
		me.AstBody.ciAstForceIntoIrABlock(f.ForDo)
		a = f
	case "IfElse":
		i := ªIf(me.IfElse.ciAstToIrAst())
		me.AstThen.ciAstForceIntoIrABlock(i.Then)
		if me.AstElse != nil {
			i.Else = ªBlock()
			me.AstElse.ciAstForceIntoIrABlock(i.Else)
			i.Else.parent = i
		}
		a = i
	case "App":
		c := ªCall(me.App.ciAstToIrAst())
		for _, carg := range me.AstApplArgs {
			arg := carg.ciAstToIrAst()
			arg.Base().parent = c
			c.CallArgs = append(c.CallArgs, arg)
		}
		a = c
	case "VariableIntroduction":
		v := ªLet("", me.VariableIntroduction, nil)
		var wastypefunc *irAFunc
		if me.AstRight != nil {
			v.LetVal = me.AstRight.ciAstToIrAst()
			vlvb := v.LetVal.Base()
			vlvb.parent = v
			if v.LetVal != nil && vlvb.RefFunc != nil {
				if istopleveldecl && ustr.BeginsUpper(me.VariableIntroduction) {
					wastypefunc = v.LetVal.(*irAFunc)
				}
			} else if vlvc, _ := v.LetVal.(*irACall); vlvc != nil {
				if vlvcb := vlvc.Callee.Base(); vlvcb.RefFunc != nil {
					if istopleveldecl && ustr.BeginsUpper(me.VariableIntroduction) {
						wastypefunc = vlvc.Callee.(*irAFunc)
					}
				}
			}
		}
		if wastypefunc != nil {
			a = &irACtor{irAFunc: *wastypefunc}
		} else {
			a = v
		}
	case "Function":
		wastypefunc := istopleveldecl && me.Function != "" && ustr.BeginsUpper(me.Function)
		f := ªFunc()
		f.RefFunc = &irATypeRefFunc{}
		f.setBothNamesFromPsName(me.Function)
		for _, fpn := range me.AstFuncParams {
			arg := &irANamedTypeRef{}
			arg.setBothNamesFromPsName(fpn)
			f.RefFunc.Args = append(f.RefFunc.Args, arg)
		}
		f.RefFunc.impl = f.FuncImpl
		me.AstBody.ciAstForceIntoIrABlock(f.FuncImpl)
		if wastypefunc {
			a = &irACtor{irAFunc: *f}
		} else {
			a = f
		}
	case "Unary":
		o := ªO1(me.AstOp, me.Unary.ciAstToIrAst())
		switch o.Op1 {
		case "Negate":
			o.Op1 = "-"
		case "Not":
			o.Op1 = "!"
		case "Positive":
			o.Op1 = "+"
		case "BitwiseNot":
			o.Op1 = "^"
		case "New":
			o.Op1 = "&"
		default:
			panic(notImplErr("Unary", o.Op1, me.root.mod.impFilePath))
		}
		a = o
	case "Binary":
		o := ªO2(me.Binary.ciAstToIrAst(), me.AstOp, me.AstRight.ciAstToIrAst())
		switch o.Op2 {
		case "Add":
			o.Op2 = "+"
		case "Subtract":
			o.Op2 = "-"
		case "Multiply":
			o.Op2 = "*"
		case "Divide":
			o.Op2 = "/"
		case "Modulus":
			o.Op2 = "%"
		case "EqualTo":
			o.Op2 = "=="
		case "NotEqualTo":
			o.Op2 = "!="
		case "LessThan":
			o.Op2 = "<"
		case "LessThanOrEqualTo":
			o.Op2 = "<="
		case "GreaterThan":
			o.Op2 = ">"
		case "GreaterThanOrEqualTo":
			o.Op2 = ">="
		case "And":
			o.Op2 = "&&"
		case "Or":
			o.Op2 = "||"
		case "BitwiseAnd":
			o.Op2 = "&"
		case "BitwiseOr":
			o.Op2 = "|"
		case "BitwiseXor":
			o.Op2 = "^"
		case "ShiftLeft":
			o.Op2 = "<<"
		case "ShiftRight":
			o.Op2 = ">>"
		case "ZeroFillShiftRight":
			o.Op2 = "&^"
		default:
			panic(notImplErr("Binary", o.Op2, me.root.mod.impFilePath))
		}
		a = o
	case "Comment":
		c := ªComments(me.Comment...)
		a = c
	case "ObjectLiteral":
		o := ªO(nil)
		for _, namevaluepair := range me.ObjectLiteral {
			for onekey, oneval := range namevaluepair {
				ofv := ªOFld(oneval.ciAstToIrAst())
				ofv.setBothNamesFromPsName(onekey)
				ofv.parent = o
				o.ObjFields = append(o.ObjFields, ofv)
				break
			}
		}
		a = o
	case "ReturnNoResult":
		r := ªRet(nil)
		a = r
	case "Return":
		r := ªRet(me.Return.ciAstToIrAst())
		a = r
	case "Throw":
		r := ªPanic(me.Throw.ciAstToIrAst())
		a = r
	case "ArrayLiteral":
		exprs := make([]irA, 0, len(me.ArrayLiteral))
		for _, v := range me.ArrayLiteral {
			exprs = append(exprs, v.ciAstToIrAst())
		}
		l := ªA(exprs...)
		a = l
	case "Assignment":
		o := ªSet(me.Assignment.ciAstToIrAst(), me.AstRight.ciAstToIrAst())
		a = o
	case "Indexer":
		if me.AstRight.AstTag != "StringLiteral" {
			a = ªIndex(me.Indexer.ciAstToIrAst(), me.AstRight.ciAstToIrAst())
		} else { // TODO will need to differentiate better between a real property or an obj-dict-key
			if me.Indexer.AstTag == "Var" {
				if mod := findModuleByPName(me.Indexer.Var); mod != nil {
					a = ªPkgSym(mod.pName, me.AstRight.StringLiteral)
				}
			}
			if a == nil {
				dv := ªSymPs(me.AstRight.StringLiteral, me.root.mod.irMeta.hasExport(me.AstRight.StringLiteral))
				a = ªDot(me.Indexer.ciAstToIrAst(), dv)
			}
		}
	case "InstanceOf":
		if me.AstRight.Var != "" {
			a = ªIs(me.InstanceOf.ciAstToIrAst(), me.AstRight.Var)
		} else if me.AstRight.Indexer != nil {
			apkgsym := me.AstRight.ciAstToIrAst().(*irAPkgSym)
			a = ªIs(me.InstanceOf.ciAstToIrAst(), findModuleByPName(apkgsym.PkgName).qName+"."+apkgsym.Symbol)
		} else {
			panic(notImplErr("InstanceOf right-hand-side", me.AstRight.AstTag, me.root.mod.impFilePath))
		}
	default:
		panic(notImplErr("CoreImp AST tag", me.AstTag, me.root.mod.impFilePath))
	}
	if ab := a.Base(); ab != nil {
		ab.Comments = me.Comment
	}
	return
}

func (me *coreImp) initAstOnLoaded() {
	me.Body = me.initSubAsts(nil, me.Body...)
}

func (me *coreImp) prepTopLevel() {
	me.namedRequires = map[string]string{}
	i := 0
	ditch := func() {
		me.Body = append(me.Body[:i], me.Body[i+1:]...)
		i--
	}
	for i = 0; i < len(me.Body); i++ {
		a := me.Body[i]
		if a.StringLiteral == "use strict" {
			//	"use strict"
			ditch()
		} else if a.Assignment != nil && a.Assignment.Indexer != nil && a.Assignment.Indexer.Var == "module" && a.Assignment.AstRight != nil && a.Assignment.AstRight.StringLiteral == "exports" {
			//	module.exports = ..
			ditch()
		} else if a.AstTag == "VariableIntroduction" {
			if a.AstRight != nil && a.AstRight.App != nil && a.AstRight.App.Var == "require" && len(a.AstRight.AstApplArgs) == 1 && len(a.AstRight.AstApplArgs[0].StringLiteral) > 0 {
				me.namedRequires[a.VariableIntroduction] = a.AstRight.AstApplArgs[0].StringLiteral
				ditch()
			} else if a.AstRight != nil && a.AstRight.AstTag == "Function" {
				// turn top-level `var foo = func()` into `func foo()`
				a.AstRight.Function = a.VariableIntroduction
				a = a.AstRight
				a.parent, me.Body[i] = nil, a
			}
		} else if a.AstTag != "Function" && a.AstTag != "VariableIntroduction" && a.AstTag != "Comment" {
			panic(notImplErr("top-level CoreImp AST tag", a.AstTag, me.mod.impFilePath))
		}
	}
}

func (me *coreImp) initSubAsts(parent *coreImpAst, asts ...*coreImpAst) coreImpAsts {
	if parent != nil {
		parent.root = me
	}
	for ai, a := range asts {
		if a != nil {
			//	we might swap out `a` in here
			if a.AstTag == "Comment" && a.AstCommentDecl != nil {
				//	decls as sub-asts of comments is handy for PureScript but not for our own traversals, we lift the inner decl outward and set its own Comment instead. hence, we never process any AstCommentDecl, after this branch they're all nil
				if a.AstCommentDecl.AstTag == "Comment" {
					panic(notImplErr("comments", "nesting", me.mod.impFilePath))
				}
				decl := a.AstCommentDecl
				a.AstCommentDecl, decl.Comment, decl.parent = nil, a.Comment, parent
				a, asts[ai] = decl, decl
			}
			//	we might swap out `a` in here
			if parent != nil && a.AstTag == "Function" && a.Function != "" {
				//	there are a handful of cases (TCO it looks like) where CoreImp function bodies contain inner "full" functions as top-level-style stand-alone defs instead of bound expressions --- we bind them to a var right here, early on.
				nuvar := &coreImpAst{AstTag: "VariableIntroduction", VariableIntroduction: a.Function, AstRight: a, parent: parent}
				a.parent, a.Function = nuvar, ""
				a, asts[ai] = nuvar, nuvar
			}
			//	we might swap out `a` in here
			if a.AstTag == "Unary" && a.AstOp == "Not" && a.Unary.AstTag == "BooleanLiteral" {
				operand := a.Unary
				operand.parent, operand.BooleanLiteral = parent, !a.Unary.BooleanLiteral
				a, asts[ai] = operand, operand
			}
			//	now proceed whatever `a` now is

			a.For = strReplUnsanitize.Replace(a.For)
			a.ForIn = strReplUnsanitize.Replace(a.ForIn)
			a.Function = strReplUnsanitize.Replace(a.Function)
			a.Var = strReplUnsanitize.Replace(a.Var)
			a.VariableIntroduction = strReplUnsanitize.Replace(a.VariableIntroduction)

			for i, mkv := range a.ObjectLiteral {
				for onename, oneval := range mkv {
					if nuname := strReplUnsanitize.Replace(onename); nuname != onename {
						mkv = map[string]*coreImpAst{}
						mkv[nuname] = oneval
						a.ObjectLiteral[i] = mkv
					}
				}
			}
			for i, afp := range a.AstFuncParams {
				a.AstFuncParams[i] = strReplUnsanitize.Replace(afp)
			}

			a.root = me
			a.parent = parent
			a.App = me.initSubAsts(a, a.App)[0]
			a.ArrayLiteral = me.initSubAsts(a, a.ArrayLiteral...)
			a.Assignment = me.initSubAsts(a, a.Assignment)[0]
			a.AstApplArgs = me.initSubAsts(a, a.AstApplArgs...)
			a.AstBody = me.initSubAsts(a, a.AstBody)[0]
			a.AstCommentDecl = me.initSubAsts(a, a.AstCommentDecl)[0]
			a.AstFor1 = me.initSubAsts(a, a.AstFor1)[0]
			a.AstFor2 = me.initSubAsts(a, a.AstFor2)[0]
			a.AstElse = me.initSubAsts(a, a.AstElse)[0]
			a.AstThen = me.initSubAsts(a, a.AstThen)[0]
			a.AstRight = me.initSubAsts(a, a.AstRight)[0]
			a.Binary = me.initSubAsts(a, a.Binary)[0]
			a.Block = me.initSubAsts(a, a.Block...)
			a.IfElse = me.initSubAsts(a, a.IfElse)[0]
			a.Indexer = me.initSubAsts(a, a.Indexer)[0]
			a.Assignment = me.initSubAsts(a, a.Assignment)[0]
			a.InstanceOf = me.initSubAsts(a, a.InstanceOf)[0]
			a.Return = me.initSubAsts(a, a.Return)[0]
			a.Throw = me.initSubAsts(a, a.Throw)[0]
			a.Unary = me.initSubAsts(a, a.Unary)[0]
			a.While = me.initSubAsts(a, a.While)[0]
			for km, m := range a.ObjectLiteral {
				for kx, expr := range m {
					m[kx] = me.initSubAsts(a, expr)[0]
				}
				a.ObjectLiteral[km] = m
			}
		}
	}
	return asts
}
