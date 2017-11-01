package main

import (
	"fmt"
)

func (me *irMeta) populateGoTypeDefs() {
	//	TYPE ALIASES / SYNONYMS
	for _, ts := range me.EnvTypeSyns {
		tc, gtd, tdict := me.tc(ts.Name), &irGoNamedTypeRef{Export: me.hasExport(ts.Name)}, map[string][]string{}
		gtd.setBothNamesFromPsName(ts.Name)
		gtd.setRefFrom(me.toIrGoTypeRef(tdict, ts.Ref))
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

	//	TYPE CLASSES + INSTANCES
	for _, tc := range me.EnvTypeClasses {
		tsynfound := false
		for _, ts := range me.EnvTypeSyns {
			if tsynfound = (ts.Name == tc.Name); tsynfound {
				break
			}
		}
		if !tsynfound {
			panic(notImplErr("lack of pre-formed type-synonym for type-class", tc.Name, me.mod.srcFilePath))
			// tdict, gtd := map[string][]string{}, &irGoNamedTypeRef{Export: me.hasExport(tc.Name)}
			// gtd.setBothNamesFromPsName(tc.Name)
			// gtd.NameGo += "ˇ"
			// gtd.RefStruct = &irGoTypeRefStruct{PassByPtr: true}
			// for _, tcm := range tc.Members {
			// 	tcmfield := &irGoNamedTypeRef{Export: true}
			// 	tcmfield.setBothNamesFromPsName(tcm.Name)
			// 	tcmfield.setRefFrom(me.toIrGoTypeRef(tdict, tcm.Ref))
			// 	gtd.RefStruct.Fields = append(gtd.RefStruct.Fields, tcmfield)
			// }
			// me.GoTypeDefs = append(me.GoTypeDefs, gtd)
		}
	}

	//	ALGEBRAIC DATA TYPES
	me.GoTypeDefs = append(me.GoTypeDefs, me.toIrGoDataDefs(me.EnvTypeDataDecls)...)
}

func (me *irMeta) toIrGoDataDefs(typedatadecls []*irPsTypeDataDef) (gtds irGoNamedTypeRefs) {
	for _, td := range typedatadecls {
		tdict := map[string][]string{}
		if numctors := len(td.Ctors); numctors == 0 {
			// panic(notImplErr(me.mod.srcFilePath+": unexpected ctor absence for", td.Name, td))
		} else {
			isnewtype, hasctorargs := false, false
			gid := &irGoNamedTypeRef{RefInterface: &irGoTypeRefInterface{xtd: td}, Export: me.hasExport(td.Name)}
			gid.setBothNamesFromPsName(td.Name)
			for _, ctor := range td.Ctors {
				if numargs := len(ctor.Args); numargs > 0 {
					if hasctorargs = true; numargs == 1 && numctors == 1 {
						if tc := ctor.Args[0].Type.Q; tc != nil && tc.QName != (me.mod.qName+"."+td.Name) {
							isnewtype = true
						}
					}
				}
			}
			if isnewtype {
				gid.RefInterface = nil
				gid.setRefFrom(me.toIrGoTypeRef(tdict, td.Ctors[0].Args[0].Type))
			} else {
				for _, ctor := range td.Ctors {
					ctor.ŧ = &irGoNamedTypeRef{Export: me.hasExport(gid.NamePs + "ĸ" + ctor.Name),
						RefStruct: &irGoTypeRefStruct{PassByPtr: (hasctorargs && len(ctor.Args) >= Proj.BowerJsonFile.Gonad.CodeGen.PtrStructMinFieldCount)}}
					ctor.ŧ.setBothNamesFromPsName(gid.NamePs + "۰" + ctor.Name)
					ctor.ŧ.NamePs = ctor.Name
					for ia, ctorarg := range ctor.Args {
						field := &irGoNamedTypeRef{}
						if field.setRefFrom(me.toIrGoTypeRef(tdict, ctorarg.Type)); field.RefAlias != nil && field.RefAlias.Q == (me.mod.qName+"."+ctor.Name) {
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

func (me *irMeta) toIrGoTypeRef(tdict map[string][]string, tref *irPsTypeRef) interface{} {
	tAppl := tref.A
	tConstr := tref.C
	tEmpty := tref.E
	tForall := tref.F
	tCtor := tref.Q
	tRow := tref.R
	tSkolem := tref.S
	// tTlStr := tref.TlS
	tVar := tref.V

	if tCtor != nil {
		return &irGoTypeRefAlias{Q: tCtor.QName}
	} else if tEmpty != nil {
		return nil
	} else if tVar != nil {
		return &irGoTypeRefInterface{isTypeVar: true}
	} else if tConstr != nil {
		return me.toIrGoTypeRef(tdict, tConstr.Ref)
	} else if tForall != nil {
		return me.toIrGoTypeRef(tdict, tForall.Ref)
	} else if tSkolem != nil {
		return fmt.Sprintf("Skolem_%s_scope%d_value%d", tSkolem.Name, tSkolem.Scope, tSkolem.Value)
	} else if tRow != nil {
		rectype := &irGoTypeRefStruct{}
		myfield := &irGoNamedTypeRef{Export: true}
		myfield.setBothNamesFromPsName(tRow.Label)
		myfield.setRefFrom(me.toIrGoTypeRef(tdict, tRow.Left))
		rectype.Fields = append(rectype.Fields, myfield)
		if nextrow, _ := me.toIrGoTypeRef(tdict, tRow.Right).(*irGoTypeRefStruct); nextrow != nil {
			rectype.Fields = append(rectype.Fields, nextrow.Fields...)
		}
		rectype.PassByPtr = len(rectype.Fields) >= Proj.BowerJsonFile.Gonad.CodeGen.PtrStructMinFieldCount
		return rectype
	} else if tAppl != nil {
		if leftctor := tAppl.Left.Q; leftctor != nil {
			if leftctor.QName == "Prim.Record" {
				return me.toIrGoTypeRef(tdict, tAppl.Right)
			} else if leftctor.QName == "Prim.Array" {
				array := &irGoTypeRefArray{Of: &irGoNamedTypeRef{}}
				array.Of.setRefFrom(me.toIrGoTypeRef(tdict, tAppl.Right))
				return array
				// } else if strings.HasPrefix(tr.TypeApp.Left.TypeConstructor, "Prim.") {
				// 	panic(notImplErr("type-app left-hand primitive", tr.TypeApp.Left.TypeConstructor, me.mod.srcFilePath))
				// } else if tr.TypeApp.Left.TypeApp != nil && tr.TypeApp.Left.TypeApp.Left.TypeConstructor == "Prim.Function" && tr.TypeApp.Left.TypeApp.Right.TypeApp != nil && tr.TypeApp.Left.TypeApp.Right.TypeApp.Left.TypeConstructor == "Prim.Record" && tr.TypeApp.Right.TypeApp != nil && tr.TypeApp.Right.TypeApp.Left != nil {
				// 	return funcyhackery(tr.TypeApp.Right.TypeApp.Left)
				// } else if tr.TypeApp.Left.TypeApp != nil && (tr.TypeApp.Left.TypeApp.Left.TypeConstructor == "Prim.Function" || /*insanely hacky*/ tr.TypeApp.Right.TypeVar != "") {
				// 	return funcyhackery(tr.TypeApp.Right)
				// } else if tr.TypeApp.Left.TypeConstructor != "" {
				// 	return me.toIrGoTypeRef(tdict, tr.TypeApp.Left)
			}
		}
	}
	return nil
}
