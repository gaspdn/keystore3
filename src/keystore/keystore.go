package keystore

import (
	"io"
	"bucket"
)

func (k Keystore) Insert(key []Key, more ...int) ([]int, error) { // @@@ watch for forkfanout/forkwidth
	uniq := make([]int, len(key))
	shorthands := 0
	//	stopped := make([]bool, len(key))

	if len(more) == 1 {
		shorthands = more[0]
	} else if len(more) > 1 {
		panic("Keystore Insert: too many args")
	}

	/*
	 * must make sure that all dimension keys supplied exhaust simultaneously.
	 */
	if shorthands > 1 { // get rid of this in real code
		shorthands = 2
	}
	return uniq, nil
}

/*
 * Delete discards blocks at tail end of the deleted string trimming keys and restarting downtree search until it hits block that keeps the last fork.
 * within that block, it deletes one branch.
 */
func (k Keystore) Delete(key []Key, more ...[]bool) error {
	exact := []bool(nil)

	if len(more) == 1 {
		exact = more[0]
	} else if len(more) > 1 {
		panic("Keystore Delete: too many args")
	}

	if exact[0] { // get rid of this in real code
		return nil
	}
	return nil
}

/*
 * Replace oldkey with newkey.
 * if exact[d] is specified, oldkey[d] needs to exactly match the stored key dimension d.
 * Regardless of exact, replace is only guaranteed to succeed when, in each dimension d, oldkey[d] is the only
 * key matching S, where S is the max length string matching both heads of oldkey[d] and newkey[d].
 */
func (k Keystore) Replace(oldkey []Key, newkey []Key, more ...[]bool) error {
	exact := []bool(nil)

	if len(more) == 1 {
		exact = more[0]
	} else if len(more) > 1 {
		panic("Keystore Replace: too many args")
	}

	if exact[0] { // get rid of this in real code
		return nil
	}
	return nil
}

func (k Keystore) Retrieve(key []Key, more ...interface{}) ([][]Key, error) {
	shorthand := false
	matchlen := map[int]int(nil)
	reverse := []bool(nil)
	maxkeys := 1

	for i := range more {
		switch v := more[i].(type) {
		case bool:
			shorthand = v
		case map[int]int:
			matchlen = v
		case []bool:
			reverse = v
		case int:
			maxkeys = v
		default:
			i = 99
		}
		if i > 2 || (shorthand && i > 0) {
			panic("Keystore retrieve: invalid args")
		}
	}

	curkey := make([]Key, len(key))

	ret := make([][]Key, maxkeys)
	/*
	 * if shorthand, follow each dimension to full key length and then follow shorthand path;
	 *    at each fork, walk the branch with shorthand match bits equal to total unmatched bits, (if such exists)
	 *    checking that the bits actually match. if matching does not produce a single branch, fail.
	 *    return a single key.
	 * In all other cases:
	 *    Walk through keys, and down the tree, pacing bits according to dimension pace,
	 *     until we have collected matchlen bits (consider exact as per interface comment); then:
	 *       if on a string, and on the wrong side of the complete key array, return.
	 *       if on a fork, sort matching entries based on the value, dimension and direction (reverse or not) of each bit/stop.
	 *          recurse through branches in sorted order, upwards on downwards and collect keys
	 *	    (until reaching a string on the wrong side of the complete array or a fork with all entries on the wrong side).
	 * never collect more than maxkeys.
	 */
	//	for i:= range ret {
	//		ret[i] = make([]Key, k.dim) // actually should dynamically append per each []Key returned
	//	}
	ret = append(ret, curkey) // ... like so

	if reverse[0] && shorthand { // get rid of this in real code
		matchlen[0] = maxkeys
	}
	return ret, nil
}

type remcomp struct { // up to last block
	keybit []int // per dimension bit num; Bitlen+1 if seen stop
	rem    remote
	link   bucket.Link // last known link; could be NOLINK
}

type segcomp struct {
	keybit []int // per dimension bit num; Bitlen+1 if seen stop
	segidx int
}

type strcomp struct {
	keybit []int // per dimension bit num; Bitlen+1 if seen stop
	strnum int   // string number in segment
}

type bitcomp struct {
	keybit []int // per dimension bit num; Bitlen+1 if seen stop
	bitnum int   // in last string, bit num where above bits were consumed
}

type forkcomp struct {
	keybit   []int // per dimension bit num; Bitlen+1 if seen stop
	first, n int   // in last fork, first matching and number of matching entries (range might contain non-matching entries if dimension keys do not exhaust together)
}

/*
@@@ downtree should iterate (no recursion) with correct backtracking in cases where retrace backtracks.
*/

type rempath []remcomp
type segpath []segcomp
type strpath []strcomp
type bitpath []bitcomp   // could have 0 or 1 elements
type forkpath []forkcomp // could have 0 or 1 elements; can't have both bitpath and forkpath

type searchstate struct {
	k        *Keystore
	rempath        // blocks
	segpath        // segments in _last_block
	strpath        // strings in _last_ segment
	bitpath        // bit in last string
	forkpath       // ... or in last fork
	forks    []int // segment numbers of forks seen during search (in last block)
}

func (state *searchstate) downtree_prep(key []Key) (startbit uint, stopmap map[uint]uint) {
	var keybit []int

	startbit = 0
	switch {
	case len(state.bitpath) > 0:
		keybit = state.bitpath[0].keybit
	case len(state.forkpath) > 0:
		keybit = state.forkpath[0].keybit
	case len(state.strpath) > 0:
		keybit = state.strpath[len(state.strpath)-1].keybit
	case len(state.segpath) > 0:
		keybit = state.segpath[len(state.segpath)-1].keybit
	case len(state.rempath) > 0:
		keybit = state.rempath[len(state.rempath)-1].keybit
	default:
		keybit = make([]int, 0)
	}
	for d, k := range keybit {
		if key[d].Bitlen < uint(k) {
			k--
			stopmap[uint(d)] = uint(k)
		}
		startbit += uint(k)
	}
	return
	// can later call state.downtree(block, key, matchstop, dim, startbit, stopmap)
}

/*
 * downtree search:
 * follow all dimensions from a given searchstate to ambiguity or exhastion. update searchstate.
 * on return, hold a referenced copy of the block where search was terminated, but not of parent blocks.
 * matchstop[d] specifies whether a stop should be matched at dimension d.
 * return in one of the following conditions:
 *   - we have matched all dimensions where stop was to be matched and at least one other dimension key is exhausted (key exhaustion)
 *   - the next key bit to be matched does not match the next one in the keystore (keystore exhastion)
 *   - we are at a fork and do not have sufficient key bits to match a unique branch (ambiguity) -- in which case state.forkmap
 *     reflects how far the fork can be matched. caller would figure out what to do further.
 */
func (state *searchstate) downtree(b block, key []Key, matchstop []bool, dim Dimpace, startbit uint, stopmap map[uint]uint) *block {
	// startbit and stopmap are eligible args for Dimpace
	return &block{}
}

func modified(bkt bucket.Bucket, bn bucket.Block, link bucket.Link) bool {
	return bkt.Replace(bn, &(bucket.Buf{}), 0, link, false) != nil
}

func (buf *buff) parseblock(compressed bool) *block {
	b, err := demarshall(buf, compressed)
	if err != nil {
		panic("demarshalling error") // @@@ handle/report better
	}
	b.buf = buf
	return b
}

/*
 * retrace a remote pointer path.
 * called with root at state.rempath[0] and a new candidate for a block to traverse at state.rempath[last].
 * attempt to re-parse modified blocks, trim searchstate to last successful point upon failure.
 * when re-parsing, only need to examine segments that have remote pointers and look for the relevant one.
 * return the parsed block with a pointer to the unparsed buff at the end of the (trimmed or completely retraced) path.
 */
func (state *searchstate) retrace() *block {
	pbufs := make([]*bucket.Buf, 0, len(state.rempath))
	lost := false

	defer func(bkt bucket.Bucket, b []*bucket.Buf) {
		if len(b) > 1 {
			bkt.Release(b[:len(b)-1]...)
		}
	}(state.k.Bucket, pbufs)

scan:
	for i := 0; ; i++ {
		buf, link, err := state.k.Bucket.Fetch(state.rempath[i].rem.bn, true)
		if err != nil {
			panic("cannot fetch block") // @@@ handle/report better
		}
		if i == 0 {
			lost = state.rempath[0].link == bucket.NOLINK || modified(state.k.Bucket, state.rempath[0].rem.bn, state.rempath[0].link)
		} else if modified(state.k.Bucket, state.rempath[i-1].rem.bn, state.rempath[i-1].link) {
			// parent has been modified; backtrack
			state.k.Bucket.Release(buf, pbufs[i-1])
			pbufs = pbufs[:i-1]
			i -= 2
			continue scan
		}
		state.rempath[i].link = link
		pbufs = append(pbufs, buf)
		// in the common case we do not return here and are not lost; avoid parsing.
		if i == len(state.rempath)-1 {
			return ((*buff)(buf)).parseblock(state.k.Compressed)
		}
		rn := state.rempath[i+1].rem
		if lost {
			// re-parse block; look for the next pointer on the list; if failed trim state and bail out
			b := ((*buff)(buf)).parseblock(state.k.Compressed)
			for _, seg := range b.seg {
				if lost = !seg.has_remote || seg.r.bn != rn.bn || seg.r.gen != rn.gen; !lost {
					state.rempath[i+1].rem.pos = seg.r.pos
					continue scan
				}
			}
			state.rempath = state.rempath[:i+1]
			state.segpath = state.segpath[:0]
			state.strpath = state.strpath[:0]
			state.bitpath = state.bitpath[:0]
			state.forkpath = state.forkpath[:0]
			state.forks = state.forks[:0]
			return b
		}

		state.rempath[i+1].rem.ReadFrom(io.ReadSeeker(&(newreader((*buff)(buf), 0, rn.pos).revreader)))
		lost = state.rempath[i+1].rem.gen != rn.gen || state.rempath[i+1].rem.bn != rn.bn
	}
}

/*
 * ready to write a subtree:
 * takes path trail to the parent, and block buffer and address at the top of the new subtree.
 * upon success, parent block will be written in place.
 *     it is the caller's responsibility to discard blocks from the old subtree.
 * upon failure, it is the caller's responsibility to free resources before restarting.
 */
