package main

import (
	"github.com/metaleap/go-util/dev/ps"
	"github.com/metaleap/go-util/str"
)

func ªA(exprs ...irA) *irALitArr {
	a := &irALitArr{ArrVals: exprs}
	a.ExprType()
	return a
}

func ªB(literal bool) *irALitBool {
	a := &irALitBool{LitBool: literal}
	a.Ref.Q = &irGoTypeRefSyn{QName: "Prim.Boolean"}
	return a
}

func ªN(literal float64) *irALitNum {
	a := &irALitNum{LitNum: literal}
	a.Ref.Q = &irGoTypeRefSyn{QName: "Prim.Number"}
	return a
}

func ªI(literal int) *irALitInt {
	a := &irALitInt{LitInt: literal}
	a.Ref.Q = &irGoTypeRefSyn{QName: "Prim.Int"}
	return a
}

func ªO(typeref *irGoNamedTypeRef, fields ...*irALitObjField) *irALitObj {
	a := &irALitObj{ObjFields: fields}
	if typeref != nil {
		a.irGoNamedTypeRef = *typeref
	}
	for _, of := range a.ObjFields {
		of.parent = a
	}
	return a
}

func ªOFld(fieldval irA) *irALitObjField {
	a := &irALitObjField{FieldVal: fieldval}
	return a
}

func ªC(literal rune) *irALitChar {
	a := &irALitChar{LitChar: literal}
	a.Ref.Q = &irGoTypeRefSyn{QName: "Prim.Char"}
	return a
}

func ªS(literal string) *irALitStr {
	a := &irALitStr{LitStr: literal}
	a.Ref.Q = &irGoTypeRefSyn{QName: "Prim.String"}
	return a
}

func ªBlock(asts ...irA) *irABlock {
	a := &irABlock{Body: asts}
	for _, expr := range a.Body {
		expr.Base().parent = a
	}
	return a
}

func ªCall(callee irA, callargs ...irA) *irACall {
	a := &irACall{Callee: callee, CallArgs: callargs}
	a.Callee.Base().parent = a
	for _, expr := range callargs {
		expr.Base().parent = a
	}
	return a
}

func ªCase(cond irA) *irACase {
	c := &irACase{CaseCond: cond, CaseBody: &irABlock{}}
	cond.Base().parent, c.CaseBody.parent = c, c
	return c
}

func ªComments(comments ...udevps.CoreComment) *irAComments {
	a := &irAComments{irABase: irABase{Comments: comments}}
	return a
}

func ªConst(name *irGoNamedTypeRef, val irA) *irAConst {
	a, v := &irAConst{ConstVal: val}, val.Base()
	v.parent, a.irGoNamedTypeRef = a, v.irGoNamedTypeRef
	if name != nil {
		a.NameGo, a.NamePs = name.NameGo, name.NamePs
	}
	return a
}

func ªDot(left irA, right irA) *irADot {
	a := &irADot{DotLeft: left, DotRight: right}
	lb, rb := left.Base(), right.Base()
	lb.parent, rb.parent = a, a
	return a
}

func ªDotNamed(left string, right string) *irADot {
	return ªDot(ªSymGo(left), ªSymGo(right))
}

func ªEq(left irA, right irA) *irAOp2 {
	return ªO2(left, "==", right)
}

func ªFor() *irAFor {
	a := &irAFor{ForDo: ªBlock()}
	a.ForDo.parent = a
	return a
}

func ªFunc(maybesig *irGoTypeRefFunc) *irAFunc {
	a := &irAFunc{FuncImpl: ªBlock()}
	a.FuncImpl.parent = a
	if a.Ref.F = maybesig; maybesig == nil {
		a.Ref.F = &irGoTypeRefFunc{impl: a.FuncImpl}
	}
	return a
}

func ªIf(cond irA) *irAIf {
	a := &irAIf{If: cond, Then: ªBlock()}
	a.If.Base().parent, a.Then.parent = a, a
	return a
}

func ªIndex(left irA, right irA) *irAIndex {
	a := &irAIndex{IdxLeft: left, IdxRight: right}
	a.IdxLeft.Base().parent, a.IdxRight.Base().parent = a, a
	return a
}

func ªIs(expr irA, typeexpr string) *irAIsType {
	a := &irAIsType{ExprToTest: expr, TypeToTest: typeexpr}
	a.names.v, a.names.t, a.ExprToTest.Base().parent = expr.(irASymStr).symStr(), typeexpr, a
	return a
}

func ªLet(namego string, nameps string, val irA) *irALet {
	a := &irALet{LetVal: val}
	if val != nil {
		vb := val.Base()
		vb.parent = a
		a.irGoNamedTypeRef = vb.irGoNamedTypeRef
	}
	if namego == "" && nameps != "" {
		a.setBothNamesFromPsName(nameps)
	} else {
		a.NameGo, a.NamePs = namego, nameps
	}
	return a
}

func ªNil(comments ...udevps.CoreComment) *irANil {
	a := &irANil{}
	a.Comments = comments
	return a
}

func ªO1(op string, operand irA) *irAOp1 {
	a := &irAOp1{Op1: op, Of: operand}
	a.Of.Base().parent = a
	return a
}

func ªO2(left irA, op string, right irA) *irAOp2 {
	a := &irAOp2{Op2: op, Left: left, Right: right}
	a.Left.Base().parent, a.Right.Base().parent = a, a
	return a
}

func ªPanic(errarg irA) *irAPanic {
	a := &irAPanic{PanicArg: errarg}
	a.PanicArg.Base().parent = a
	return a
}

func ªPkgSym(pkgname string, symbol string) *irAPkgSym {
	if pkgname != "" {
		if mod := findModuleByPName(pkgname); mod != nil {
			symbol = ustr.Upper.Ensure(symbol, 0)
		}
	}
	a := &irAPkgSym{PkgName: pkgname, Symbol: symbol}
	return a
}

func ªRet(retarg irA) *irARet {
	a := &irARet{RetArg: retarg}
	if a.RetArg != nil {
		a.RetArg.Base().parent = a
	}
	return a
}

func ªSet(left irA, right irA) *irASet {
	a := &irASet{SetLeft: left, ToRight: right}
	a.SetLeft.Base().parent = a
	if right != nil {
		rb := right.Base()
		if rb.parent = a; rb.hasTypeInfo() {
			a.irGoNamedTypeRef = rb.irGoNamedTypeRef
		}
	}
	return a
}

func ªsetVarInGroup(namego string, right irA, typespec *irGoNamedTypeRef) *irASet {
	a := ªSet(ªSymGo(namego), right)
	if typespec != nil && typespec.hasTypeInfo() {
		a.copyTypeInfoFrom(typespec) // a.irGoNamedTypeRef = *typespec
	}
	a.isInVarGroup = true
	return a
}

func ªSwitch(on irA, cases []*irACase) *irASwitch {
	s := &irASwitch{On: on, Cases: cases}
	on.Base().parent = s
	for _, c := range cases {
		c.parent = s
	}
	return s
}

func ªSymGo(namego string) *irASym {
	a := &irASym{}
	a.NameGo = namego
	return a
}

func ªSymPs(nameps string, exported bool) *irASym {
	a := &irASym{}
	a.Export = exported
	a.setBothNamesFromPsName(nameps)
	return a
}

func ªTo(expr irA, pname string, tname string) *irAToType {
	a := &irAToType{ExprToConv: expr, TypePkg: pname, TypeName: tname}
	a.ExprToConv.Base().parent = a
	return a
}
