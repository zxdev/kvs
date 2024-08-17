package kvs

// Option provides settings to alter the default system setting
// to further optimize for density and compaction at the expense
// on insert performance pushing toward a minimum perfect hash table
type Option struct {

	// Density represents the compaction pading factor which can affect
	// 	the perforamance of the Shuffler,Tracker settings
	//  2  = 0.02%  99.98% 10,000 adds 20 buckets
	//	5  = 0.5%   99.95% 10,000 adds 50 buckets
	//	10 = 1.0%   99.00% 10,000 adds 100 buckets
	//	15 = 1.5%   98.50% 10,000 adds 150 buckets
	//	20 = 2.0%   98.00% 10,000 adsd 200 buckets
	// 	25 = 2.5%   97.50% 10,000 adds 250 buckets
	//	1000 = 0.0% 100%   10,000 adds 0 buckets; perfect hash
	Density uint64 // 15 default

	// Width is the representative size of the buckets at index
	// key location for the hash bucket blocks [ key|key|key ]
	Width uint64

	// Shuffler and Tracker configure the .Insert(bool) methods internal dynamic
	// item shuffler that makes space by rotating items into alternate locations
	Shuffler uint64 // 500 shuffle cycles of up Tracker movements
	Tracker  int    // (~17*width) possible movements per shuffle; cyclic detection aborts track

}

// confgure sets default assurances
func (c *Option) configure() {

	if c.Density == 0 {
		c.Density = 25 // 2.5% padding
	}
	if c.Density == 1000 {
		c.Density = 0
	}

	if c.Width == 0 {
		c.Width = 3
	}

	if c.Shuffler == 0 {
		c.Shuffler = 500
		c.Tracker = 50
	}

	if c.Tracker == 0 {
		c.Tracker = 17 * int(c.Width)
	}

}
