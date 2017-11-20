package main

type funcIra2Bool func(irA) bool

type funcIra2Ira func(irA) irA

func irALookupInAncestorBlocks(a irA, check funcIra2Bool) irA {
	for nextparent := a.Parent(); nextparent != nil; nextparent = nextparent.Parent() {
		switch p := nextparent.(type) {
		case *irABlock:
			for _, stmt := range p.Body {
				if check(stmt) {
					return stmt
				}
			}
		}
	}
	return nil
}

func irALookupBelow(me irA, intofuncvals bool, check func(irA) bool) (all []irA) {
	walk(me, intofuncvals, func(a irA) irA {
		if check(a) {
			all = append(all, a)
		}
		return a
	})
	return
}

func irALookupBelowˇIsType(me irA, intofuncvals bool) (all []*irAIsType) {
	irALookupBelow(me, intofuncvals, func(a irA) bool {
		if ax, _ := a.(*irAIsType); ax != nil {
			all = append(all, ax)
		}
		return false
	})
	return
}

func irALookupBelowˇRet(me irA, intofuncvals bool) (all []*irARet) {
	irALookupBelow(me, intofuncvals, func(a irA) bool {
		if ax, _ := a.(*irARet); ax != nil {
			all = append(all, ax)
		}
		return false
	})
	return
}

func (me *irABase) outerFunc() *irAFunc {
	for nextup := me.parent; nextup != nil; nextup = nextup.Parent() {
		if nextfn, _ := nextup.(*irAFunc); nextfn != nil {
			return nextfn
		}
	}
	return nil
}

func (me *irABlock) perFuncDown(on func(*irAFunc)) {
	walk(me, false, func(a irA) irA { // false == don't recurse into inner func-vals
		switch ax := a.(type) {
		case *irAFunc: // we hit a func-val in the current block
			on(ax)                      // invoke handler for it
			ax.FuncImpl.perFuncDown(on) // only now recurse into itself
		}
		return a
	})
}

func (me *irABase) perFuncUp(on func(*irAFunc)) {
	for nextup := me.parent; nextup != nil; nextup = nextup.Parent() {
		if nextfn, _ := nextup.(*irAFunc); nextfn != nil {
			on(nextfn)
		}
	}
}

func (me *irAst) topLevelDefs(okay funcIra2Bool) (defs []irA) {
	for _, ast := range me.Block.Body {
		if okay == nil || okay(ast) {
			defs = append(defs, ast)
		}
	}
	return
}

func (me *irAst) walkTopLevelDefs(on func(irA)) {
	for _, ast := range me.Block.Body {
		on(ast)
	}
}

func (me *irAst) walk(on funcIra2Ira) {
	for i, a := range me.Block.Body {
		if a != nil {
			me.Block.Body[i] = walk(a, true, on)
		}
	}
	for _, tr := range me.irM.GoTypeDefs {
		for _, trm := range tr.Methods {
			if trm.Ref.F.impl != nil {
				trm.Ref.F.impl, _ = walk(trm.Ref.F.impl, true, on).(*irABlock)
			}
		}
	}
	for i, tcf := range me.culled.typeCtorFuncs {
		me.culled.typeCtorFuncs[i] = walk(tcf, true, on).(*irACtor)
	}
}

func walk(ast irA, intofuncvals bool, on funcIra2Ira) irA {
	if ast != nil {
		switch a := ast.(type) {
		// why extra nil checks some places below: we do have the rare case of ast!=nil and ast.(type) set and still holding a null-ptr
		case *irABlock:
			if a != nil {
				for i := range a.Body {
					a.Body[i] = walk(a.Body[i], intofuncvals, on)
				}
			}
		case *irACall:
			a.Callee = walk(a.Callee, intofuncvals, on)
			for i := range a.CallArgs {
				a.CallArgs[i] = walk(a.CallArgs[i], intofuncvals, on)
			}
		case *irAConst:
			a.ConstVal = walk(a.ConstVal, intofuncvals, on)
		case *irADot:
			a.DotLeft, a.DotRight = walk(a.DotLeft, intofuncvals, on), walk(a.DotRight, intofuncvals, on)
		case *irAFor:
			a.ForCond = walk(a.ForCond, intofuncvals, on)
			if tmp, _ := walk(a.ForRange, intofuncvals, on).(*irALet); tmp != nil {
				a.ForRange = tmp
			}
			if tmp, _ := walk(a.ForDo, intofuncvals, on).(*irABlock); tmp != nil {
				a.ForDo = tmp
			}
			for i, fi := range a.ForInit {
				if tmp, _ := walk(fi, intofuncvals, on).(*irALet); tmp != nil {
					a.ForInit[i] = tmp
				}
			}
			for i, fs := range a.ForStep {
				if tmp, _ := walk(fs, intofuncvals, on).(*irASet); tmp != nil {
					a.ForStep[i] = tmp
				}
			}
		case *irACtor:
			if tmp, _ := walk(a.FuncImpl, intofuncvals, on).(*irABlock); tmp != nil {
				a.FuncImpl = tmp
			}
		case *irAFunc:
			if intofuncvals {
				if tmp, _ := walk(a.FuncImpl, intofuncvals, on).(*irABlock); tmp != nil {
					a.FuncImpl = tmp
				}
			}
		case *irAIf:
			a.If = walk(a.If, intofuncvals, on)
			if tmp, _ := walk(a.Then, intofuncvals, on).(*irABlock); tmp != nil {
				a.Then = tmp
			}
			if tmp, _ := walk(a.Else, intofuncvals, on).(*irABlock); tmp != nil {
				a.Else = tmp
			}
		case *irAIndex:
			a.IdxLeft, a.IdxRight = walk(a.IdxLeft, intofuncvals, on), walk(a.IdxRight, intofuncvals, on)
		case *irAOp1:
			a.Of = walk(a.Of, intofuncvals, on)
		case *irAOp2:
			a.Left, a.Right = walk(a.Left, intofuncvals, on), walk(a.Right, intofuncvals, on)
		case *irAPanic:
			a.PanicArg = walk(a.PanicArg, intofuncvals, on)
		case *irARet:
			a.RetArg = walk(a.RetArg, intofuncvals, on)
		case *irASet:
			a.SetLeft, a.ToRight = walk(a.SetLeft, intofuncvals, on), walk(a.ToRight, intofuncvals, on)
		case *irALet:
			if a != nil {
				a.LetVal = walk(a.LetVal, intofuncvals, on)
			}
		case *irAIsType:
			a.ExprToTest = walk(a.ExprToTest, intofuncvals, on)
		case *irAToType:
			a.ExprToConv = walk(a.ExprToConv, intofuncvals, on)
		case *irALitArr:
			for i, av := range a.ArrVals {
				a.ArrVals[i] = walk(av, intofuncvals, on)
			}
		case *irALitObj:
			for i, av := range a.ObjFields {
				if tmp, _ := walk(av, intofuncvals, on).(*irALitObjField); tmp != nil {
					a.ObjFields[i] = tmp
				}
			}
		case *irALitObjField:
			a.FieldVal = walk(a.FieldVal, intofuncvals, on)
		case *irASwitch:
			a.On = walk(a.On, intofuncvals, on)
			for i, cc := range a.Cases {
				if tmp, _ := walk(cc, intofuncvals, on).(*irACase); tmp != nil {
					a.Cases[i] = tmp
				}
			}
		case *irACase:
			a.CaseCond = walk(a.CaseCond, intofuncvals, on)
			if tmp, _ := walk(a.CaseBody, intofuncvals, on).(*irABlock); tmp != nil {
				a.CaseBody = tmp
			}
		case *irAComments, *irAPkgSym, *irANil, *irALitBool, *irALitNum, *irALitInt, *irALitChar, *irALitStr, *irASym:
		default:
			panicWithType(ast.Base().srcFilePath(), ast, "walk")
		}
		if nuast := on(ast); nuast != ast {
			if oldp := ast.Parent(); nuast != nil {
				nuast.Base().parent = oldp
			}
			ast = nuast
		}
	}
	return ast
}
