package main

import (
	"fmt"
	"strings"
)

func (me *irMeta) populateGoTypeDefs() {
	cfg := &ProjCfg.CodeGen

	//	TYPE SYNONYMS
	for _, ts := range me.EnvTypeSyns {
		if tc := me.tc(ts.Name); tc == nil {
			gtd, tdict := &irGoNamedTypeRef{Export: me.hasExport(ts.Name)}, map[string][]string{}
			gtd.setBothNamesFromPsName(ts.Name)
			gtd.Ref.setFrom(me.toIrGoTypeRef(tdict, ts.Ref))
			gtd.Ref.origs = irPsTypeRefs{ts.Ref}
			me.GoTypeDefs = append(me.GoTypeDefs, gtd)
		}
	}

	//	TYPE-CLASSES
	for _, tc := range me.EnvTypeClasses {
		tdict, gtd := map[string][]string{}, &irGoNamedTypeRef{Export: me.hasExport(tc.Name)}
		gtd.setBothNamesFromPsName(tc.Name)
		gtd.NameGo = fmt.Sprintf(cfg.Fmt.IfaceName_TypeClass, gtd.NameGo)
		gtd.Ref.I = &irGoTypeRefInterface{origClass: tc}
		for _, tcm := range tc.Members {
			method := &irGoNamedTypeRef{Export: true, Ref: irGoTypeRef{F: &irGoTypeRefFunc{origTcMem: tcm}}}
			method.setBothNamesFromPsName(tcm.Name)
			method.Ref.F.copyArgTypesOnlyFrom(false, nil, me.toIrGoTypeRef(tdict, tcm.Ref))
			method.Ref.origs = irPsTypeRefs{tcm.Ref}
			gtd.Methods = append(gtd.Methods, method)
		}
		me.GoTypeDefs = append(me.GoTypeDefs, gtd)
	}

	//	TYPE-CLASS INSTANCES
	for _, tci := range me.EnvTypeClassInsts {
		gtd := &irGoNamedTypeRef{Export: false, Ref: irGoTypeRef{S: &irGoTypeRefStruct{origInst: tci}}}
		gtd.setBothNamesFromPsName(tci.Name)
		gtd.NameGo = fmt.Sprintf(cfg.Fmt.StructName_InstImpl, gtd.NameGo)
		me.GoTypeDefs = append(me.GoTypeDefs, gtd)
	}

	//	ALGEBRAIC DATA TYPES
	me.GoTypeDefs = append(me.GoTypeDefs, me.toIrGoDataDefs(me.EnvTypeDataDecls)...)

	//	POST TYPE-GEN FIXUPS
	if cfg.TypeSynsForSingletonStructs {
		modpref := me.mod.qName + "."
		for _, gtd := range me.GoTypeDefs {
			if gtdrefstruct := gtd.Ref.S; gtdrefstruct != nil && len(gtdrefstruct.Fields) == 1 {
				field, cando := gtdrefstruct.Fields[0], len(gtd.Methods) == 0 // we need additional precautions below only if struct has methods
				for tref := &field.Ref; (!cando) && tref != nil; {
					if issyn := tref.Q != nil; !issyn { // field type isn't synonym:
						tref, cando = nil, tref.A != nil || tref.F != nil || tref.E != nil // then it's ok if array or func
					} else if strings.HasPrefix(tref.Q.QName, "Prim.") { // if it's synonym, a prim is always ok
						tref, cando = nil, true
					} else if strings.HasPrefix(tref.Q.QName, modpref) { // if it's synonym for package-local type?
						if gtdr := me.goTypeDefByPsName(tref.Q.QName[len(modpref):], false); gtdr == nil { // but it doesn't exist (not ever likely but hey)
							tref, cando = nil, false
						} else { // we capture that package-local type being referenced, to perform the same checks again in the next iteration
							tref = &gtdr.Ref
						}
					} else { // synonym for package-external type, we don't ditch the struct then because
						tref, cando = nil, false
					}
				}
				if cando {
					gtd.Ref.setFrom(&field.Ref)
				}
			}
		}
	}
	for i := 0; i < len(me.GoTypeDefs); i++ {
		if gtd := me.GoTypeDefs[i]; gtd.Ref.Q != nil && strings.HasPrefix(gtd.Ref.Q.QName, "Prim.") {
			tname := gtd.Ref.Q.QName[5:]
			switch tname {
			case "Char", "String", "Int", "Number", "Boolean":
			case "Array":
				me._primArrAliases = append(me._primArrAliases, gtd)
				me.GoTypeDefs.removeAt(i)
				i--
			default:
				panic(notImplErr("prim type", tname, me.mod.srcFilePath))
			}
		}
	}
}

func (me *irMeta) toIrGoDataDefs(typedatadecls []*irPsTypeDataDef) (gtds irGoNamedTypeRefs) {
	for _, td := range typedatadecls {
		tdict := map[string][]string{}
		isnewtype, hasctorargs, numctors := false, false, len(td.Ctors)
		gid := &irGoNamedTypeRef{Export: me.hasExport(td.Name), Ref: irGoTypeRef{origData: td}}
		gtds = append(gtds, gid)
		if numctors == 0 {
			gid.Ref.S = &irGoTypeRefStruct{}
		} else {
			gid.Ref.I = &irGoTypeRefInterface{}
		}
		gid.setBothNamesFromPsName(td.Name)
		for _, ctor := range td.Ctors {
			if numargs := len(ctor.Args); numargs > 0 {
				if hasctorargs = true; numargs == 1 && numctors == 1 {
					isnewtype = true
					if ct := ctor.Args[0].Type; ct != nil {
						if ctq := ct.Q; ctq != nil && ctq.QName == (me.mod.qName+"."+td.Name) {
							isnewtype = false
						}
					}
				}
			}
		}
		if cfg := &ProjCfg.CodeGen; cfg.TypeSynsForNewtypes && isnewtype {
			gid.Ref.clear(false)
			gid.Ref.setFrom(me.toIrGoTypeRef(tdict, td.Ctors[0].Args[0].Type))
		} else {
			isdataenum := cfg.DataAsEnumsWherePossible && (!hasctorargs) && numctors > 0
			if isdataenum {
				gid.Ref.clear(false)
				gid.Ref.E = &irGoTypeRefEnum{}
			}
			ctorlabel := func(ctor *irPsTypeDataCtor) string {
				return strings.NewReplacer("{D}", gid.NamePs, "{C}", ctor.Name).Replace(cfg.Fmt.StructName_DataCtor)
			}
			for _, ctor := range td.Ctors {
				if isdataenum {
					numem := &irGoNamedTypeRef{Export: ctor.Export}
					numem.setBothNamesFromPsName(ctorlabel(ctor))
					gid.Ref.E.Names = append(gid.Ref.E.Names, numem)
				} else {
					numargs := len(ctor.Args)
					ctor.ŧ = &irGoNamedTypeRef{Export: ctor.Export}
					ctor.ŧ.Ref.origCtor, ctor.ŧ.Ref.S = ctor, &irGoTypeRefStruct{PassByPtr: (hasctorargs && numargs >= cfg.PtrStructMinFieldCount)}
					ctor.ŧ.setBothNamesFromPsName(ctorlabel(ctor))
					ctor.ŧ.NamePs = "ĸ" + ctor.Name
					for ia, ctorarg := range ctor.Args {
						field := &irGoNamedTypeRef{}
						if field.Ref.setFrom(me.toIrGoTypeRef(tdict, ctorarg.Type)); field.Ref.Q != nil && field.Ref.Q.QName == (me.mod.qName+"."+ctor.Name) {
							//	an inconstructable self-recursive type, aka Data.Void
							field.turnRefIntoRefPtr()
						}
						field.NameGo = strings.NewReplacer("{C}", sanitizeSymbolForGo(ctor.Name, true), "{I}", fmt.Sprint(ia)).Replace(cfg.Fmt.FieldName_DataCtor)
						field.NamePs = fmt.Sprintf("value%d", ia)
						ctor.ŧ.Ref.S.Fields = append(ctor.ŧ.Ref.S.Fields, field)
					}
					gtds = append(gtds, ctor.ŧ)
				}
				if cfg.DataTypeAssertMethods {
					ifacemethod := &irGoNamedTypeRef{Export: ctor.Export}
					ifacemethod.setBothNamesFromPsName(ctor.Name)
					ifacemethod.Ref.origCtor, ifacemethod.Ref.F = ctor, &irGoTypeRefFunc{Rets: irGoNamedTypeRefs{&irGoNamedTypeRef{}}}
					if isdataenum {
						ifacemethod.Ref.F.Rets[0].Ref.Q = &irGoTypeRefSyn{QName: "Prim.Boolean"}
					} else {
						ifacemethod.Ref.F.Rets[0].Ref.P = &irGoTypeRefPtr{Of: &irGoNamedTypeRef{Ref: irGoTypeRef{Q: &irGoTypeRefSyn{QName: ctor.ŧ.NameGo}}}}
					}
					gid.Methods = append(gid.Methods, ifacemethod)
				}
			}
			if cfg.DataTypeAssertMethods {
				for _, ctor := range td.Ctors {
					if isdataenum {
						structmethod := gid.Methods.byPsName(ctor.Name).Ref.F
						structmethod.hasthis = true
						stmtret := ªRet(ªEq(ªSymGo(cfg.Fmt.Method_ThisName), ªSymPs(ctorlabel(ctor), ctor.Export)))
						structmethod.impl = &irABlock{Body: []irA{stmtret}}
					} else {
						for _, gidm := range gid.Methods {
							if gidm.Ref.origCtor != nil {
								structmethod := gidm.Ref.F.clone()
								structmethod.hasthis = (ctor == gidm.Ref.origCtor)
								stmtret := ªRet(ªNil())
								if structmethod.hasthis {
									stmtret.RetArg = ªSymGo(cfg.Fmt.Method_ThisName)
								}
								structmethod.impl = &irABlock{Body: []irA{stmtret}}
								ctor.ŧ.Methods = append(ctor.ŧ.Methods, &irGoNamedTypeRef{NameGo: gidm.NameGo, Export: gidm.Export, Ref: irGoTypeRef{origCtor: gidm.Ref.origCtor, F: structmethod}})
							}
						}
					}
				}
			}
		}
	}
	return
}

func (me *irMeta) toIrGoTypeRef(tdict map[string][]string, tref *irPsTypeRef) *irGoTypeRef {
	gtr := &irGoTypeRef{}
	if tref != nil {
		tAppl := tref.A
		tConstr := tref.C
		tCtor := tref.Q
		tForall := tref.F
		tRow := tref.R

		origs := irPsTypeRefs{tref}
		if tCtor != nil {
			gtr.Q = &irGoTypeRefSyn{QName: tCtor.QName}
		} else if tConstr != nil {
			gtr = me.toIrGoTypeRef(tdict, tConstr.Ref)
		} else if tForall != nil {
			gtr = me.toIrGoTypeRef(tdict, tForall.Ref)
		} else if tRow != nil {
			refstruc := &irGoTypeRefStruct{}
			myfield := &irGoNamedTypeRef{Export: true}
			myfield.setBothNamesFromPsName(tRow.Label)
			myfield.Ref.setFrom(me.toIrGoTypeRef(tdict, tRow.Left))
			refstruc.Fields = append(refstruc.Fields, myfield)
			if nextrow := me.toIrGoTypeRef(tdict, tRow.Right); nextrow != nil && nextrow.S != nil {
				refstruc.Fields = append(refstruc.Fields, nextrow.S.Fields...)
			}
			refstruc.PassByPtr = len(refstruc.Fields) >= ProjCfg.CodeGen.PtrStructMinFieldCount
			gtr.S = refstruc
		} else if tAppl != nil {
			if leftctor := tAppl.Left.Q; leftctor != nil {
				if leftctor.QName == "Prim.Record" {
					gtr = me.toIrGoTypeRef(tdict, tAppl.Right)
				} else if leftctor.QName == "Prim.Array" {
					refarr := &irGoTypeRefArray{Of: &irGoNamedTypeRef{}}
					refarr.Of.Ref.setFrom(me.toIrGoTypeRef(tdict, tAppl.Right))
					gtr.A = refarr
				} else { // unary known-type app (like Maybe, List, Array etc)
					gtr.Q = &irGoTypeRefSyn{QName: leftctor.QName}
				}
			} else if leftappl := tAppl.Left.A; leftappl != nil {
				if leftappl.Left.Q != nil {
					if leftappl.Left.Q.QName == "Prim.Function" {
						gtr.F = &irGoTypeRefFunc{}
						gtr.F.Args = irGoNamedTypeRefs{&irGoNamedTypeRef{}}
						gtr.F.Args[0].Ref.setFrom(me.toIrGoTypeRef(tdict, leftappl.Right))
						gtr.F.Rets = irGoNamedTypeRefs{&irGoNamedTypeRef{}}
						gtr.F.Rets[0].Ref.setFrom(me.toIrGoTypeRef(tdict, tAppl.Right))
					} else { // n>1-ary type app (like Either)
						gtr.Q = &irGoTypeRefSyn{QName: leftappl.Left.Q.QName}
					}
				} else {
					if strings.HasPrefix(me.mod.srcFilePath, "bower_components/purescript-prelude") {
						println(me.mod.srcFilePath + "\t\t\t" + tref.String())
					}
				}
			}
		}
		gtr.origs = append(origs, gtr.origs...) // prepend "ours" in front, in case it has any from one of the above branches
	}
	return gtr
}
