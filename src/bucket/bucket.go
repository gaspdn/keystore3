package bucket

type Buf []byte
type Block uint64

const NOBLOCK = ^Block(0)

type Gen uint64
type Link uint64

const NOLINK = Link(0)

/*
 * multiple implementations are planned:
 *  - shared memory based (@@@use open(/dev/shm/xxx) + mmap), with refcounts and copyless buffers + goroutines
 *  - private memory based, copyful, with concurrent goroutines for coalescing
 *  - memory only, primitive -- for debugging
 *  - proxy client/server that enable using any of the above in a separate address space/process/container/host
 */
type Bucket interface {
	/*
	 * Write the contents of Buf to a newly allocated block;
	 * return the written block address and a gen number.
	 * Gen is guaranteed never to re-occur for the same Block.
	 * decrement refcount if decref is set.
	 * an implementation may choose to panic if Keeping a Discarded block.
	 * NOTE: reference counts are for _buffers_ (memory addresses),
	 *   whereas allocation is for _blocks_ (disk addresses).
	 *   a buffer will keep its reference count also after being written to a new block.
	 *   in particular, a referenced anonymous buffer (obtained by Fetch(NOBLOCK)) can be
	 *   assigned a block address and retain its reference when passed to Keep.
	 */
	Keep(b *Buf, decref bool) (Block, Gen, error)

	/*
	 * Read a block from the specified address and increment refcount.
	 * linking enables LL/SC like synchronization:
	 *    subsequent Replace fails if block might have been modified or Discarded since the matching linked Fetch.
	 * Fetching NOBLOCK returns a buffer with no associated block.
	 */
	Fetch(d Block, withlink bool) (*Buf, Link, error)

	/*
	 * write in place: may write a partial block; offset is from beginning of the block.
	 * writing a partial block that is not cached by the keeper may either RMW or fail, at keeper's discretion;
	 *   however if not implementing RMW, the keeper must implement some minimal caching.
	 * if multiple replaces are coalesced, the keeper first fails all replaces with expired links.
	 *   then, linked operations that write disjoint ranges are coalseced,
	 *   together with all unlinked ones, written in a single opeation and return successfully.
	 *   linked replaces with ranges overlapping successful ones will fail.
	 * Replace with a zero length buf can be used to verify that the link is still valid
	 *    without modifying the block, and will not fail subsequent linked Replaces.
	 * decrement refcount if decref is set.
	 */
	Replace(d Block, b *Buf, off uint, l Link, decref bool) error

	/*
	 * decrefs buffers and frees the underlying blocks when refcount drops to 0
	 * (also as a result of a subsequent Release)
	 */
	Discard(d ...Block) error

	/* decrefs the buffers but keeps the stored blocks (unless previously marked as discarded) */
	Release(b ...*Buf) error
}

type Buckette struct {
	Bufsize int
}

func (l Link) Error() string {
	return "block rewritten under our feet" // change to something more meaningful describing l
}

func (b *Buf) Bytes() []byte {
	return []byte(*b)
}
