package main

import (
	"fmt"
	"strings"

	"github.com/metaleap/go-util/dev/ps"
	"github.com/metaleap/go-util/str"
)

func (me *irAst) fromˇDeclBind(parent irA, bind *udevps.CoreFnDeclBind) (a irA) {
	x := me.fromˇExpr(parent, &bind.Expression)
	v := ªLet("", bind.Identifier, x)
	if parent == &me.irABlock {
		v.Export = me.irM.hasExport(bind.Identifier)
	}
	v.setBothNamesFromPsName(bind.Identifier)
	if ustr.BeginsUpper(bind.Identifier) {
		v.NameGo = fmt.Sprintf(ProjCfg.CodeGen.Fmt.IfaceName_TypeClass, v.NameGo)
	}
	a = v
	return
}

func (me *irAst) fromˇExpr(parent irA, expr *udevps.CoreFnExpr) irA {
	if expr.Abs != nil {
	} else if expr.Accessor != nil {
	} else if expr.App != nil {
	} else if expr.Case != nil {
	} else if expr.Constructor != nil {
	} else if expr.Let != nil {
	} else if expr.Literal != nil {
		return me.fromˇExprLit(parent, expr.Literal)
	} else if expr.ObjectUpdate != nil {

	} else if expr.Var != nil {
		return me.fromˇExprVar(parent, expr.Var)
	}
	return nil
}

func (me *irAst) fromˇExprLit(parent irA, lit *udevps.CoreFnExprLit) (a irA) {
	switch lit.Val.Type {
	case "BooleanLiteral":
		b := ªB(lit.Val.Boolean)
		b.parent, a = parent, b
	case "CharLiteral":
		c := ªC(lit.Val.Char)
		c.parent, a = parent, c
	case "IntLiteral":
		i := ªI(lit.Val.Int)
		i.parent, a = parent, i
	case "NumberLiteralLiteral":
		n := ªN(lit.Val.Number)
		n.parent, a = parent, n
	case "StringLiteral":
		s := ªS(lit.Val.Str)
		s.parent, a = parent, s
	case "ArrayLiteral":
		if len(lit.Val.ArrayOfBinders) > 0 {
			return ªSymGo("MUH_HOW_WHY_AoB")
		} else {
			arr, l := ªA(), len(lit.Val.Array)
			arr.parent = parent
			all := make([]irA, l, l)
			for i, x := range lit.Val.Array {
				all[i] = me.fromˇExpr(arr, &x)
			}
		}
	case "ObjectLiteral":
		objctor := ªO(nil)
		for _, litobjfld := range lit.Val.Obj {
			if litobjfld.Binder != nil {
				return ªSymGo("MUH_HOW_WHY_OoB")
			} else {
				ofld := ªOFld(nil)
				ofld.parent = objctor
				ofld.FieldVal = me.fromˇExpr(ofld, litobjfld.Val)
				ofld.Export = true
				ofld.setBothNamesFromPsName(litobjfld.Name)
			}
		}
		objctor.parent, a = parent, objctor
	}
	return
}

func (me *irAst) fromˇExprVar(parent irA, v *udevps.CoreFnExprVar) (a irA) {
	return me.fromˇIdent(&v.Value)
}

func (me *irAst) fromˇIdent(id *udevps.CoreFnIdent) irA {
	if len(id.ModuleName) == 0 {
		return ªSymPs(id.Identifier, me.irM.hasExport(id.Identifier))
	}
	return ªPkgSym(strings.Join(id.ModuleName, "."), id.Identifier)
}
