package kvs

import (
	"bufio"
	"fmt"
	"os"
)

// Info will read and return the *KVS file header information.
//
//	Signature types
//	0xff01 keon
//	0xff02 keva
func Info(path string) (info struct {
	Signature, Checksum, Timestamp, Count, Max uint64 // externals
	Depth, Width, Density, Shuffler, Tracker   uint64 // internals
	Ok                                         bool   // status
}) {

	f, err := os.Open(path)
	if err == nil {
		buf := bufio.NewReader(f)
		_, err = fmt.Fscanln(buf, &info.Signature, &info.Checksum, &info.Timestamp, &info.Count, &info.Max,
			&info.Depth, &info.Width, &info.Density, &info.Shuffler, &info.Tracker)
		f.Close()
	}

	// validate the header was readable and the header with a valid checksum, capacity, and signature
	info.Ok = err == nil && info.Checksum > 0 && info.Max > 0 && info.Signature > 0xff00
	return

}
