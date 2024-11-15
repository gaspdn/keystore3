package bucket_mem

import . "bucket"

type Bucket_mem Buckette

/*
 * a full implementation will keep refcounts on buffers; this one merely creates copies.
 * also, in a full implementation we may want to force a context switch and a rendezvous just before issuing a write, to enable more coalescing.
 */

func (k Bucket_mem) Keep(b *Buf, decref bool) (Block, Gen, error) {
	return Block(0), Gen(0), nil
}

func (k Bucket_mem) Fetch(d Block, withlink bool) (*Buf, Link, error) {
	b := Buf(make([]byte, k.Bufsize))
	l := NOLINK

	return &b, l, nil
}

func (k Bucket_mem) Replace(d Block, b *Buf, off uint, l Link, decref bool) error {
	if l != NOLINK {
		return l // this is how to fail a store-linked; other errors are handled according to their types.
	}
	return nil
}

func (k Bucket_mem) Discard(d ...Block) error {
	return nil
}

func (k Bucket_mem) Release(b ...*Buf) error {
	return nil
}
