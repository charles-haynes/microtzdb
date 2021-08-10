package main

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/alecthomas/kong"
)

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

var cli struct {
	Dir string `arg:"" type:"existingdir" help:"Root of the zoneinfo db." default:"/usr/share/zoneinfo"`
}

const FNV_PRIME = uint32(16777619)
const OFFSET_BASIS = uint32(2166136261)

func fnvHash(str string) uint32 {
	hash := OFFSET_BASIS
	for i := 0; i < len(str); i++ {
		hash ^= uint32(str[i])
		hash *= FNV_PRIME
	}
	return hash
}

type NameEnt struct {
	Name  string
	Hash  uint32
	Posix int
}

type By func(n1, n2 *NameEnt) bool

func (by By) Sort(names []NameEnt) {
	ns := &nameSorter{
		names: names,
		by:    by,
	}
	sort.Sort(ns)
}

type nameSorter struct {
	names []NameEnt
	by    func(n1, n2 *NameEnt) bool
}

func (s *nameSorter) Len() int           { return len(s.names) }
func (s *nameSorter) Swap(i, j int)      { s.names[i], s.names[j] = s.names[j], s.names[i] }
func (s *nameSorter) Less(i, j int) bool { return s.by(&s.names[i], &s.names[j]) }

var names = map[string]NameEnt{}
var posix = map[string]int{}

var mask = uint32(1)

func walkDirFn(path string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if !d.Type().IsRegular() {
		return nil
	}
	base, err := filepath.Rel(cli.Dir, path)
	checkErr(err)
	f, err := os.Open(path)
	checkErr(err)
	defer f.Close()
	b, err := ioutil.ReadFile(path)
	checkErr(err)
	if string(b[0:4]) != "TZif" {
		return nil
	}
	// the last line of a TZif file is the posix timzone string
	i := strings.LastIndexByte(string(b[:len(b)-1]), '\n')
	if i < 0 {
		return nil // no posix TZ spec in this file
	}

	posixTZ := b[i+1 : len(b)-1]
	if _, ok := posix[string(posixTZ)]; !ok {
		posix[string(posixTZ)] = len(posix)
		fmt.Printf("  /* %3d */ \"%s\",\n", posix[string(posixTZ)], posixTZ)
	}
	h := fnvHash(base)
	// is it unique?
	for _, v := range names {
		for (h & mask) == (v.Hash & mask) {
			// not unique, extend the mask till it is
			mask = (mask << 1) | 1
		}
	}
	names[base] = NameEnt{Name: base, Hash: h, Posix: posix[string(posixTZ)]}
	return nil
}

func main() {
	ctx := kong.Parse(&cli)
	fmt.Print("const char *posix[] = {\n")
	err := filepath.WalkDir(cli.Dir, walkDirFn)
	ctx.FatalIfErrorf(err)
	fmt.Println("};")
	fmt.Printf("\nconst uint32_t mask = 0x%x;\n", mask)
	sortedNames := make([]NameEnt, 0, len(names))
	for _, k := range names {
		k.Hash &= mask
		sortedNames = append(sortedNames, k)
	}
	hash := func(n1, n2 *NameEnt) bool { return n1.Hash < n2.Hash }
	By(hash).Sort(sortedNames)
	fmt.Print("\nconst struct {uint32_t hash:24; uint8_t posix:8;} zones[] = {\n")
	for _, k := range sortedNames {
		fmt.Printf("  {%7d, %3d}, // %s\n", k.Hash, k.Posix, k.Name)
	}
	fmt.Println("};")
}
