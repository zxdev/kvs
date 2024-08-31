package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"

	"github.com/zxdev/kvs"
)

// kvs-keon
//	build a keon from an \n list source

func main() {

	switch len(os.Args) {
	case 1:
		fmt.Println("DENSITY= WIDTH= kvs {file}")
		return

	case 2:

		density, _ := strconv.Atoi(os.Getenv("DENSITY"))
		width, _ := strconv.Atoi(os.Getenv("WIDTH"))

		if density == 0 {
			density += 5
		}
		if width == 0 {
			width += 3
		}

		f, err := os.Open(os.Args[1])
		if err == nil {
			defer f.Close()

			var count uint64
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				count++
			}

			kv := kvs.NewKEON(count, &kvs.Option{Density: uint64(density), Width: uint64(width)})
			insert := kv.Insert(false)

			f.Seek(0, 0) // rewind
			scanner = bufio.NewScanner(f)
			for scanner.Scan() {
				if insert(scanner.Bytes()).NoSpace {
					fmt.Printf("failure: count[%d] density[%d], width[%d]", count, density, width)
					return
				}
			}

			kv.Write(os.Args[1] + ".keon")
		}

	}

}
