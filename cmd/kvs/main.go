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
//
//	inspect kvs resources
//	provide kvs lookup service
func main() {

	switch len(os.Args) {
	case 1:
		fmt.Println("kvs {file} {key,key,key}")
		return

	case 2:

		// sense the kvs file extensions for user convienence
		info := kvs.Info(os.Args[1])
		if !info.Ok {
			info = kvs.Info(os.Args[1] + ".keon")
			if !info.Ok {
				info = kvs.Info(os.Args[1] + ".keva")
				if !info.Ok {
					fmt.Println("kvs: invalid resource")
					return
				}
			}
		}

		const unit = 1024 // IEC units
		var size uint64   // bytes per type
		var kind string   // kvs type
		switch info.Signature {
		case 0xff01:
			kind = "keon"
			size += 8
		case 0xff02:
			kind = "keva"
			size += 16
		}

		fmt.Println("\n ", filepath.Base(os.Args[1]))
		fmt.Println("---------------------------------")
		fmt.Println("checksum   :", info.Checksum)
		fmt.Println("timestamp  :", kind, info.Timestamp)
		fmt.Println("capacity   :", info.Max)
		fmt.Println("count      :", info.Count)
		fmt.Printf("format     : %d x %x\n", info.Depth, info.Width)
		fmt.Printf("density    : %d %d [%d]\n", info.Density, info.Depth*info.Width, (info.Depth*info.Width)-info.Count)
		fmt.Printf("shuffler   : %d x %d\n", info.Shuffler, info.Tracker)

		var b = info.Depth * info.Width * size
		if b > unit {
			div, exp := int64(unit), 0
			for n := b / unit; n >= unit; n /= unit {
				div *= unit
				exp++
			}
			fmt.Printf("memory     : %.2f %ciB\n", float64(b)/float64(div), "KMGTPE"[exp])
		}

		fmt.Println()

	case 3:

		// the header signature is what determines the kvs structure type and
		// not the kvs file extension which is sensed for convienence only
		info := kvs.Info(os.Args[1])
		if !info.Ok {
			info = kvs.Info(os.Args[1] + ".keon")
			if info.Ok {
				os.Args[1] += ".keon"
			} else {
				info = kvs.Info(os.Args[1] + ".keva")
				if info.Ok {
					os.Args[1] += ".keva"
				} else {
					fmt.Println("kvs: invalid resource")
					return
				}
			}
		}

		switch info.Signature {
		case 0xff01: // keon
			kv, ok := kvs.LoadKEON(os.Args[1])
			if ok {
				lookup := kv.Lookup()
				for _, v := range strings.Split(os.Args[2], ",") {
					fmt.Println("keon:", v, lookup([]byte(v)))
				}
			}

		case 0xff02: // keva
			kv, ok := kvs.LoadKEVA(os.Args[1])
			if ok {
				lookup := kv.Lookup()
				for _, v := range strings.Split(os.Args[2], ",") {
					item := lookup([]byte(v))
					var b [8]byte
					binary.LittleEndian.PutUint64(b[:], item.Value)
					if item.Value == 0 {
						fmt.Printf("keva: %s %v\n", v, item.Ok)
						continue
					}
					fmt.Printf("keva: %s %v %v\n", v, item.Ok, b)
				}
			}
		}

	}
}
