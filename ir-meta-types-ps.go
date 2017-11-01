package main

import (
	"github.com/metaleap/go-util/dev/ps"
	"github.com/metaleap/go-util/fs"
)

type irPsNamedTypeRef struct {
	Name string       `json:"ntn,omitempty"`
	Ref  *irPsTypeRef `json:"ntr,omitempty"`
}

type irPsTypeClass struct {
	Name           string                 `json:"tcn,omitempty"`
	Args           []string               `json:"tca,omitempty"`
	Members        []*irPsTypeClassMember `json:"tcm,omitempty"`
	CoveringSets   [][]int                `json:"tccs,omitempty"`
	DeterminedArgs []int                  `json:"tcda,omitempty"`
	Superclasses   irMConstraints         `json:"tcsc,omitempty"`
	Dependencies   []irPsTypeClassDep     `json:"tcd,omitempty"`
}

func (me *irPsTypeClass) memberBy(name string) *irPsTypeClassMember {
	for _, m := range me.Members {
		if m.Name == name {
			return m
		}
	}
	return nil
}

type irPsTypeClassDep struct {
	Determiners []int `json:"tcdDeterminers,omitempty"`
	Determined  []int `json:"tcdDetermined,omitempty"`
}

type irMConstraints []*irMConstraint

func (me irMConstraints) equiv(cmp irMConstraints) bool {
	if l := len(me); l == len(cmp) {
		for i := 0; i < l; i++ {
			if !me[i].equiv(cmp[i]) {
				return false
			}
		}
		return true
	}
	return false
}

type irMConstraint struct {
	Class string       `json:"cc,omitempty"`
	Args  irPsTypeRefs `json:"ca,omitempty"`
	Data  interface{}  `json:"cd,omitempty"`
}

func (me *irMConstraint) equiv(cmp *irMConstraint) bool {
	return (me == nil && cmp == nil) || (me != nil && cmp != nil && me.Class == cmp.Class && me.Data == cmp.Data && me.Args.equiv(cmp.Args))
}

type irPsTypeClassInst struct {
	Name         string                  `json:"tcin,omitempty"`
	ClassName    string                  `json:"tcicn,omitempty"`
	Types        irPsTypeRefs            `json:"tcit,omitempty"`
	Chain        []string                `json:"tcic,omitempty"`
	Index        int                     `json:"tcii,omitempty"`
	Value        string                  `json:"tciv,omitempty"`
	Path         []irPsTypeClassInstPath `json:"tcip,omitempty"`
	Dependencies irMConstraints          `json:"tcid,omitempty"`
}

type irPsTypeClassInstPath struct {
	Cls string `json:"tciPc,omitempty"`
	Idx int    `json:"tciPi,omitempty"`
}

type irPsTypeClassMember struct {
	irPsNamedTypeRef

	parent *irPsTypeClass
}

type irPsTypeDataDef struct {
	Name  string              `json:"tdn,omitempty"`
	Ctors []*irPsTypeDataCtor `json:"tdc,omitempty"`
	Args  []irPsTypeDataArg   `json:"tda,omitempty"`
}

type irPsTypeDataArg struct {
	Name string
	Kind udevps.CoreTagKind
}

type irPsTypeDataCtor struct {
	Name         string                 `json:"tdcn,omitempty"`
	Args         []*irPsTypeDataCtorArg `json:"tdca,omitempty"`
	DataTypeName string                 `json:"tdct,omitempty"`
	Ctor         *irPsTypeRef           `json:"tdcc,omitempty"`
	IsNewType    bool                   `json:"tdcnt,omitempty"`

	ŧ *irGoNamedTypeRef
}

type irPsTypeDataCtorArg struct {
	Name string       `json:"tdcan,omitempty"`
	Type *irPsTypeRef `json:"tdcat,omitempty"`
}

type irPsTypeRefs []*irPsTypeRef

func (me irPsTypeRefs) equiv(cmp irPsTypeRefs) bool {
	if l := len(me); l == len(cmp) {
		for i := 0; i < l; i++ {
			if !me[i].equiv(cmp[i]) {
				return false
			}
		}
		return true
	}
	return false
}

type irPsTypeRef struct {
	A   *irPsTypeRefAppl
	C   *irPsTypeRefConstrained
	E   *irPsTypeRefEmpty
	F   *irPsTypeRefForall
	Q   *irPsTypeRefConstruct
	R   *irPsTypeRefRow
	S   *irPsTypeRefSkolem
	TlS *irPsTypeRefTlStr
	V   *irPsTypeRefVar
}

func (me *irPsTypeRef) equiv(cmp *irPsTypeRef) bool {
	return (me == nil && cmp == nil) || (me != nil && cmp != nil && me.A.equiv(cmp.A) && me.C.equiv(cmp.C) && me.E.equiv(cmp.E) && me.F.equiv(cmp.F) && me.Q.equiv(cmp.Q) && me.R.equiv(cmp.R) && me.S.equiv(cmp.S) && me.TlS.equiv(cmp.TlS) && me.V.equiv(cmp.V))
}

type irPsTypeRefAppl struct {
	Left  *irPsTypeRef `json:"t1,omitempty"`
	Right *irPsTypeRef `json:"t2,omitempty"`
}

func (me *irPsTypeRefAppl) equiv(cmp *irPsTypeRefAppl) bool {
	return (me == nil && cmp == nil) || (me != nil && cmp != nil && me.Left.equiv(cmp.Left) && me.Right.equiv(cmp.Right))
}

type irPsTypeRefConstrained struct {
	Ref    *irPsTypeRef   `json:"trcr,omitempty"`
	Constr irMConstraints `json:"trcc,omitempty"`
}

func (me *irPsTypeRefConstrained) equiv(cmp *irPsTypeRefConstrained) bool {
	return (me == nil && cmp == nil) || (me != nil && cmp != nil && me.Ref.equiv(cmp.Ref) && me.Constr.equiv(cmp.Constr))
}

func (me *irPsTypeRefConstrained) flatten() {
	for next := me.Ref.C; next != nil; next = me.Ref.C {
		me.Constr = append(me.Constr, next.Constr[0])
		me.Ref = next.Ref
	}
}

func (me *irPsTypeRefConstrained) final() (lastinchain *irPsTypeRefConstrained) {
	lastinchain = me
	for next := lastinchain.Ref.C; next != nil; next = lastinchain.Ref.C {
		lastinchain = next
	}
	return
}

type irPsTypeRefConstruct struct {
	QName string
}

func (me *irPsTypeRefConstruct) equiv(cmp *irPsTypeRefConstruct) bool {
	return (me == nil && cmp == nil) || (me != nil && cmp != nil && me.QName == cmp.QName)
}

type irPsTypeRefEmpty struct {
}

func (me *irPsTypeRefEmpty) equiv(cmp *irPsTypeRefEmpty) bool {
	return (me == nil) == (cmp == nil)
}

type irPsTypeRefForall struct {
	Name        string       `json:"en,omitempty"`
	Ref         *irPsTypeRef `json:"er,omitempty"`
	SkolemScope *int         `json:"es,omitempty"`
}

func (me *irPsTypeRefForall) equiv(cmp *irPsTypeRefForall) bool {
	return (me == nil && cmp == nil) || (me != nil && cmp != nil && me.Name == cmp.Name && me.Ref.equiv(cmp.Ref) && ((me.SkolemScope == nil && cmp.SkolemScope == nil) || (me.SkolemScope != nil && cmp.SkolemScope != nil && *me.SkolemScope == *cmp.SkolemScope)))
}

type irPsTypeRefRow struct {
	Label string       `json:"rl,omitempty"`
	Left  *irPsTypeRef `json:"r1,omitempty"`
	Right *irPsTypeRef `json:"r2,omitempty"`
}

func (me *irPsTypeRefRow) equiv(cmp *irPsTypeRefRow) bool {
	return (me == nil && cmp == nil) || (me != nil && cmp != nil && me.Label == cmp.Label && me.Left.equiv(cmp.Left) && me.Right.equiv(cmp.Right))
}

type irPsTypeRefSkolem struct {
	Name  string `json:"sn,omitempty"`
	Value int    `json:"sv,omitempty"`
	Scope int    `json:"ss,omitempty"`
}

func (me *irPsTypeRefSkolem) equiv(cmp *irPsTypeRefSkolem) bool {
	return (me == nil && cmp == nil) || (me != nil && cmp != nil && me.Name == cmp.Name && me.Value == cmp.Value && me.Scope == cmp.Scope)
}

type irPsTypeRefTlStr struct {
	Text string
}

func (me *irPsTypeRefTlStr) equiv(cmp *irPsTypeRefTlStr) bool {
	return (me == nil && cmp == nil) || (me != nil && cmp != nil && me.Text == cmp.Text)
}

type irPsTypeRefVar struct {
	Name string
}

func (me *irPsTypeRefVar) equiv(cmp *irPsTypeRefVar) bool {
	return (me == nil && cmp == nil) || (me != nil && cmp != nil && me.Name == cmp.Name)
}

func (me *irMeta) tc(name string) *irPsTypeClass {
	for _, tc := range me.EnvTypeClasses {
		if tc.Name == name {
			return tc
		}
	}
	return nil
}

func (me *irMeta) tcInst(name string) *irPsTypeClassInst {
	for _, tci := range me.EnvTypeClassInsts {
		if tci.Name == name {
			return tci
		}
	}
	return nil
}

func (me *irMeta) tcMember(name string) *irPsTypeClassMember {
	for _, tc := range me.EnvTypeClasses {
		for _, tcm := range tc.Members {
			if tcm.Name == name {
				return tcm
			}
		}
	}
	return nil
}

func (me *irMeta) newConstr(from *udevps.CoreConstr) (c *irMConstraint) {
	c = &irMConstraint{Class: from.Cls, Data: from.Data}
	for _, fromarg := range from.Args {
		c.Args = append(c.Args, me.newTRefFrom(fromarg))
	}
	return

}

func (me *irMeta) newTRefFrom(t interface{}) *irPsTypeRef {
	if t != nil {
		var tref irPsTypeRef
		switch r := t.(type) {
		case *udevps.CoreTagType:
			if tc := r; tc.IsTypeConstructor() {
				tref.Q = &irPsTypeRefConstruct{QName: tc.Text}
			} else if tc.IsTypeVar() {
				tref.V = &irPsTypeRefVar{Name: tc.Text}
			} else if tc.IsREmpty() {
				tref.E = &irPsTypeRefEmpty{}
			} else if tc.IsRCons() {
				tref.R = &irPsTypeRefRow{
					Label: tc.Text, Left: me.newTRefFrom(tc.Type0), Right: me.newTRefFrom(tc.Type1)}
			} else if tc.IsForAll() {
				forall := &irPsTypeRefForall{Name: tc.Text, Ref: me.newTRefFrom(tc.Type0)}
				if tc.Skolem >= 0 {
					forall.SkolemScope = &tc.Skolem
				}
				tref.F = forall
			} else if tc.IsSkolem() {
				tref.S = &irPsTypeRefSkolem{Name: tc.Text, Value: tc.Num, Scope: tc.Skolem}
			} else if tc.IsTypeApp() {
				tref.A = &irPsTypeRefAppl{Left: me.newTRefFrom(tc.Type0), Right: me.newTRefFrom(tc.Type1)}
			} else if tc.IsConstrainedType() {
				tref.C = &irPsTypeRefConstrained{Constr: irMConstraints{me.newConstr(tc.Constr)}, Ref: me.newTRefFrom(tc.Type0)}
			} else if tc.IsTypeLevelString() {
				tref.TlS = &irPsTypeRefTlStr{Text: tc.Text}
			} else {
				panic(notImplErr("tagged-type", tc.Tag, me.mod.srcFilePath))
			}
		case *irPsTypeRefAppl:
			tref.A = r
		case *irPsTypeRefConstrained:
			tref.C = r
		case *irPsTypeRefConstruct:
			tref.Q = r
		case *irPsTypeRefEmpty:
			tref.E = r
		case *irPsTypeRefForall:
			tref.F = r
		case *irPsTypeRefRow:
			tref.R = r
		case *irPsTypeRefSkolem:
			tref.S = r
		case *irPsTypeRefVar:
			tref.V = r
		case *irPsTypeRefTlStr:
			tref.TlS = r
		default:
			panic(notImplErr("`ref` for", "newTRefFrom", me.mod.srcFilePath))
		}
		return &tref
	}
	return nil
}

func (me *irMeta) populateEnvFuncsAndVals() {
	for fname, fdef := range me.mod.coreimp.DeclEnv.Functions {
		me.EnvValDecls = append(me.EnvValDecls, &irPsNamedTypeRef{Name: fname, Ref: me.newTRefFrom(fdef.Type)})
	}
}

func (me *irMeta) populateEnvTypeDataDecls() {
	for tdefname, tdef := range me.mod.coreimp.DeclEnv.TypeDefs {
		if tdef.Decl.TypeSynonym {
			//	type-aliases handled separately in populateEnvTypeSyns already, nothing to do here
		} else if tdef.Decl.ExternData {
			if ffigofilepath := me.mod.srcFilePath[:len(me.mod.srcFilePath)-len(".purs")] + ".go"; ufs.FileExists(ffigofilepath) {
				panic(me.mod.srcFilePath + ": time to handle FFI " + ffigofilepath)
			} else {
				//	special case for official purescript core libs: alias to applicable struct from gonad's default ffi packages
				ta := &irPsNamedTypeRef{Name: tdefname, Ref: me.newTRefFrom(&irPsTypeRefConstruct{QName: prefixDefaultFfiPkgNs + strReplDot2ˈ.Replace(me.mod.qName) + "." + tdefname})}
				me.EnvTypeSyns = append(me.EnvTypeSyns, ta)
			}
		} else {
			dt := &irPsTypeDataDef{Name: tdefname}
			for _, dtarg := range tdef.Decl.DataType.Args {
				dt.Args = append(dt.Args, irPsTypeDataArg{Name: dtarg.Name, Kind: *dtarg.Kind})
			}
			for _, dtctor := range tdef.Decl.DataType.Ctors {
				dcdef := me.mod.coreimp.DeclEnv.DataCtors[dtctor.Name]
				if len(dcdef.Args) != len(dtctor.Types) {
					panic(notImplErr("ctor-args count mismatch", tdefname+"|"+dtctor.Name, me.mod.impFilePath))
				}
				dtc := &irPsTypeDataCtor{Name: dtctor.Name, DataTypeName: dcdef.Type, IsNewType: dcdef.IsDeclNewtype(), Ctor: me.newTRefFrom(dcdef.Ctor)}
				for i, dtcargtype := range dtctor.Types {
					dtc.Args = append(dtc.Args, &irPsTypeDataCtorArg{Name: dcdef.Args[i], Type: me.newTRefFrom(dtcargtype)})
				}
				dt.Ctors = append(dt.Ctors, dtc)
			}
			me.EnvTypeDataDecls = append(me.EnvTypeDataDecls, dt)
		}
	}
}

func (me *irMeta) populateEnvTypeSyns() {
	for tsname, tsdef := range me.mod.coreimp.DeclEnv.TypeSyns {
		ts := &irPsNamedTypeRef{Name: tsname}
		ts.Ref = me.newTRefFrom(tsdef.Type)
		me.EnvTypeSyns = append(me.EnvTypeSyns, ts)
	}
}

func (me *irMeta) populateEnvTypeClasses() {
	for tcname, tcdef := range me.mod.coreimp.DeclEnv.Classes {
		tc := &irPsTypeClass{Name: tcname}
		for _, tcarg := range tcdef.Args {
			tc.Args = append(tc.Args, tcarg.Name)
		}
		for _, tcmdef := range tcdef.Members {
			tref := me.newTRefFrom(tcmdef.Type)
			tc.Members = append(tc.Members, &irPsTypeClassMember{parent: tc, irPsNamedTypeRef: irPsNamedTypeRef{Name: tcmdef.Ident, Ref: tref}})
		}
		for _, tcsc := range tcdef.Superclasses {
			tc.Superclasses = append(tc.Superclasses, me.newConstr(tcsc))
		}
		tc.CoveringSets = tcdef.CoveringSets
		tc.DeterminedArgs = tcdef.DeterminedArgs
		for _, fdep := range tcdef.Dependencies {
			tc.Dependencies = append(tc.Dependencies, irPsTypeClassDep{Determiners: fdep.Determiners, Determined: fdep.Determined})
		}
		me.EnvTypeClasses = append(me.EnvTypeClasses, tc)
	}
	for _, m := range me.mod.coreimp.DeclEnv.ClassDicts {
		for tciclass, tcinsts := range m {
			for tciname, tcidef := range tcinsts {
				tci := &irPsTypeClassInst{Name: tciname, ClassName: tciclass, Chain: tcidef.Chain, Index: tcidef.Index, Value: tcidef.Value}
				for _, tcid := range tcidef.Dependencies {
					tci.Dependencies = append(tci.Dependencies, me.newConstr(tcid))
				}
				for _, tcip := range tcidef.Path {
					tci.Path = append(tci.Path, irPsTypeClassInstPath{Cls: tcip.Class, Idx: tcip.Int})
				}
				for _, tcit := range tcidef.InstanceTypes {
					tci.Types = append(tci.Types, me.newTRefFrom(tcit))
				}
				me.EnvTypeClassInsts = append(me.EnvTypeClassInsts, tci)
			}
		}
	}
}
