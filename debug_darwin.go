package kvs

import "fmt"

// Dump and format the kn.key
func (kn *KEON) Dump() {
	for i := 0; i < len(kn.key); i++ {
		fmt.Printf("%016x ", kn.key[i])
		if (i+1)%int(kn.width) == 0 {
			fmt.Println()
		}
	}
}
