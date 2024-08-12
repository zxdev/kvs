package kvs

// width is the static representative size of the index
// key for the hash bucket location block [ key|key|key ]
//const width = 3

// Option provides settings to alter the default system setting
// to further optimize for density and compaction at the expense
// on insert performance pushing toward a minimum perfect hash table
type Option struct {

	// Shuffler and Tracker configure the .Insert() method movement shuffler that
	// makes space by rotating items into their alternate hash location
	Shuffler uint64 // 500 large shuffle cycles
	Tracker  int    // 50 (~17*width) cyclic detection tracks

	// Density represents the compaction pad factor which can be
	// 	affected by the Shuffler,Tracker settings
	// 	20 = 95.00% +10,000/20 adds 5.00% +500 pad
	// 	40 = 97.50% +10,000/40 adds 2.50% +250 pad
	// 	80 = 99.75% +10,000/80 adds 1.25% +125 pad
	// 	100 = 100%; perfect hash table
	Density uint64 // 40 default

	// Width is the static representative size of the index
	// key for the hash bucket location block [ key|key|key ]
	Width uint64
}

// confgure sets default assurances
func (c *Option) configure() {

	if c.Shuffler == 0 {
		c.Shuffler = 500
		c.Tracker = 50
	}

	if c.Tracker == 0 {
		c.Tracker = 50
	}

	if c.Density == 0 {
		c.Density = 40
	}

	if c.Width == 0 {
		c.Width = 3
	}
}
