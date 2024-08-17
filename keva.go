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
	KE:VA is a cookoo style hash table that distributes and rebalances keys
	and values across alternative index locations for fast lookup. It is
	similar to a map[string]uint64 only faster and more memory efficient and
	offers a density compaction factor as well as file functions and tuning.

	key|key|key
	key|key|key
	...
	key|key|key

	Note: this is not mutex protected so it is not safe to read/write
	at the time same time, but it is safe for concurrent reads.

	kn := kvs.NewKEVA(n,nil)
	insert := kn.Insert()
	lookup := kn.Lookup()
	remove := kn.Remove()
	for ... {
		if !insert(key).Ok {
			// handle error
		}
	}
*/

// KEVA is a set-only hash table structure
type KEVA struct {
	path              string   // path to file
	count, max        uint64   // count of items, and max items
	depth, width      uint64   // depth and width to establish hash bucket locations [ key|key|key ]
	hloc              uint64   // [4]uint64 indexer hash key location
	density, shuffler uint64   // options
	tracker           int      // options
	key               []uint64 // key slice
	value             []uint64 // value slice
}

/*
	KEVA package level functions
		NewKEVA, Info, Load

*/

// NewKEON is the *KEON constructor that accepts optional configuration settings.
func NewKEVA(n uint64, opt *Option) *KEVA {

	if n == 0 {
		return nil
	}

	if opt == nil {
		opt = new(Option)
	}
	opt.configure()

	var kn = &KEVA{
		width:    opt.Width,    // [ key|key|key ]
		hloc:     3,            // static; .calculate(&idx) [4]uint64 hash key index location
		density:  opt.Density,  // density pading factor
		shuffler: opt.Shuffler, // shuffler large cycle
		tracker:  opt.Tracker,  // shuffler cycling tracker
	}

	return kn.sizer(n)
}

// Load a *KEVA from disk and the checksum validation status.
func LoadKEVA(path string) (*KEVA, bool) {

	kn := &KEVA{path: path}
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
		&kn.depth, &kn.density, &kn.shuffler, &kn.tracker)

	var kv [16]byte // uint64x2 k:8 v:8
	var i uint64
	kn.sizer(0) // kn.max configured via header load
	for {
		_, err = io.ReadFull(buf, kv[:])
		if err != nil {
			// io.EOF or io.UnexpectedEOF
			return kn, checksum == kn.validation()
		}
		kn.key[i] = binary.LittleEndian.Uint64(kv[:8])
		kn.value[i] = binary.LittleEndian.Uint64(kv[8:])
		i++
	}

}

/*
	KEVA file i/o methods
		KEVA.Load
		kn.Create, kn.Save

*/

// ext validates the file has a .KEVA extension
func (kn *KEVA) ext() {
	if len(kn.path) == 0 {
		kn.path = "kvs.keva"
	}
	if !strings.HasSuffix(kn.path, ".keva") {
		kn.path += ".keva"
	}
}

// Write *KEVA to disk at path.
func (kn *KEVA) Write(path string) error {
	kn.path = path
	kn.ext()
	return kn.Save()
}

// Save *KEVA to disk at prior Load/Write path
func (kn *KEVA) Save() error {

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
		binary.LittleEndian.PutUint64(b[:], kn.value[i])
		buf.Write(b[:])
	}

	buf.Flush()
	return f.Sync()
}

/*
	KEVA utility and information methods
		sizer
		Len, Cap, Ratio, Ident

*/

// sizer configures KEVA.key slice based on size requirement and density
func (kn *KEVA) sizer(n uint64) *KEVA {

	if n != 0 {
		kn.max = n
	}

	kn.depth = kn.max / kn.width              // calculate depth
	if kn.depth*kn.width < kn.max || n == 0 { // ensure space requirements
		kn.depth++
	}
	kn.depth += (kn.depth * kn.density) / 1000 // add density factor padding space
	kn.key = make([]uint64, kn.depth*kn.width)
	kn.value = make([]uint64, kn.depth*kn.width)

	return kn
}

// validation generates a checksum number for data integrity validation
// using the current keys and their sequential ordering
func (kn *KEVA) validation() (checksum uint64) {
	for i := range kn.key {
		checksum = kn.key[i] ^ checksum // XOR
	}
	return checksum
}

// calculate target index locations using the current key hash via XOR with prime mixing
func (kn *KEVA) calculate(idx *[4]uint64) {
	// idx[3:kn.hloc] holds hash of key
	idx[0] = kn.width * (idx[kn.hloc] % kn.depth)
	idx[1] = kn.width * ((idx[kn.hloc] ^ 11400714785074694791) % kn.depth) // prime1 11400714785074694791
	idx[2] = kn.width * ((idx[kn.hloc] ^ 9650029242287828579) % kn.depth)  // prime4 9650029242287828579
}

// Len is number of current entries.
func (kn *KEVA) Len() uint64 { return kn.count }

// Cap is max capacity of *KEVA.
func (kn *KEVA) Cap() uint64 { return kn.max }

// Ratio is fill ratio of *KEVA.
func (kn *KEVA) Ratio() uint64 {
	if kn.max == 0 {
		return 0
	}
	return kn.count * 100 / kn.max
}

/*
	KEVA primary management methods
		Lookup, Remove, Insert

*/

// Lookup key in *KEVA.
func (kn *KEVA) Lookup() func(key []byte) (item struct {
	Value uint64
	Ok    bool
}) {

	var idx [4]uint64
	var n, i, j uint64

	return func(key []byte) (item struct {
		Value uint64
		Ok    bool
	}) {

		idx[kn.hloc] = xxhash.Sum(key)
		kn.calculate(&idx)

		for i = 0; i < kn.hloc; i++ {
			for j = 0; j < kn.width; j++ {
				n = idx[i] + j
				if kn.key[n] == idx[kn.hloc] {
					item.Value = kn.value[n]
					item.Ok = true
					return
				}
			}
		}
		return
	}
}

// Remove key from *KEVA.
func (kn *KEVA) Remove() func(key []byte) bool {

	var idx [4]uint64
	var n, i, j uint64

	return func(key []byte) bool {

		idx[kn.hloc] = xxhash.Sum((key))
		kn.calculate(&idx)

		for i = 0; i < kn.hloc; i++ {
			for j = 0; j < kn.width; j++ {
				n = idx[i] + j
				if kn.key[n] == idx[kn.hloc] {
					copy(kn.key[n:n+kn.width-j], kn.key[n+1:n+1+kn.width-j])     // shift segment
					kn.key[idx[i]+kn.width-1] = 0                                // wipe tail
					copy(kn.value[n:n+kn.width-j], kn.value[n+1:n+1+kn.width-j]) // shift segment
					kn.value[idx[i]+kn.width-1] = 0                              // wipe tail
					kn.count--
				}
			}
		}

		return false
	}
}

// Insert into *KEVA.
//
//	Ok flag on insert success
//	Exist flag when already present (or collision) or updated with update boolean
//	NoSpace flag with at capacity or shuffler failure
func (kn *KEVA) Insert(update bool) func(key []byte, value uint64) struct{ Ok, Exist, NoSpace bool } {

	var idx [4]uint64
	var n, i, j uint64
	var ix, jx uint64
	var empty bool

	var node [2]uint64
	var cyclic map[[2]uint64]uint8

	return func(key []byte, value uint64) (item struct{ Ok, Exist, NoSpace bool }) {

		item.NoSpace = kn.count == kn.max
		if item.NoSpace {
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
					item.Exist = true
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
			kn.value[idx[ix]+jx] = value
			kn.count++
			item.Ok = true
			return
		}

		// shuffle and displace a key to allow for current key insertion using an
		// outer loop composed of many short inner shuffles that succeed or fail quickly
		// to cycle over many alternate short path swaps that abort on cyclic movements
		var random [8]byte
		var displace = value
		for jx = 0; jx < kn.shuffler; jx++ { // 500 cycles of up to 50 smaller swap tracks
			cyclic = make(map[[2]uint64]uint8, kn.tracker) // cyclic movement tracker

			for {
				rand.Read(random[:])
				ix = idx[binary.LittleEndian.Uint64(random[:8])%kn.hloc] // select random altenate index to use
				n = ix + (uint64(random[7]) % kn.width)                  // select random key to displace and swap
				node = [2]uint64{ix, idx[kn.hloc]}                       // cyclic node generation; index and key
				cyclic[node]++                                           // cyclic recurrent node movement tracking
				if cyclic[node] > uint8(kn.width) || len(cyclic) == kn.tracker {
					break // reset cyclic path tracker and jump shuffle by picking a new random index
					// and key to displace, as this gives us about ~2x faster performance boost by
					// locating an open slot faster rather than cycling back over prior shifts
				}

				kn.key[n], idx[kn.hloc] = idx[kn.hloc], kn.key[n] // swap keys to displace the key
				kn.value[n], displace = displace, kn.value[n]     // swap values to displace the value
				kn.calculate(&idx)                                // generate index set for displaced key

				for i = 0; i < kn.hloc-1; i++ { // attempt to insert displaced key in alternate location
					if idx[i] != ix { // avoid the common index between key and displaced key
						for j = 0; j < kn.width; j++ {
							n = idx[i] + j
							if kn.key[n] == 0 { // a new location for displaced key and value
								kn.key[n] = idx[kn.hloc]
								kn.value[n] = displace
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
