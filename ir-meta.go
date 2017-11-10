package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/metaleap/go-util/slice"
	"github.com/metaleap/go-util/str"
)

type irMeta struct {
	Exports           []string             `json:",omitempty"`
	Imports           irMPkgRefs           `json:",omitempty"`
	EnvTypeSyns       irPsNamedTypeRefs    `json:",omitempty"`
	EnvTypeClasses    []*irPsTypeClass     `json:",omitempty"`
	EnvTypeClassInsts []*irPsTypeClassInst `json:",omitempty"`
	EnvTypeDataDecls  []*irPsTypeDataDef   `json:",omitempty"`
	EnvValDecls       irPsNamedTypeRefs    `json:",omitempty"`
	GoTypeDefs        irGoNamedTypeRefs    `json:",omitempty"`
	GoValDecls        irGoNamedTypeRefs    `json:",omitempty"`
	ForeignImp        *irMPkgRef           `json:",omitempty"`

	imports []*modPkg

	mod             *modPkg
	_primArrAliases irGoNamedTypeRefs
}

type irMPkgRefs []*irMPkgRef

func (me irMPkgRefs) Len() int { return len(me) }
func (me irMPkgRefs) Less(i, j int) bool {
	if u1, u2 := me[i].isUriForm(), me[j].isUriForm(); u1 != u2 {
		return u2
	}
	return me[i].ImpPath < me[j].ImpPath
}
func (me irMPkgRefs) Swap(i, j int) { me[i], me[j] = me[j], me[i] }

func (me *irMPkgRefs) addIfMissing(lname, imppath, qname string) (pkgref *irMPkgRef) {
	if imppath == "" {
		if strings.HasPrefix(lname, prefixDefaultFfiPkgNs) {
			imppath = prefixDefaultFfiPkgImpPath + strReplˈ2Slash.Replace(lname[len(prefixDefaultFfiPkgNs):])
			lname, qname = "", ""
		} else {
			imppath = lname
		}
	}
	if pkgref = me.byImpPath(imppath); pkgref == nil {
		pkgref = &irMPkgRef{GoName: lname, ImpPath: imppath, PsModQName: qname}
		*me = append(*me, pkgref)
	}
	return
}

func (me irMPkgRefs) byImpPath(imppath string) *irMPkgRef {
	for _, imp := range me {
		if imp.ImpPath == imppath {
			return imp
		}
	}
	return nil
}

func (me irMPkgRefs) byImpName(pkgname string) *irMPkgRef {
	if pkgname != "" {
		for _, imp := range me {
			if imp.GoName == pkgname || (imp.GoName == "" && imp.ImpPath == pkgname) {
				return imp
			}
		}
	}
	return nil
}

type irMPkgRef struct {
	GoName     string
	PsModQName string
	ImpPath    string

	emitted bool
}

func (me *irMPkgRef) isUriForm() bool {
	id, is := strings.IndexRune(me.ImpPath, '.'), strings.IndexRune(me.ImpPath, '/')
	return id > 0 && id < is
}

func (me *modPkg) newModImp() *irMPkgRef {
	return &irMPkgRef{GoName: me.pName, PsModQName: me.qName, ImpPath: me.impPath()}
}

func (me *irMeta) ensureImp(lname, imppath, qname string) *irMPkgRef {
	if imp := me.Imports.byImpName(lname); imp != nil {
		return imp
	}
	if imppath == "" && (ustr.BeginsUpper(lname) || ustr.BeginsUpper(qname)) {
		var mod *modPkg
		if qname != "" {
			mod = findModuleByQName(qname)
		} else if lname != "" {
			mod = findModuleByPName(lname)
		}
		if mod != nil {
			lname, qname, imppath = mod.pName, mod.qName, mod.impPath()
		}
	}
	imp := me.Imports.addIfMissing(lname, imppath, qname)
	return imp
}

func (me *irMeta) envTypeDataDeclByPsName(nameps string) *irPsTypeDataDef {
	for _, tdd := range me.EnvTypeDataDecls {
		if tdd.Name == nameps {
			return tdd
		}
	}
	return nil
}

func (me *irMeta) hasExport(name string) bool {
	return uslice.StrHas(me.Exports, name)
}

func (me *irMeta) populateFromCore() {
	if ProjCfg.In.UseLegacyCoreImp && me.mod.coreImp != nil {
		me.populateFromCoreImp()
	} else {
		me.populateFromCoreFn()
	}
}

func (me *irMeta) populateFromCoreExt() {
	for _, extexp := range me.mod.coreExt.Exports {
		if len(extexp.TypeRef) > 1 {
			tname := extexp.TypeRef[1].(string)
			me.Exports = append(me.Exports, tname)
			if len(extexp.TypeRef) > 2 {
				if ctornames, _ := extexp.TypeRef[2].([]interface{}); len(ctornames) > 0 {
					for _, ctorname := range ctornames {
						cn := ctorname.(string)
						me.Exports = append(me.Exports, cn)
						me.Exports = append(me.Exports, tname+"ĸ"+cn)
					}
				} else if me.mod.coreImp != nil {
					if td := me.mod.coreImp.DeclEnv.TypeDefs[tname]; td != nil && td.Decl.DataType != nil {
						for _, dtctor := range td.Decl.DataType.Ctors {
							me.Exports = append(me.Exports, tname+"ĸ"+dtctor.Name)
						}
					}
				}
			}
		} else if len(extexp.TypeClassRef) > 1 {
			me.Exports = append(me.Exports, extexp.TypeClassRef[1].(string))
		} else if len(extexp.ValueRef) > 1 {
			me.Exports = append(me.Exports, extexp.ValueRef[1].(map[string]interface{})["Ident"].(string))
		} else if len(extexp.TypeInstanceRef) > 1 {
			me.Exports = append(me.Exports, extexp.TypeInstanceRef[1].(map[string]interface{})["Ident"].(string))
		}
	}
}

func (me *irMeta) populateFromCoreFn() {
	usecfnexports := (!ProjCfg.In.UseExterns) || me.mod.coreExt == nil
	if usecfnexports {
		me.Exports = me.mod.coreFn.Exports
	} else {
		me.populateFromCoreExt()
	}
	for i := 0; i < len(me.mod.coreFn.Decls); i++ {
		decl := &me.mod.coreFn.Decls[i]
		for j, _ := range decl.Binds {
			declbind := &decl.Binds[j]
			var dd *irPsTypeDataDef
			if ctor := declbind.Expression.Constructor; ctor != nil {
				if ctor.Annotation.Meta != nil {
					panic(notImplErr("CoreFnExpr.Constructor.Annotation.Meta", ctor.Annotation.Meta.String(), *ctor.Annotation.Meta))
				}
				if dd = me.envTypeDataDeclByPsName(ctor.TypeName); dd == nil {
					dd = &irPsTypeDataDef{Name: ctor.TypeName}
					me.EnvTypeDataDecls = append(me.EnvTypeDataDecls, dd)
				}
				ddctor := irPsTypeDataCtor{Name: ctor.ConstructorName, Export: me.hasExport(ctor.ConstructorName), DataTypeName: ctor.TypeName}
				if expĸ := ctor.TypeName + "ĸ" + ctor.ConstructorName; ddctor.Export && !me.hasExport(expĸ) {
					me.Exports = append(me.Exports, expĸ)
				}
				ddctor.Args = make([]*irPsTypeDataCtorArg, 0, len(ctor.FieldNames))
				for _, cfn := range ctor.FieldNames {
					ddctor.Args = append(ddctor.Args, &irPsTypeDataCtorArg{Name: cfn})
				}
				dd.Ctors = append(dd.Ctors, &ddctor)

				me.mod.coreFn.RemoveAt(i)
				i--
				break // so far true across 700+ real-world PS modules: whenever there's a Constructor inside decl.Binds, the latter has len=1
			} else if abs := declbind.Expression.Abs; abs != nil {
				if absv := abs.Body.Var; absv != nil && absv.Value.IsModuleNameNil() && abs.Argument == absv.Value.Identifier {
					if meta := abs.Meta(); meta != nil && meta.IsNewtype() {
						if ProjCfg.In.UseExterns && me.mod.coreExt != nil {
							for _, exp := range me.Exports {
								if suff := "ĸ" + declbind.Identifier; strings.HasSuffix(exp, suff) {
									ddtname := exp[:len(exp)-len(suff)]
									if dd = me.envTypeDataDeclByPsName(ddtname); dd == nil {
										dd = &irPsTypeDataDef{Name: ddtname}
										me.EnvTypeDataDecls = append(me.EnvTypeDataDecls, dd)
									}
								}
							}
						}
						if dd == nil {
							tsyn := &irPsNamedTypeRef{Name: declbind.Identifier, Ref: &irPsTypeRef{}}
							me.EnvTypeSyns = append(me.EnvTypeSyns, tsyn)
						} else {
							ddctor := irPsTypeDataCtor{Name: declbind.Identifier, Export: me.hasExport(declbind.Identifier), DataTypeName: dd.Name}
							ddctor.Args = []*irPsTypeDataCtorArg{&irPsTypeDataCtorArg{Name: abs.Argument}}
							dd.Ctors = append(dd.Ctors, &ddctor)
						}

						me.mod.coreFn.RemoveAt(i)
						i--
						break // so far true across 700+ real-world PS modules: whenever there's an Abs with Meta.IsNewtype inside decl.Binds, the latter has len=1
					}
				}
			}
		}
	}
	if usecfnexports {
		for _, tdd := range me.EnvTypeDataDecls {
			anyctorsexported := false
			if len(tdd.Ctors) == 1 && len(tdd.Ctors[0].Args) == 1 {
				tdd.Ctors[0].IsNewType = true
			}
			if !me.hasExport(tdd.Name) {
				for _, ctor := range tdd.Ctors {
					if ctor.Export {
						anyctorsexported = true
						break
					}
				}
				if anyctorsexported {
					me.Exports = append(me.Exports, tdd.Name)
				}
			}
		}
	}
	me.populateGoTypeDefs()
}

func (me *irMeta) populateFromCoreImp() {
	me.mod.coreImp.Prep()
	if ProjCfg.In.UseExterns && me.mod.coreExt != nil {
		me.populateFromCoreExt()
	}
	// transform 100% complete coreimp structures
	// into lean, only-what-we-use irMeta structures (still representing PS-not-Go decls)
	me.populateEnvTypeSyns()
	me.populateEnvTypeClasses()
	me.populateEnvTypeDataDecls()
	me.populateEnvFuncsAndVals()

	// then transform those into Go decls
	me.populateGoTypeDefs()
	me.populateGoValDecls()
}

func (me *irMeta) populateImportsEarly() {
	//	from core files?
	if me.mod.coreFn != nil {
		// discover and store imports
		for _, imp := range me.mod.coreFn.Imports {
			if impname := strings.Join(imp.ModuleName, "."); impname != "Prim" && impname != "Prelude" && impname != me.mod.qName {
				me.imports = append(me.imports, findModuleByQName(impname))
			}
		}
		for _, impmod := range me.imports {
			me.Imports = append(me.Imports, impmod.newModImp())
		}
	} else { // or restored from gonad.json file?
		me.imports = nil
		for _, imp := range me.Imports {
			if !strings.HasPrefix(imp.ImpPath, prefixDefaultFfiPkgImpPath) {
				if impmod := findModuleByQName(imp.PsModQName); impmod != nil {
					me.imports = append(me.imports, impmod)
				} else if imp.PsModQName != "" {
					panic(fmt.Errorf("%s: bad import %s", me.mod.srcFilePath, imp.PsModQName))
				}
			}
		}
	}
}

func (me *irMeta) populateFromLoaded() {
	for _, tc := range me.EnvTypeClasses {
		for _, tcm := range tc.Members {
			tcm.parent = tc
		}
	}
}

func (me *irMeta) populateGoValDecls() {
	for _, evd := range me.EnvValDecls {
		tdict := map[string][]string{}
		gvd := &irGoNamedTypeRef{Export: me.hasExport(evd.Name)}
		gvd.setBothNamesFromPsName(evd.Name)
		for gtd := me.goTypeDefByGoName(gvd.NameGo); gtd != nil; gtd = me.goTypeDefByGoName(gvd.NameGo) {
			gvd.NameGo += "ˆ"
		}
		for gvd2 := me.goValDeclByGoName(gvd.NameGo); gvd2 != nil; gvd2 = me.goValDeclByGoName(gvd.NameGo) {
			gvd.NameGo += "ˇ"
		}
		gvd.Ref.setFrom(me.toIrGoTypeRef(tdict, evd.Ref))
		if gvd.Ref.S != nil && len(gvd.Ref.S.Fields) > 0 {
			for _, gtd := range me.GoTypeDefs {
				if gtd.Ref.S != nil && gtd.Ref.S.equiv(gvd.Ref.S) {
					gvd.Ref.S = nil
					gvd.Ref.Q = &irGoTypeRefSyn{QName: me.mod.qName + "." + gtd.NamePs}
				}
			}
		}
		me.GoValDecls = append(me.GoValDecls, gvd)
	}
}

func (me *irMeta) goValDeclByGoName(goname string) *irGoNamedTypeRef {
	for _, gvd := range me.GoValDecls {
		if gvd.NameGo == goname {
			return gvd
		}
	}
	return nil
}

func (me *irMeta) goValDeclByPsName(psname string) *irGoNamedTypeRef {
	for _, gvd := range me.GoValDecls {
		if gvd.NamePs == psname {
			return gvd
		}
	}
	return nil
}

func (me *irMeta) writeAsJsonTo(w io.Writer) error {
	jsonenc := json.NewEncoder(w)
	return jsonenc.Encode(me)
}
