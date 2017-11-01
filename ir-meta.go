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
	EnvTypeSyns       []*irPsNamedTypeRef  `json:",omitempty"`
	EnvTypeClasses    []*irPsTypeClass     `json:",omitempty"`
	EnvTypeClassInsts []*irPsTypeClassInst `json:",omitempty"`
	EnvTypeDataDecls  []*irPsTypeDataDef   `json:",omitempty"`
	EnvValDecls       []*irPsNamedTypeRef  `json:",omitempty"`
	GoTypeDefs        irGoNamedTypeRefs    `json:",omitempty"`
	GoValDecls        irGoNamedTypeRefs    `json:",omitempty"`
	ForeignImp        *irMPkgRef           `json:",omitempty"`

	imports []*modPkg

	mod     *modPkg
	proj    *psBowerProject
	isDirty bool
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

func (me *irMPkgRefs) addIfMissing(lname, imppath, qname string) (pkgref *irMPkgRef, added bool) {
	if imppath == "" {
		if strings.HasPrefix(lname, prefixDefaultFfiPkgNs) {
			imppath = prefixDefaultFfiPkgImpPath + strReplˈ2Slash.Replace(lname[len(prefixDefaultFfiPkgNs):])
			lname, qname = "", ""
		} else {
			imppath = lname
		}
	}
	if pkgref = me.byImpPath(imppath); pkgref == nil {
		added, pkgref = true, &irMPkgRef{GoName: lname, ImpPath: imppath, PsModQName: qname}
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
	imp, haschanged := me.Imports.addIfMissing(lname, imppath, qname)
	if haschanged {
		me.isDirty = true
	}
	return imp
}

func (me *irMeta) hasExport(name string) bool {
	return uslice.StrHas(me.Exports, name)
}

func (me *irMeta) populateFromCoreImp() {
	me.mod.coreimp.Prep()
	// discover and store exports
	for _, exp := range me.mod.ext.EfExports {
		if len(exp.TypeRef) > 1 {
			tname := exp.TypeRef[1].(string)
			me.Exports = append(me.Exports, tname)
			if len(exp.TypeRef) > 2 {
				if ctornames, _ := exp.TypeRef[2].([]interface{}); len(ctornames) > 0 {
					for _, ctorname := range ctornames {
						if cn, _ := ctorname.(string); cn != "" && !me.hasExport(cn) {
							me.Exports = append(me.Exports, tname+"ĸ"+cn)
						}
					}
				} else {
					if td := me.mod.coreimp.DeclEnv.TypeDefs[tname]; td != nil && td.Decl.DataType != nil {
						for _, dtctor := range td.Decl.DataType.Ctors {
							me.Exports = append(me.Exports, tname+"ĸ"+dtctor.Name)
						}
					}
				}
			}
		} else if len(exp.TypeClassRef) > 1 {
			me.Exports = append(me.Exports, exp.TypeClassRef[1].(string))
		} else if len(exp.ValueRef) > 1 {
			me.Exports = append(me.Exports, exp.ValueRef[1].(map[string]interface{})["Ident"].(string))
		} else if len(exp.TypeInstanceRef) > 1 {
			me.Exports = append(me.Exports, exp.TypeInstanceRef[1].(map[string]interface{})["Ident"].(string))
		}
	}

	// discover and store imports
	for _, imp := range me.mod.coreimp.Imps {
		if impname := strings.Join(imp, "."); impname != "Prim" && impname != "Prelude" && impname != me.mod.qName {
			me.imports = append(me.imports, findModuleByQName(impname))
		}
	}
	for _, impmod := range me.imports {
		me.Imports = append(me.Imports, impmod.newModImp())
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

func (me *irMeta) populateFromLoaded() {
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
					gvd.Ref.Q = &irGoTypeRefAlias{QName: me.mod.qName + "." + gtd.NamePs}
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
	jsonenc.SetIndent("", "\t")
	return jsonenc.Encode(me)
}
