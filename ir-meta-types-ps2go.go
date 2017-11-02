package main

import (
	"fmt"
	"strings"
)

func (me *irMeta) populateGoTypeDefs() {
	//	TYPE ALIASES / SYNONYMS
	for _, ts := range me.EnvTypeSyns {
		if tc := me.tc(ts.Name); tc == nil {
			gtd, tdict := &irGoNamedTypeRef{Export: me.hasExport(ts.Name)}, map[string][]string{}
			gtd.setBothNamesFromPsName(ts.Name)
			gtd.Ref.setFrom(me.toIrGoTypeRef(tdict, ts.Ref))
			gtd.Ref.Orig = ts.Ref
			me.GoTypeDefs = append(me.GoTypeDefs, gtd)
		}
	}

	//	TYPE-CLASSES
	for _, tc := range me.EnvTypeClasses {
		tdict, gtd := map[string][]string{}, &irGoNamedTypeRef{Export: me.hasExport(tc.Name)}
		gtd.setBothNamesFromPsName(tc.Name)
		gtd.NameGo = fmt.Sprintf(Proj.BowerJsonFile.Gonad.CodeGen.Fmt.IfaceName_TypeClass, gtd.NameGo)
		gtd.Ref.I = &irGoTypeRefInterface{origClass: tc}
		for _, tcm := range tc.Members {
			method := &irGoNamedTypeRef{Export: true, Ref: irGoTypeRef{F: &irGoTypeRefFunc{origTcMem: tcm}}}
			method.setBothNamesFromPsName(tcm.Name)
			method.Ref.F.copyArgTypesOnlyFrom(false, me.toIrGoTypeRef(tdict, tcm.Ref).F)
			method.Ref.Orig = tcm.Ref
			gtd.Ref.I.Methods = append(gtd.Ref.I.Methods, method)
		}
		me.GoTypeDefs = append(me.GoTypeDefs, gtd)
	}

	//	TYPE-CLASS INSTANCES
	for _, tci := range me.EnvTypeClassInsts {
		gtd := &irGoNamedTypeRef{Export: false, Ref: irGoTypeRef{S: &irGoTypeRefStruct{origInst: tci}}}
		gtd.setBothNamesFromPsName(tci.Name)
		gtd.NameGo = fmt.Sprintf(Proj.BowerJsonFile.Gonad.CodeGen.Fmt.StructName_InstImpl, gtd.NameGo)
		me.GoTypeDefs = append(me.GoTypeDefs, gtd)
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
			gid := &irGoNamedTypeRef{Ref: irGoTypeRef{I: &irGoTypeRefInterface{origData: td}}, Export: me.hasExport(td.Name)}
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
				gid.Ref.I = nil
				gid.Ref.setFrom(me.toIrGoTypeRef(tdict, td.Ctors[0].Args[0].Type))
			} else {
				for _, ctor := range td.Ctors {
					ctor.ŧ = &irGoNamedTypeRef{Export: me.hasExport(gid.NamePs + "ĸ" + ctor.Name)}
					ctor.ŧ.Ref.S = &irGoTypeRefStruct{PassByPtr: (hasctorargs && len(ctor.Args) >= Proj.BowerJsonFile.Gonad.CodeGen.PtrStructMinFieldCount)}
					ctor.ŧ.setBothNamesFromPsName(strings.NewReplacer("{D}", gid.NamePs, "{C}", ctor.Name).Replace(Proj.BowerJsonFile.Gonad.CodeGen.Fmt.StructName_DataCtor))
					ctor.ŧ.NamePs = ctor.Name
					for ia, ctorarg := range ctor.Args {
						field := &irGoNamedTypeRef{}
						if field.Ref.setFrom(me.toIrGoTypeRef(tdict, ctorarg.Type)); field.Ref.Q != nil && field.Ref.Q.QName == (me.mod.qName+"."+ctor.Name) {
							//	an inconstructable self-recursive type, aka Data.Void
							field.turnRefIntoRefPtr()
						}
						field.NameGo = fmt.Sprintf("%s%d", sanitizeSymbolForGo(ctor.Name, true), ia)
						field.NamePs = fmt.Sprintf("value%d", ia)
						ctor.ŧ.Ref.S.Fields = append(ctor.ŧ.Ref.S.Fields, field)
					}
					gtds = append(gtds, ctor.ŧ)
				}
			}
			gtds = append(gtds, gid)
		}
	}
	return
}

func (me *irMeta) toIrGoTypeRef(tdict map[string][]string, tref *irPsTypeRef) *irGoTypeRef {
	tAppl := tref.A
	tCtor := tref.Q
	tRow := tref.R

	gtr := irGoTypeRef{Orig: tref}
	if tCtor != nil {
		gtr.Q = &irGoTypeRefAlias{QName: tCtor.QName}
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
				return me.toIrGoTypeRef(tdict, tAppl.Right)
			} else if leftctor.QName == "Prim.Array" {
				refarr := &irGoTypeRefArray{Of: &irGoNamedTypeRef{}}
				refarr.Of.Ref.setFrom(me.toIrGoTypeRef(tdict, tAppl.Right))
				gtr.A = refarr
			} else { // the well-known type-app (Maybe, Either, List, etcpp)
			}
		} else if leftappl := tAppl.Left.A; leftappl != nil {
			if leftappl.Left.A != nil && leftappl.Left.A.Left.Q != nil && leftappl.Left.A.Left.Q.QName == "Prim.Function" {
				gtr.F = &irGoTypeRefFunc{}
				gtr.F.Args = irGoNamedTypeRefs{&irGoNamedTypeRef{}}
				gtr.F.Args[0].Ref.setFrom(me.toIrGoTypeRef(tdict, leftappl.Left.A.Right))
				gtr.F.Rets = irGoNamedTypeRefs{&irGoNamedTypeRef{}}
				gtr.F.Rets[0].Ref.setFrom(me.toIrGoTypeRef(tdict, leftappl.Right))
			}
		}
	}
	return &gtr
}
