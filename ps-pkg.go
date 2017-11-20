package main

import (
	"errors"
	"path/filepath"
	"strings"
	"sync"

	"github.com/metaleap/go-util/dev/bower"
	"github.com/metaleap/go-util/fs"
)

type psBowerFile struct {
	udevbower.BowerFile
	Gonad Cfg // likely empty for most (and ignored for all) deps. BUT the Proj itself will typically contain this in its bower.json file to configure our transpilation
}

type psPkg struct {
	BowerJsonFile     psBowerFile
	BowerJsonFilePath string
	DepsDirPath       string
	SrcDirPath        string
	Modules           []*modPkg
	GoOut             struct {
		PkgDirPath string
	}

	importedDirectlyOrIndirectlyFromProj bool
}

func (me *psPkg) loadFromJsonFile() {
	err := udevbower.LoadFromFile(me.BowerJsonFilePath, &me.BowerJsonFile)
	if err == nil {
		isdep := (me != &Proj)
		if !isdep {
			ProjCfg = &me.BowerJsonFile.Gonad
			ProjCfg.populateDefaultsUponLoaded()
			ProjCfg.loadedFromJson = true
			err = ufs.EnsureDirExists(ProjCfg.Out.GoDirSrcPath)
		}
		if err == nil {
			// proceed
			me.GoOut.PkgDirPath = ProjCfg.Out.GoNamespaceProj
			if isdep && ProjCfg.Out.GoNamespaceDeps != "" {
				me.GoOut.PkgDirPath = ProjCfg.Out.GoNamespaceDeps
				if repourl := me.BowerJsonFile.RepositoryURLParsed(); repourl != nil && repourl.Path != "" {
					if i := strings.LastIndex(repourl.Path, "."); i > 0 {
						me.GoOut.PkgDirPath = filepath.Join(ProjCfg.Out.GoNamespaceDeps, repourl.Path[:i])
					} else {
						me.GoOut.PkgDirPath = filepath.Join(ProjCfg.Out.GoNamespaceDeps, repourl.Path)
					}
				}
				if me.GoOut.PkgDirPath = strings.Trim(me.GoOut.PkgDirPath, "/\\"); !strings.HasSuffix(me.GoOut.PkgDirPath, me.BowerJsonFile.Name) {
					me.GoOut.PkgDirPath = filepath.Join(me.GoOut.PkgDirPath, me.BowerJsonFile.Name)
				}
				if me.BowerJsonFile.Version != "" {
					me.GoOut.PkgDirPath = filepath.Join(me.GoOut.PkgDirPath, me.BowerJsonFile.Version)
				}
			}
			gopkgdir := filepath.Join(ProjCfg.Out.GoDirSrcPath, me.GoOut.PkgDirPath)
			ufs.WalkAllFiles(me.SrcDirPath, func(relpath string) bool {
				if relpath = strings.TrimLeft(relpath[len(me.SrcDirPath):], "\\/"); strings.HasSuffix(relpath, ".purs") {
					me.addModPkgFromPsSrcFileIfCoreFiles(relpath, gopkgdir)
				}
				return true
			})
		}
	}
	if err != nil {
		panic(errors.New(me.BowerJsonFilePath + ": " + err.Error()))
	}
}

func (me *psPkg) addModPkgFromPsSrcFileIfCoreFiles(relpath string, gopkgdir string) {
	i, l := strings.LastIndexAny(relpath, "/\\"), len(relpath)-5
	modinfo := &modPkg{
		parentPkg: me, srcFilePath: filepath.Join(me.SrcDirPath, relpath),
		qName: strReplFsSlash2Dot.Replace(relpath[:l]), lName: relpath[i+1 : l],
	}
	modinfo.pName = strReplDot2ꓸ.Replace(modinfo.qName)
	if modinfo.cfnFilePath = filepath.Join(ProjCfg.In.CoreFilesDirPath, modinfo.qName, "corefn.json"); ufs.FileExists(modinfo.cfnFilePath) {
		modinfo.impFilePath = filepath.Join(ProjCfg.In.CoreFilesDirPath, modinfo.qName, "coreimp.json")
		modinfo.extFilePath = filepath.Join(ProjCfg.In.CoreFilesDirPath, modinfo.qName, "externs.json")
		modinfo.irMetaFilePath = filepath.Join(ProjCfg.In.CoreFilesDirPath, modinfo.qName, "gonad.json")
		modinfo.goOutDirPath = relpath[:l]
		modinfo.goOutFilePath = filepath.Join(modinfo.goOutDirPath, modinfo.qName) + ".go"
		modinfo.gopkgfilepath = filepath.Join(gopkgdir, modinfo.goOutFilePath)
		if ufs.FileExists(modinfo.irMetaFilePath) && ufs.FileExists(modinfo.gopkgfilepath) {
			stalemetaˇcfn, _ := ufs.IsNewerThan(modinfo.cfnFilePath, modinfo.irMetaFilePath)
			stalepkgˇcfn, _ := ufs.IsNewerThan(modinfo.cfnFilePath, modinfo.gopkgfilepath)
			if modinfo.reGenIr = stalemetaˇcfn || stalepkgˇcfn; !modinfo.reGenIr {
				if ProjCfg.In.UseExterns && ufs.FileExists(modinfo.extFilePath) {
					stalemetaˇext, _ := ufs.IsNewerThan(modinfo.extFilePath, modinfo.irMetaFilePath)
					stalepkgˇext, _ := ufs.IsNewerThan(modinfo.extFilePath, modinfo.gopkgfilepath)
					modinfo.reGenIr = modinfo.reGenIr || stalemetaˇext || stalepkgˇext
				}
				if ProjCfg.In.UseLegacyCoreImp && ufs.FileExists(modinfo.impFilePath) {
					stalemetaˇimp, _ := ufs.IsNewerThan(modinfo.impFilePath, modinfo.irMetaFilePath)
					stalepkgˇimp, _ := ufs.IsNewerThan(modinfo.impFilePath, modinfo.gopkgfilepath)
					modinfo.reGenIr = modinfo.reGenIr || stalemetaˇimp || stalepkgˇimp
				}
			}
		} else {
			modinfo.reGenIr = true
		}
		me.Modules = append(me.Modules, modinfo)
	}
}

func (me *psPkg) ensureOutDirs() {
	dirpath := filepath.Join(ProjCfg.Out.GoDirSrcPath, me.GoOut.PkgDirPath)
	err := ufs.EnsureDirExists(dirpath)
	if err == nil {
		for _, depmod := range me.Modules {
			if err = ufs.EnsureDirExists(filepath.Join(dirpath, depmod.goOutDirPath)); err != nil {
				break
			}
		}
	}
	if err != nil {
		panic(err)
	}
}

func (me *psPkg) ensureModPkgIrMetas() {
	me.forAll(func(wg *sync.WaitGroup, modinfo *modPkg) {
		defer wg.Done()
		var err error
		if modinfo.reGenIr || Flag.ForceAll {
			err = modinfo.reGenPkgIrMeta()
		} else if err = modinfo.loadPkgIrMeta(); err != nil {
			modinfo.reGenIr = true // we capture this so the .go file later also gets re-gen'd from the re-gen'd IRs
			println(modinfo.qName + ": regenerating due to error when loading " + modinfo.irMetaFilePath + ": " + err.Error())
			err = modinfo.reGenPkgIrMeta()
		}
		if err != nil {
			panic(err)
		}
	})
}

func (me *psPkg) forAll(op func(*sync.WaitGroup, *modPkg)) {
	var wg sync.WaitGroup
	for _, modinfo := range me.Modules {
		wg.Add(1)
		go op(&wg, modinfo)
	}
	wg.Wait()
}

func (me *psPkg) moduleByQName(qname string) *modPkg {
	if qname != "" {
		for _, m := range me.Modules {
			if m.qName == qname {
				return m
			}
		}
	}
	return nil
}

func (me *psPkg) moduleByPName(pname string) *modPkg {
	if pname != "" {
		pᛌname := strReplUnderscore2ꓸ.Replace(pname)
		for _, m := range me.Modules {
			if m.pName == pᛌname || m.pName == pname {
				return m
			}
		}
	}
	return nil
}

func (me *psPkg) populateModPkgIrMetas() {
	me.forAll(func(wg *sync.WaitGroup, modinfo *modPkg) {
		defer wg.Done()
		modinfo.populatePkgIrMeta()
	})
}

func (me *psPkg) prepModPkgIrAsts() {
	me.forAll(func(wg *sync.WaitGroup, modinfo *modPkg) {
		defer wg.Done()
		if modinfo.reGenIr || Flag.ForceAll {
			modinfo.prepIrAst()
		}
	})
}

func (me *psPkg) reGenModPkgIrAsts() {
	me.forAll(func(wg *sync.WaitGroup, modinfo *modPkg) {
		defer wg.Done()
		if modinfo.reGenIr || Flag.ForceAll {
			modinfo.reGenPkgIrAst()
		}
	})
}

func (me *psPkg) shakeOutStaleDeps() {
	if me == &Proj {
		me.importedDirectlyOrIndirectlyFromProj = true
	}
	pkgs2check := map[*psPkg]bool{}
	for _, pkgmod := range me.Modules {
		for _, impmod := range pkgmod.irMeta.imports {
			if impmod.parentPkg != me && !impmod.parentPkg.importedDirectlyOrIndirectlyFromProj {
				impmod.parentPkg.importedDirectlyOrIndirectlyFromProj = true
				pkgs2check[impmod.parentPkg] = true
			}
		}
	}
	for subdep := range pkgs2check {
		subdep.shakeOutStaleDeps()
	}

	if me == &Proj {
		for depname, dep := range Deps {
			if !dep.importedDirectlyOrIndirectlyFromProj {
				delete(Deps, depname)
			}
		}
	}
}

func (me *psPkg) writeOutFiles() {
	me.forAll(func(wg *sync.WaitGroup, m *modPkg) {
		defer wg.Done()
		if m.reGenIr || Flag.ForceAll {
			//	maybe gonad.json
			err := m.writeIrMetaFile()
			if err == nil && (m.reGenIr || Flag.ForceAll) {
				//	maybe gonad.ast.json
				if ProjCfg.Out.DumpAst {
					err = m.writeIrAstFile()
				}
				//	maybe .go file
				if err == nil {
					err = m.writeGoFile()
				}
			}
			if err != nil {
				panic(err)
			}
		}
	})
	return
}
