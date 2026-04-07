package kvs

import (
	"crypto/rand"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/zxdev/xxhash"
)

/*
	KE:ON is a cookoo style hash table that distributes and rebalances keys
	across alternative index locations for membership testing. It is
	similar to a map[string]bool only faster and more memory efficient and
	offers a density compaction factor as well as file functions and tuning.

	key|key|key
	key|key|key
	...
	key|key|key

	Note: this is not mutex protected so it is not safe to read/write
	at the time same time, but it is safe for concurrent reads.

	insert := kn.Insert()
	lookup := kn.Lookup()
	remove := kn.Remove()
	for ... {
		if !insert(key).Ok {
			// handle error
		}
	}
*/

// Keon signature
const KeonSignature = 0xff01

// KEON is a set-only hash table structure
type KEON struct {
	name              string   // source name
	origin            int64    // origin timestamp
	count, max        uint64   // count of items, and max items
	depth, width      uint64   // depth and width to establish hash bucket locations [ key|key|key ]
	density, shuffler uint64   // options
	tracker           int      // options
	key               []uint64 // key slice
}

/*

	kvs package level generational functions
		NewKEON, LoadKEON, SaveKEON

*/

// NewKEON constructor that accepts optional configuration settings.
func NewKEON(n int, opt *Option) *KEON {

	if opt == nil {
		opt = new(Option)
	}
	opt.configure()

	var kn = &KEON{
		name:     "kvs.keon",
		origin:   time.Now().Unix(), // origin timestamp
		max:      uint64(n),         // maximum size
		width:    opt.Width,         // [ key|key|key ]
		density:  opt.Density,       // density pading factor
		shuffler: opt.Shuffler,      // shuffler large cycle
		tracker:  opt.Tracker,       // shuffler cycling tracker
	}

	return kn.sizer(true)
}

// LoadKEON from disk and validate the checksum and signature.
func LoadKEON(path string, kn *KEON) (ok bool) {

	f, err := os.Open(path)
	if err != nil {
		return // bad file
	}
	defer f.Close()

	kn.name = filepath.Base(path)
	return kn.Importer(f)
}

// SaveKEON to disk with checksum and timestamp
func SaveKEON(path string, kn *KEON) (ok bool) {

	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()

	return kn.Exporter(f)
}

/*
keon package level generational functions

	Importer, Exporter
	Packager, Patcher
*/

// Importer reads the header:keon from the io.Reader
func (kn *KEON) Importer(r io.Reader) (ok bool) {

	var header [80]byte
	r.Read(header[:])

	// validate keon signature 0xff01
	if binary.BigEndian.Uint64(header[:8]) != KeonSignature {
		return
	}

	// read the header before processing the payload because we
	// can abort when we are attempting to load the same object
	// and can be detected by testing CHECKSUM values
	if binary.BigEndian.Uint64(header[8:16]) == kn.Checksum() {
		return
	}

	// configure keon settings from header metadata
	if len(kn.name) == 0 {
		kn.name = "stream"
	}
	kn.origin = int64(binary.BigEndian.Uint64(header[16:24]))
	kn.count = binary.BigEndian.Uint64(header[24:32])
	kn.max = binary.BigEndian.Uint64(header[32:40])
	kn.depth = binary.BigEndian.Uint64(header[40:48])
	kn.width = binary.BigEndian.Uint64(header[48:56])
	kn.density = binary.BigEndian.Uint64(header[56:64])
	kn.shuffler = binary.BigEndian.Uint64(header[64:72])
	kn.tracker = int(binary.BigEndian.Uint64(header[72:]))
	kn.sizer(false)

	var b [8]byte
	var n int
	var err error
	for i := range kn.key {
		n, err = r.Read(b[:])
		if n != 8 || err != nil {
			break
		}
		kn.key[i] = binary.BigEndian.Uint64(b[:])
	}

	// validate the header and object CHECKSUM match
	if binary.BigEndian.Uint64(header[8:16]) != kn.Checksum() {
		return
	}

	return true
}

// Exporter writes the header:keon to the io.Writer
func (kn *KEON) Exporter(w io.Writer) (ok bool) {

	// write the header
	var n int
	var err error
	var b [8]byte
	for _, v := range []uint64{
		KeonSignature, kn.Checksum(), uint64(time.Now().Unix()),
		kn.count, kn.max, kn.depth, kn.width, kn.density, kn.shuffler, uint64(kn.tracker),
	} {
		binary.BigEndian.PutUint64(b[:], v)
		n, err = w.Write(b[:])
		if n != 8 || err != nil {
			return
		}
	}

	// write the data
	for i := uint64(0); i < uint64(len(kn.key)); i++ {
		binary.BigEndian.PutUint64(b[:], kn.key[i])
		n, err = w.Write(b[:])
		if n != 8 || err != nil {
			return
		}
	}

	return true
}

// Write a disk image
func (kn *KEON) Write(path string) bool {
	f, err := os.Open(path)
	if err == nil {
		defer f.Close()
		defer kn.Exporter(f)
	}
	return false
}

// Packager exports a patch data package excluding empty buckets
// for the designated action (1:insert, 0:remove)
//
//	action: 1 insert (0xff21)
//	action: 0 remove (0xff11)
func (kn *KEON) Packager(w io.Writer, action int) {

	// write the patch header
	var n int
	var err error
	var b [8]byte
	for _, v := range []uint64{
		KeonSignature | uint64(action+1)<<4, kn.Checksum(), uint64(time.Now().Unix()), kn.count,
	} {
		binary.BigEndian.PutUint64(b[:], v)
		n, err = w.Write(b[:])
		if n != 8 || err != nil {
			return
		}
	}

	// write the patch data
	for i := uint64(0); i < uint64(len(kn.key)); i++ {
		if kn.key[i] == 0 {
			continue
		}
		binary.BigEndian.PutUint64(b[:], kn.key[i])
		n, err = w.Write(b[:])
		if n != 8 || err != nil {
			return
		}
	}

}

// Patcher applies a patch data package and updates the origin
func (kn *KEON) Patcher(r io.Reader) (info struct {
	Signature uint64
	Checksum  uint64
	Origin    uint64
	Count     uint64
}) {

	var header [32]byte
	r.Read(header[:])

	info.Signature = binary.BigEndian.Uint64(header[:8])
	info.Checksum = binary.BigEndian.Uint64(header[8:16])
	info.Origin = binary.BigEndian.Uint64(header[16:24])
	info.Count = binary.BigEndian.Uint64(header[24:32])
	kn.origin = int64(info.Origin)

	// detect action by signature
	switch info.Signature {
	case KeonSignature | 1<<4: // 0xff11 remove

		remove := kn.patchRemove()
		var b [8]byte
		var n int
		var err error
		for i := 0; i < int(info.Count); i++ {
			n, err = r.Read(b[:])
			if n != 8 || err != nil {
				break
			}
			remove(b[:])
		}

	case KeonSignature | 2<<4: // 0xff21 insert

		insert := kn.patchInsert(true) // allow overwrites
		var b [8]byte
		var n int
		var err error
		for i := 0; i < int(info.Count); i++ {
			n, err = r.Read(b[:])
			if n != 8 || err != nil {
				break
			}
			insert(b[:])
		}
	}

	return
}

/*

	KEON utility and information methods
		sizer, Checksum, Origin
		Len, Cap, Ratio, Ident

*/

// sizer configures KEON.key slice based on size requirement and density factor
func (kn *KEON) sizer(calculate bool) *KEON {

	if calculate {
		kn.depth = kn.max / kn.width                   // calculate depth
		if kn.depth*kn.width < kn.max || kn.max == 0 { // ensure space requirements
			kn.depth++
		}
		kn.depth += (kn.depth * kn.density) / 1000 // add density factor padding space
	}
	kn.key = make([]uint64, kn.depth*kn.width)

	return kn
}

// Checksum generates an order independant numeric
// using the KEON key; empty buckets have no impact
func (kn *KEON) Checksum() (checksum uint64) {
	for i := range kn.key {
		checksum ^= kn.key[i] // XOR
	}
	return checksum
}

// Origin timestamp of the *KEVA
//
//	this represents the creation or the base disk image loaded
func (kn *KEON) Origin() int64 { return kn.origin }

// Len is number of current entries.
func (kn *KEON) Len() uint64 { return kn.count }

// Cap is max capacity of *KEON.
func (kn *KEON) Cap() uint64 { return kn.max }

// Ratio is fill ratio of *KEON.
func (kn *KEON) Ratio() uint64 {
	if kn.max == 0 {
		return 0
	}
	return kn.count * 100 / kn.max
}

// calculate target index locations using the current key hash via XOR with prime mixing
func (kn *KEON) calculate(idx *[4]uint64) {
	// idx[0] holds hash of key
	idx[1] = kn.width * (idx[0] % kn.depth)
	idx[2] = kn.width * ((idx[0] ^ 11400714785074694791) % kn.depth) // prime1 11400714785074694791
	idx[3] = kn.width * ((idx[0] ^ 9650029242287828579) % kn.depth)  // prime4 9650029242287828579
}

/*

	KEON primary management methods
		Lookup, Remove, Insert

*/

// Lookup key in *KEON.
func (kn *KEON) Lookup() func(key []byte) (ok bool) {

	var idx [4]uint64 // key,index locations
	var n, i, j uint64

	return func(key []byte) bool {

		idx[0] = xxhash.Sum(key)
		kn.calculate(&idx)

		for i = 1; i < 4; i++ {
			for j = 0; j < kn.width; j++ {
				n = idx[i] + j
				if kn.key[n] == idx[0] {
					return true
				}
			}
		}

		return false
	}
}

// Remove key from *KEON.
//
//	Ok    key is valid
//	Exist found in table
func (kn *KEON) Remove() func([]byte) struct{ Ok, Exist bool } { return kn.remove(xxhash.Sum) }
func (kn *KEON) patchRemove() func([]byte) struct{ Ok, Exist bool } {
	return kn.remove(func(raw []byte) uint64 { return binary.BigEndian.Uint64(raw) })
}

func (kn *KEON) remove(encoder func([]byte) uint64) func(key []byte) struct{ Ok, Exist bool } {

	var idx [4]uint64  // index locations + key
	var n, i, j uint64 // counters

	return func(key []byte) (item struct{ Ok, Exist bool }) {

		idx[0] = encoder(key) // eg. xxhash.Sum(key)
		kn.calculate(&idx)
		item.Ok = idx[0] != 0

		for i = 1; i < 4; i++ {
			for j = 0; j < kn.width; j++ {
				n = idx[i] + j
				if kn.key[n] == idx[0] {
					if j != kn.width-1 {
						// [ a b c ] -> [ a b 0 ] remove c by clear tail
						// [ a b c ] -> [ a c 0 ] remove b by c << 1 and clear tail
						// [ a b c ] -> [ b c 0 ] remove a by b,c << 1 and clear tail
						copy(kn.key[n:n+kn.width-j], kn.key[n+1:n+kn.width-j]) // shift segment over
					}
					kn.key[n+kn.width-j-1] = 0 // clear tail

					kn.count--
					item.Exist = true
					return
				}
			}
		}

		return
	}
}

// Insert into *KEON.
//
//	boolean for updateable
//	nil to bypass key hasher and use raw [8]byte
//
//	Ok      flag on insert success
//	Exist   flag when already present (or collision)
//	NoSpace flag with at capacity or shuffler failure
func (kn *KEON) Insert(update bool) func([]byte) struct{ Ok, Exist, NoSpace bool } {
	return kn.insert(update, xxhash.Sum)
}
func (kn *KEON) patchInsert(update bool) func([]byte) struct{ Ok, Exist, NoSpace bool } {
	return kn.insert(update, func(raw []byte) uint64 { return binary.BigEndian.Uint64(raw) })
}
func (kn *KEON) insert(update bool, encoder func([]byte) uint64) func([]byte) struct{ Ok, Exist, NoSpace bool } {

	var idx [4]uint64  // index locations + key
	var n, i, j uint64 // counters
	var ix, jx uint64  // counters
	var empty bool     // flags

	var node [2]uint64
	var cyclic map[[2]uint64]uint8

	return func(key []byte) (item struct{ Ok, Exist, NoSpace bool }) {

		if kn.count == kn.max {
			item.NoSpace = true
			return
		}

		idx[0] = encoder(key) // xxhash.Sum(key)
		kn.calculate(&idx)
		empty = false

		// verify not already present in any target index location
		// and record the next empty insertion point during check
		for i = 1; i < 4; i++ {
			for j = 0; j < kn.width; j++ {
				n = idx[i] + j
				if kn.key[n] == 0 {
					if !empty {
						empty = true
						ix, jx = i, j
					}
					continue
				}
				if kn.key[n] == idx[0] {
					item.Exist = true
					item.Ok = update
					return
				}
			}
		}

		// insert the new key at ix,jx target
		if empty {
			kn.key[idx[ix]+jx] = idx[0]
			kn.count++
			item.Ok = true
			return
		}

		// shuffle and displace a random key to allow for current key insertion using an
		// outer loop composed of many short inner shuffles that succeed or fail quickly
		// to cycle over many alternate short path swaps that abort on cyclic movements
		var random [8]byte
		for jx = 0; jx < kn.shuffler; jx++ { // 500 cycles of up to ~17*3 smaller swap tracks
			cyclic = make(map[[2]uint64]uint8, kn.tracker) // cyclic movement tracker

			for {
				rand.Read(random[:])
				ix = idx[1+binary.BigEndian.Uint64(random[:8])%3] // select random altenate index to use
				n = ix + (uint64(random[7]) % kn.width)           // select random key to displace and swap
				node = [2]uint64{ix, idx[0]}                      // cyclic node generation; index and key
				cyclic[node]++                                    // cyclic recurrent node movement tracking
				if cyclic[node] > uint8(kn.width) || len(cyclic) == kn.tracker {
					break // reset cyclic path tracker and jump tracks by picking a new random index
					// and key to displace as this gives us about ~2x faster performance boost by
					// locating an open slot faster for some reason
				}

				kn.key[n], idx[0] = idx[0], kn.key[n] // swap keys to displace the key
				kn.calculate(&idx)                    // generate index set for displaced key

				for i = 1; i < 4; i++ { // attempt to insert displaced key in alternate location
					if idx[i] != ix { // avoid the common index between key and displaced key
						for j = 0; j < kn.width; j++ {
							n = idx[i] + j
							if kn.key[n] == 0 { // a new location for displaced key
								kn.key[n] = idx[0]
								kn.count++
								item.Ok = true
								return
							}
						}
					}
				}

			}
		}

		// ran out of key shuffle options
		item.NoSpace = true
		return
	}
}
