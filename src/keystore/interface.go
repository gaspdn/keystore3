package keystore

import "bucket"

type Keyelem uint8 // must be unsigned

const Keyelembits = uint((^Keyelem(0)^(^Keyelem(0)>>1))>>57&64 |
	(^Keyelem(0)^(^Keyelem(0)>>1))>>26&32 |
	(^Keyelem(0)^(^Keyelem(0)>>1))>>11&16 |
	(^Keyelem(0)^(^Keyelem(0)>>1))>>4&8)

type Key struct {
	Bitlen uint
	Bits   []Keyelem
}

// @@@ should delete and replace take a shorthand arg?

type KeyStore interface {
	/*
	 * input:
	 *   key[d] is the key for dimension d
	 *   shorthands (optional) is minumum total # of bits for a shorthand.
	 *      Given the minimum per dimension, it can be computed by Shorthandlen.
	 *      No shorthands are generated if shorthands == 0 or is missing.
	 * output:
	 *   uniq[d] = position of first unique bit (with no common prefix) in dimension d.
	 *      if shorthands are used, this is also the shorthand length for the given dimension.
	 *      when a key is a substring of a previously stored key, uniq[n] will be set to the dimension key length
	 *        (to include the "stop"), e.g.: A 1d store already contains "aa";
	 *           storing "a" sets uniq[0] to 8, meaning bits 0:7 are common, and (non-existent) bit 8 is unique.
	 */
	Insert(key []Key, shorthands ...int) (uniq []int, err error)

	/*
	 * if exact[d] (optional), delete keys exactly matching key[], or with key[] prefix in dimensions not setting exact[].
	 * not guaranteed to succeed unless all non-exact keys are paced such that they exhaust simultaneously.
	 */
	Delete(key []Key, exact ...[]bool) error

	/*
	 * Replace oldkey with newkey.
	 * if exact[d] (optional) is specified, oldkey[d] needs to exactly match the stored key dimension d.
	 * Regardless of exact, replace is only guaranteed to succeed when, in each dimension d, oldkey[d] is the only
	 * key matching S, where S is the max length string matching both heads of oldkey[d] and newkey[d].
	 */
	Replace(key []Key, withkey []Key, exact ...[]bool) error

	/*
	 * Retrieve(key []Key, shorthand bool, matchlen map[int] int, reverse []bool, maxkeys int) ([][]Key, error)
	 * key is the mandatory. other arguments are optional and can be supplied at any order.
	 * matchlen specifies minimum # of bits to match per dimension
	 *   entire dimension key is matched if matchlen == nil or matchlen[d] is not set.
	 *   exact match for the dimension (including stop) is required if matchlen[d] > Key[d].Bitlen.
	 *   There is no "exact" argument since it overlaps in functionality with matchlen
	 *      (can't have both exact[d] and (matchlen[d] < Key[d].Bitlen))
	 * reverse[d] specifies backwards search in dimension d.
	 * shorthand specifies shorthand match.
	 * maxkeys specifies max number of keys to return; omitting maxkeys means all keys; - use with caution!
	 * in each dimension d, only keys lexicographically >= (or <= if reverse[d]) key[d] will be returned.
	 * if shorthand is set, maxkeys is assumed 1; reverse and matchlen should not be supplied.
	 */
	Retrieve(key []Key, moreargs ...interface{}) ([][]Key, error)
}

/*
 * Given a bit number and a map of dimension key lengths,
 * return the dimension of the bit argument and number of subsequent bits that are in the same dimension.
 */
type Dimpace func(uint, map[uint]uint) (uint, uint)

type Keystore struct {
	Dimpace
	Bucket     bucket.Bucket
	Root       bucket.Block
	Bufsize    int // must match that of underlying bucket
	Compressed bool
	forkfanout uint
	forkwidth  uint
}
