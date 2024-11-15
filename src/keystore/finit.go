package keystore

import (
	"bucket"
	"bucket_mem"
)

func (k Keystore) f() { // what a user might do to initialize
	k.Dimpace = func(b uint, stopmap map[uint]uint) (uint, uint) {
		return b % 4, 0 // simple 4d with fixed length keys
	}

	block := bucket.Buf(make([]byte, 512))
	k.Bucket = &bucket_mem.Bucket_mem{Bufsize: len(block)}
	k.Bufsize = len(block)
	k.Root, _, _ = k.Bucket.Keep(&block, false)
	k.Compressed = true
	k.Init()
	k.Bucket.Release(&block) // test only
	k.Retrieve([]Key{})  // test only
}
