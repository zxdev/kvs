# KVS

* KEON - key only membership testing hash table.

This is a membership tesing hash table. It is similar to map[uint64]bool, but provides faster performance, uses about 1/6th of the RAM, and can be tuned and distrubuted with binary validation.

* KEVA - key:value encodeable hash table database.

This is a key:value hash table that links the key:value unit. It is akin to a map[uint64]uint64, but provides faster performance, uses about 1/6th of the RAM, can be tuned and distributed with binary validation.

* The basic implementation builds a static reference hash table that can be build from a set of data and can be and used locally or distributed; runs in memory.
* To be MRSW would require an external wrapper to implement sync.RW to safeguard operations to prevent data races and should be size according to the intended use case.
* Once created the table size is static.

The defaut configuration utilizes a cuckoo style hash table that has been optimized with and and internal shuffler that optimizes the table density while providing constant lookup performace expectations.

---

# Options

Modifying the ```shuffler``` (default 500) manages the shuffler large loop, so higher track movement trials means higher density at longer insert times when near capacity. Modifying the ```tracker``` (default 50) manages the cyclic movement detection within an individual track trial with the current bucket width (default 3) quickly aborts to a new track trial when cyclice movement has been detects. This cyclic tracker  has proven to be worth about ~2x performance gain.

* Shuffler ```(default 500)```
* Tracker ```(default 50)```

Shuffler and Tracker configure the .Insert() methods movement shuffler that makes space for the new itme by rotating current items into their alternate hash index location. 

* Density ```(default 40)```

The ```density math compaction factor``` is effectively adding a percentage of empty padding spaces while only using integer numerics to calculate it. This impacts insert performance and the default density ```40``` sets the compaction factor to 97.50%.

  20 = 95.00% +depth/20 10,000 adds 500
  40 = 97.50% +depth/40 10,000 adds 250
  80 = 99.75% +depth/80 10,000 adds 125
  100 = 100% 

The compaction ```density``` is currently coded to 97.5% (40) with the current defaults as this provides a reasonable trade off between memory, insert performance, and table utilization. The key shuffler and cyclic movement tracker will shuffle a randomly selected item within a scope of smaller cyclic tracks with monitored movements, and this has proven to be adequate up to 99.75% (80) table density on tables of 100MM items. Small tables such those with 1e6 items can be arranged into a minimal-perfect-hash table when table density is configured at (100) which essentially doubles in insertion time an increases the risk of MPH related failures which can be furter mitigated with additional tuning of shuffler and tracker. On failuare, give the random nature of the internal movement, it be may possible to simply rebuild to find an alternate solution.  


* Width ```(default 3)```

This specifies is a three bucket wide per index arraged logically like in a row/bucket pattern:

  ```shell 
  key|key|key
  key|key|key
  ...
  key|key|key
  ```

---

.Insert(bool) reports three conditions

* Ok, insert was successful
* Exist, key already exists 
  * update 
  * collision 
* NoSpace
  * At max capacity, when kn.Count == kn.Max
  * Shuffer failure, when kn.Count < kn.Max

---

# Example 

With 1e6 records trial, the following code performance was observed on an Apple 2023 M2 Pro Mac Mini with 16GB ram.

## Keon
Default Options

```shell
kvs % go test -v -run KEON
=== RUN   TestKEON
    kvs_test.go:28: insert 360.640125ms
    kvs_test.go:37: lookup 42.709625ms
    kvs_test.go:39: stats 1000000 1000000 100
--- PASS: TestKEON (0.40s)
```

* 2.7MM insert/sec @97.5% density.
* 23MM lookup/sec.
* 1e6 items in 7.63mb ram.

## Keon MPH
kvs.NewKEON(size, &kvs.Option{Density: 100})

```shell
go test -v -run Option
=== RUN   TestOption
    kvs_test.go:88: insert 670.870375ms
    kvs_test.go:97: lookup 44.618333ms
    kvs_test.go:99: stats 1000000 1000000 100
```

* 1.4MM insert/sec @100% density
* 23MM lookup/sec.
* 1e6 items in 7.63mb ram; perfect hash.


## Keva
Default Options 

```shell
kvs % go test -v -run KEVA
=== RUN   TestKEVA
    kvs_test.go:58: insert 582.784458ms
    kvs_test.go:67: lookup 51.280666ms
    kvs_test.go:69: stats 1000000 1000000 100
--- PASS: TestKEVA (0.63s)
```
* 1.7MM insert/sec @97.5% density.
* 19MM lookup/sec.
* 1e6 items in 15.26mb ram.


---

Scaling factor for go routine readers has been observed to approximately 1.5x per CPU, so in theory a 4 core machine could support 6 concurrent go routines with a theoretical rate of 
reads of approximately 138MM/sec with 6 go routine readers accessing a KEON or 114MM/sec accessing a KEVA.

# MRSW

Adding a sync.RWMutex around the Insert, Lookup, and Remove methods for concurrent Read/Write access has been observered to impose an approximate 10% performace hit when placed in the overlay management layer as shown.

```golang

	size := uint64(1000000)
	kn := kvs.NewKEON(size, nil)
	insert := kn.Insert()
	lookup := kn.Lookup()
  var mutex sync.RWMutex

	t0 := time.Now()
	for i := uint64(0); i < size; i++ {
    mutex.Lock()
		if !insert([]byte{byte(i % 255), byte(i % 126), byte(i % 235), byte(i % 254), byte(i % 249), byte(i % 197), byte(i % 17), byte(i % 99)}).Ok {
			t.Log("insert failure", i)
			t.FailNow()
		}
    mutex.Unlock()
	}
	t.Log("insert", time.Since(t0))


	t0 = time.Now()
	for i := uint64(0); i < size; i++ {
    mutex.RLock()
		if !lookup([]byte{byte(i % 255), byte(i % 126), byte(i % 235), byte(i % 254), byte(i % 249), byte(i % 197), byte(i % 17), byte(i % 99)}) {
			t.Log("lookup failure", i)
			t.FailNow()
		}
    mutex.RUnlock()
	}
	t.Log("lookup", time.Since(t0))

	t.Log("stats", kn.Len(), kn.Cap(), kn.Ratio())


```
