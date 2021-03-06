package main

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/metaleap/go-util/dev/ps"
	"github.com/metaleap/go-util/slice"
	"github.com/metaleap/go-util/str"
)

type never struct{}

const (
	prefixDefaultFfiPkgImpPath = "github.com/golamb/da/ffi/ps2go/"
	prefixDefaultFfiPkgNs      = "𝙜ˈ"
	msgfmt                     = "encountered un-anticipated %s '%s' in %v,\n\tplease report the case with the *.purs code(base) so that I can support it"
)

var (
	//ꓸ۰٠ߺ  nope: ᛌ ᚲ
	strReplˈ2Slash      = strings.NewReplacer("ˈ", "/")
	strReplDot2ˈ        = strings.NewReplacer(".", "ˈ")
	strReplDot2ꓸ        = strings.NewReplacer(".", "ꓸ")
	strReplDot2Slash    = strings.NewReplacer(".", "/")
	strReplFsSlash2Dot  = strings.NewReplacer("\\", ".", "/", ".")
	strReplUnderscore2ꓸ = strings.NewReplacer("_", "ꓸ")

	strReplSanitizer  = strings.NewReplacer("'", "ˈ", "$", "ᵒ", " ", "ˉ", ":", "ꓽ")
	strReplUnsanitize = strings.NewReplacer("$prime", "'", "$$", "")
)

func deferr(err *error, fn func() error) {
	if e := fn(); e != nil && (*err) == nil {
		*err = e
	}
}

func init() {
	udevps.NotImplErr = notImplErr
	udevps.StrReplUnsanitize = strReplUnsanitize
}

func notImplErr(cat string, name string, in interface{}) error {
	return fmt.Errorf(msgfmt, cat, name, in)
}

func panicWithType(in string, v interface{}, of string) {
	panic(fmt.Errorf("%s: unexpected value %v (type %v) for '%s'", in, v, reflect.TypeOf(v), of))
}

func ensureIfaceForTvar(tdict map[string][]string, tvar string, ifacetname string) {
	if ifaces4tvar := tdict[tvar]; !uslice.StrHas(ifaces4tvar, ifacetname) {
		ifaces4tvar = append(ifaces4tvar, ifacetname)
		tdict[tvar] = ifaces4tvar
	}
}

func findPsTypeByQName(qname string) (mod *modPkg, tr interface{}) {
	var pname, tname string
	i := strings.LastIndex(qname, ".")
	if tname = qname[i+1:]; i > 0 {
		pname = qname[:i]
		if mod = findModuleByQName(pname); mod == nil {
			panic(notImplErr("module qname", pname, qname))
		} else {
			for _, ets := range mod.irMeta.EnvTypeSyns {
				if ets.Name == tname {
					tr = ets
					return
				}
			}
			for _, etc := range mod.irMeta.EnvTypeClasses {
				if etc.Name == tname {
					tr = etc
					return
				}
			}
			for _, eti := range mod.irMeta.EnvTypeClassInsts {
				if eti.Name == tname {
					tr = eti
					return
				}
			}
			for _, etd := range mod.irMeta.EnvTypeDataDecls {
				if etd.Name == tname {
					tr = etd
					return
				}
			}
		}
	} else {
		panic(notImplErr("non-qualified type-name", qname, "a *.purs file of yours"))
	}
	return
}

func findGoTypeByGoQName(curmod *modPkg, qname string) (mod *modPkg, tref *irGoNamedTypeRef) {
	pname, tname := ustr.SplitOnce(qname, '.')
	if mod = findModuleByPName(pname); mod == nil {
		mod = curmod
	}
	tref = mod.irMeta.goTypeDefByGoName(tname)
	return
}

func findGoTypeByPsQName(curmod *modPkg, qname string) (*modPkg, *irGoNamedTypeRef) {
	var pname, tname string
	mod, i := curmod, strings.LastIndex(qname, ".")
	if tname = qname[i+1:]; i > 0 {
		pname = qname[:i]
		if mod = findModuleByQName(pname); mod == nil {
			mod = findModuleByPName(pname)
		}
		if mod == nil {
			if pname == "Prim" {
				return nil, nil
			}
			panic(notImplErr("module qname", pname, qname))
		}
	}
	return mod, mod.irMeta.goTypeDefByPsName(tname, false)
}

func irASymStrOr(me irA, or string) string {
	if asymstr, _ := me.(irASymStr); asymstr != nil {
		return asymstr.symStr()
	}
	return or
}

func sanitizeSymbolForGo(name string, upper bool) string {
	if name == "" {
		return name
	}
	if upper {
		if !ustr.BeginsUpper(name) {
			runes := []rune(name)
			runes[0] = unicode.ToUpper(runes[0])
			name = string(runes)
		}
	} else {
		if ustr.BeginsUpper(name) {
			runes := []rune(name)
			runes[0] = unicode.ToLower(runes[0])
			name = string(runes)
		}
		switch name {
		case "break", "case", "chan", "const", "continue", "default", "defer", "else", "fallthrough", "for", "func", "go", "goto", "if", "import", "interface", "map", "package", "range", "return", "select", "struct", "switch", "type", "var":
			return fmt.Sprintf(ProjCfg.CodeGen.Fmt.Reserved_Keywords, name)
		case "append", "bool", "byte", "cap", "close", "complex", "copy", "delete", "error", "false", "float32", "float64", "imag", "int", "int16", "int32", "int64", "int8", "iota", "len", "make", "new", "nil", "panic", "print", "println", "real", "rune", "recover", "string", "true", "uint", "uint16", "uint32", "uint64", "uint8", "uintptr":
			return fmt.Sprintf(ProjCfg.CodeGen.Fmt.Reserved_Identifiers, name)
		}
	}
	return strReplSanitizer.Replace(name)
}

func typeNameWithPkgName(pkgname string, typename string) (fullname string) {
	if fullname = typename; pkgname != "" {
		fullname = pkgname + "." + fullname
	}
	return
}
