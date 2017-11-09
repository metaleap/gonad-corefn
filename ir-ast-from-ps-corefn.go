package main

import (
	"fmt"
	"strings"

	"github.com/metaleap/go-util/dev/ps"
	"github.com/metaleap/go-util/str"
)

func (me *irAst) intoFromˇDecl(into *irABlock, decl *udevps.CoreFnDecl) {
	if decl.CoreFnDeclBind != nil {
		into.add(me.fromˇDeclBind(into.parent == nil, decl.CoreFnDeclBind))
	} else {
		for i, _ := range decl.Binds {
			into.add(me.fromˇDeclBind(into.parent == nil, &decl.Binds[i]))
		}
	}
}

func (me *irAst) fromˇDeclBind(istoplevel bool, bind *udevps.CoreFnDeclBind) (a irA) {
	var av *irALet
	var af *irAFunc
	expr := me.fromˇExpr(&bind.Expression)
	if istoplevel && bind.Expression.Abs != nil {
		af, _ = expr.(*irAFunc)
		af.Export = me.irM.hasExport(bind.Identifier)
		af.setBothNamesFromPsName(bind.Identifier)
		a = af
	} else {
		av = ªLet("", "", expr)
		if istoplevel {
			av.Export = me.irM.hasExport(bind.Identifier)
		}
		av.setBothNamesFromPsName(bind.Identifier)
		if ustr.BeginsUpper(bind.Identifier) {
			av.NameGo = fmt.Sprintf(ProjCfg.CodeGen.Fmt.IfaceName_TypeClass, av.NameGo)
		}
		a = av
	}
	return
}

func (me *irAst) fromˇExpr(expr *udevps.CoreFnExpr) irA {
	if expr.Abs != nil {
		return me.fromˇExprAbs(expr.Abs)
	} else if expr.Accessor != nil {
		return me.fromˇExprAcc(expr.Accessor)
	} else if expr.App != nil {
		return me.fromˇExprApp(expr.App)
	} else if expr.Case != nil {
		return me.fromˇExprCase(expr.Case)
	} else if expr.Constructor != nil {
		return me.fromˇExprCtor(expr.Constructor)
	} else if expr.Let != nil {
		return me.fromˇExprLet(expr.Let)
	} else if expr.Literal != nil {
		return me.fromˇExprLit(expr.Literal)
	} else if expr.ObjectUpdate != nil {
		return me.fromˇExprObjUpd(expr.ObjectUpdate)
	} else if expr.Var != nil {
		return me.fromˇExprVar(expr.Var)
	}
	return ªSymGo("NOT_YET_IMPL")
}

func (me *irAst) fromˇExprAbs(xabs *udevps.CoreFnExprAbs) *irAFunc {
	af := ªFunc(nil)
	af.Ref.F.Rets = irGoNamedTypeRefs{&irGoNamedTypeRef{}}
	af.Ref.F.Args = irGoNamedTypeRefs{&irGoNamedTypeRef{}}
	af.Ref.F.Args[0].setBothNamesFromPsName(xabs.Argument)
	af.FuncImpl.add(me.fromˇExpr(&xabs.Body))
	return af
}

func (me *irAst) fromˇExprApp(xapp *udevps.CoreFnExprApp) *irACall {
	callee := me.fromˇExpr(&xapp.Abstraction)
	callarg := me.fromˇExpr(&xapp.Argument)
	return ªCall(callee, callarg)
}

func (me *irAst) fromˇExprAcc(xacc *udevps.CoreFnExprAcc) *irADot {
	dotl := me.fromˇExpr(&xacc.Expression)
	dotr := ªSymPs(":DOT:"+xacc.FieldName, true)
	return ªDot(dotl, dotr)
}

func (me *irAst) fromˇExprCase(xcase *udevps.CoreFnExprCase) *irASwitch {
	alts := make([]*irACase, 0, len(xcase.Alternatives))
	exprs := make([]irA, 0, len(xcase.Expressions))
	for i, _ := range xcase.Alternatives {
		alts = append(alts, me.fromˇExprCaseAlt(&xcase.Alternatives[i]))
	}
	for i, _ := range xcase.Expressions {
		exprs = append(exprs, me.fromˇExpr(&xcase.Expressions[i]))
	}
	return ªSwitch(ªSymGo("exprs0?"), alts)
}

func (me *irAst) fromˇExprCtor(xctor *udevps.CoreFnExprCtor) *irASym {
	return ªSymGo("CTOR_" + xctor.TypeName + "___" + xctor.ConstructorName)
}

func (me *irAst) fromˇExprCaseAlt(xcasealt *udevps.CoreFnExprCaseAlt) *irACase {
	return ªCase(ªSymGo("casecond"))
}

func (me *irAst) fromˇExprLet(xlet *udevps.CoreFnExprLet) *irACall {
	af := ªFunc(nil)
	af.Ref.F.Rets = irGoNamedTypeRefs{&irGoNamedTypeRef{}}
	for i, _ := range xlet.Binds {
		me.intoFromˇDecl(af.FuncImpl, &xlet.Binds[i])
	}
	af.FuncImpl.add(ªRet(me.fromˇExpr(&xlet.Expression)))
	return ªCall(af)
}

func (me *irAst) fromˇExprLit(xlit *udevps.CoreFnExprLit) irA {
	xlv := &xlit.Val
	switch xlv.Type {
	case "BooleanLiteral":
		return ªB(xlv.Boolean)
	case "CharLiteral":
		return ªC(xlv.Char)
	case "IntLiteral":
		return ªI(xlv.Int)
	case "NumberLiteral":
		return ªN(xlv.Number)
	case "StringLiteral":
		return ªS(xlv.Str)
	case "ArrayLiteral":
		if len(xlv.ArrayOfBinders) > 0 {
			return ªSymGo("MUH_HOW_WHY_AoB")
		} else {
			arr, l := ªA(), len(xlv.Array)
			all := make([]irA, l, l)
			for i, _ := range xlv.Array {
				all[i] = me.fromˇExpr(&xlv.Array[i])
				all[i].Base().parent = arr
			}
			return arr
		}
	case "ObjectLiteral":
		objctor := ªO(nil)
		for _, litobjfld := range xlv.Obj {
			if litobjfld.Binder != nil {
				return ªSymGo("MUH_HOW_WHY_OoB")
			} else {
				ofld := ªOFld(nil)
				ofld.parent = objctor
				ofld.FieldVal = me.fromˇExpr(litobjfld.Val)
				ofld.FieldVal.Base().parent = ofld
				ofld.Export = true
				ofld.setBothNamesFromPsName(litobjfld.Name)
			}
		}
		return objctor
	default:
		panic(notImplErr("CoreFnExprLit.Val.Type", xlv.Type, me.mod.cfnFilePath))
	}
}

func (me *irAst) fromˇExprObjUpd(xobjupd *udevps.CoreFnExprObjUpd) irA {
	return ªSymGo("OBJUPD")
}

func (me *irAst) fromˇExprVar(xvar *udevps.CoreFnExprVar) irA {
	return me.fromˇIdent(&xvar.Value)
}

func (me *irAst) fromˇIdent(xid *udevps.CoreFnIdent) irA {
	mod := me.mod
	if len(xid.ModuleName) > 0 {
		if qname := strings.Join(xid.ModuleName, "."); qname == "Prim" {
			if xid.Identifier == "undefined" {
				return ªNil(udevps.CoreComment{BlockComment: "undefined"})
			} else {
				panic(notImplErr("Prim ident", xid.Identifier, me.mod.cfnFilePath))
			}
		} else if mod = findModuleByQName(qname); mod == nil {
			panic(notImplErr("unresolvable module reference", qname, me.mod.cfnFilePath))
			// } else /*not needed: whenever this is true the below is already in-place*/ if mod.qName == me.mod.qName {
			// 	mod = me.mod
		}
	}
	if mod == me.mod {
		return ªSymPs(xid.Identifier, me.irM.hasExport(xid.Identifier))
	}
	return ªPkgSym(mod.pName, xid.Identifier)
}
