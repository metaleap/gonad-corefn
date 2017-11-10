package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/metaleap/go-util/str"
)

const (
	areOverlappingInterfacesSupportedByGo = true // technically would be false, see https://github.com/golang/go/issues/6977 --- in practice keep true until it becomes an actual issue in generated code
)

func (_ *irAst) codeGenCommaIf(w io.Writer, i int) {
	if i > 0 {
		fmt.Fprint(w, ", ")
	}
}

func (_ *irAst) codeGenComments(w io.Writer, singlelineprefix string, withcomments *irABase) (err error) {
	for _, c := range withcomments.Comments {
		if c.BlockComment != "" {
			_, err = fmt.Fprintf(w, "/*%s*/", c.BlockComment)
		} else if c.LineComment != "" {
			_, err = fmt.Fprintf(w, "%s//%s\n", singlelineprefix, c.LineComment)
		}
		if err != nil {
			break
		}
	}
	return
}

func (me *irAst) codeGenAst(w io.Writer, indent int, ast irA) {
	if ast == nil {
		return
	}
	tabs := ""
	if indent > 0 {
		tabs = strings.Repeat("\t", indent)
	}
	switch a := ast.(type) {
	case *irALitStr:
		fmt.Fprintf(w, "%q", a.LitStr)
	case *irALitBool:
		fmt.Fprintf(w, "%t", a.LitBool)
	case *irALitChar:
		fmt.Fprintf(w, "%q", a.LitChar)
	case *irALitNum:
		s := fmt.Sprintf("%f", a.LitNum)
		for strings.HasSuffix(s, "0") {
			s = s[:len(s)-1]
		}
		fmt.Fprint(w, s)
	case *irALitInt:
		fmt.Fprintf(w, "%d", a.LitInt)
	case *irALitArr:
		me.codeGenTypeRef(w, &a.irGoNamedTypeRef, indent)
		fmt.Fprint(w, "{")
		for i, expr := range a.ArrVals {
			me.codeGenCommaIf(w, i)
			me.codeGenAst(w, indent, expr)
		}
		fmt.Fprint(w, "}")
	case *irALitObj:
		me.codeGenTypeRef(w, &a.irGoNamedTypeRef, -1)
		fmt.Fprint(w, "{")
		for i, namevaluepair := range a.ObjFields {
			me.codeGenCommaIf(w, i)
			if namevaluepair.NameGo != "" {
				fmt.Fprintf(w, "%s: ", namevaluepair.NameGo)
			}
			me.codeGenAst(w, indent, namevaluepair.FieldVal)
		}
		fmt.Fprint(w, "}")
	case *irAConst:
		fmt.Fprintf(w, "%sconst %s ", tabs, a.NameGo)
		if ProjCfg.CodeGen.ForceExplicitTypeAnnotations {
			me.codeGenTypeRef(w, a.ExprType(), -1)
			fmt.Fprint(w, " ")
		}
		fmt.Fprint(w, "= ")
		me.codeGenAst(w, indent, a.ConstVal)
		fmt.Fprint(w, "\n")
	case *irASym:
		fmt.Fprint(w, a.NameGo)
	case *irALet:
		switch ato := a.LetVal.(type) {
		case *irAToType:
			fmt.Fprint(w, tabs)
			if a.typeConv.okname == "" {
				fmt.Fprint(w, a.NameGo)
			} else {
				if a.typeConv.vused {
					fmt.Fprint(w, a.NameGo)
				} else {
					fmt.Fprint(w, "_")
				}
				fmt.Fprint(w, ", "+a.typeConv.okname)
			}
			fmt.Fprint(w, " := ")
			me.codeGenAst(w, indent, ato)
		default:
			if at := a.ExprType(); at.Ref.F != nil && a.LetVal != nil {
				fmt.Fprintf(w, "%s%s := ", tabs, a.NameGo)
				me.codeGenAst(w, indent, a.LetVal)
			} else {
				fmt.Fprintf(w, "%svar %s ", tabs, a.NameGo)
				if ProjCfg.CodeGen.ForceExplicitTypeAnnotations {
					me.codeGenTypeRef(w, at, -1)
					fmt.Fprint(w, " ")
				}
				if a.LetVal != nil {
					fmt.Fprint(w, "= ")
					me.codeGenAst(w, indent, a.LetVal)
				}
				if a.isTopLevel() {
					fmt.Fprint(w, "\n")
				}
			}
		}
		fmt.Fprint(w, "\n")
	case *irABlock:
		if len(a.Body) == 0 {
			fmt.Fprint(w, "{}")
		} else if len(a.Body) == 1 { // one-liner
			fmt.Fprint(w, "{ ")
			me.codeGenAst(w, -1, a.Body[0])
			fmt.Fprint(w, " }")
		} else {
			fmt.Fprint(w, "{\n")
			ind1 := indent + 1
			for _, expr := range a.Body {
				me.codeGenAst(w, ind1, expr)
			}
			fmt.Fprintf(w, "%s}", tabs)
		}
	case *irAIf:
		fmt.Fprintf(w, "%sif ", tabs)
		me.codeGenAst(w, indent, a.If)
		fmt.Fprint(w, " ")
		me.codeGenAst(w, indent, a.Then)
		if a.Else != nil {
			fmt.Fprint(w, " else ")
			me.codeGenAst(w, indent, a.Else)
		}
		fmt.Fprint(w, "\n")
	case *irACall:
		me.codeGenAst(w, indent, a.Callee)
		fmt.Fprint(w, "(")
		for i, expr := range a.CallArgs {
			if i > 0 {
				fmt.Fprint(w, ", ")
			}
			me.codeGenAst(w, indent, expr)
		}
		fmt.Fprint(w, ")")
	case *irAFunc:
		me.codeGenTypeRef(w, &a.irGoNamedTypeRef, indent)
		fmt.Fprint(w, " ")
		me.codeGenAst(w, indent, a.FuncImpl)
	case *irAComments:
		me.codeGenComments(w, tabs, &a.irABase)
	case *irARet:
		if a.RetArg == nil {
			fmt.Fprintf(w, "%sreturn", tabs)
		} else {
			fmt.Fprintf(w, "%sreturn ", tabs)
			me.codeGenAst(w, indent, a.RetArg)
		}
		if indent >= 0 {
			fmt.Fprint(w, "\n")
		}
	case *irAPanic:
		fmt.Fprintf(w, "%spanic(", tabs)
		me.codeGenAst(w, indent, a.PanicArg)
		fmt.Fprint(w, ")\n")
	case *irADot:
		me.codeGenAst(w, indent, a.DotLeft)
		fmt.Fprint(w, ".")
		me.codeGenAst(w, indent, a.DotRight)
	case *irAIndex:
		me.codeGenAst(w, indent, a.IdxLeft)
		fmt.Fprint(w, "[")
		me.codeGenAst(w, indent, a.IdxRight)
		fmt.Fprint(w, "]")
	case *irAIsType:
		fmt.Fprint(w, "ː"+a.names.v+"ᐧ"+a.names.t)
		// fmt.Fprint(w, typeNameWithPkgName(me.resolveGoTypeRefFromQName(a.TypeToTest)))
	case *irAToType:
		me.codeGenAst(w, indent, a.ExprToConv)
		fmt.Fprintf(w, ".(%s)", typeNameWithPkgName(me.resolveGoTypeRefFromQName(ustr.PrefixWithSep(a.TypePkg, ".", a.TypeName))))
	case *irAPkgSym:
		if a.PkgName != "" {
			if pkgimp := me.irM.ensureImp(a.PkgName, "", ""); pkgimp != nil {
				pkgimp.emitted = true
			}
			fmt.Fprintf(w, "%s.", a.PkgName)
		}
		fmt.Fprint(w, a.Symbol)
	case *irASet:
		fmt.Fprint(w, tabs)
		me.codeGenAst(w, indent, a.SetLeft)
		if a.isInVarGroup && ProjCfg.CodeGen.ForceExplicitTypeAnnotations {
			fmt.Fprint(w, " ")
			me.codeGenTypeRef(w, &a.irGoNamedTypeRef, indent)
		}
		fmt.Fprint(w, " = ")
		me.codeGenAst(w, indent, a.ToRight)
		fmt.Fprint(w, "\n")
	case *irAOp1:
		po1, po2 := a.parentOp()
		parens := po2 != nil || po1 != nil
		if parens {
			fmt.Fprint(w, "(")
		}
		fmt.Fprint(w, a.Op1)
		me.codeGenAst(w, indent, a.Of)
		if parens {
			fmt.Fprint(w, ")")
		}
	case *irAOp2:
		po1, po2 := a.parentOp()
		parens := po1 != nil || (po2 != nil && (po2.Op2 != a.Op2 || (a.Op2 != "+" && a.Op2 != "*" && a.Op2 != "&&" && a.Op2 != "&" && a.Op2 != "||" && a.Op2 != "|")))
		if parens {
			fmt.Fprint(w, "(")
		}
		me.codeGenAst(w, indent, a.Left)
		fmt.Fprintf(w, " %s ", a.Op2)
		me.codeGenAst(w, indent, a.Right)
		if parens {
			fmt.Fprint(w, ")")
		}
	case *irANil:
		fmt.Fprint(w, "nil")
	case *irAFor:
		if a.ForRange != nil {
			fmt.Fprintf(w, "%sfor _, %s := range ", tabs, a.ForRange.NameGo)
			me.codeGenAst(w, indent, a.ForRange.LetVal)
			me.codeGenAst(w, indent, a.ForDo)
		} else if len(a.ForInit) > 0 || len(a.ForStep) > 0 {
			fmt.Fprint(w, "for ")

			for i, finit := range a.ForInit {
				me.codeGenCommaIf(w, i)
				fmt.Fprint(w, finit.NameGo)
			}
			fmt.Fprint(w, " := ")
			for i, finit := range a.ForInit {
				me.codeGenCommaIf(w, i)
				me.codeGenAst(w, indent, finit.LetVal)
			}
			fmt.Fprint(w, "; ")

			me.codeGenAst(w, indent, a.ForCond)
			fmt.Fprint(w, "; ")

			for i, fstep := range a.ForStep {
				me.codeGenCommaIf(w, i)
				me.codeGenAst(w, indent, fstep.SetLeft)
			}
			fmt.Fprint(w, " = ")
			for i, fstep := range a.ForStep {
				me.codeGenCommaIf(w, i)
				me.codeGenAst(w, indent, fstep.ToRight)
			}
			me.codeGenAst(w, indent, a.ForDo)
		} else {
			fmt.Fprintf(w, "%sfor ", tabs)
			me.codeGenAst(w, indent, a.ForCond)
			fmt.Fprint(w, " ")
			me.codeGenAst(w, indent, a.ForDo)
		}
		fmt.Fprint(w, "\n")
	default:
		b, _ := json.Marshal(&ast)
		fmt.Fprintf(w, "/*****%v*****/", string(b))
	}
}

func (me *irAst) codeGenGroupedVals(w io.Writer, consts bool, asts []irA) {
	if l := len(asts); l > 0 {
		if l == 1 || ProjCfg.CodeGen.NoTopLevelDeclParenBlocks {
			for _, a := range asts {
				me.codeGenAst(w, 0, a)
			}
		} else {
			if consts {
				fmt.Fprint(w, "const (\n")
			} else {
				fmt.Fprint(w, "var (\n")
			}
			valˇnameˇtype := func(a irA) (val irA, name string, typeref *irGoNamedTypeRef) {
				if ac, _ := a.(*irAConst); ac != nil && consts {
					val, name, typeref = ac.ConstVal, ac.NameGo, ac.ExprType()
				} else if av, _ := a.(*irALet); av != nil {
					val, name, typeref = av.LetVal, av.NameGo, &av.irGoNamedTypeRef
				}
				return
			}
			for i, a := range asts {
				if val, name, typeref := valˇnameˇtype(a); val != nil {
					setgroup := ªsetVarInGroup(name, val, typeref)
					setgroup.parent = &me.irABlock
					me.codeGenAst(w, 1, setgroup)
					if i < (len(asts) - 1) {
						if _, ok := asts[i+1].(*irAComments); ok {
							fmt.Fprint(w, "\n")
						}
					}
				}
			}
			fmt.Fprint(w, ")\n\n")
		}
	}
}

func (me *irAst) codeGenFuncArgs(w io.Writer, indent int, methodargs irGoNamedTypeRefs, isretargs bool, withnames bool) {
	parens := (!isretargs) || len(methodargs) > 1 || (len(methodargs) == 1 && len(methodargs[0].NameGo) > 0)
	if parens {
		fmt.Fprint(w, "(")
	}
	if len(methodargs) > 0 {
		for i, arg := range methodargs {
			me.codeGenCommaIf(w, i)
			if withnames && arg.NameGo != "" {
				fmt.Fprintf(w, "%s ", arg.NameGo)
			}
			me.codeGenTypeRef(w, arg, indent+1)
		}
	}
	if parens {
		fmt.Fprint(w, ")")
	}
	if !isretargs {
		fmt.Fprint(w, " ")
	}
}

func (me *irAst) codeGenModImps(w io.Writer) (err error) {
	if len(me.irM.Imports) > 0 {
		modimps := make(irMPkgRefs, 0, len(me.irM.Imports))
		for _, modimp := range me.irM.Imports {
			if modimp.emitted {
				modimps = append(modimps, modimp)
			}
		}
		if len(modimps) > 0 {
			sort.Sort(modimps)
			if _, err = fmt.Fprint(w, "import (\n"); err == nil {
				wasuriform := modimps[0].isUriForm()
				for _, modimp := range modimps {
					if modimp.isUriForm() != wasuriform {
						wasuriform = !wasuriform
						_, err = fmt.Fprint(w, "\n")
					}
					if err == nil {
						if modimp.GoName == modimp.ImpPath || /*for the time being*/ true {
							_, err = fmt.Fprintf(w, "\t%q\n", modimp.ImpPath)
						} else {
							_, err = fmt.Fprintf(w, "\t%s %q\n", modimp.GoName, modimp.ImpPath)
						}
					}
					if err != nil {
						break
					}
				}
				if err == nil {
					_, err = fmt.Fprint(w, ")\n\n")
				}
			}
		}
	}
	return
}

func (me *irAst) codeGenPkgDecl(w io.Writer) (err error) {
	_, err = fmt.Fprintf(w, "package %s\n\n", me.mod.pName)
	return
}

func (me *irAst) codeGenStructMethods(w io.Writer, tr *irGoNamedTypeRef) {
	for _, method := range tr.Methods {
		mthis := "_"
		if method.Ref.F.hasthis {
			mthis = ProjCfg.CodeGen.Fmt.Method_ThisName
		}
		tthis := tr.NameGo
		if tr.Ref.E == nil && (method.Ref.origCtor != nil || (tr.Ref.S != nil && tr.Ref.S.PassByPtr)) {
			tthis = "*" + tthis
		}
		fmt.Fprintf(w, "func (%s %s) %s", mthis, tthis, method.NameGo)
		me.codeGenFuncArgs(w, -1, method.Ref.F.Args, false, true)
		me.codeGenFuncArgs(w, -1, method.Ref.F.Rets, true, true)
		fmt.Fprint(w, " ")
		me.codeGenAst(w, 0, method.Ref.F.impl)
		fmt.Fprint(w, "\n\n")
	}
}

func (me *irAst) codeGenTypeDef(w io.Writer, gtd *irGoNamedTypeRef) {
	fmt.Fprintf(w, "type %s ", gtd.NameGo)
	me.codeGenTypeRef(w, gtd, 0)
	fmt.Fprint(w, "\n\n")
	if gtdenum := gtd.Ref.E; gtdenum != nil {
		fmt.Fprint(w, "const (\n")
		for i, member := range gtdenum.Names {
			fmt.Fprintf(w, "\t%s", member.NameGo)
			if i == 0 {
				fmt.Fprintf(w, " %s = iota\n", gtd.NameGo)
			} else {
				fmt.Fprint(w, "\n")
			}
		}
		fmt.Fprint(w, ")\n\n")
	}
}

func (me *irAst) codeGenTypeRef(w io.Writer, gtd *irGoNamedTypeRef, indlevel int) {
	fmtembeds := "\t%s\n"
	if gtd == nil { // TODO: remove this check ultimately
		return
	}
	isfuncwithbodynotjustsig := gtd.Ref.F != nil && gtd.Ref.F.impl != nil
	if gtd.Ref.Q != nil {
		me.codeGenAst(w, -1, ªPkgSym(me.resolveGoTypeRefFromQName(gtd.Ref.Q.QName)))
	} else if gtd.Ref.E != nil {
		fmt.Fprint(w, "int")
	} else if gtd.Ref.A != nil {
		fmt.Fprint(w, "[]")
		me.codeGenTypeRef(w, gtd.Ref.A.Of, -1)
	} else if gtd.Ref.P != nil {
		fmt.Fprint(w, "*")
		me.codeGenTypeRef(w, gtd.Ref.P.Of, -1)
	} else if gtd.Ref.I != nil {
		if len(gtd.Ref.I.Embeds) == 0 && len(gtd.Methods) == 0 {
			fmt.Fprint(w, "interface{}")
		} else {
			var tabind string
			if indlevel > 0 {
				tabind = strings.Repeat("\t", indlevel)
			}
			fmt.Fprint(w, "interface {\n")
			if areOverlappingInterfacesSupportedByGo {
				for _, ifembed := range gtd.Ref.I.Embeds {
					fmt.Fprint(w, tabind+"\t")
					me.codeGenAst(w, -1, ªPkgSym(me.resolveGoTypeRefFromQName(ifembed)))
					fmt.Fprint(w, "\n")
				}
			}
			var buf bytes.Buffer
			for _, ifmethod := range gtd.Methods {
				fmt.Fprint(&buf, ifmethod.NameGo)
				if ifmethod.Ref.F == nil {
					panic(notImplErr("interface-method (not a func)", ifmethod.NamePs, gtd.NamePs))
				} else {
					me.codeGenFuncArgs(&buf, indlevel, ifmethod.Ref.F.Args, false, false)
					me.codeGenFuncArgs(&buf, indlevel, ifmethod.Ref.F.Rets, true, false)
				}
				fmt.Fprint(w, tabind)
				fmt.Fprintf(w, fmtembeds, buf.String())
				buf.Reset()
			}
			fmt.Fprintf(w, "%s}", tabind)
		}
	} else if gtd.Ref.S != nil {
		var tabind string
		if indlevel > 0 {
			tabind = strings.Repeat("\t", indlevel)
		}
		if len(gtd.Ref.S.Embeds) == 0 && len(gtd.Ref.S.Fields) == 0 {
			fmt.Fprint(w, "struct{}")
		} else {
			fmt.Fprint(w, "struct {\n")
			for _, structembed := range gtd.Ref.S.Embeds {
				fmt.Fprint(w, tabind)
				fmt.Fprintf(w, fmtembeds, structembed)
			}
			fnlen := 0
			for _, structfield := range gtd.Ref.S.Fields {
				if l := len(structfield.NameGo); l > fnlen {
					fnlen = l
				}
			}
			var buf bytes.Buffer
			for _, structfield := range gtd.Ref.S.Fields {
				me.codeGenTypeRef(&buf, structfield, indlevel+1)
				fmt.Fprint(w, tabind)
				fmt.Fprintf(w, fmtembeds, ustr.PadRight(structfield.NameGo, fnlen)+" "+buf.String())
				buf.Reset()
			}
			fmt.Fprintf(w, "%s}", tabind)
		}
	} else if gtd.Ref.F != nil {
		fmt.Fprint(w, "func")
		if isfuncwithbodynotjustsig && gtd.NameGo != "" {
			fmt.Fprintf(w, " %s", gtd.NameGo)
		}
		me.codeGenFuncArgs(w, indlevel, gtd.Ref.F.Args, false, isfuncwithbodynotjustsig)
		me.codeGenFuncArgs(w, indlevel, gtd.Ref.F.Rets, true, isfuncwithbodynotjustsig)
	} else {
		if ProjCfg.CodeGen.NoAliasForEmptyInterface {
			fmt.Fprintf(w, "interface{} /* %s */", gtd.Ref.origs.String())
		} else {
			fmt.Fprintf(w, "𝒈.𝑻 /* %s */", gtd.Ref.origs.String())
			me.irM.ensureImp("", "github.com/golamb/da", "").emitted = true
		}
	}
}
