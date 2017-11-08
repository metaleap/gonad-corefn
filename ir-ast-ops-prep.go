package main

import (
	"github.com/metaleap/go-util/dev/ps"
)

func (me *irAst) prepFromCore() {
	me.irABlock.root = me
	me.mod.coreFn.Prep()
	me.Comments = me.mod.coreFn.Comments

	for _, cfndecl := range me.mod.coreFn.Decls {
		if cfndecl.CoreFnDeclBind != nil {
			me.add(me.fromCoreFnˇDecl(cfndecl.CoreFnDeclBind))
		} else {
			for _, cdb := range cfndecl.Binds {
				me.add(me.fromCoreFnˇDecl(&cdb))
			}
		}
	}
}

func (me *irAst) fromCoreFnˇDecl(bind *udevps.CoreFnDeclBind) (a irA) {
	v := ªLet("", bind.Identifier, ªI(1))
	v.Export = me.irM.hasExport(bind.Identifier)
	v.setBothNamesFromPsName(bind.Identifier)
	bstr := bind.Expression.String()
	v.NameGo += "ˇ" + bstr + "ˇ" + bind.Identifier
	if bstr != "Abs" {
		println(bstr)
	}
	a = v
	return
}
