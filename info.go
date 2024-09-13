package kvs

import (
	"encoding/binary"
	"io"
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
		var header [80]byte
		var n int
		n, err = io.ReadFull(f, header[:])
		if n == 80 && err == nil {
			info.Signature = binary.BigEndian.Uint64(header[:8])
			info.Checksum = binary.BigEndian.Uint64(header[8:16])
			info.Timestamp = binary.BigEndian.Uint64(header[16:24])
			info.Count = binary.BigEndian.Uint64(header[24:32])
			info.Max = binary.BigEndian.Uint64(header[32:40])
			info.Depth = binary.BigEndian.Uint64(header[40:48])
			info.Width = binary.BigEndian.Uint64(header[48:56])
			info.Density = binary.BigEndian.Uint64(header[56:64])
			info.Shuffler = binary.BigEndian.Uint64(header[64:72])
			info.Tracker = binary.BigEndian.Uint64(header[72:])
		}
		f.Close()
	}

	// validate the header was readable and the header has a valid signature, checksum, and capacity
	info.Ok = err == nil && info.Signature > 0xff00 && info.Checksum > 0 && info.Timestamp > 0 && info.Max > 0
	return

}
