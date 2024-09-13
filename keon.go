package kvs

import (
	"bufio"
	"crypto/rand"
	"encoding/binary"
	"io"
	"os"
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
		hloc:     3,            // idx hash location in [4]uint64 for kn.calulate
		max:      n,            // maximum size
		width:    opt.Width,    // [ key|key|key ]
		density:  opt.Density,  // density pading factor
		shuffler: opt.Shuffler, // shuffler large cycle
		tracker:  opt.Tracker,  // shuffler cycling tracker
	}

	return kn.sizer(true)
}

// LoadKEON a *KEON from disk and validate the checksum and signature.
func LoadKEON(path string) (*KEON, bool) {

	f, err := os.Open(path)
	if err != nil {
		return nil, false // bad file
	}
	defer f.Close()

	var buf = bufio.NewReader(f)
	var signature, checksum, index uint64
	var header [80]byte
	var k [8]byte
	io.ReadFull(buf, header[:])
	signature = binary.BigEndian.Uint64(header[:8])
	checksum = binary.BigEndian.Uint64(header[8:16])
	// timestamp = binary.BigEndian.Uint64(header[16:24])
	kn := &KEON{
		path:     path,
		hloc:     3,
		count:    binary.BigEndian.Uint64(header[24:32]),
		max:      binary.BigEndian.Uint64(header[32:40]),
		depth:    binary.BigEndian.Uint64(header[40:48]),
		width:    binary.BigEndian.Uint64(header[48:56]),
		density:  binary.BigEndian.Uint64(header[56:64]),
		shuffler: binary.BigEndian.Uint64(header[64:72]),
		tracker:  int(binary.BigEndian.Uint64(header[72:])),
	}
	kn.sizer(false)

	for {
		_, err = io.ReadFull(buf, k[:])
		if err != nil {
			// io.EOF or io.UnexpectedEOF
			return kn, signature == 0xff01 && checksum == kn.Checksum()
		}
		kn.key[index] = binary.BigEndian.Uint64(k[:])
		index++
	}

}

/*
	KEON file i/o methods
		keon.Load
		kn.Write, kn.Save

*/

// Write *KEON to disk at path.
func (kn *KEON) Write(path string) error {
	kn.path = path
	return kn.Save()
}

// Save *KEON to disk at prior Load/Write path
func (kn *KEON) Save() error {

	if len(kn.path) == 0 {
		kn.path = "kvs.keon"
	}

	f, err := os.Create(kn.path)
	if err != nil {
		return err
	}
	defer f.Close()

	// 0xff01 is the keon header signature type
	var buf = bufio.NewWriter(f)
	var b [8]byte
	for _, v := range []uint64{
		0xff01, kn.Checksum(), uint64(time.Now().Unix()),
		kn.count, kn.max, kn.depth, kn.width, kn.density, kn.shuffler, uint64(kn.tracker),
	} {
		binary.BigEndian.PutUint64(b[:], v)
		buf.Write(b[:])
	}

	for i := uint64(0); i < uint64(len(kn.key)); i++ {
		binary.BigEndian.PutUint64(b[:], kn.key[i])
		buf.Write(b[:])
	}

	buf.Flush()
	return f.Sync()
}

// Export all bucket hash data excluding empty buckets
func (kn *KEON) Export() func(b *[8]byte) bool {
	var item int
	var checksum uint64
	return func(b *[8]byte) bool {
		for item < len(kn.key) {
			if kn.key[item] == 0 {
				item++
				continue
			}
			binary.BigEndian.PutUint64(b[:], kn.key[item])
			checksum ^= kn.key[item]
			item++
			return true
		}
		return false
	}
}

/*
	KEON utility and information methods
		sizer, Checksum
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
//
//	Ok    key is valid
//	Exist found in table
func (kn *KEON) Remove() func([]byte) struct{ Ok, Exist bool } { return kn.remove(xxhash.Sum) }
func (kn *KEON) RawRemove() func([]byte) struct{ Ok, Exist bool } {
	return kn.remove(func(raw []byte) uint64 { return binary.BigEndian.Uint64(raw) })
}

func (kn *KEON) remove(encoder func([]byte) uint64) func(key []byte) struct{ Ok, Exist bool } {

	var idx [4]uint64  // index locations + key
	var n, i, j uint64 // counters

	return func(key []byte) (item struct{ Ok, Exist bool }) {

		idx[kn.hloc] = encoder(key) // eg. xxhash.Sum(key)
		kn.calculate(&idx)
		item.Ok = idx[kn.hloc] != 0

		for i = 0; i < kn.hloc; i++ {
			for j = 0; j < kn.width; j++ {
				n = idx[i] + j
				if kn.key[n] == idx[kn.hloc] {
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
func (kn *KEON) RawInsert(update bool) func([]byte) struct{ Ok, Exist, NoSpace bool } {
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

		idx[kn.hloc] = encoder(key) // xxhash.Sum(key)
		kn.calculate(&idx)
		empty = false

		// verify not already present in any target index location
		// and record the next empty insertion point during check
		for i = 0; i < kn.hloc; i++ {
			for j = 0; j < kn.width; j++ {
				n = idx[i] + j
				if kn.key[n] == 0 {
					if !empty {
						empty = true
						ix, jx = i, j
					}
					continue
				}
				if kn.key[n] == idx[kn.hloc] {
					item.Exist = true
					item.Ok = update
					return
				}
			}
		}

		// insert the new key at ix,jx target
		if empty {
			kn.key[idx[ix]+jx] = idx[kn.hloc]
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
				ix = idx[binary.BigEndian.Uint64(random[:8])%kn.hloc] // select random altenate index to use
				n = ix + (uint64(random[7]) % kn.width)               // select random key to displace and swap
				node = [2]uint64{ix, idx[kn.hloc]}                    // cyclic node generation; index and key
				cyclic[node]++                                        // cyclic recurrent node movement tracking
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
