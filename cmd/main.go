package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zxdev/kvs"
)

func main() {

	switch len(os.Args) {
	case 1:
		fmt.Println("kvs {file}")
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

	}
}
