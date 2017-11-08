package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/metaleap/go-util/fs"
)

type mainWorker struct {
	sync.WaitGroup
}

func (me *mainWorker) forAllDeps(fn func(*psPkg)) {
	f := func(dep *psPkg) {
		defer me.Done()
		fn(dep)
	}
	for _, dep := range Deps {
		me.Add(1)
		go f(dep)
	}
	me.Wait()
}

func (_ *mainWorker) maybeFilterDepsThenEnsureDepOutDirs() {
	if !ProjCfg.Out.IncludeUnusedDeps {
		prevcount := len(Deps) - 1 // Deps minus the Proj, just for the below msg
		Proj.shakeOutStaleDeps()
		if curcount := len(Deps) - 1; curcount != prevcount {
			fmt.Printf("(Ignoring %d unused dependency packages out of %d candidates in %s, processing just %d)\n", prevcount-curcount, prevcount, Proj.DepsDirPath, curcount)
		}
	}
	confirmNoOutDirConflicts() // before we create numerous out-dir hierarchies, so as to not abort half-way through..
	for _, dep := range Deps { // not in parallel because many sub-path overlaps
		dep.ensureOutDirs()
	}
}

func (me *mainWorker) populateDeps() {
	var mutex sync.Mutex

	checkifdepdirhasbowerjsonfile := func(reldirpath string) {
		defer me.Done()
		jsonfilepath := filepath.Join(reldirpath, ".bower.json")
		if depname := strings.TrimLeft(reldirpath[len(Proj.DepsDirPath):], "\\/"); ufs.FileExists(jsonfilepath) {
			bproj := &psPkg{
				DepsDirPath: Proj.DepsDirPath, BowerJsonFilePath: jsonfilepath, SrcDirPath: filepath.Join(reldirpath, "src"),
			}
			defer mutex.Unlock()
			mutex.Lock()
			Deps[depname] = bproj
		}
	}

	ufs.WalkDirsIn(Proj.DepsDirPath, func(reldirpath string) (keepwalking bool) {
		me.Add(1)
		go checkifdepdirhasbowerjsonfile(reldirpath)
		return true
	})
	me.Wait()
}
