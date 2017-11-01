package main

import (
	"github.com/metaleap/go-util/dev/ps"
	"github.com/metaleap/go-util/fs"
)

type irMNamedTypeRef struct {
	Name string     `json:"ntn,omitempty"`
	Ref  irMTypeRef `json:"ntr,omitempty"`
}

type irMTypeClass struct {
	Name           string                `json:"tcn,omitempty"`
	Args           []string              `json:"tca,omitempty"`
	Members        []*irMTypeClassMember `json:"tcm,omitempty"`
	CoveringSets   [][]int               `json:"tccs,omitempty"`
	DeterminedArgs []int                 `json:"tcda,omitempty"`
	Superclasses   []*irMConstraint      `json:"tcsc,omitempty"`
	Dependencies   []irMTypeClassDep     `json:"tcd,omitempty"`
}

func (me *irMTypeClass) memberBy(name string) *irMTypeClassMember {
	for _, m := range me.Members {
		if m.Name == name {
			return m
		}
	}
	return nil
}

type irMTypeClassDep struct {
	Determiners []int `json:"tcdDeterminers,omitempty"`
	Determined  []int `json:"tcdDetermined,omitempty"`
}

type irMConstraint struct {
	Class string      `json:"cc,omitempty"`
	Args  irMTypeRefs `json:"ca,omitempty"`
	Data  interface{} `json:"cd,omitempty"`
}

type irMTypeClassInst struct {
	Name         string                 `json:"tcin,omitempty"`
	ClassName    string                 `json:"tcicn,omitempty"`
	Types        irMTypeRefs            `json:"tcit,omitempty"`
	Chain        []string               `json:"tcic,omitempty"`
	Index        int                    `json:"tcii,omitempty"`
	Value        string                 `json:"tciv,omitempty"`
	Path         []irMTypeClassInstPath `json:"tcip,omitempty"`
	Dependencies []*irMConstraint       `json:"tcid,omitempty"`
}

type irMTypeClassInstPath struct {
	Cls string `json:"tciPc,omitempty"`
	Idx int    `json:"tciPi,omitempty"`
}

type irMTypeClassMember struct {
	irMNamedTypeRef

	parent *irMTypeClass
}

type irMTypeDataDef struct {
	Name  string             `json:"tdn,omitempty"`
	Ctors []*irMTypeDataCtor `json:"tdc,omitempty"`
	Args  []irMTypeDataArg   `json:"tda,omitempty"`
}

type irMTypeDataArg struct {
	Name string
	Kind udevps.CoreTagKind
}

type irMTypeDataCtor struct {
	Name         string                `json:"tdcn,omitempty"`
	Args         []*irMTypeDataCtorArg `json:"tdca,omitempty"`
	DataTypeName string                `json:"tdct,omitempty"`
	Ctor         irMTypeRef            `json:"tdcc,omitempty"`
	IsNewType    bool                  `json:"tdcnt,omitempty"`

	ŧ *irANamedTypeRef
}

type irMTypeDataCtorArg struct {
	Name string     `json:"tdcan,omitempty"`
	Type irMTypeRef `json:"tdcat,omitempty"`
}

type irMTypeRefs []irMTypeRef

type irMTypeRef interface {
}

type irMTypeRefAppl struct {
	Left  irMTypeRef `json:"t1,omitempty"`
	Right irMTypeRef `json:"t2,omitempty"`
}

type irMTypeRefConstrained struct {
	Ref    irMTypeRef       `json:"trcr,omitempty"`
	Constr []*irMConstraint `json:"trcc,omitempty"`
}

func (me *irMTypeRefConstrained) flatten() {
	for next, _ := me.Ref.(*irMTypeRefConstrained); next != nil; next, _ = me.Ref.(*irMTypeRefConstrained) {
		me.Constr = append(me.Constr, next.Constr[0])
		me.Ref = next.Ref
	}
}

func (me *irMTypeRefConstrained) final() (lastinchain *irMTypeRefConstrained) {
	lastinchain = me
	for next, _ := lastinchain.Ref.(*irMTypeRefConstrained); next != nil; next, _ = lastinchain.Ref.(*irMTypeRefConstrained) {
		lastinchain = next
	}
	return
}

type irMTypeRefConstruct struct {
	QName string
}

type irMTypeRefEmpty struct {
}

type irMTypeRefForall struct {
	Name        string     `json:"en,omitempty"`
	Ref         irMTypeRef `json:"er,omitempty"`
	SkolemScope *int       `json:"es,omitempty"`
}

type irMTypeRefRow struct {
	Label string     `json:"rl,omitempty"`
	Left  irMTypeRef `json:"r1,omitempty"`
	Right irMTypeRef `json:"r2,omitempty"`
}

type irMTypeRefSkolem struct {
	Name  string `json:"sn,omitempty"`
	Value int    `json:"sv,omitempty"`
	Scope int    `json:"ss,omitempty"`
}

type irMTypeRefTlStr struct {
	Text string
}

type irMTypeRefVar struct {
	Name string
}

func (me *irMeta) tc(name string) *irMTypeClass {
	for _, tc := range me.EnvTypeClasses {
		if tc.Name == name {
			return tc
		}
	}
	return nil
}

func (me *irMeta) tcInst(name string) *irMTypeClassInst {
	for _, tci := range me.EnvTypeClassInsts {
		if tci.Name == name {
			return tci
		}
	}
	return nil
}

func (me *irMeta) tcMember(name string) *irMTypeClassMember {
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
		c.Args = append(c.Args, me.newTypeRefFromEnvTag(fromarg))
	}
	return

}

func (me *irMeta) newTypeRefFromEnvTag(tc *udevps.CoreTagType) (tref irMTypeRef) {
	if tc != nil {
		if tc.IsTypeConstructor() {
			tref = &irMTypeRefConstruct{QName: tc.Text}
		} else if tc.IsTypeVar() {
			tref = &irMTypeRefVar{Name: tc.Text}
		} else if tc.IsREmpty() {
			tref = &irMTypeRefEmpty{}
		} else if tc.IsRCons() {
			tref = &irMTypeRefRow{
				Label: tc.Text, Left: me.newTypeRefFromEnvTag(tc.Type0), Right: me.newTypeRefFromEnvTag(tc.Type1)}
		} else if tc.IsForAll() {
			forall := &irMTypeRefForall{Name: tc.Text, Ref: me.newTypeRefFromEnvTag(tc.Type0)}
			if tc.Skolem >= 0 {
				forall.SkolemScope = &tc.Skolem
			}
			tref = forall
		} else if tc.IsSkolem() {
			tref = &irMTypeRefSkolem{Name: tc.Text, Value: tc.Num, Scope: tc.Skolem}
		} else if tc.IsTypeApp() {
			tref = &irMTypeRefAppl{Left: me.newTypeRefFromEnvTag(tc.Type0), Right: me.newTypeRefFromEnvTag(tc.Type1)}
		} else if tc.IsConstrainedType() {
			tref = &irMTypeRefConstrained{Constr: []*irMConstraint{me.newConstr(tc.Constr)}, Ref: me.newTypeRefFromEnvTag(tc.Type0)}
		} else if tc.IsTypeLevelString() {
			tref = &irMTypeRefTlStr{Text: tc.Text}
		} else {
			panic(notImplErr("tagged-type", tc.Tag, me.mod.srcFilePath))
		}
	}
	return
}

func (me *irMeta) populateEnvFuncsAndVals() {
	for fname, fdef := range me.mod.coreimp.DeclEnv.Functions {
		me.EnvValDecls = append(me.EnvValDecls, &irMNamedTypeRef{Name: fname, Ref: me.newTypeRefFromEnvTag(fdef.Type)})
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
				ta := &irMNamedTypeRef{Name: tdefname, Ref: &irMTypeRefConstruct{QName: prefixDefaultFfiPkgNs + strReplDot2ˈ.Replace(me.mod.qName) + "." + tdefname}}
				me.EnvTypeSyns = append(me.EnvTypeSyns, ta)
			}
		} else {
			dt := &irMTypeDataDef{Name: tdefname}
			for _, dtarg := range tdef.Decl.DataType.Args {
				dt.Args = append(dt.Args, irMTypeDataArg{Name: dtarg.Name, Kind: *dtarg.Kind})
			}
			for _, dtctor := range tdef.Decl.DataType.Ctors {
				dcdef := me.mod.coreimp.DeclEnv.DataCtors[dtctor.Name]
				if len(dcdef.Args) != len(dtctor.Types) {
					panic(notImplErr("ctor-args count mismatch", tdefname+"|"+dtctor.Name, me.mod.impFilePath))
				}
				dtc := &irMTypeDataCtor{Name: dtctor.Name, DataTypeName: dcdef.Type, IsNewType: dcdef.IsDeclNewtype(), Ctor: me.newTypeRefFromEnvTag(dcdef.Ctor)}
				for i, dtcargtype := range dtctor.Types {
					dtc.Args = append(dtc.Args, &irMTypeDataCtorArg{Name: dcdef.Args[i], Type: me.newTypeRefFromEnvTag(dtcargtype)})
				}
				dt.Ctors = append(dt.Ctors, dtc)
			}
			me.EnvTypeDataDecls = append(me.EnvTypeDataDecls, dt)
		}
	}
}

func (me *irMeta) populateEnvTypeSyns() {
	for tsname, tsdef := range me.mod.coreimp.DeclEnv.TypeSyns {
		ts := &irMNamedTypeRef{Name: tsname}
		ts.Ref = me.newTypeRefFromEnvTag(tsdef.Type)
		me.EnvTypeSyns = append(me.EnvTypeSyns, ts)
	}
}

func (me *irMeta) populateEnvTypeClasses() {
	for tcname, tcdef := range me.mod.coreimp.DeclEnv.Classes {
		tc := &irMTypeClass{Name: tcname}
		for _, tcarg := range tcdef.Args {
			tc.Args = append(tc.Args, tcarg.Name)
		}
		for _, tcmdef := range tcdef.Members {
			tref := me.newTypeRefFromEnvTag(tcmdef.Type)
			tc.Members = append(tc.Members, &irMTypeClassMember{parent: tc, irMNamedTypeRef: irMNamedTypeRef{Name: tcmdef.Ident, Ref: tref}})
		}
		for _, tcsc := range tcdef.Superclasses {
			tc.Superclasses = append(tc.Superclasses, me.newConstr(tcsc))
		}
		tc.CoveringSets = tcdef.CoveringSets
		tc.DeterminedArgs = tcdef.DeterminedArgs
		for _, fdep := range tcdef.Dependencies {
			tc.Dependencies = append(tc.Dependencies, irMTypeClassDep{Determiners: fdep.Determiners, Determined: fdep.Determined})
		}
		me.EnvTypeClasses = append(me.EnvTypeClasses, tc)
	}
	for _, m := range me.mod.coreimp.DeclEnv.ClassDicts {
		for tciclass, tcinsts := range m {
			for tciname, tcidef := range tcinsts {
				tci := &irMTypeClassInst{Name: tciname, ClassName: tciclass, Chain: tcidef.Chain, Index: tcidef.Index, Value: tcidef.Value}
				for _, tcid := range tcidef.Dependencies {
					tci.Dependencies = append(tci.Dependencies, me.newConstr(tcid))
				}
				for _, tcip := range tcidef.Path {
					tci.Path = append(tci.Path, irMTypeClassInstPath{Cls: tcip.Class, Idx: tcip.Int})
				}
				for _, tcit := range tcidef.InstanceTypes {
					tci.Types = append(tci.Types, me.newTypeRefFromEnvTag(tcit))
				}
				me.EnvTypeClassInsts = append(me.EnvTypeClassInsts, tci)
			}
		}
	}
}
