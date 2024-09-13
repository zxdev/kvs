package kvs

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
)

// MergeKEON current KEON with another
//
//	action nil,true  insert
//	action false     remove
func MergeKEON(dst *KEON, path string, action interface{}) (result struct {
	Ok, Invalid, NoSpace bool
	Items, Checksum      uint64
}) {

	var header [80]byte
	var src struct{ signature, checksum, count uint64 }
	var current = dst.Checksum()

	r, err := os.Open(path)
	result.Ok = err == nil
	if result.Ok {
		defer r.Close()

		r.Read(header[:])
		src.signature = binary.BigEndian.Uint64(header[:8])
		src.checksum = binary.BigEndian.Uint64(header[8:16])
		//timestamp = binary.BigEndian.Uint64((header[16:24]))
		src.count = binary.BigEndian.Uint64((header[24:32]))
	}

	// valid signature type with content and available space
	result.Invalid = src.signature != 0xff01 || src.count == 0 || src.checksum == 0
	result.NoSpace = dst.count+src.count > dst.max
	result.Ok = !result.Invalid && !result.NoSpace
	if result.Ok {

		var b [8]byte
		var n int
		var err error
		var k uint64

		if action == nil || action.(bool) {

			// use an assurance that we can only add new items
			// so that we can track the new items
			insert := dst.RawInsert(false)
			for {
				n, err = r.Read(b[:])
				if n == 0 || errors.Is(err, io.EOF) {
					break
				}
				k = binary.BigEndian.Uint64(b[:])
				if k != 0 {
					r := insert(b[:])
					if r.Exist {
						continue
					}
					if r.NoSpace {
						// the current format can not support the
						// new additional keys; insert failed
						result.Ok = false
						result.NoSpace = true
						return
					}
					if !r.Ok {
						break
					}
					result.Checksum ^= k
					result.Items++
				}
			}
			result.Ok = dst.Checksum() == current^result.Checksum

		} else {

			remove := dst.RawRemove()
			for {
				n, err = r.Read(b[:])
				if n == 0 || errors.Is(err, io.EOF) {
					break
				}
				k = binary.BigEndian.Uint64(b[:])
				if k != 0 {
					if remove(b[:]).Exist {
						result.Checksum ^= k
						result.Items++
					}
				}
			}
			result.Ok = dst.Checksum() == current^result.Checksum

		}
	}

	return
}

// MergeKEVA current KEON with another
//
//	action nil,true  insert
//	action false     remove
func MergeKEVA(dst *KEVA, path string, action interface{}) (result struct {
	Ok, Invalid, NoSpace bool
	Items, Checksum      uint64
}) {

	var header [80]byte
	var src struct{ signature, checksum, count uint64 }
	var current = dst.Checksum()

	r, err := os.Open(path)
	result.Ok = err == nil
	if result.Ok {
		defer r.Close()

		r.Read(header[:])
		src.signature = binary.BigEndian.Uint64(header[:8])
		src.checksum = binary.BigEndian.Uint64(header[8:16])
		//timestamp = binary.BigEndian.Uint64((header[16:24]))
		src.count = binary.BigEndian.Uint64((header[24:32]))
	}

	// valid signature type with content and available space
	result.Invalid = src.signature != 0xff02 || src.count == 0 || src.checksum == 0
	result.NoSpace = dst.count+src.count > dst.max
	result.Ok = !result.Invalid && !result.NoSpace
	if result.Ok {

		var b [16]byte
		var n int
		var err error
		var k, v uint64

		if action == nil || action.(bool) {

			// we allow updates but keep track of the
			// updated items for our new checksum
			insert := dst.RawInsert(true)
			for {
				n, err = r.Read(b[:])
				if n == 0 || errors.Is(err, io.EOF) {
					break
				}
				k = binary.BigEndian.Uint64(b[:8])
				v = binary.BigEndian.Uint64(b[8:])
				if k != 0 {
					r := insert(b[:8], v)
					if r.Exist {
						continue
					}
					if r.NoSpace {
						// the current format can not support the
						// new additional keys; insert failed
						result.Ok = false
						result.NoSpace = true
						return
					}
					if !r.Ok {
						break
					}
					result.Checksum ^= k
					result.Items++
				}
			}
			result.Ok = dst.Checksum() == current^result.Checksum

		} else {

			remove := dst.RawRemove()
			for {
				n, err = r.Read(b[:])
				if n == 0 || errors.Is(err, io.EOF) {
					break
				}
				k = binary.BigEndian.Uint64(b[:8])
				if k != 0 {
					if remove(b[:]).Exist {
						result.Checksum ^= k
						result.Items++
					}
				}
			}
			result.Ok = dst.Checksum() == current^result.Checksum

		}
	}

	return
}
