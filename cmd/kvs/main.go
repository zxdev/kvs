package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/zxdev/kvs"
)

// kvs-tool
//	inspect kvs resources
//	provide kvs lookup service

func main() {

	switch len(os.Args) {
	case 1:
		fmt.Println("kvs {file} {key,key,key}")
		return

	case 2:

		detect(&os.Args[1])
		info := kvs.Info(os.Args[1])
		if !info.Ok {
			fmt.Println("kvs: invalid resource")
			return
		}

		fmt.Println("\n ", filepath.Base(os.Args[1]))
		fmt.Println("---------------------------------")
		fmt.Println("checksum   :", info.Checksum)
		fmt.Println("capacity   :", info.Max)
		fmt.Println("count      :", info.Count)
		fmt.Printf("format     : %d x %x\n", info.Depth, info.Width)
		fmt.Printf("density    : %d %d [%d]\n", info.Density, info.Depth*info.Width, (info.Depth*info.Width)-info.Count)
		fmt.Printf("shuffler   : %d x %d\n\n", info.Shuffler, info.Tracker)

	case 3:

		kind := detect(&os.Args[1])
		switch {
		case kind.Keon:
			kv, ok := kvs.LoadKEON(os.Args[1])
			if ok {
				lookup := kv.Lookup()
				for _, v := range strings.Split(os.Args[2], ",") {
					fmt.Println("lookup:", v, lookup([]byte(v)))
				}
			}

		case kind.Keva:
			kv, ok := kvs.LoadKEVA(os.Args[1])
			if ok {
				lookup := kv.Lookup()
				for _, v := range strings.Split(os.Args[2], ",") {
					item := lookup([]byte(v))
					var b []byte
					binary.LittleEndian.PutUint64(b, item.Value)
					fmt.Printf("lookup: %s %v %v", v, item.Ok, b)
				}
			}
		}

	}
}

// detect kvs type; assurance
func detect(fn *string) (kind struct {
	Keon, Keva bool
}) {

	switch {
	case strings.HasSuffix(*fn, ".keon"):
		kind.Keon = !kind.Keon
	case strings.HasSuffix(*fn, ".keva"):
		kind.Keva = !kind.Keva
	default:
		_, err := os.Stat(*fn + ".keon")
		if kind.Keon = !errors.Is(err, fs.ErrNotExist); kind.Keon {
			*fn += ".keon"
			return
		}
		_, err = os.Stat(*fn + ".keva")
		if kind.Keva = !errors.Is(err, fs.ErrNotExist); kind.Keva {
			*fn += ".keva"
		}
	}
	return
}
