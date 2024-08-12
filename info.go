package kvs

import (
	"bufio"
	"fmt"
	"os"
)

// Info will read and return the *KVS file header information.
func Info(path string) (result struct {
	Checksum, Count, Max                     uint64
	Depth, Width, Density, Shuffler, Tracker uint64
	Ok                                       bool
}) {

	f, err := os.Open(path)
	if err == nil {
		buf := bufio.NewReader(f)
		_, err = fmt.Fscanln(buf, &result.Checksum, &result.Count, &result.Max,
			&result.Depth, &result.Width, &result.Density, &result.Shuffler, &result.Tracker)
		f.Close()
	}

	// validate the header was readable and the header appears valid
	result.Ok = err == nil && result.Checksum > 0 && result.Max > 0
	return

}
