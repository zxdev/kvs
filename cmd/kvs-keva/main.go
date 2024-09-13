package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/zxdev/kvs"
)

// kvs-keva
//	build a keva from an \n comma delimited key,value source
// 	key must be text
//	value but be a unint64 encoded number

func main() {

	switch len(os.Args) {
	case 1:
		fmt.Println("DENSITY={n} WIDTH={n} DELIMITER={v} kvs {file}")
		return

	case 2:

		density, _ := strconv.Atoi(os.Getenv("DENSITY"))
		width, _ := strconv.Atoi(os.Getenv("WIDTH"))
		delimiter := os.Getenv("DELIMITER")

		if density == 0 {
			density += 5
		}
		if width == 0 {
			width += 3
		}
		if len(delimiter) == 0 {
			delimiter = ","
		}

		f, err := os.Open(os.Args[1])
		if err == nil {
			defer f.Close()

			var count uint64
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				count++
			}

			f.Seek(0, 0) // rewind

			var seg []string
			var value int
			var kv = kvs.NewKEVA(count, &kvs.Option{Density: uint64(density), Width: uint64(width)})
			insert := kv.Insert(false)
			scanner = bufio.NewScanner(f)
			for scanner.Scan() {
				seg = strings.Split(scanner.Text(), delimiter)
				if len(seg) != 2 || len(seg[0]) == 0 {
					continue
				}
				value, _ = strconv.Atoi(seg[1])
				if insert([]byte(seg[0]), uint64(value)).NoSpace {
					fmt.Printf("failure: count[%d] density[%d], width[%d]", count, density, width)
					return
				}
			}

			kv.Write(os.Args[1] + ".keva")
		}

	}

}
