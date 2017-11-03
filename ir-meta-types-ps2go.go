package main

import (
	"fmt"
	"strings"
)

func (me *irMeta) populateGoTypeDefs() {
	cfg := Proj.BowerJsonFile.Gonad.CodeGen

	//	TYPE ALIASES / SYNONYMS
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
	if cfg.TypeAliasesForSingletonStructs {
		modpref := me.mod.qName + "."
		for _, gtd := range me.GoTypeDefs {
			if gtd.Ref.S != nil && len(gtd.Ref.S.Fields) == 1 {
				field, cando := gtd.Ref.S.Fields[0], len(gtd.Methods) == 0 // we need additional precautions below only if struct has methods
				for tref := &field.Ref; (!cando) && tref != nil; {
					if isalias := tref.Q != nil; !isalias { // field type isn't alias:
						tref, cando = nil, tref.A != nil || tref.F != nil // then it's ok if array or func
					} else if strings.HasPrefix(tref.Q.QName, "Prim.") { // if it's alias, a prim is always ok
						tref, cando = nil, true
					} else if strings.HasPrefix(tref.Q.QName, modpref) { // if it's aliasing to package-local type?
						if gtdr := me.goTypeDefByPsName(tref.Q.QName[len(modpref):], false); gtdr == nil { // but it doesn't exist (not ever likely but hey)
							tref, cando = nil, false
						} else { // we capture that package-local type being referenced, to perform the same checks again in the next iteration
							tref = &gtdr.Ref
						}
					} else { // aliasing to external type, more likely than not an interface, we don't ditch the struct then
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
					if tc := ctor.Args[0].Type.Q; tc == nil || tc.QName != (me.mod.qName+"."+td.Name) {
						isnewtype = true
					}
				}
			}
		}
		if cfg := &Proj.BowerJsonFile.Gonad.CodeGen; cfg.TypeAliasesForNewtypes && isnewtype {
			gid.Ref.I = nil
			gid.Ref.setFrom(me.toIrGoTypeRef(tdict, td.Ctors[0].Args[0].Type))
		} else {
			for _, ctor := range td.Ctors {
				numargs := len(ctor.Args)
				ctor.ŧ = &irGoNamedTypeRef{Export: me.hasExport(gid.NamePs + "ĸ" + ctor.Name)}
				ctor.ŧ.Ref.origCtor, ctor.ŧ.Ref.S = ctor, &irGoTypeRefStruct{PassByPtr: (hasctorargs && numargs >= cfg.PtrStructMinFieldCount)}
				ctor.ŧ.setBothNamesFromPsName(strings.NewReplacer("{D}", gid.NamePs, "{C}", ctor.Name).Replace(cfg.Fmt.StructName_DataCtor))
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
				if cfg.DataTypeAssertMethods {
					ifacemethod := &irGoNamedTypeRef{Export: ctor.ŧ.Export}
					ifacemethod.setBothNamesFromPsName(ctor.Name)
					ifacemethod.Ref.origCtor, ifacemethod.Ref.F = ctor, &irGoTypeRefFunc{Rets: irGoNamedTypeRefs{&irGoNamedTypeRef{Ref: irGoTypeRef{
						P: &irGoTypeRefPtr{Of: &irGoNamedTypeRef{Ref: irGoTypeRef{Q: &irGoTypeRefAlias{QName: ctor.ŧ.NameGo}}}}}}}}

					gid.Methods = append(gid.Methods, ifacemethod)
				}
				gtds = append(gtds, ctor.ŧ)
			}
			if cfg.DataTypeAssertMethods {
				for _, ctor := range td.Ctors {
					for _, gidm := range gid.Methods {
						if gidm.Ref.origCtor != nil {
							structmethod := gidm.Ref.F.clone()
							structmethod.hasthis = (ctor == gidm.Ref.origCtor)
							stmtret := ªRet(ªNil())
							if structmethod.hasthis {
								stmtret.RetArg = ªSymGo(Proj.BowerJsonFile.Gonad.CodeGen.Fmt.Method_ThisName)
							}
							structmethod.impl = &irABlock{Body: []irA{stmtret}}
							ctor.ŧ.Methods = append(ctor.ŧ.Methods, &irGoNamedTypeRef{NameGo: gidm.NameGo, Export: gidm.Export, Ref: irGoTypeRef{origCtor: gidm.Ref.origCtor, F: structmethod}})
						}
					}
				}
			}
		}
	}
	return
}

func (me *irMeta) toIrGoTypeRef(tdict map[string][]string, tref *irPsTypeRef) *irGoTypeRef {
	tAppl := tref.A
	tConstr := tref.C
	tCtor := tref.Q
	tForall := tref.F
	tRow := tref.R

	origs := irPsTypeRefs{tref}
	gtr := &irGoTypeRef{}
	if tCtor != nil {
		gtr.Q = &irGoTypeRefAlias{QName: tCtor.QName}
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
		refstruc.PassByPtr = len(refstruc.Fields) >= Proj.BowerJsonFile.Gonad.CodeGen.PtrStructMinFieldCount
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
				gtr.Q = &irGoTypeRefAlias{QName: leftctor.QName}
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
					gtr.Q = &irGoTypeRefAlias{QName: leftappl.Left.Q.QName}
				}
			} else {
				// println(tref.String())
			}
		}
	}
	gtr.origs = append(origs, gtr.origs...) // prepend "ours" in front, in case it has any from one of the above branches
	return gtr
}
