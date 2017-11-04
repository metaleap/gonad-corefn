package main

import (
	"fmt"
	"strings"

	"github.com/metaleap/go-util/slice"
)

type irGoNamedTypeRefs []*irGoNamedTypeRef

func (me irGoNamedTypeRefs) byPsName(psname string) *irGoNamedTypeRef {
	for _, gntr := range me {
		if gntr.NamePs == psname {
			return gntr
		}
	}
	return nil
}

func (me irGoNamedTypeRefs) equiv(cmp irGoNamedTypeRefs) bool {
	if l := len(me); l == len(cmp) {
		for i := 0; i < l; i++ {
			if !me[i].Ref.equiv(&cmp[i].Ref) {
				return false
			}
		}
		return true
	}
	return false
}

func (me *irGoNamedTypeRefs) removeAt(i int) {
	self := *me
	*me = append(self[:i], self[i+1:]...)
}

type irGoNamedTypeRef struct {
	NamePs  string            `json:",omitempty"`
	NameGo  string            `json:",omitempty"`
	Export  bool              `json:",omitempty"`
	Ref     irGoTypeRef       `json:",omitempty"`
	Methods irGoNamedTypeRefs `json:",omitempty"`
}

func (me *irGoNamedTypeRef) copyFrom(from *irGoNamedTypeRef, names bool, trefs bool, export bool) {
	if names {
		me.NameGo, me.NamePs = from.NameGo, from.NamePs
	}
	if trefs {
		me.Ref.Q, me.Ref.I, me.Ref.F, me.Ref.S, me.Ref.A, me.Ref.P, me.Ref.E = from.Ref.Q, from.Ref.I, from.Ref.F, from.Ref.S, from.Ref.A, from.Ref.P, from.Ref.E
	}
	if export {
		me.Export = from.Export
	}
}

func (me *irGoNamedTypeRef) copyTypeInfoFrom(from *irGoNamedTypeRef) {
	me.copyFrom(from, false, true, false)
}

func (me *irGoNamedTypeRef) nameless() (copy *irGoNamedTypeRef) {
	copy = &irGoNamedTypeRef{}
	copy.copyTypeInfoFrom(me)
	return
}

func (me *irGoNamedTypeRef) hasName() bool {
	return me.NamePs != ""
}

func (me *irGoNamedTypeRef) hasTypeInfoBeyondEmptyIface() (welltyped bool) {
	if welltyped = me.hasTypeInfo(); welltyped && me.Ref.I != nil {
		welltyped = len(me.Ref.I.Embeds) > 0 || len(me.Methods) > 0
	}
	return
}

func (me *irGoNamedTypeRef) hasTypeInfo() bool {
	return me != nil && (me.Ref.Q != nil || me.Ref.A != nil || me.Ref.F != nil || me.Ref.I != nil || me.Ref.P != nil || me.Ref.S != nil || me.Ref.E != nil)
}

func (me *irGoNamedTypeRef) setBothNamesFromPsName(psname string) {
	me.NamePs = psname
	me.NameGo = sanitizeSymbolForGo(psname, me.Export)
}

func (me *irGoNamedTypeRef) turnRefIntoRefPtr() {
	refptr := &irGoTypeRefPtr{Of: &irGoNamedTypeRef{}}
	refptr.Of.copyTypeInfoFrom(me)
	me.Ref.Q, me.Ref.A, me.Ref.F, me.Ref.I, me.Ref.P, me.Ref.S, me.Ref.E = nil, nil, nil, nil, refptr, nil, nil
}

type irGoTypeRef struct {
	//	"native" Go type kinds
	A *irGoTypeRefArray     `json:",omitempty"`
	E *irGoTypeRefEnum      `json:",omitempty"`
	F *irGoTypeRefFunc      `json:",omitempty"`
	I *irGoTypeRefInterface `json:",omitempty"`
	P *irGoTypeRefPtr       `json:",omitempty"`
	Q *irGoTypeRefSyn       `json:",omitempty"`
	S *irGoTypeRefStruct    `json:",omitempty"`

	origs    irPsTypeRefs
	origCtor *irPsTypeDataCtor
	origData *irPsTypeDataDef
}

func (me *irGoTypeRef) clear(origstoo bool) {
	if me.Q, me.I, me.F, me.S, me.A, me.P, me.E = nil, nil, nil, nil, nil, nil, nil; origstoo {
		me.origs, me.origCtor, me.origData = nil, nil, nil
	}
}

func (me *irGoTypeRef) equiv(cmp *irGoTypeRef) bool {
	return (me == nil && cmp == nil) ||
		(me != nil && cmp != nil && me.Q.equiv(cmp.Q) && me.E.equiv(cmp.E) && me.I.equiv(cmp.I) && me.F.equiv(cmp.F) && me.S.equiv(cmp.S) && me.A.equiv(cmp.A) && me.P.equiv(cmp.P))
}

func (me *irGoTypeRef) allNil() bool {
	return me.A == nil && me.F == nil && me.I == nil && me.P == nil && me.Q == nil && me.S == nil && me.E == nil
}

func (me *irGoTypeRef) setFrom(tref interface{}) {
	me.clear(true)
	switch tr := tref.(type) {
	case *irGoTypeRef:
		me.Q = tr.Q
		me.A = tr.A
		me.F = tr.F
		me.I = tr.I
		me.P = tr.P
		me.S = tr.S
		me.origs, me.origCtor, me.origData = tr.origs, tr.origCtor, tr.origData
	case *irGoTypeRefInterface:
		me.I = tr
	case *irGoTypeRefFunc:
		me.F = tr
	case *irGoTypeRefStruct:
		me.S = tr
	case *irGoTypeRefArray:
		me.A = tr
	case *irGoTypeRefPtr:
		me.P = tr
	case *irGoTypeRefSyn:
		me.Q = tr
	case *irGoTypeRefEnum:
		me.E = tr
	default:
		panicWithType("setFrom", tref, "tref")
	}
}

type irGoTypeRefSyn struct {
	QName string
}

func (me *irGoTypeRefSyn) equiv(cmp *irGoTypeRefSyn) bool {
	return (me == nil && cmp == nil) || (me != nil && cmp != nil && me.QName == cmp.QName)
}

type irGoTypeRefArray struct {
	Of *irGoNamedTypeRef
}

func (me *irGoTypeRefArray) equiv(cmp *irGoTypeRefArray) bool {
	return (me == nil && cmp == nil) || (me != nil && cmp != nil && me.Of.Ref.equiv(&cmp.Of.Ref))
}

type irGoTypeRefEnum struct {
	Names []*irGoNamedTypeRef
}

func (me *irGoTypeRefEnum) equiv(cmp *irGoTypeRefEnum) bool {
	if me != nil && cmp != nil && len(me.Names) == len(cmp.Names) {
		for i, m := range me.Names {
			if m.NameGo != cmp.Names[i].NameGo {
				return false
			}
		}
		return true
	}
	return me == nil && cmp == nil
}

type irGoTypeRefPtr struct {
	Of *irGoNamedTypeRef
}

func (me *irGoTypeRefPtr) equiv(cmp *irGoTypeRefPtr) bool {
	return (me == nil && cmp == nil) || (me != nil && cmp != nil && me.Of.Ref.equiv(&cmp.Of.Ref))
}

type irGoTypeRefInterface struct {
	Embeds []string `json:",omitempty"`

	origClass *irPsTypeClass
}

func (me *irGoTypeRefInterface) equiv(cmp *irGoTypeRefInterface) bool {
	return (me == nil && cmp == nil) || (me != nil && cmp != nil && uslice.StrEq(me.Embeds, cmp.Embeds))
}

type irGoTypeRefFunc struct {
	Args irGoNamedTypeRefs `json:",omitempty"`
	Rets irGoNamedTypeRefs `json:",omitempty"`

	origTcMem *irPsTypeClassMember
	hasthis   bool
	impl      *irABlock
}

func (me *irGoTypeRefFunc) clone() *irGoTypeRefFunc {
	clone := &irGoTypeRefFunc{origTcMem: me.origTcMem, impl: me.impl, hasthis: me.hasthis}
	clone.copyArgTypesOnlyFrom(true, me, nil)
	return clone
}

func (me *irGoTypeRefFunc) copyArgTypesOnlyFrom(namesIfNil bool, eitherfromfunc *irGoTypeRefFunc, orfromtyperef *irGoTypeRef) {
	copyargs := func(meargs irGoNamedTypeRefs, fromargs irGoNamedTypeRefs) irGoNamedTypeRefs {
		if numargsme := len(meargs); numargsme == 0 {
			for _, arg := range fromargs {
				mearg := &irGoNamedTypeRef{}
				mearg.copyFrom(arg, namesIfNil, true, false)
				meargs = append(meargs, mearg)
			}
		} else if numargsfrom := len(fromargs); numargsme != numargsfrom {
			panic(notImplErr("args-num mismatch", fmt.Sprintf("%v vs %v", numargsme, numargsfrom), "copyArgTypesFrom"))
		} else {
			for i := 0; i < numargsme; i++ {
				meargs[i].copyTypeInfoFrom(fromargs[i])
			}
		}
		return meargs
	}
	if eitherfromfunc == nil && orfromtyperef != nil && orfromtyperef.F != nil {
		eitherfromfunc = orfromtyperef.F
	}
	if eitherfromfunc != nil {
		me.Args = copyargs(me.Args, eitherfromfunc.Args)
		me.Rets = copyargs(me.Rets, eitherfromfunc.Rets)
	} else if orfromtyperef != nil {
		me.Args = irGoNamedTypeRefs{}
		me.Rets = irGoNamedTypeRefs{&irGoNamedTypeRef{Ref: *orfromtyperef}}
	}
}

func (me *irGoTypeRefFunc) equiv(cmp *irGoTypeRefFunc) bool {
	return (me == nil && cmp == nil) || (me != nil && cmp != nil && me.Args.equiv(cmp.Args) && me.Rets.equiv(cmp.Rets))
}

func (me *irGoTypeRefFunc) forEachArgAndRet(on func(*irGoNamedTypeRef)) {
	for _, a := range me.Args {
		on(a)
	}
	for _, r := range me.Rets {
		on(r)
	}
}

func (me *irGoTypeRefFunc) haveAllArgsTypeInfo() bool {
	for _, arg := range me.Args {
		if !arg.hasTypeInfo() {
			return false
		}
	}
	for _, ret := range me.Rets {
		if !ret.hasTypeInfo() {
			return false
		}
	}
	return true
}

func (me *irGoTypeRefFunc) haveAnyArgsTypeInfo() bool {
	for _, arg := range me.Args {
		if arg.hasTypeInfo() {
			return true
		}
	}
	for _, ret := range me.Rets {
		if ret.hasTypeInfo() {
			return true
		}
	}
	return false
}

func (me *irGoTypeRefFunc) toSig(forceretarg bool) (rf *irGoTypeRefFunc) {
	rf = &irGoTypeRefFunc{}
	for _, arg := range me.Args {
		rf.Args = append(rf.Args, arg.nameless())
	}
	if forceretarg && len(me.Rets) == 0 {
		rf.Rets = append(rf.Rets, &irGoNamedTypeRef{})
	} else {
		for _, ret := range me.Rets {
			rf.Rets = append(rf.Rets, ret.nameless())
		}
	}
	return
}

type irGoTypeRefStruct struct {
	Embeds    []string          `json:",omitempty"`
	Fields    irGoNamedTypeRefs `json:",omitempty"`
	PassByPtr bool              `json:",omitempty"`

	origInst *irPsTypeClassInst
}

func (me *irGoTypeRefStruct) equiv(cmp *irGoTypeRefStruct) bool {
	return (me == nil && cmp == nil) || (me != nil && cmp != nil && uslice.StrEq(me.Embeds, cmp.Embeds) && me.Fields.equiv(cmp.Fields))
}

func (me *irMeta) goTypeDefByGoName(goname string) *irGoNamedTypeRef {
	for _, gtd := range me.GoTypeDefs {
		if gtd.NameGo == goname {
			return gtd
		}
	}
	return nil
}

func (me *irMeta) goTypeDefByPsName(psname string, isctor bool) *irGoNamedTypeRef {
	isnoctor := !isctor
	for _, gtd := range me.GoTypeDefs {
		if gtd.NamePs == psname {
			if isnoctor || gtd.Ref.S != nil {
				return gtd
			}
		}
	}
	return nil
}

func (me *irAst) resolveGoTypeRefFromQName(tref string) (pname string, tname string) {
	var mod *modPkg
	wasprim := false
	i := strings.LastIndex(tref, ".")
	if tname = tref[i+1:]; i > 0 {
		pname = tref[:i]
		if pname == me.mod.qName {
			pname = ""
			mod = me.mod
		} else if wasprim = (pname == "Prim"); wasprim {
			pname = ""
			switch tname {
			case "Char":
				tname = "rune"
			case "String":
				tname = "string"
			case "Boolean":
				tname = "bool"
			case "Number":
				tname = "float64"
			case "Int":
				tname = "int"
			default:
				panic(notImplErr("prim type", tname, me.mod.srcFilePath))
			}
		} else {
			qn, foundimport, isffi := pname, false, strings.HasPrefix(pname, prefixDefaultFfiPkgNs)
			if !isffi {
				if mod = findModuleByQName(qn); mod == nil {
					if mod = findModuleByPName(qn); mod == nil {
						panic(notImplErr("module qname", qn, me.mod.srcFilePath))
					}
				}
				pname = mod.pName
			}
			for _, imp := range me.irM.Imports {
				if imp.PsModQName == qn {
					foundimport = true
					break
				}
			}
			if !foundimport {
				var imp *irMPkgRef
				if isffi {
					imp = &irMPkgRef{ImpPath: prefixDefaultFfiPkgImpPath + strReplDot2Slash.Replace(qn)}
				} else {
					imp = mod.newModImp()
				}
				me.irM.imports, me.irM.Imports = append(me.irM.imports, mod), append(me.irM.Imports, imp)
			}
		}
	} else {
		mod = me.mod
	}
	if (!wasprim) && mod != nil {
		if gtd := mod.irMeta.goTypeDefByPsName(tname, false); gtd != nil {
			tname = gtd.NameGo
		}
	}
	return
}
