package kvs_test

import (
	"bufio"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/zxdev/kvs"
)

func TestKEON(t *testing.T) {

	size := uint64(10000000)
	var opt = &kvs.Option{Density: 25, Width: 3, Shuffler: 1000} // 97.50% starting density factor
	kn := kvs.NewKEON(size, opt)
	insert := kn.Insert(false)
	lookup := kn.Lookup()

	t0 := time.Now()
	for i := uint64(0); i < size; i++ {
		if !insert([]byte{byte(i % 255), byte(i % 126), byte(i % 235), byte(i % 254), byte(i % 249), byte(i % 197), byte(i % 17), byte(i % 99)}).Ok {
			t.Log("insert failure", i)
			t.FailNow()
		}
	}
	t.Log("insert", time.Since(t0))

	t0 = time.Now()
	for i := uint64(0); i < size; i++ {
		if !lookup([]byte{byte(i % 255), byte(i % 126), byte(i % 235), byte(i % 254), byte(i % 249), byte(i % 197), byte(i % 17), byte(i % 99)}) {
			t.Log("lookup failure", i)
			t.FailNow()
		}
	}
	t.Log("lookup", time.Since(t0))

	t.Log("stats", kn.Len(), kn.Cap(), kn.Ratio())

}

func TestKEVA(t *testing.T) {

	size := uint64(10000000)

	kn := kvs.NewKEVA(size, nil)
	insert := kn.Insert(false) // no update
	lookup := kn.Lookup()

	t0 := time.Now()
	for i := uint64(0); i < size; i++ {
		if item := insert([]byte{byte(i % 255), byte(i % 127), byte(i % 235), byte(i % 254), byte(i % 249), byte(i % 197), byte(i % 17), byte(i % 99)}, i); !item.Ok || item.Exist || item.NoSpace {
			t.Log("insert failure", i)
			t.FailNow()
		}
	}
	t.Log("insert", time.Since(t0))

	for i := uint64(0); i < size; i++ {
		if item := lookup([]byte{byte(i % 255), byte(i % 127), byte(i % 235), byte(i % 254), byte(i % 249), byte(i % 197), byte(i % 17), byte(i % 99)}); !item.Ok || item.Value != i {
			t.Log("lookup failure", i, item)
			t.FailNow()
		}
	}

	t.Log("stats", kn.Len(), kn.Cap(), kn.Ratio())

}

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

	var size = uint64(1000000)
	var opt = &kvs.Option{Density: 1, Width: 4, Shuffler: 1500} // 99.98% starting density factor
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

func TestOption(t *testing.T) {

	size := uint64(1000000)
	var density uint64 = 2 // 0.02% padding factor

	kn := kvs.NewKEON(size, &kvs.Option{Density: density, Shuffler: 500, Width: 5})
	insert := kn.Insert(false)
	lookup := kn.Lookup()

	t0 := time.Now()
	for i := uint64(0); i < size; i++ {
		if insert([]byte{byte(i % 255), byte(i % 128), byte(i % 235), byte(i % 254), byte(i % 249), byte(i % 197), byte(i % 17), byte(i % 99)}).NoSpace {
			t.Log("insert failure", i)
			t.FailNow()
		}
	}
	t.Log("insert", time.Since(t0), "@density=", density)

	t0 = time.Now()
	for i := uint64(0); i < size; i++ {
		if !lookup([]byte{byte(i % 255), byte(i % 128), byte(i % 235), byte(i % 254), byte(i % 249), byte(i % 197), byte(i % 17), byte(i % 99)}) {
			t.Log("lookup failure", i)
			t.FailNow()
		}
	}
	t.Log("lookup", time.Since(t0))

	t.Log("stats", kn.Len(), kn.Cap(), kn.Ratio())

}

func TestInfo(t *testing.T) {

	size := uint64(1000000)

	kn := kvs.NewKEON(size, nil)
	insert := kn.Insert(false)

	for i := uint64(0); i < size; i++ {
		if !insert([]byte{byte(i % 255), byte(i % 126), byte(i % 235), byte(i % 254), byte(i % 249), byte(i % 197), byte(i % 17), byte(i % 99)}).Ok {
			t.Log("insert failure", i)
			t.FailNow()
		}
	}

	t.Log("stats", kn.Len(), kn.Cap(), kn.Ratio())

	kn.Write("sandbox/test.keon")

	info := kvs.Info("sandbox/test.keon")
	t.Log(info.Checksum, info.Count, info.Max, info.Ok)
	os.Remove("sandbox/test.keon")

}

func TestUpdater(t *testing.T) {

	f1 := "sandbox/test1.keon"
	f2 := "sandbox/test2.keon"

	size := uint64(10000)

	kn1 := kvs.NewKEON(size, nil)
	insert1 := kn1.Insert(false)
	for i := uint64(0); i < size; i++ {
		if !insert1([]byte{byte(i % 255), byte(i % 26), byte(i % 235), byte(i % 254), byte(i % 249), byte(i % 197), byte(i % 17), byte(i % 99)}).Ok {
			t.Log("insert failure", i)
			t.FailNow()
		}
	}
	t.Log("stats", kn1.Len(), kn1.Cap(), kn1.Ratio())
	kn1.Write(f1) // checksum 4937243075915459616

	kn2 := kvs.NewKEON(size, nil)
	insert2 := kn2.Insert(false)
	for i := uint64(0); i < size; i++ {
		if !insert2([]byte{byte(i % 195), byte(i % 26), byte(i % 253), byte(i % 254), byte(i % 249), byte(i % 197), byte(i % 17), byte(i % 119)}).Ok {
			t.Log("insert failure", i)
			t.FailNow()
		}
	}
	t.Log("stats", kn2.Len(), kn2.Cap(), kn2.Ratio())
	kn2.Write(f2) // checksum 4927319418560608743

}

var fileDB = "kvs.DB"

func TestFileDB1(t *testing.T) {

	file := "ddump1e6"
	path := filepath.Join(os.Getenv("HOME"), "Development", "_data", file)
	show := false

	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var count uint64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		count++
	}
	t.Log(count)
	f.Seek(0, 0)

	kn := kvs.NewKEON(uint64(count), nil)
	insert := kn.Insert(false)
	lookup := kn.Lookup()

	count = 0
	t0 := time.Now()
	scanner = bufio.NewScanner(f)
	for scanner.Scan() {
		if !insert(scanner.Bytes()).Ok {
			t.Log("keon: lookup insert", count, scanner.Text()) //, xxHash(scanner.Bytes()))
			t.Fail()
		}
		count++
	}
	t.Log(count, time.Since(t0))

	f.Seek(0, 0)
	scanner = bufio.NewScanner(f)
	count = 0
	t0 = time.Now()
	for scanner.Scan() {
		if !lookup(scanner.Bytes()) {
			t.Log("keon: lookup failure", count, scanner.Text())
			t.Fail()
		}
		count++
	}
	t.Log(count, time.Since(t0))

	if show {
		dump := kn.Dump()
		for i := 0; i < len(dump); i += 3 {
			row := []uint64{}
			for j := 0; j < 3; j++ {
				row = append(row, dump[i+j])
			}
			t.Log(i, row)
		}
	}

	t0 = time.Now()
	kn.Write(fileDB)
	t.Log("keon: save", time.Since(t0))
}

func TestFileDB2(t *testing.T) {

	file := "ddump1e6"
	path := filepath.Join(os.Getenv("HOME"), "Development", "_data", file)

	t0 := time.Now()
	kn, ok := kvs.LoadKEON(fileDB)
	if !ok {
		panic("no " + fileDB)
	}
	defer os.Remove(fileDB)
	t.Log("keon: load", time.Since(t0))
	lookup := kn.Lookup()

	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	count := 0
	t0 = time.Now()
	for scanner.Scan() {
		if !lookup(scanner.Bytes()) {
			t.Log("keon: lookup fail", count, scanner.Text())
			t.Fail()
		}
		count++
	}
	t.Log("keon: summary", count, time.Since(t0))

}

func TestAlexa(t *testing.T) {

	file := "alexa-5129"
	path := filepath.Join(os.Getenv("HOME"), "Development", "testdata", file)

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
	t.Log("write sandbox/alexa")
	kv.Write("sandbox/alexa")
	t.Log(kv.Cap(), kv.Len())

	f.Seek(0, 0)
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

func TestAlexaKeon(t *testing.T) {

	file := "alexa-5129"
	path := filepath.Join(os.Getenv("HOME"), "Development", "testdata", file)

	f, _ := os.Open(path)
	defer f.Close()

	t.Log("load sandbox/alexa")
	kv, ok := kvs.LoadKEON("sandbox/alexa.keon")
	if !ok {
		t.Log("load failure")
		t.FailNow()
	}

	t.Log(kv.Cap(), kv.Len())

	lookup := kv.Lookup()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if !lookup(scanner.Bytes()) {
			t.Log("fail -", scanner.Text())
			t.FailNow()
		}
	}

}
