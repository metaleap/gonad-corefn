package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/metaleap/go-util/dev/go"
	"github.com/metaleap/go-util/fs"
)

func main() {
	ffidir := (udevgo.GopathSrcGithub("golamb", "da", "ffi", "ps2go"))
	needle := ": undefined: 𝙜ˈ"
	l := len(needle)
	ˈ2slash := strings.NewReplacer("ˈ", "/")
	ˈ2dot := strings.NewReplacer("ˈ", ".")

	readln := bufio.NewScanner(os.Stdin)
	for readln.Scan() {
		for _, msgln := range strings.Split(readln.Text(), "\n") { // should only ever be len 1 but.. play it safe anyway =)
			if i := strings.Index(msgln, needle); i > 0 {
				if pkgnameˇtypename := strings.Split(msgln[i+l:], "."); len(pkgnameˇtypename) != 2 {
					panic("unexpected err-msg format, might have changed?")
				} else {
					pkgname, typename := pkgnameˇtypename[0], pkgnameˇtypename[1]
					ffioutfile := filepath.Join(ffidir, ˈ2slash.Replace(pkgname), ˈ2dot.Replace(pkgname)+".ffi.go")
					if txt := strings.TrimSpace(ufs.ReadTextFile(ffioutfile, false, "")); txt == "" {
						panic("couldn't read file: " + ffioutfile + "; re-run gonad-ps2go-ffi-gen-scaffolds")
					} else {
						txt = txt + "\n\ntype " + typename + " struct{}\n"
						if err := ufs.WriteTextFile(ffioutfile, txt); err != nil {
							panic(err)
						}
					}
				}
			}
		}
	}

}
