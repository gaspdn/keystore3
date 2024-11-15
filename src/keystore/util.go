package keystore

import (
	"math/bits"
	"bucket"
)

func forkfan(bs int, compressed bool) (uint, uint, uint) { // return max fork fanout, bit distance (including stops) and ptrsize
	p3, p2 := 1, 1

	bs -= 2 + 1 // block header + fork header
	if compressed {
		bs -= 18 // gzip header
	}
	maxsegbits := bits.Len(uint(bs/3 - 1)) //  3 is minimum size of a segment; @@@ handle corner case where compression would decrease this

	for bd := uint(1); ; bd++ {
		w := 2*p3 + p2
		s := (w*bits.Len(bd+1) + 7) / 8     // bits for shorthands
		if 18*w+s+(maxsegbits*w+7)/8 > bs { // 18 is size of a segment holding 1 remote ptr
			return uint(2*p3/3 + p2/2), bd - 1, uint(maxsegbits)
		}
		p3 *= 3
		p2 *= 2
	}
}

func (k Key) Substr(from, length uint) Key {
	if (from + length) > k.Bitlen {
		if from >= k.Bitlen {
			return Key{}
		}
		length = k.Bitlen - from
	}

	shift := from & (Keyelembits - 1)
	ret := Key{Bitlen: length, Bits: make([]Keyelem, ((length+shift)+(Keyelembits-1))/Keyelembits)}
	ks := k.Bits[from/Keyelembits:]

	for i := range ret.Bits {
		ret.Bits[i] = (ks[i] << shift)
		if shift > 0 {
			ret.Bits[i] |= ks[i+1] >> (Keyelembits - shift)
		}
	}

	return ret
}

/*
 * Given a minimum shorthand length in each dimension, return the minimum total shorthand length.
 * for meaningful results, zero members should be omitted from the shorthands map.
 */
func Shorthandlen(dimpace func(uint) uint, shorthands map[uint]uint) uint {
	short := make(map[uint]uint)

	for i, v := range shorthands {
		short[i] = v
	}

	for i := uint(0); ; i++ {
		dim := dimpace(i)
		if short[dim] > 1 {
			short[dim]--
		} else if delete(short, dim); len(short) == 0 {
			return i + 1
		}
	}
}

/*
 * Takes no args, rather normalizes and checks values preassigned to k.
 * Should be called once, before any insert/delete/replace/retrieve ops are attempted.
 * @@@ is this really needed??? @@@
 */
func (k Keystore) Init() {
	if k.Bucket == nil {
		panic("uninitialized bucket")
	}
	if k.Root == bucket.NOBLOCK {
		panic("unknown root")
	}
	k.forkfanout, k.forkwidth, _ = forkfan(k.Bufsize, k.Compressed)
}
