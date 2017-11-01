package main

import (
	"fmt"
)

func (me *irMeta) populateGoTypeDefs() {
	for _, ts := range me.EnvTypeSyns {
		tc, gtd, tdict := me.tc(ts.Name), &irANamedTypeRef{Export: me.hasExport(ts.Name)}, map[string][]string{}
		gtd.setBothNamesFromPsName(ts.Name)
		gtd.setRefFrom(me.toIrATypeRef(tdict, ts.Ref))
		if tc != nil {
			if gtd.NameGo += "ᛌ"; gtd.RefStruct != nil {
				gtd.RefStruct.PassByPtr = true
				for _, gtdf := range gtd.RefStruct.Fields {
					if gtdf.Export != gtd.Export {
						gtdf.Export = gtd.Export
						gtdf.setBothNamesFromPsName(gtdf.NamePs)
					}
					if tcm := tc.memberBy(gtdf.NamePs); tcm == nil {
						if rfn := gtdf.RefFunc; rfn == nil {
							// panic(notImplErr("non-func super-class-referencing-struct-field type for", gtdf.NamePs, me.mod.srcFilePath))
						} else {
							for retfunc := rfn.Rets[0].RefFunc; retfunc != nil; retfunc = rfn.Rets[0].RefFunc {
								rfn = retfunc
							}
							rfn.Rets[0].turnRefIntoRefPtr()
						}
					}
				}
			}
		}
		me.GoTypeDefs = append(me.GoTypeDefs, gtd)
	}
	for _, tc := range me.EnvTypeClasses {
		tsynfound := false
		for _, ts := range me.EnvTypeSyns {
			if tsynfound = (ts.Name == tc.Name); tsynfound {
				break
			}
		}
		if !tsynfound {
			panic(notImplErr("lack of pre-formed type-synonym for type-class", tc.Name, me.mod.srcFilePath))
			// tdict, gtd := map[string][]string{}, &irANamedTypeRef{Export: me.hasExport(tc.Name)}
			// gtd.setBothNamesFromPsName(tc.Name)
			// gtd.NameGo += "ˇ"
			// gtd.RefStruct = &irATypeRefStruct{PassByPtr: true}
			// for _, tcm := range tc.Members {
			// 	tcmfield := &irANamedTypeRef{Export: true}
			// 	tcmfield.setBothNamesFromPsName(tcm.Name)
			// 	tcmfield.setRefFrom(me.toIrATypeRef(tdict, tcm.Ref))
			// 	gtd.RefStruct.Fields = append(gtd.RefStruct.Fields, tcmfield)
			// }
			// me.GoTypeDefs = append(me.GoTypeDefs, gtd)
		}
	}
	me.GoTypeDefs = append(me.GoTypeDefs, me.toIrADataTypeDefs(me.EnvTypeDataDecls)...)
}

func (me *irMeta) toIrADataTypeDefs(typedatadecls []*irMTypeDataDef) (gtds irANamedTypeRefs) {
	for _, td := range typedatadecls {
		tdict := map[string][]string{}
		if numctors := len(td.Ctors); numctors == 0 {
			// panic(notImplErr(me.mod.srcFilePath+": unexpected ctor absence for", td.Name, td))
		} else {
			isnewtype, hasctorargs := false, false
			gid := &irANamedTypeRef{RefInterface: &irATypeRefInterface{xtd: td}, Export: me.hasExport(td.Name)}
			gid.setBothNamesFromPsName(td.Name)
			for _, ctor := range td.Ctors {
				if numargs := len(ctor.Args); numargs > 0 {
					if hasctorargs = true; numargs == 1 && numctors == 1 {
						if tc, _ := ctor.Args[0].Type.(*irMTypeRefConstruct); tc != nil && tc.QName != (me.mod.qName+"."+td.Name) {
							isnewtype = true
						}
					}
				}
			}
			if isnewtype {
				gid.RefInterface = nil
				gid.setRefFrom(me.toIrATypeRef(tdict, td.Ctors[0].Args[0].Type))
			} else {
				for _, ctor := range td.Ctors {
					ctor.ŧ = &irANamedTypeRef{Export: me.hasExport(gid.NamePs + "ĸ" + ctor.Name),
						RefStruct: &irATypeRefStruct{PassByPtr: (hasctorargs && len(ctor.Args) >= Proj.BowerJsonFile.Gonad.CodeGen.PtrStructMinFieldCount)}}
					ctor.ŧ.setBothNamesFromPsName(gid.NamePs + "۰" + ctor.Name)
					ctor.ŧ.NamePs = ctor.Name
					for ia, ctorarg := range ctor.Args {
						field := &irANamedTypeRef{}
						if field.setRefFrom(me.toIrATypeRef(tdict, ctorarg.Type)); field.RefAlias == (me.mod.qName + "." + ctor.Name) {
							//	an inconstructable self-recursive type, aka Data.Void
							field.turnRefIntoRefPtr()
						}
						field.NameGo = fmt.Sprintf("%s%d", sanitizeSymbolForGo(ctor.Name, true), ia)
						field.NamePs = fmt.Sprintf("value%d", ia)
						ctor.ŧ.RefStruct.Fields = append(ctor.ŧ.RefStruct.Fields, field)
					}
					gtds = append(gtds, ctor.ŧ)
				}
			}
			gtds = append(gtds, gid)
		}
	}
	return
}

func (me *irMeta) toIrATypeRef(tdict map[string][]string, tref irMTypeRef) interface{} {
	// funcyhackery := func(ret *irMTypeRef) interface{} {
	// 	funtype := &irATypeRefFunc{}
	// 	funtype.Args = irANamedTypeRefs{&irANamedTypeRef{}}
	// 	funtype.Args[0].setRefFrom(me.toIrATypeRef(tdict, tr.TypeApp.Left.TypeApp.Right))
	// 	funtype.Rets = irANamedTypeRefs{&irANamedTypeRef{}}
	// 	funtype.Rets[0].setRefFrom(me.toIrATypeRef(tdict, ret))
	// 	return funtype
	// }
	switch tr := tref.(type) {
	case *irMTypeRefConstruct:
		return tr.QName
	case *irMTypeRefEmpty:
		return nil
	case *irMTypeRefVar:
		return &irATypeRefInterface{isTypeVar: true}
	case *irMTypeRefConstrained:
		return me.toIrATypeRef(tdict, tr.Ref)
	case *irMTypeRefForall:
		return me.toIrATypeRef(tdict, tr.Ref)
	case *irMTypeRefSkolem:
		return fmt.Sprintf("Skolem_%s_scope%d_value%d", tr.Name, tr.Scope, tr.Value)
	case *irMTypeRefRow:
		rectype := &irATypeRefStruct{}
		myfield := &irANamedTypeRef{Export: true}
		myfield.setBothNamesFromPsName(tr.Label)
		myfield.setRefFrom(me.toIrATypeRef(tdict, tr.Left))
		rectype.Fields = append(rectype.Fields, myfield)
		if nextrow, _ := me.toIrATypeRef(tdict, tr.Right).(*irATypeRefStruct); nextrow != nil {
			rectype.Fields = append(rectype.Fields, nextrow.Fields...)
		}
		rectype.PassByPtr = len(rectype.Fields) >= Proj.BowerJsonFile.Gonad.CodeGen.PtrStructMinFieldCount
		return rectype
	case *irMTypeRefAppl:
		if lc, _ := tr.Left.(*irMTypeRefConstruct); lc != nil {
			if lc.QName == "Prim.Record" {
				return me.toIrATypeRef(tdict, tr.Right)
			} else if lc.QName == "Prim.Array" {
				array := &irATypeRefArray{Of: &irANamedTypeRef{}}
				array.Of.setRefFrom(me.toIrATypeRef(tdict, tr.Right))
				return array
				// } else if strings.HasPrefix(tr.TypeApp.Left.TypeConstructor, "Prim.") {
				// 	panic(notImplErr("type-app left-hand primitive", tr.TypeApp.Left.TypeConstructor, me.mod.srcFilePath))
				// } else if tr.TypeApp.Left.TypeApp != nil && tr.TypeApp.Left.TypeApp.Left.TypeConstructor == "Prim.Function" && tr.TypeApp.Left.TypeApp.Right.TypeApp != nil && tr.TypeApp.Left.TypeApp.Right.TypeApp.Left.TypeConstructor == "Prim.Record" && tr.TypeApp.Right.TypeApp != nil && tr.TypeApp.Right.TypeApp.Left != nil {
				// 	return funcyhackery(tr.TypeApp.Right.TypeApp.Left)
				// } else if tr.TypeApp.Left.TypeApp != nil && (tr.TypeApp.Left.TypeApp.Left.TypeConstructor == "Prim.Function" || /*insanely hacky*/ tr.TypeApp.Right.TypeVar != "") {
				// 	return funcyhackery(tr.TypeApp.Right)
				// } else if tr.TypeApp.Left.TypeConstructor != "" {
				// 	return me.toIrATypeRef(tdict, tr.TypeApp.Left)
			}
		}
	}
	return nil
}
