# KVS

* KEON - key only membership testing hash table.

This is a membership tesing hash table. It is similar to map[uint64]bool, but provides faster performance, uses about 1/6th of the RAM, and can be tuned and distrubuted with binary validation.

* KEVA - key:value encodeable hash table database.

This is a key:value hash table that links the key:value unit. It is akin to a map[uint64]uint64, but provides faster performance, uses about 1/6th of the RAM, can be tuned and distributed with binary validation.

* The basic implementation builds a static reference hash table that can be build from a set of data and can be and used locally or distributed; runs in memory.
* To be MRSW would require an external wrapper to implement sync.RWMutex to safeguard existing data integrity operations and prevent data races due to the dynamic nature of the design and internal movement of items.
* Once created, the table is a static sized container space, meaning the max capacity once specified can not be altered without creating a new a new container.

The defaut configuration utilizes a cuckoo style hash table that has been optimized with an internal shuffler that optimizes the table density while providing constant lookup performace expectations and assurances.


```shell
$ kvs testdata/test

  alexa-948061.keon
---------------------------------
checksum   : 1288379988661879870
timestamp  : keon 1725119832
capacity   : 948061
count      : 948061
format     : 317601 x 3
density    : 5 952803 [4742]
shuffler   : 500 x 50

```
---

# Options

Modifying the ```shuffler``` (default 500) adjusts the internal item shuffle internally while outside the internal cyclic track detection shuffler, so a higher value helps gain higher density at longer insert times when nearing capacity. Modifying the ```tracker``` (default 50) monitors movements for cyclic recurrences for specified shuffles as an individual shuffler track with the current bucket width (default 3) for recurrent item movement back to to a prior index location, and when cyclic movement is detected the shuffler aborts to a new trial track with a new randomly selected items. This cyclic tracker has proven to be worth about ~2x performance gain and setting around 17 x (the bucket size) seems to be ideal.

* Shuffler ```(default 500)```
* Tracker ```(default 50)```

Shuffler and Tracker configure the .Insert() methods movement shuffler that makes space for the new items by rotating current items into their alternate hash index location. 

* Density ```(default 25)``` 2.5% padding

The ```density math compaction factor``` is effectively adding a percentage of empty padding spaces while only using integer numerics to calculate it. This impacts insert performance and the default density ```25``` sets the compaction padding factor of +2.5%.

	5  = 0.5%   99.95% 10,000 adds 50 buckets
	10 = 1.0%   99.00% 10,000 adds 100 buckets
	15 = 1.5%   98.50% 10,000 adds 150 buckets
	20 = 2.0%   98.00% 10,000 adsd 200 buckets
	25 = 2.5%   97.50% 10,000 adds 250 buckets
	1000 = 0.0% 100%   10,000 adds 0 buckets; perfect hash attempt

The compaction ```density``` is currently coded to 97.5% (25) with the current defaults as this provides a reasonable trade off between memory, insert performance, and table utilization. The key shuffler and cyclic movement tracker will shuffle a randomly selected item within a scope of smaller cyclic tracks with monitored movements, and this has proven to be adequate up to 99.75% (80) table density on tables of 100MM items. Small tables such those with 1e6 items can be arranged into a minimal-perfect-hash table when table density is configured at (100) which essentially doubles in insertion time an increases the risk of MPH related failures which can be further mitigated with additional tuning of shuffler and tracker. On failuare, give the random nature of the internal movement, it be may possible to simply rebuild to find an alternate solution.  


* Width ```(default 3)```

This specifies the number of buckets per index arraged logically like in a row/bucket pattern:

  ```shell 
  key|key|key
  key|key|key
  ...
  key|key|key
  ```

## density and shuffler considerations

The size requirement and performance tuning needs to consider the voulme of data, table density, and format. To determine optimal settings, tuning tests will need to be performed.

Insert 10MM entires with padding factor of 2.5% (97.5% density). Memory requirement 78.201mb.

```shell
=== RUN   TestOption
    kvs_test.go:89: insert 5.312242291s @density=25
    kvs_test.go:98: lookup 1.065102708s
    kvs_test.go:100: stats 10000000 10000000 100
--- PASS: TestOption (6.38s)
```

Insert 10MM entries with padding facor of 0.04% (99.96% density) Memory requirement 76.324mb. bytes.
```shell
2024/08/16 18:54:34 3346667 10040001 4 10000000
    kvs_test.go:89: insert 20.696769125s @density=4
    kvs_test.go:98: lookup 1.066863625s
    kvs_test.go:100: stats 10000000 10000000 100
--- PASS: TestOption (21.76s)
```

Insert 10MM entries with padding facotr of 0.03% (99.97% density) with default shuffler of 500 this fails, however when increasing the Shuffer to 5000 the table is successfully created at 2x the time requirement for 1000 fewer padding bucket (8k) gain.
```shell
=== RUN   TestOption
    kvs_test.go:89: insert 43.8051215s @density=3
    kvs_test.go:98: lookup 1.151619667s
    kvs_test.go:100: stats 10000000 10000000 100
--- PASS: TestOption (44.96s)
```

Insert 10MM entries with padding factor of 0.03% (99.97% density) with default Shuffler:500 and Width:5 is successful and 4x faster with slight impact on lookup time due to table with. In effect, instead of 3x3=9 possible locations, the possible locatations shift from 3x5=15 (a gain of 40% more possible options).
```shell
=== RUN   TestOption
    kvs_test.go:89: insert 7.168545125s @density=3
    kvs_test.go:98: lookup 1.250907083s
    kvs_test.go:100: stats 10000000 10000000 100
--- PASS: TestOption (8.42s)
```

As the density increases the creation/insert time increases as the system moves toward a perfect hash table solution for the table data. The is a balance between space utiliztion, width, vs time of insertion, all these parameters can be tuned for the use case, as shown above.

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

## Keon MPH (very close)
kvs.NewKEON(size, &kvs.Option{Density: 2})

```shell
go test -v -run Option
=== RUN   TestOption
    kvs_test.go:89: insert 578.899334ms @density=2
    kvs_test.go:98: lookup 43.508791ms
    kvs_test.go:100: stats 1000000 1000000 100
--- PASS: TestOption (0.62s)
```

* 1.7MM insert/sec @99.98% density and width:5
* 23MM lookup/sec.
* 1e6 items in 7.63mb ram; 99.98% perfect hash.


## Keva
Default Options 

```shell
go test -v -run KEVA
=== RUN   TestKEVA
    kvs_test.go:58: insert 573.638458ms
    kvs_test.go:67: lookup 50.265083ms
    kvs_test.go:69: stats 1000000 1000000 100
--- PASS: TestKEVA (0.62s)
```
* 1.7MM insert/sec @97.5% density
* 20MM lookup/sec.
* 1e6 items in 15.26mb ram.


---

Scaling factor for concurrent go routine readers has been observed to approximately 1.5x per CPU, so in theory a 4 core machine could support 6 concurrent go routines with a theoretical rate of reads of approximately 138MM/sec accessing a KEON or 114MM/sec accessing a KEVA.

# MRSW

Adding a sync.RWMutex around the Insert, Lookup, and Remove methods for concurrent Read/Write access has been observered to impose an approximate 10% performace hit when placed in the overlay management layer as shown.

```golang

	size := uint64(1000000)
	kn := kvs.NewKEON(size, nil)
	insert := kn.Insert(false)
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
