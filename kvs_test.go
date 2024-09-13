package kvs_test

import (
	"bufio"
	"encoding/binary"
	"os"
	"testing"
	"time"

	"github.com/zxdev/kvs"
	"github.com/zxdev/xxhash"
)

// go test -v -run KEON
func TestKEON(t *testing.T) {

	// 	=== RUN   TestKEON
	//     kvs_test.go:39: insert 5.35s 1.869e+06
	//     kvs_test.go:49: lookup 1.18s 8.439e+06
	//     kvs_test.go:51: opt   &{25 3 500 50}
	//     kvs_test.go:52: stats 10000000 10000000 18446423210106567862
	//     kvs_test.go:62: remove 1.25s 7.971e+06
	//     kvs_test.go:63: stats 0 10000000 0
	// --- PASS: TestKEON (7.80s)

	size := uint64(10000000)           // 10MM
	var opt = &kvs.Option{Density: 25} // 97.50%, default starting density factor
	kn := kvs.NewKEON(size, opt)
	insert := kn.Insert(false)
	lookup := kn.Lookup()
	remove := kn.Remove()

	t0 := time.Now()
	for i := uint64(0); i < size; i++ {
		if !insert([]byte{byte(i % 255), byte(i % 126), byte(i % 235), byte(i % 254), byte(i % 249), byte(i % 197), byte(i % 17), byte(i % 99)}).Ok {
			t.Log("insert failure", i)
			t.FailNow()
		}
	}
	t1 := time.Since(t0)
	t.Logf("insert %.4vs %.4v", t1, float64(size)/t1.Seconds())

	t0 = time.Now()
	for i := uint64(0); i < size; i++ {
		if !lookup([]byte{byte(i % 255), byte(i % 126), byte(i % 235), byte(i % 254), byte(i % 249), byte(i % 197), byte(i % 17), byte(i % 99)}) {
			t.Log("lookup failure", i)
			t.FailNow()
		}
	}
	t1 = time.Since(t0)
	t.Logf("lookup %.4vs %.4v", t1, float64(size)/t1.Seconds())

	t.Log("opt  ", opt)
	t.Log("stats", kn.Len(), kn.Cap(), kn.Checksum())

	t0 = time.Now()
	for i := uint64(0); i < size; i++ {
		if !remove([]byte{byte(i % 255), byte(i % 126), byte(i % 235), byte(i % 254), byte(i % 249), byte(i % 197), byte(i % 17), byte(i % 99)}).Exist {
			t.Log("remove failure", i)
			t.FailNow()
		}
	}
	t1 = time.Since(t0)
	t.Logf("remove %.4vs %.4v", t1, float64(size)/t1.Seconds())
	t.Log("stats", kn.Len(), kn.Cap(), kn.Checksum())

}

// go test -v -run KEVA
func TestKEVA(t *testing.T) {

	// 	=== RUN   TestKEVA
	//     kvs_test.go:76: insert 8.60s 1.162e+06
	//     kvs_test.go:86: lookup 10.1s 9.891e+05
	//     kvs_test.go:87: opt   &{25 3 500 50}
	//     kvs_test.go:88: stats 10000000 10000000 16259743542975096488
	// --- PASS: TestKEVA (10.11s)

	size := uint64(10000000) // 10MM
	var opt = &kvs.Option{}  // 97.50%, default starting density factor
	kn := kvs.NewKEVA(size, opt)
	insert := kn.Insert(false) // no update
	lookup := kn.Lookup()

	t0 := time.Now()
	for i := uint64(0); i < size; i++ {
		if item := insert([]byte{byte(i % 255), byte(i % 127), byte(i % 235), byte(i % 254), byte(i % 249), byte(i % 197), byte(i % 17), byte(i % 99)}, i); !item.Ok || item.Exist || item.NoSpace {
			t.Log("insert failure", i, item)
			t.FailNow()
		}
	}
	t1 := time.Since(t0)
	t.Logf("insert %.4vs %.4v", t1, float64(size)/t1.Seconds())

	for i := uint64(0); i < size; i++ {
		if item := lookup([]byte{byte(i % 255), byte(i % 127), byte(i % 235), byte(i % 254), byte(i % 249), byte(i % 197), byte(i % 17), byte(i % 99)}); !item.Ok || item.Value != i {
			t.Log("lookup failure", i, item)
			t.FailNow()
		}
	}

	t1 = time.Since(t0)
	t.Logf("lookup %.4vs %.4v", t1, float64(size)/t1.Seconds())
	t.Log("opt  ", opt)
	t.Log("stats", kn.Len(), kn.Cap(), kn.Checksum())
}

// go test -v -run Format
func TestFormat(t *testing.T) {

	// 	=== RUN   TestFormat
	// 		kvs_test.go:132: == BUILD ==
	// 78f5654d6efd7dbf 38b81600d8a95c42 824c7e712de3865e
	// 47a91d3e9d51e21e 592a234f2db10146 eadfa4c58bb47dbd
	// 351e21dd92557ab6 28e7c088c6cdfaf0 acc45ce58f964c67
	// d5d3e44b28b8af2f 4998a77f66cbc7a6 64a7b5245e1c752e
	// 817406c49c828bf7 bc2a9da3c158e347 124dcae06e9a7606
	// 6d82b39f1a55d1a2 faf494717f230000 9982dcf3b330ea64
	// 4ab4b72392bd3fa6 9a7a43d8071b1ccd e7f8e92714e1b80d
	// add53ced6ee9cfa2 dbda3b6de1046bb6 fd389a6ac4571746
	// 7661a718233a7663 acfc90fb7c562127 898ee134ffe8df66
	// 053c232042e3b909 cfe8d9d936564c38 3a720afd4278fee5
	// 		kvs_test.go:141: == CLEAR ==
	// 78f5654d6efd7dbf 38b81600d8a95c42 824c7e712de3865e
	// 47a91d3e9d51e21e 592a234f2db10146 eadfa4c58bb47dbd
	// 351e21dd92557ab6 28e7c088c6cdfaf0 acc45ce58f964c67
	// d5d3e44b28b8af2f 4998a77f66cbc7a6 64a7b5245e1c752e
	// 817406c49c828bf7 bc2a9da3c158e347 124dcae06e9a7606
	// 6d82b39f1a55d1a2 faf494717f230000 9982dcf3b330ea64
	// 4ab4b72392bd3fa6 9a7a43d8071b1ccd e7f8e92714e1b80d
	// add53ced6ee9cfa2 fd389a6ac4571746 0000000000000000
	// 7661a718233a7663 acfc90fb7c562127 898ee134ffe8df66
	// 053c232042e3b909 cfe8d9d936564c38 3a720afd4278fee5
	// --- PASS: TestFormat (0.00s)

	size := uint64(30)
	kn := kvs.NewKEON(size, &kvs.Option{Density: 1000}) // perfect hash
	insert := kn.Insert(false)
	remove := kn.Remove()

	var bb [][8]byte
	for i := uint64(0); i < size; i++ {
		bb = append(bb, [8]byte{0, 0, 0, byte(i) + 1, 0, 0, 0, 0})
	}

	t.Log("== BUILD ==")
	for i := range bb {
		if !insert(bb[i][:]).Ok {
			t.Log("insert failure", i)
			t.FailNow()
		}
	}
	kn.Dump()

	t.Log("== CLEAR ==")
	// remove dbda3b6de1046bb6
	if !remove(bb[1][:]).Exist {
		t.Log("remove failure")
		t.FailNow()
	}
	kn.Dump()

}

// go test -v -run Best
func TestBestFit(t *testing.T) {

	// === RUN   TestBestFit Find Solution 99.97% compression
	// 	kvs_test.go:87:  kvs: best for 1000000
	// 	kvs_test.go:106: kvs: generation [1,500,3] failure
	// 	kvs_test.go:106: kvs: generation [2,500,3] failure
	// 	kvs_test.go:116: kvs: build [3,1500,3] 371327/sec
	// 	kvs_test.go:126: kvs: lookup [3,1500,3] 23048910/sec
	// --- PASS: TestBestFit (6.73s)
	// PASS

	// 	=== RUN   TestBestFit Averrage Build 99.90% compression
	//   kvs_test.go:87:  kvs: best for 1000000
	//   kvs_test.go:116: kvs: build [10,500,3] 1517464/sec
	//   kvs_test.go:126: kvs: lookup [10,500,3] 23476796/sec
	// --- PASS: TestBestFit (0.70s)
	// PASS

	// 	=== RUN   TestBestFit Fast Build 97.50% compression
	//  kvs_test.go:94:  kvs: best for 1000000
	//  kvs_test.go:123: kvs: build [25,500,3] 2615753/sec
	//  kvs_test.go:133: kvs: lookup [25,500,3] 23209732/sec
	// --- PASS: TestBestFit (0.43s)
	// PASS

	// 	=== RUN   TestBestFit 99.99% compression
	//     kvs_test.go:100: kvs: best for 1000000
	//     kvs_test.go:129: kvs: build [1,1500,4] 516023/sec
	//     kvs_test.go:139: kvs: lookup [1,1500,4] 23236315/sec
	// --- PASS: TestBestFit (1.98s)
	// PASS

	var size = uint64(1000000)                                 // 1MM
	var opt = &kvs.Option{Density: 1, Width: 3, Shuffler: 500} // 99.98% starting density factor
	var t0 time.Time
	var t1 time.Duration
	t.Log("kvs: best for", size)

	for fail := true; fail; {
		kn := kvs.NewKEON(size, opt)
		insert := kn.Insert(false) // no update
		lookup := kn.Lookup()
		t0 = time.Now()
		for i := uint64(0); i < size; i++ {
			if fail = insert([]byte{byte(i % 255), byte(i % 127), byte(i % 235), byte(i % 254), byte(i % 249), byte(i % 197), byte(i % 127), byte(i % 99)}).NoSpace; fail {
				break
			}
		}
		t1 = time.Since(t0)

		switch {
		case opt.Density > 25:
			t.Log("kvs: generation failure")
			return
		case fail:
			t.Logf("kvs: generation [%d,%d,%d] failure", opt.Density, opt.Shuffler, opt.Width)
			opt.Density += 1        // increase density padding factor
			if opt.Density%3 == 0 { // bump shuffler every 0.03% increase
				opt.Shuffler += 1000
			}
			if opt.Density > 10 && opt.Width < 5 {
				opt.Width++ // bump width
			}
			continue
		default:
			t.Logf("kvs: build [%d,%d,%d] %.0f/sec", opt.Density, opt.Shuffler, opt.Width, float64(kn.Len())/t1.Seconds())

			t0 = time.Now()
			for i := uint64(0); i < size; i++ {
				if !lookup([]byte{byte(i % 255), byte(i % 127), byte(i % 235), byte(i % 254), byte(i % 249), byte(i % 197), byte(i % 127), byte(i % 99)}) {
					t.Log("lookup failure")
					t.FailNow()
				}
			}
			t1 = time.Since(t0)
			t.Logf("kvs: lookup [%d,%d,%d] %.0f/sec", opt.Density, opt.Shuffler, opt.Width, float64(kn.Len())/t1.Seconds())

			return
		}
	}

}

// go test -v -run Info
func TestInfo(t *testing.T) {

	// 	=== RUN   TestInfo
	//     kvs_test.go:212: checksum 14029875144168523218 1000000 1000000
	// --- PASS: TestInfo (0.42s)
	// PASS

	size := uint64(1000000)
	os.Mkdir("sandbox", 0755)
	k1 := "sandbox/info.keon"
	defer os.Remove(k1)

	kn := kvs.NewKEON(size, nil)
	insert := kn.Insert(false)
	for i := uint64(0); i < size; i++ {
		if !insert([]byte{byte(i % 255), byte(i % 126), byte(i % 235), byte(i % 254), byte(i % 249), byte(i % 197), byte(i % 17), byte(i % 99)}).Ok {
			t.Log("insert failure", i)
			t.FailNow()
		}
	}
	kn.Write(k1)

	// info.keon
	// ---------------------------------
	// checksum   : 14029875144168523218
	// timestamp  : keon 1725993287
	// capacity   : 1000000
	// count      : 1000000
	// format     : 341667 x 3
	// density    : 25 1025001 [25001]
	// shuffler   : 500 x 50
	// memory     : 7.82 MiB

	info := kvs.Info(k1)
	if !info.Ok {
		t.Log("info: failed")
		t.FailNow()
	}
	t.Log("checksum", info.Checksum, info.Count, info.Max)

}

// go test -v -run Export
func TestExport(t *testing.T) {

	// 	=== RUN   TestExport
	// kvs_test.go:233: == export ==
	// kvs_test.go:314: [31 195 84 228 51 63 96 131]
	// kvs_test.go:314: [121 46 62 219 47 170 189 234]
	// 	...
	// kvs_test.go:314: [221 72 254 87 73 92 145 245]
	// kvs_test.go:314: [220 28 228 214 84 204 103 159]
	// --- PASS: TestExport (0.00s)
	// PASS

	size := uint64(50)
	kn := kvs.NewKEON(size, nil)
	insert := kn.Insert(false)
	for i := uint64(0); i < size; i++ {
		if !insert([]byte{byte(i % 255), byte(i % 126), byte(i % 235), byte(i % 254), byte(i % 249), byte(i % 197), byte(i % 17), byte(i % 99)}).Ok {
			t.Log("insert failure", i)
			t.FailNow()
		}
	}

	var count int
	t.Log("== export ==")
	next := kn.Export()
	for b := [8]byte{}; next(&b); {
		count++
		t.Log(count, b)
	}

}

// go test -v -run Alexa
func TestAlexa(t *testing.T) {

	// 	=== RUN   TestAlexa
	//     kvs_test.go:285: insert 9.76075ms
	//     kvs_test.go:286: write sandbox/alexa
	//     kvs_test.go:288: 5129 5129
	//     kvs_test.go:302: lookup 334.458µs
	// --- PASS: TestAlexa (0.02s)

	path := "testdata/alexa-5129"
	alexa := "sandbox/alexa"
	defer os.Remove(alexa)

	f, _ := os.Open(path)
	defer f.Close()

	var count uint64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		count++
	}

	f.Seek(0, 0)

	kv := kvs.NewKEON(count, &kvs.Option{Density: 3})
	insert := kv.Insert(false)
	scanner = bufio.NewScanner(f)
	t0 := time.Now()
	for scanner.Scan() {
		if r := insert(scanner.Bytes()); !r.Ok {
			t.Log(scanner.Text(), r)
			t.FailNow()
		}
	}
	t.Log("insert", time.Since(t0))
	t.Log("write", alexa)
	kv.Write(alexa)
	t.Log(kv.Cap(), kv.Len())

	f.Seek(0, 0)

	kv, _ = kvs.LoadKEON(alexa)
	lookup := kv.Lookup()
	scanner = bufio.NewScanner(f)
	t0 = time.Now()
	for scanner.Scan() {
		if !lookup(scanner.Bytes()) {
			t.Log("fail -", scanner.Text())
			t.FailNow()
		}
	}
	t.Log("lookup", time.Since(t0))

}

// go test -v -run Update
func TestUpdate(t *testing.T) {

	// 	=== RUN   TestUpdate
	//     kvs_test.go:322: stats 50 100 4853821449203168217
	//     kvs_test.go:335: == UPDATE ==
	//     kvs_test.go:366: stats 75 100 179520822399970802
	//     kvs_test.go:367: exist 25
	// --- PASS: TestUpdate (0.01s)

	f1 := "sandbox/test1.keon"
	f2 := "sandbox/test2.keon"

	size := uint64(100)

	kn1 := kvs.NewKEON(size, nil)
	insert1 := kn1.Insert(false) // disallow updates
	for i := uint64(0); i < size/2; i++ {
		if !insert1([]byte{byte(i + 1), 0, 0, 0, 0, 0, 0, 0}).Ok {
			t.Log("insert failure", i)
			t.FailNow()
		}
	}
	t.Log("stats", kn1.Len(), kn1.Cap(), kn1.Checksum())
	kn1.Write(f1)

	// test1
	// ---------------------------------
	// checksum   : 4853821449203168217
	// timestamp  : keon 1726000636
	// capacity   : 100
	// count      : 50
	// format     : 34 x 3
	// density    : 25 102 [52]
	// shuffler   : 500 x 50

	t.Log("== UPDATE ==")
	var exist int
	var ok bool
	kn1, ok = kvs.LoadKEON(f1)
	if !ok {
		t.Log("failed", f1)
		t.FailNow()
	}
	insert1 = kn1.Insert(true) // allow updates, flags r.Exist
	// half already exist, other half are new
	for i := uint64(25); i < size-25; i++ {
		r := insert1([]byte{byte(i + 1), 0, 0, 0, 0, 0, 0, 0})
		if r.Exist {
			exist++
		}
		if !r.Ok {
			t.Log(i, r)
			t.FailNow()
		}
	}

	// test2
	// ---------------------------------
	// checksum   : 179520822399970802
	// timestamp  : keon 1726000636
	// capacity   : 100
	// count      : 75
	// format     : 34 x 3
	// density    : 25 102 [27]
	// shuffler   : 500 x 50

	t.Log("stats", kn1.Len(), kn1.Cap(), kn1.Checksum())
	t.Log("exist", exist)
	kn1.Write(f2) // checksum 4927319418560608743

}

func TestRaw(t *testing.T) {

	// 	=== RUN   TestMerge
	//     kvs_test.go:597: stats 50 100 4853821449203168217
	//     kvs_test.go:623: stats 50 100 5373087661334553314
	//     kvs_test.go:626: == MERGE KEONS ==
	//     kvs_test.go:634: 25 4693302597208153643
	//     kvs_test.go:646: stats 75 100 179520822399970802
	// --- PASS: TestMerge (0.02s)

	f1 := "sandbox/test1.keon"
	f2 := "sandbox/test2.keon"
	defer os.Remove(f2)
	defer os.Remove(f1)

	size := uint64(100)

	kn1 := kvs.NewKEON(size, nil)
	insert1 := kn1.Insert(false) // disallow updates
	for i := uint64(0); i < size/2; i++ {
		if !insert1([]byte{byte(i + 1), 0, 0, 0, 0, 0, 0, 0}).Ok {
			t.Log("insert failure", i)
			t.FailNow()
		}
	}
	t.Log("stats", kn1.Len(), kn1.Cap(), kn1.Checksum())
	kn1.Write(f1)

	// test1
	// ---------------------------------
	// checksum   : 4853821449203168217
	// timestamp  : keon 1726000636
	// capacity   : 100
	// count      : 50
	// format     : 34 x 3
	// density    : 25 102 [52]
	// shuffler   : 500 x 50

	t.Log("== RAW UPDATE ==")
	var exist int
	var ok bool
	kn1, ok = kvs.LoadKEON(f1)
	if !ok {
		t.Log("failed", f1)
		t.FailNow()
	}

	// generate a raw update set; a raw
	// set requires [8]byte segments
	var patch [][8]byte
	var b [8]byte
	for i := uint64(25); i < size-25; i++ {
		// generate the hashed bytes for the patch, store as uint64 set
		binary.BigEndian.PutUint64(b[:], xxhash.Sum([]byte{byte(i + 1), 0, 0, 0, 0, 0, 0, 0}))
		patch = append(patch, b)
	}

	insert1 = kn1.RawInsert(true) // signals raw with updateable keys
	// half already exists, other half are new
	for i := range patch {
		r := insert1(patch[i][:])
		if r.Exist {
			exist++
		}
		if !r.Ok {
			t.Log(i, r)
			t.FailNow()
		}
	}

	// test2
	// ---------------------------------
	// checksum   : 179520822399970802
	// timestamp  : keon 1726000636
	// capacity   : 100
	// count      : 75
	// format     : 34 x 3
	// density    : 25 102 [27]
	// shuffler   : 500 x 50

	t.Log("stats", kn1.Len(), kn1.Cap(), kn1.Checksum())
	t.Log("exist", exist)
	kn1.Write(f2)

	t.Log("== RAW REMOVE ==")
	remove1 := kn1.RawRemove()
	if !remove1(patch[0][:]).Exist {
		t.Log("remove raw[0] failure")
		t.FailNow()
	}
	t.Log("stats", kn1.Len(), kn1.Cap(), kn1.Checksum())
	kn1.Write(f2)

	// test2
	// ---------------------------------
	// checksum   : 2082901812776260895
	// timestamp  : keon 1726012661
	// capacity   : 100
	// count      : 74
	// format     : 34 x 3
	// density    : 25 102 [28]
	// shuffler   : 500 x 50

}

func TestMerge(t *testing.T) {

	// 	=== RUN   TestMerge
	//     kvs_test.go:597: stats 50 100 4853821449203168217
	//     kvs_test.go:623: stats 50 100 5373087661334553314
	//     kvs_test.go:626: == MERGE KEONS ==
	//     kvs_test.go:645: stats 75 100 179520822399970802
	// --- PASS: TestMerge (0.02s)

	f1 := "sandbox/test1.keon"
	f2 := "sandbox/test2.keon"
	f3 := "sandbox/test3.keon"
	defer os.Remove(f1)
	defer os.Remove(f2)
	defer os.Remove(f3)

	size := uint64(100)

	kn1 := kvs.NewKEON(size, nil)
	insert1 := kn1.Insert(false) // disallow updates
	for i := uint64(0); i < size/2; i++ {
		if !insert1([]byte{byte(i + 1), 0, 0, 0, 0, 0, 0, 0}).Ok {
			t.Log("insert failure", i)
			t.FailNow()
		}
	}

	// test1
	// ---------------------------------
	// checksum   : 4853821449203168217
	// timestamp  : keon 1726000636
	// capacity   : 100
	// count      : 50
	// format     : 34 x 3
	// density    : 25 102 [52]
	// shuffler   : 500 x 50

	t.Log("stats", kn1.Len(), kn1.Cap(), kn1.Checksum())
	kn1.Write(f1)

	// create new KEON where half the items
	// already exist and other half are new

	kn2 := kvs.NewKEON(size, nil)
	insert2 := kn2.Insert(false)
	for i := uint64(25); i < size-25; i++ {
		r := insert2([]byte{byte(i + 1), 0, 0, 0, 0, 0, 0, 0})
		if !r.Ok {
			t.Log(i, r)
			t.FailNow()
		}
	}

	// test2
	// ---------------------------------
	// checksum   : 5373087661334553314
	// timestamp  : keon 1726158571
	// capacity   : 100
	// count      : 50
	// format     : 34 x 3
	// density    : 25 102 [52]
	// shuffler   : 500 x 50

	t.Log("stats", kn2.Len(), kn2.Cap(), kn2.Checksum())
	kn2.Write(f2)

	t.Log("== MERGE KEONS ==")

	r := kvs.MergeKEON(kn1, f2, nil)
	if !r.Ok {
		t.Log(r)
		t.Log("merge failure")
		t.FailNow()
	}
	t.Log(r.Items, r.Checksum)

	// test3
	// ---------------------------------
	// checksum   : 179520822399970802
	// timestamp  : keon 1726190895
	// capacity   : 100
	// count      : 75
	// format     : 34 x 3
	// density    : 25 102 [27]
	// shuffler   : 500 x 50

	t.Log("stats", kn1.Len(), kn1.Cap(), kn1.Checksum())
	kn1.Write(f3)
}
