package main

import (
	"path/filepath"
	"strings"

	"github.com/metaleap/go-util/dev/go"
	"github.com/metaleap/go-util/fs"
)

func main() {
	ffidir := (udevgo.GopathSrcGithub("gonadz", "-", "ffi", "ps2go"))
	l := len("output/")
	dot2slash := strings.NewReplacer(".", "/")
	dot2ˈ := strings.NewReplacer(".", "ˈ")
	if !ufs.DirExists("output") {
		panic("no ./output directory exists here")
	}
	ufs.WalkDirsIn("output", func(dir string) bool {
		modqname := dir[l:]
		ffisubdir := dot2slash.Replace(modqname)
		ffioutdir := filepath.Join(ffidir, ffisubdir)
		if err := ufs.EnsureDirExists(ffioutdir); err != nil {
			panic(err)
		}
		ffifilename := filepath.Join(ffioutdir, modqname+".ffi.go")
		if !ufs.FileExists(ffifilename) {
			if err := ufs.WriteTextFile(ffifilename, "package 𝙜ˈ"+dot2ˈ.Replace(modqname)+"\n"); err != nil {
				panic(err)
			}
			println(ffifilename)
		}
		return true
	})
}
