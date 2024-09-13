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

  test
---------------------------------
checksum   : 1288379988661879870
timestamp  : keon 1725119832
capacity   : 948061
count      : 948061
format     : 317601 x 3
density    : 5 952803 [4742]
shuffler   : 500 x 50
memory     : 7.27 MiB

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

The size requirement and performance tuning needs to consider the volume of data, table density, and format. To determine optimal settings, tuning tests will need to be performed, ```TestBestFit``` provides a basic tuning formula approach for a best compression build.

# Examples

With 10 million record trils, as shown below, the following code performance was observed on an Apple 2023 M2 Pro Mac Mini with 16GB ram.

* insert 97.50% ~ 1.7MM/sec degrading to ~200k/sec at 99.97% table fill density.
* lookup 97.50% ~ 7.32MM/sec degrading to 5.85MM/sec based on table column architecure.

## samples

Insert 10MM entires with padding factor of 2.5% (97.5% density).

```shell
=== RUN   TestKEON
    kvs_test.go:38: insert 5.9s 1.69e+06
    kvs_test.go:48: lookup 1.3s 7.32e+06
    kvs_test.go:50: opt   &{25 3 500 50}
    kvs_test.go:51: stats 10000000 10000000 18446423210106567862
--- PASS: TestKEON (7.29s)
```

Insert 10MM entries with padding facor of 0.04% (99.96% density).
```shell
=== RUN   TestKEON
    kvs_test.go:38: insert 22.s 4.42e+05
    kvs_test.go:48: lookup 1.4s 7.02e+06
    kvs_test.go:50: opt   &{4 3 500 50}
    kvs_test.go:51: stats 10000000 10000000 18446423210106567862
--- PASS: TestKEON (24.07s)
```

Insert 10MM entries with padding facotr of 0.03% (99.97% density) with default shuffler of 500 this fails, however when increasing the Shuffer to 5000 the table is successfully created at 2x the time requirement (22s vs 48s) but requiring fewer pad buckets for an overall memory savings vs the previous 99.96% density.
```shell
=== RUN   TestKEON
    kvs_test.go:38: insert 48.s 2.05e+05
    kvs_test.go:48: lookup 1.3s 7.15e+06
    kvs_test.go:50: opt   &{3 3 5000 51}
    kvs_test.go:51: stats 10000000 10000000 18446423210106567862
```

Insert 10MM entries with padding factor of 0.03% (99.97% density) and width:4 is successful and almost 5x faster with an impact on lookup performance due to table width. In effect, instead of 3x3=9 possible locations, the possible locatations shift from 3x4=12 providing 25% additional internal addressing, 3x5=15 provides an additional 40% internal addressing at the expense of lookup performance.
```shell
=== RUN   TestKEON
    kvs_test.go:38: insert 11.s 8.88e+05
    kvs_test.go:48: lookup 1.5s 6.46e+06
    kvs_test.go:50: opt   &{3 4 500 50}
    kvs_test.go:51: stats 10000000 10000000 18446423210106567862
--- PASS: TestKEON (12.81s)

=== RUN   TestKEON
    kvs_test.go:38: insert 8.7s 1.14e+06
    kvs_test.go:48: lookup 1.7s 5.85e+06
    kvs_test.go:50: opt   &{3 5 500 50}
    kvs_test.go:51: stats 10000000 10000000 18446423210106567862
--- PASS: TestKEON (10.49s)
```

As the density increases the creation/insert time increases as the system moves toward or approaches a perfect hash table solution for the given table data. There is a balance between space utiliztion, width, shuffle cycles vs time of insertion, all these parameters can be tuned for the specif use case, as shown above.

Notice in the above examples the checksum value of ```18446423210106567862``` is consistent across all table formats and density factors. The checksum in not order dependant, the checksum is item dependant, meaning regardless of the table format configurations, the same set of data will ALWAYS generate the same checksum regardless of where the item is physically located inside the structure. This provides an assurance that the expected items are present somewhere in the table. 

---

```golang
insert := kv.Insert(bool) 
  ...
result := insert(key) // result reports three conditions
  result.Ok
  result.Exist
  result.NoSpace
```

* Ok, insert was successful
* Exist, key already exists 
  * update flag
  * collision flag
* NoSpace
  * At max capacity, when kn.Count() == kn.Max()
  * Shuffer failure, when kn.Count() < kn.Max()

## considerations

The current design does not implement a rollback feature for ```result.NoSpace```. The shuffler is random, so a random entery, not the current key, has been ejected from the data. For this reason, the table must be rebuilt from source. This generally means the current table architecure needs to be adjusted to allow more shuffle cycles to seek for a solution and/or altering the density and the table architecture.

---

Scaling factor for concurrent go routine readers has been observed to approximately 1.5x per CPU, so in theory a 4 core machine could support 6 concurrent go routines.

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

# Export

The internal content of the KVS object can be exported, however be aware that the export will consist of the raw internal data.

When using an export to rebuild or resize a data object the ```RawInsert``` method allows the table to be rebuilt without computing the hash, which is why it is provided. 

```golang
  
  next := kn.Export()
	for b := [8]byte{}; next(&b); {
		w.Write(b[:])
	}

  var kn := kvs.NewKEON(size,&opt)
  var insert = kn.RawInsert(false)
  var scanner = bufio.NewScanner(reader)
  for scanner.Scan() {
    if !insert(scanner.Bytes()).Ok {
      return
    }
  }

```

To resize a KVS object simply export the data to a file or buffer and create a new KVS container object and use the ```RawInsert(bool)``` method as shown above. The internal structure and where items can be found is based on the KVS object format that was/is establised at the creation time of the KVS object.

# Merge KVS Objects

While any regular file can be used to add or remove items using the applicable ```Insert(bool)``` methods, it is possible to create smaller update files that can be configured to add, update, or remove itmes. The only requirement is that the KVS objects be of the same type and that there is space available in the primary KVS object to handle the new items. A composite checksum of new impacts will be generated, meaning new items added (not just updated) and items thaere were removed.

```golang

  // r = struct{Ok bool; Invalid bool; NoSpace bool; Items uint64; Checksum uint64}
  r := kvs.MergeKEON(kn1, f2, nil)

```

To apply a patch in real-time with inflight queries the integrator must have coded the design for a MSRW useage (as shown above) or otherwise take the KVS service should be taken offline to prevent data races and placed into a maintence mode, apply the patch updates, then retore the system to an online status. The second approach is more easly handled when the system is part of a cluster. 

If the patch update failes, the origional source fails with an ejected random key; unrecoverable. It is trivial to reload the current state, export the current contents in a raw form, enlarge and/or KVS option for the appropriate size or format using options settngs, and then populate the new data object table using the raw export and then merge the patch data and save the update. Because the checksum is order independent of the key location within the table and the table format, it is trivial to create a new table and generate a a composite checkum for validation of all keys present.
