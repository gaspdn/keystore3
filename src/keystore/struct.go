package keystore

import "bucket"

type buff bucket.Buf

type str struct {
	has_stop bool
	bitlen   uint
	align    uint
	bits     []byte
}

type forkelem struct {
	segidx         uint
	shorthandmatch uint
}

type fork struct {
	fe []forkelem // len is # of pointers
}

type remote struct {
	bn  bucket.Block
	gen bucket.Gen
	pos uint // offset of this remote pointer from end of the original block
}

type segment struct {
	has_remote bool
	has_fork   bool
	has_stop   bool
	stralign   uint
	strings    []str
	f          fork
	r          remote
}

type block struct {
	seg     []segment
	address bucket.Block
	buf     *buff
}
