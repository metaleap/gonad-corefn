package main

import (
	"fmt"
	"strings"

	"github.com/metaleap/go-util/slice"
)

type irANamedTypeRefs []*irANamedTypeRef

func (me irANamedTypeRefs) Len() int { return len(me) }
func (me irANamedTypeRefs) Less(i, j int) bool {
	if me[i].sortIndex != me[j].sortIndex {
		return me[i].sortIndex < me[j].sortIndex
	}
	return strings.ToLower(me[i].NameGo) < strings.ToLower(me[j].NameGo)
}
func (me irANamedTypeRefs) Swap(i, j int) { me[i], me[j] = me[j], me[i] }

func (me irANamedTypeRefs) byPsName(psname string) *irANamedTypeRef {
	for _, gntr := range me {
		if gntr.NamePs == psname {
			return gntr
		}
	}
	return nil
}

func (me irANamedTypeRefs) equiv(cmp irANamedTypeRefs) bool {
	if l := len(me); l != len(cmp) {
		return false
	} else {
		for i := 0; i < l; i++ {
			if !me[i].equiv(cmp[i]) {
				return false
			}
		}
	}
	return true
}

type irANamedTypeRef struct {
	NamePs string `json:",omitempty"`
	NameGo string `json:",omitempty"`

	RefAlias     string               `json:",omitempty"`
	RefUnknown   int                  `json:",omitempty"`
	RefInterface *irATypeRefInterface `json:",omitempty"`
	RefFunc      *irATypeRefFunc      `json:",omitempty"`
	RefStruct    *irATypeRefStruct    `json:",omitempty"`
	RefArray     *irATypeRefArray     `json:",omitempty"`
	RefPtr       *irATypeRefPtr       `json:",omitempty"`

	Export bool `json:",omitempty"`

	sortIndex int
}

func (me *irANamedTypeRef) clearTypeInfo() {
	me.RefAlias, me.RefUnknown, me.RefInterface, me.RefFunc, me.RefStruct, me.RefArray, me.RefPtr = "", 0, nil, nil, nil, nil, nil
}

func (me *irANamedTypeRef) copyFrom(from *irANamedTypeRef, names bool, trefs bool, export bool) {
	if names {
		me.NameGo, me.NamePs = from.NameGo, from.NamePs
	}
	if trefs {
		me.RefAlias, me.RefUnknown, me.RefInterface, me.RefFunc, me.RefStruct, me.RefArray, me.RefPtr = from.RefAlias, from.RefUnknown, from.RefInterface, from.RefFunc, from.RefStruct, from.RefArray, from.RefPtr
	}
	if export {
		me.Export = from.Export
	}
}

func (me *irANamedTypeRef) copyTypeInfoFrom(from *irANamedTypeRef) {
	me.copyFrom(from, false, true, false)
}

func (me *irANamedTypeRef) nameless() (copy *irANamedTypeRef) {
	copy = &irANamedTypeRef{}
	copy.copyTypeInfoFrom(me)
	return
}

func (me *irANamedTypeRef) equiv(cmp *irANamedTypeRef) bool {
	return (me == nil && cmp == nil) || (me != nil && cmp != nil && me.RefAlias == cmp.RefAlias && me.RefUnknown == cmp.RefUnknown && me.RefInterface.equiv(cmp.RefInterface) && me.RefFunc.equiv(cmp.RefFunc) && me.RefStruct.equiv(cmp.RefStruct) && me.RefArray.equiv(cmp.RefArray) && me.RefPtr.equiv(cmp.RefPtr))
}

func (me *irANamedTypeRef) hasName() bool {
	return me.NamePs != ""
}

func (me *irANamedTypeRef) hasTypeInfoBeyondEmptyIface() (welltyped bool) {
	if welltyped = me.hasTypeInfo(); welltyped && me.RefInterface != nil {
		welltyped = len(me.RefInterface.Embeds) > 0 || len(me.RefInterface.Methods) > 0
	}
	return
}

func (me *irANamedTypeRef) hasTypeInfo() bool {
	return me != nil && me.RefAlias != "" || me.RefArray != nil || me.RefFunc != nil || me.RefInterface != nil || me.RefPtr != nil || me.RefStruct != nil || me.RefUnknown != 0
}

func (me *irANamedTypeRef) setBothNamesFromPsName(psname string) {
	me.NamePs = psname
	me.NameGo = sanitizeSymbolForGo(psname, me.Export)
}

func (me *irANamedTypeRef) setRefFrom(tref interface{}) {
	switch tr := tref.(type) {
	case *irANamedTypeRef:
		me.RefAlias = tr.RefAlias
		me.RefArray = tr.RefArray
		me.RefFunc = tr.RefFunc
		me.RefInterface = tr.RefInterface
		me.RefPtr = tr.RefPtr
		me.RefStruct = tr.RefStruct
		me.RefUnknown = tr.RefUnknown
	case *irATypeRefInterface:
		me.RefInterface = tr
	case *irATypeRefFunc:
		me.RefFunc = tr
	case *irATypeRefStruct:
		me.RefStruct = tr
	case *irATypeRefArray:
		me.RefArray = tr
	case *irATypeRefPtr:
		me.RefPtr = tr
	case int:
		me.RefUnknown = tr
	case string:
		me.RefAlias = tr
	case nil:
	default:
		panicWithType("setRefFrom", tref, "tref")
	}
}

func (me *irANamedTypeRef) turnRefIntoRefPtr() {
	refptr := &irATypeRefPtr{Of: &irANamedTypeRef{}}
	refptr.Of.copyTypeInfoFrom(me)
	me.RefAlias, me.RefArray, me.RefFunc, me.RefInterface, me.RefPtr, me.RefStruct, me.RefUnknown = "", nil, nil, nil, refptr, nil, 0
}

type irATypeRefArray struct {
	Of *irANamedTypeRef
}

func (me *irATypeRefArray) equiv(cmp *irATypeRefArray) bool {
	return (me == nil && cmp == nil) || (me != nil && cmp != nil && me.Of.equiv(cmp.Of))
}

type irATypeRefPtr struct {
	Of *irANamedTypeRef
}

func (me *irATypeRefPtr) equiv(cmp *irATypeRefPtr) bool {
	return (me == nil && cmp == nil) || (me != nil && cmp != nil && me.Of.equiv(cmp.Of))
}

type irATypeRefInterface struct {
	Embeds  []string         `json:",omitempty"`
	Methods irANamedTypeRefs `json:",omitempty"`

	isTypeVar        bool
	xtc              *irMTypeClass
	xtd              *irMTypeDataDef
	inheritedMethods irANamedTypeRefs
}

func (me *irATypeRefInterface) equiv(cmp *irATypeRefInterface) bool {
	return (me == nil && cmp == nil) || (me != nil && cmp != nil && me.isTypeVar == cmp.isTypeVar && uslice.StrEq(me.Embeds, cmp.Embeds) && me.Methods.equiv(cmp.Methods))
}

type irATypeRefFunc struct {
	Args irANamedTypeRefs `json:",omitempty"`
	Rets irANamedTypeRefs `json:",omitempty"`

	impl *irABlock
}

func (me *irATypeRefFunc) copyArgTypesOnlyFrom(namesIfMeNil bool, from *irATypeRefFunc) {
	copyargs := func(meargs irANamedTypeRefs, fromargs irANamedTypeRefs) irANamedTypeRefs {
		if numargsme := len(meargs); numargsme == 0 {
			for _, arg := range fromargs {
				mearg := &irANamedTypeRef{}
				mearg.copyFrom(arg, namesIfMeNil, true, false)
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
	if from != nil {
		me.Args = copyargs(me.Args, from.Args)
		me.Rets = copyargs(me.Rets, from.Rets)
	}
}

func (me *irATypeRefFunc) equiv(cmp *irATypeRefFunc) bool {
	return (me == nil && cmp == nil) || (me != nil && cmp != nil && me.Args.equiv(cmp.Args) && me.Rets.equiv(cmp.Rets))
}

func (me *irATypeRefFunc) forEachArgAndRet(on func(*irANamedTypeRef)) {
	for _, a := range me.Args {
		on(a)
	}
	for _, r := range me.Rets {
		on(r)
	}
}

func (me *irATypeRefFunc) haveAllArgsTypeInfo() bool {
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

func (me *irATypeRefFunc) haveAnyArgsTypeInfo() bool {
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

func (me *irATypeRefFunc) toSig(forceretarg bool) (rf *irATypeRefFunc) {
	rf = &irATypeRefFunc{}
	for _, arg := range me.Args {
		rf.Args = append(rf.Args, arg.nameless())
	}
	if len(me.Rets) == 0 && forceretarg {
		rf.Rets = append(rf.Rets, &irANamedTypeRef{})
	} else {
		for _, ret := range me.Rets {
			rf.Rets = append(rf.Rets, ret.nameless())
		}
	}
	return
}

type irATypeRefStruct struct {
	Embeds    []string         `json:",omitempty"`
	Fields    irANamedTypeRefs `json:",omitempty"`
	PassByPtr bool             `json:",omitempty"`
	Methods   irANamedTypeRefs `json:",omitempty"`
}

func (me *irATypeRefStruct) equiv(cmp *irATypeRefStruct) bool {
	return (me == nil && cmp == nil) || (me != nil && cmp != nil && uslice.StrEq(me.Embeds, cmp.Embeds) && me.Fields.equiv(cmp.Fields))
}

func (me *irATypeRefStruct) memberByPsName(nameps string) (mem *irANamedTypeRef) {
	if mem = me.Fields.byPsName(nameps); mem == nil {
		mem = me.Methods.byPsName(nameps)
	}
	return
}

func (me *irMeta) goTypeDefByGoName(goname string) *irANamedTypeRef {
	for _, gtd := range me.GoTypeDefs {
		if gtd.NameGo == goname {
			return gtd
		}
	}
	return nil
}

func (me *irMeta) goTypeDefByPsName(psname string) *irANamedTypeRef {
	var gtdi *irANamedTypeRef
	for _, gtd := range me.GoTypeDefs {
		if gtd.NamePs == psname {
			if gtd.RefInterface != nil {
				gtdi = gtd
			} else {
				return gtd
			}
		}
	}
	return gtdi
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
				tname = "interface{/*Prim." + tname + "*/}"
				println(me.mod.srcFilePath + "\t" + tref + "\t" + tname)
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
		if gtd := mod.irMeta.goTypeDefByPsName(tname); gtd != nil {
			tname = gtd.NameGo
		}
	}
	return
}
