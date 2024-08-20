package kvs

import (
	"bufio"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"

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

// KEON is a set-only hash table structure
type KEON struct {
	path              string   // path to file
	count, max        uint64   // count of items, and max items
	depth, width      uint64   // depth and width to establish hash bucket locations [ key|key|key ]
	density, shuffler uint64   // options
	tracker           int      // options
	hloc              uint64   // idx hash key location in [4]uint64; 3
	key               []uint64 // key slice
}

/*
	keon package level functions
		NewKEON, Info, Load

*/

// NewKEON is the *KEON constructor that accepts optional configuration settings.
func NewKEON(n uint64, opt *Option) *KEON {

	if n == 0 {
		return nil
	}

	if opt == nil {
		opt = new(Option)
	}
	opt.configure()

	var kn = &KEON{
		max:      n,            // maximum size
		width:    opt.Width,    // [ key|key|key ]
		density:  opt.Density,  // density pading factor
		shuffler: opt.Shuffler, // shuffler large cycle
		tracker:  opt.Tracker,  // shuffler cycling tracker
		hloc:     3,            // idx hash location in [4]uint64 for kn.calulate
	}

	return kn.sizer()
}

// LoadKEON a *KEON from disk and the checksum validation status.
func LoadKEON(path string) (*KEON, bool) {

	kn := &KEON{path: path, hloc: 3}
	kn.ext()

	f, err := os.Open(kn.path)
	if err != nil {
		return nil, false // bad file
	}
	defer f.Close()

	kn.path = path
	var checksum uint64
	buf := bufio.NewReader(f)
	fmt.Fscanln(buf, &checksum, &kn.count, &kn.max,
		&kn.depth, &kn.width, &kn.density, &kn.shuffler, &kn.tracker)

	var k [8]byte
	var i uint64
	kn.sizer()
	for {
		_, err = io.ReadFull(buf, k[:])
		if err != nil {
			// io.EOF or io.UnexpectedEOF
			return kn, checksum == kn.validation()
		}
		kn.key[i] = binary.LittleEndian.Uint64(k[:])
		i++
	}

}

/*
	KEON file i/o methods
		keon.Load
		kn.Write, kn.Save

*/

// ext validates the file has a .keon extension
func (kn *KEON) ext() {
	if len(kn.path) == 0 {
		kn.path = "kvs.keon"
	}
	if !strings.HasSuffix(kn.path, ".keon") {
		kn.path += ".keon"
	}
}

// Write *KEON to disk at path.
func (kn *KEON) Write(path string) error {
	kn.path = path
	kn.ext()
	return kn.Save()
}

// Save *KEON to disk at prior Load/Write path
func (kn *KEON) Save() error {

	kn.ext()

	f, err := os.Create(kn.path)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := bufio.NewWriter(f)
	fmt.Fprintln(buf, kn.validation(), kn.count, kn.max,
		kn.depth, kn.width, kn.density, kn.shuffler, kn.tracker)

	var b [8]byte
	for i := uint64(0); i < uint64(len(kn.key)); i++ {
		binary.LittleEndian.PutUint64(b[:], kn.key[i])
		buf.Write(b[:])
	}

	buf.Flush()
	return f.Sync()
}

/*
	KEON utility and information methods
		sizer, validation
		Len, Cap, Ratio, Ident

*/

// sizer configures KEON.key slice based on size requirement and density factor
func (kn *KEON) sizer() *KEON {

	kn.depth = kn.max / kn.width                   // calculate depth
	if kn.depth*kn.width < kn.max || kn.max == 0 { // ensure space requirements
		kn.depth++
	}
	kn.depth += (kn.depth * kn.density) / 1000 // add density factor padding space
	kn.key = make([]uint64, kn.depth*kn.width)

	return kn
}

// validation generates a checksum number for data integrity validation
func (kn *KEON) validation() (checksum uint64) {
	for i := range kn.key {
		checksum = kn.key[i] ^ checksum // XOR
	}
	return checksum
}

// calculate target index locations using the current key hash via XOR with prime mixing
func (kn *KEON) calculate(idx *[4]uint64) {
	// idx[3:kn.hloc] holds hash of key
	idx[0] = kn.width * (idx[kn.hloc] % kn.depth)
	idx[1] = kn.width * ((idx[kn.hloc] ^ 11400714785074694791) % kn.depth) // prime1 11400714785074694791
	idx[2] = kn.width * ((idx[kn.hloc] ^ 9650029242287828579) % kn.depth)  // prime4 9650029242287828579
}

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

/*
	KEON primary management methods
		Lookup, Remove, Insert

*/

// Lookup key in *KEON.
func (kn *KEON) Lookup() func(key []byte) bool {

	var idx [4]uint64 // index locations
	var n, i, j uint64

	return func(key []byte) bool {

		idx[kn.hloc] = xxhash.Sum(key)
		kn.calculate(&idx)

		for i = 0; i < kn.hloc; i++ {
			for j = 0; j < kn.width; j++ {
				n = idx[i] + j
				if kn.key[n] == idx[kn.hloc] {
					return true
				}
			}
		}

		return false
	}
}

// Remove key from *KEON.
func (kn *KEON) Remove() func(key []byte) bool {

	var idx [4]uint64 // index locations
	var n, i, j uint64

	return func(key []byte) bool {

		idx[kn.hloc] = xxhash.Sum((key))
		kn.calculate(&idx)

		for i = 0; i < kn.hloc; i++ {
			for j = 0; j < kn.width; j++ {
				n = idx[i] + j
				if kn.key[n] == idx[kn.hloc] {
					copy(kn.key[n:n+kn.width-j], kn.key[n+1:n+1+kn.width-j]) // shift segment
					kn.key[idx[i]+kn.width-1] = 0                            // wipe tail
					kn.count--
				}
			}
		}

		return false
	}
}

// Insert into *KEON.
//
//	Ok flag on insert success
//	Exist flag when already present (or collision)
//	NoSpace flag with at capacity or shuffler failure
func (kn *KEON) Insert(update bool) func(key []byte) struct{ Ok, Exist, NoSpace bool } {

	var idx [4]uint64 // index locations
	var n, i, j uint64
	var ix, jx uint64
	var empty bool

	var node [2]uint64
	var cyclic map[[2]uint64]uint8

	return func(key []byte) (result struct{ Ok, Exist, NoSpace bool }) {

		if kn.count == kn.max {
			result.NoSpace = true
			return
		}

		idx[kn.hloc] = xxhash.Sum(key)
		kn.calculate(&idx)
		empty = false

		// verify not already present in any target index location
		// and record the next insertion point while checking
		for i = 0; i < kn.hloc; i++ {
			for j = 0; j < kn.width; j++ {
				n = idx[i] + j
				if kn.key[n] == idx[kn.hloc] {
					result.Exist = true
					if !update {
						return
					}
				}
				if kn.key[n] == 0 && !empty {
					empty = true
					ix, jx = i, j
				}
			}
		}

		// insert the new key at ix,jx target
		if empty {
			kn.key[idx[ix]+jx] = idx[kn.hloc]
			kn.count++
			result.Ok = true
			return
		}

		// shuffle and displace a key to allow for current key insertion using an
		// outer loop composed of many short inner shuffles that succeed or fail quickly
		// to cycle over many alternate short path swaps that abort on cyclic movements
		var random [8]byte
		for jx = 0; jx < kn.shuffler; jx++ { // 500 cycles of up to ~17*3 smaller swap tracks
			cyclic = make(map[[2]uint64]uint8, kn.tracker) // cyclic movement tracker

			for {
				rand.Read(random[:])
				ix = idx[binary.LittleEndian.Uint64(random[:8])%kn.hloc] // select random altenate index to use
				n = ix + (uint64(random[7]) % kn.width)                  // select random key to displace and swap
				node = [2]uint64{ix, idx[kn.hloc]}                       // cyclic node generation; index and key
				cyclic[node]++                                           // cyclic recurrent node movement tracking
				if cyclic[node] > uint8(kn.width) || len(cyclic) == kn.tracker {
					break // reset cyclic path tracker and jump tracks by picking a new random index
					// and key to displace as this gives us about ~2x faster performance boost by
					// locating an open slot faster for some reason
				}

				kn.key[n], idx[kn.hloc] = idx[kn.hloc], kn.key[n] // swap keys to displace the key
				kn.calculate(&idx)                                // generate index set for displaced key

				for i = 0; i < kn.hloc; i++ { // attempt to insert displaced key in alternate location
					if idx[i] != ix { // avoid the common index between key and displaced key
						for j = 0; j < kn.width; j++ {
							n = idx[i] + j
							if kn.key[n] == 0 { // a new location for displaced key
								kn.key[n] = idx[kn.hloc]
								kn.count++
								result.Ok = true
								return
							}
						}
					}
				}

			}
		}

		// ran out of key shuffle options
		result.NoSpace = true
		return
	}
}
