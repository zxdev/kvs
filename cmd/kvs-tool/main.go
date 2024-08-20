package main

import (
	"encoding/binary"
	"fmt"
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

		switch {
		case strings.HasSuffix(os.Args[1], ".keon"):
			kv, ok := kvs.LoadKEON(os.Args[1])
			if ok {
				lookup := kv.Lookup()
				for _, v := range strings.Split(os.Args[2], ",") {
					fmt.Println("lookup:", v, lookup([]byte(v)))
				}
			}

		case strings.HasSuffix(os.Args[1], ".keva"):
			kv, ok := kvs.LoadKEVA(os.Args[1])
			if ok {
				lookup := kv.Lookup()
				for _, v := range strings.Split(os.Args[2], ",") {
					item := lookup([]byte(v))
					var b []byte
					binary.LittleEndian.PutUint64(b, item.Value)
					fmt.Printf("lookup: %s %v %d", v, item.Ok, b)
				}
			}
		}

	}
}
