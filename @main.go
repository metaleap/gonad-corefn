package main

import (
	"bytes"
	"fmt"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/go-forks/pflag"
	"github.com/metaleap/go-util/fs"
	"github.com/metaleap/go-util/slice"
)

var (
	Proj    psPkg
	ProjCfg *Cfg // nil UNTIL set once after successful load by Proj --- then it points to Proj.BowerJsonFile.Gonad field
	Deps    = map[string]*psPkg{}
	Flag    struct {
		ForceAll bool
		NoPrefix bool
		Comments bool
	}
)

func main() {
	debug.SetGCPercent(-1) // turns off GC altogether: we're not a long-running process
	starttime := time.Now()

	// args match those of purs and/or pulp where there's overlap, other config goes in bower.json's `Gonad` field (see `psBowerFile`)
	pflag.StringVar(&Proj.SrcDirPath, "src-path", "src", "Project-sources directory path")
	pflag.StringVar(&Proj.DepsDirPath, "dependency-path", "bower_components", "Dependencies directory path")
	pflag.StringVar(&Proj.BowerJsonFilePath, "bower-file", "bower.json", "Project file path (further configuration options possible in the Gonad field)")
	pflag.BoolVar(&Flag.NoPrefix, "no-prefix", false, "Do not include comment header")
	pflag.BoolVar(&Flag.Comments, "comments", true, "Include comments in the generated code")
	pflag.BoolVar(&Flag.ForceAll, "force", false, "Force-regenerate all *.go & *.json files, not just the outdated or missing ones")
	pflag.Parse()

	var err error
	if !ufs.DirExists(Proj.DepsDirPath) {
		err = fmt.Errorf("No such `dependency-path` directory: %s", Proj.DepsDirPath)
	} else if !ufs.DirExists(Proj.SrcDirPath) {
		err = fmt.Errorf("No such `src-path` directory: %s", Proj.SrcDirPath)
	} else {
		var do mainWorker
		Proj.loadFromJsonFile() // from now on ProjCfg is non-nil & points to Proj.BowerJsonFile.Gonad field
		do.populateDeps()
		do.forAllDeps((*psPkg).loadFromJsonFile)
		Deps[""] = &Proj // from now on, all Deps and the main Proj are handled in parallel and equivalently

		//	each stage runs for all modpkgs in parallel, but in-between stages we wait so that the next one has all needed inputs
		do.forAllDeps((*psPkg).ensureModPkgIrMetas) // per mod: if regenerate then load PS core*.json files, else load existing gonad.json
		do.maybeFilterDepsThenEnsureDepOutDirs()
		do.forAllDeps((*psPkg).populateModPkgIrMetas) // per mod: if regenerate then populate irMeta anew from loaded PS core*.json files, else minimal preprocessing of restored/deserialized irMeta from gonad.json
		do.forAllDeps((*psPkg).prepModPkgIrAsts)
		do.forAllDeps((*psPkg).reGenModPkgIrAsts)
		do.forAllDeps((*psPkg).writeOutFiles)
		dur := time.Since(starttime)

		//	done, just some misc stuff remains
		allpkgimppaths := map[string]bool{}
		numregen, numtotal := countNumOfReGendModules(allpkgimppaths) // do this even when ForceAll to have the map filled for writeTestMainGo
		if Flag.ForceAll {                                            // if so, numregen right now is a "would be" fictitious count
			numregen = numtotal
		}
		if ProjCfg.Out.MainDepLevel > 0 {
			err = writeTestMainGo(allpkgimppaths)
		}
		if err == nil {
			fmt.Printf("Processing %d modules (re-generating %d) took me %v\n", numtotal, numregen, dur)
		}
	}
	if err != nil {
		panic(err.Error())
	}
}

func countNumOfReGendModules(allpkgimppaths map[string]bool) (numregen int, numtotal int) {
	for _, dep := range Deps {
		for _, mod := range dep.Modules {
			if allpkgimppaths[mod.impPath()], numtotal = mod.reGenIr, numtotal+1; mod.reGenIr {
				numregen++
			}
		}
	}
	return
}

func writeTestMainGo(allpkgimppaths map[string]bool) (err error) {
	okpkgs := []string{}
	for i := 0; i < ProjCfg.Out.MainDepLevel; i++ {
		thisok := []string{}
		for _, dep := range Deps {
			for _, mod := range dep.Modules {
				if modimppath := mod.impPath(); !uslice.StrHas(okpkgs, modimppath) {
					isthisok := true
					for _, imp := range mod.irMeta.Imports {
						if /*imp.emitted &&*/ !uslice.StrHas(okpkgs, imp.ImpPath) {
							if !(imp.PsModQName == "" || strings.HasPrefix(imp.ImpPath, prefixDefaultFfiPkgImpPath)) {
								isthisok = false
								break
							}
						}
					}
					if isthisok {
						// fmt.Printf("dep level #%d\t%s\n", i+1, modimppath)
						thisok = append(thisok, modimppath)
					}
				}
			}
		}
		okpkgs = append(okpkgs, thisok...)
	}
	for pkgimppath, _ := range allpkgimppaths {
		if !uslice.StrHas(okpkgs, pkgimppath) {
			delete(allpkgimppaths, pkgimppath)
		}
	}

	//	we sort them
	pkgimppaths := sort.StringSlice{}
	for pkgimppath, _ := range allpkgimppaths {
		pkgimppaths = append(pkgimppaths, pkgimppath)
	}
	sort.Strings(pkgimppaths)

	w := bytes.NewBufferString("package main\n\nimport (\n")
	for _, pkgimppath := range pkgimppaths {
		if _, err = fmt.Fprintf(w, "\t_ %q\n", pkgimppath); err != nil {
			return
		}
	}
	if _, err = fmt.Fprintf(w, ")\n\nfunc main() { println(%q) }\n", "Looks like this compiled just fine!"); err == nil {
		err = ufs.WriteTextFile(filepath.Join(ProjCfg.Out.GoDirSrcPath, Proj.GoOut.PkgDirPath, "check-if-all-gonad-generated-packages-compile.go"), w.String())
	}
	return
}
