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

		var kind string
		switch info.Signature {
		case 0xff01:
			kind = "keon"
		case 0xff02:
			kind = "keva"
		}

		fmt.Println("\n ", filepath.Base(os.Args[1]))
		fmt.Println("---------------------------------")
		fmt.Println("checksum   :", info.Checksum)
		fmt.Println("timestamp  :", kind, info.Timestamp)
		fmt.Println("capacity   :", info.Max)
		fmt.Println("count      :", info.Count)
		fmt.Printf("format     : %d x %x\n", info.Depth, info.Width)
		fmt.Printf("density    : %d %d [%d]\n", info.Density, info.Depth*info.Width, (info.Depth*info.Width)-info.Count)
		fmt.Printf("shuffler   : %d x %d\n\n", info.Shuffler, info.Tracker)

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

// // detect kvs type; assurance
// func detect(fn *string) (kind struct {
// 	Keon, Keva bool
// }) {

// 	switch {
// 	case strings.HasSuffix(*fn, ".keon"):
// 		kind.Keon = !kind.Keon
// 	case strings.HasSuffix(*fn, ".keva"):
// 		kind.Keva = !kind.Keva
// 	default:
// 		_, err := os.Stat(*fn + ".keon")
// 		if kind.Keon = !errors.Is(err, fs.ErrNotExist); kind.Keon {
// 			*fn += ".keon"
// 			return
// 		}
// 		_, err = os.Stat(*fn + ".keva")
// 		if kind.Keva = !errors.Is(err, fs.ErrNotExist); kind.Keva {
// 			*fn += ".keva"
// 		}
// 	}
// 	return
// }
