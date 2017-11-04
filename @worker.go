package main

import (
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

func (me *mainWorker) checkIfDepDirHasBowerFile(locker sync.Locker, reldirpath string) {
	defer me.Done()
	jsonfilepath := filepath.Join(reldirpath, ".bower.json")
	if depname := strings.TrimLeft(reldirpath[len(Proj.DepsDirPath):], "\\/"); ufs.FileExists(jsonfilepath) {
		bproj := &psPkg{
			DepsDirPath: Proj.DepsDirPath, BowerJsonFilePath: jsonfilepath, SrcDirPath: filepath.Join(reldirpath, "src"),
		}
		defer locker.Unlock()
		locker.Lock()
		Deps[depname] = bproj
	}
}
