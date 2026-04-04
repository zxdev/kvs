package router

import "strings"

func a2i(s string) (n uint64) {
	var c byte
	for _, c = range []byte(strings.TrimSpace(s)) {
		if c > 47 && c < 58 {
			n = n*10 + uint64(c-48)
		} else {
			n = 0
			break
		}
	}
	return
}
