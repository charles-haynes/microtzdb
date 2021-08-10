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
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer/stateful"
)

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// Ident { '/' Ident } ' ' Abbrev Time [ Abbrev [Time] ',' Date ['/' Time] ',' Date ['/' Time] ]

type TZSpec struct {
	Name string `parser:"@(FileNameComponent ('/' FileNameComponent)*) Whitespace"`
	// Posix PosixSpec `parser:"@@"`
	Posix PosixSpec `parser:"@@"`
}

type PosixSpec struct {
	Std       string   `parser:"@Abbrev"`
	StdOffset string   `parser:"@Time"`
	Dst       *DSTSpec `parser:"@@?"`
}

type DSTSpec struct {
	Dst     string  `parser:"@Abbrev"`
	Offset  *string `parser:"@Time?"`
	StdDate string  `parser:"',' @Date"`
	StdTime *string `parser:"('/' @Time)?"`
	DstDate string  `parser:"',' @Date"`
	DstTime *string `parser:"('/' @Time)?"`
}

var (
	posixTZlexer = stateful.Must(stateful.Rules{
		"Root": {
			// FilenameComponents arent's supposed to have numbers or + in them
			// {Name: "FileNameComponent", Pattern: "[A-Za-z._][A-Za-z._-]*", Action: nil},
			{Name: "FileNameComponent", Pattern: "[A-Za-z._][A-Za-z0-9._+-]*", Action: nil},
			{Name: "Whitespace", Pattern: "\\s+", Action: stateful.Push("TZSpec")},
			{Name: "Special", Pattern: "[/,]", Action: nil},
		},
		"TZSpec": {
			{Name: "Abbrev", Pattern: "(<[^>]+>|[^+,:0-9-][^+,0-9-]+)", Action: nil},
			{Name: "Time", Pattern: "[[+-]?\\d{1,3}(:\\d{2}(:\\d{2})?)?", Action: nil},
			{Name: "Date", Pattern: "J\\d{1,3}|\\d{1,3}|M\\d{1,2}\\.\\d\\.\\d", Action: nil},
			{Name: "Whitespace", Pattern: "\\s+", Action: stateful.Pop()},
			{Name: "Special", Pattern: "[/,]", Action: nil},
		},
	})

	posixTZparser = participle.MustBuild(
		&TZSpec{},
		participle.Lexer(posixTZlexer))

	cli struct {
		Dir string `arg:"" type:"existingdir" help:"Root of the zoneinfo db." default:"/usr/share/zoneinfo"`
	}
)

// func printMap(m map[string]int) {
// 	for k, v := range m {
// 		fmt.Printf("%2d %s\n", v, k)
// 	}
// }

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

// var abbrevs = map[string]int{}
// var offsets = map[string]int{}
// var dates = map[string]int{}
// var times = map[string]int{}

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
	// tzSpec := &TZSpec{}
	// err = posixTZparser.ParseBytes(cli.Dir, b[i+1:], tzSpec)
	// checkErr(err)

	posixTZ := b[i+1 : len(b)-1]
	if _, ok := posix[string(posixTZ)]; !ok {
		posix[string(posixTZ)] = len(posix)
		fmt.Printf("  /* %3d */ \"%s\",\n", posix[string(posixTZ)], posixTZ)
	}
	// h := fnvHash(tzSpec.Name)
	h := fnvHash(base)
	// is it unique?
	for _, v := range names {
		for (h & mask) == (v.Hash & mask) {
			// not unique, extend the mask till it is
			mask = (mask << 1) | 1
		}
	}
	names[base] = NameEnt{Name: base, Hash: h, Posix: posix[string(posixTZ)]}
	// abbrevs[tzSpec.Std]++
	// offsets[tzSpec.StdOffset]++
	// if tzSpec.Dst != nil {
	// 	abbrevs[tzSpec.Dst.Dst]++
	// 	if tzSpec.Dst.Offset != nil {
	// 		offsets[*tzSpec.Dst.Offset]++
	// 	}
	// 	dates[tzSpec.Dst.StdDate]++
	// 	if tzSpec.Dst.StdTime != nil {
	// 		times[*tzSpec.Dst.StdTime]++
	// 	}
	// 	dates[tzSpec.Dst.DstDate]++
	// 	if tzSpec.Dst.DstTime != nil {
	// 		times[*tzSpec.Dst.DstTime]++
	// 	}
	// }
	return nil
}

func main() {
	ctx := kong.Parse(&cli)
	fmt.Print("const char *posix[] = {\n")
	err := filepath.WalkDir(cli.Dir, walkDirFn)
	ctx.FatalIfErrorf(err)
	fmt.Println("};")
	fmt.Printf("\nconst uint32_t mask = 0x%x;\n", mask)
	// fmt.Printf("\nnames[%d]:\n", len(names))
	// printMap(names)
	// fmt.Printf("\nabbrevs[%d]:\n", len(abbrevs))
	// printMap(abbrevs)
	// fmt.Printf("\noffsets[%d]:\n", len(offsets))
	// printMap(offsets)
	// fmt.Printf("\ndates[%d]:\n", len(dates))
	// printMap(dates)
	// fmt.Printf("\ntimes[%d]:\n", len(times))
	// printMap(times)
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
