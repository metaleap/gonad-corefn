package main

func (me *irAst) prepFromCore() {
	me.irABlock.root = me
	me.mod.coreFn.Prep()
	me.Comments = me.mod.coreFn.Comments

	for i, _ := range me.mod.coreFn.Decls {
		cfndecl := &me.mod.coreFn.Decls[i]
		if cfndecl.CoreFnDeclBind != nil {
			me.add(me.fromˇDeclBind(&me.irABlock, cfndecl.CoreFnDeclBind))
		} else {
			for i, _ := range cfndecl.Binds {
				me.add(me.fromˇDeclBind(&me.irABlock, &cfndecl.Binds[i]))
			}
		}
	}
}
