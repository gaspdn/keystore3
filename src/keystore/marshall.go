package keystore

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"math/bits"
	"bucket"
)

type segwrap struct {
	segment
	rback    io.ReadSeeker
	wback    io.Writer
	ptrwidth uint
}

type forkwrap struct {
	fork
	ptrwidth uint // from block.segidxbits
}

type blockwrap struct {
	block
	segidxbits uint
	rback      io.ReadSeeker
	wback      io.Writer
}

var ErrCorrupt = errors.New("corrupt data")

// @@@ TODO: use pointers crossover in Writers (ErrShortWrite) as trigger for split when marshalling

func marshall_basic(x interface{}, w io.Writer) (int64, error) {
	switch v := x.(type) {
	case bucket.Block:
		return marshall_basic(uint64(v), w)
	case bucket.Gen:
		return marshall_basic(uint64(v), w)
	case uint64:
		m, _ := marshall_basic(uint32(v&0xffffffff), w)
		n, err := marshall_basic(uint32(v>>32), w)
		return m + n, err
	case uint32:
		m, _ := marshall_basic(uint16(v&0xffff), w)
		n, err := marshall_basic(uint16(v>>16), w)
		return m + n, err
	case uint16:
		n, err := w.Write([]byte{byte(v & 0xff), byte(v >> 8)})
		return int64(n), err
	default:
		panic(fmt.Sprintf("unknown type %T", v))
	}
}

func (r *remote) WriteTo(w io.Writer) (int64, error) {
	n, _ := marshall_basic(r.bn, w)
	m, err := marshall_basic(r.gen, w)
	return m + n, err
}

func (f *forkwrap) WriteTo(w io.Writer) (n int64, err error) {
	bitnum, prevbyte := uint(0), byte(0)
	appendbits := func(v, width uint) (n int) {
		var now uint
		n = 0

		for left := width; left > 0; left -= now {
			if now = 8 - bitnum; now > left {
				now = left
			}
			prevbyte |= byte((v & ((1 << now) - 1)) << bitnum)
			v >>= now
			if bitnum = (bitnum + now) & 7; bitnum == 0 {
				_, err = w.Write([]byte{prevbyte})
				n++
				prevbyte = 0
			}
		}
		return
	}

	n += int64(appendbits(uint(len(f.fe)), f.ptrwidth))
	for _, e := range f.fe {
		n += int64(appendbits(e.segidx, f.ptrwidth))
		if err != nil {
			return
		}
	}
	for _, e := range f.fe {
		n += int64(appendbits(e.shorthandmatch, 4))
		if err != nil {
			return
		}
	}
	if bitnum != 0 {
		_, err = w.Write([]byte{prevbyte})
		n++
	}
	return
}

func (s *str) WriteTo(w io.Writer) (n int64, err error) {
	b := uint16(map[bool]uint{false: 0, true: 1}[s.has_stop] | s.bitlen<<2)
	var k int

	if s.bitlen < 64 {
		w.Write([]byte{byte(b & 0xff)})
		n = 1
	} else {
		n, _ = marshall_basic(b|2, w)
	}
	k, err = w.Write(s.bits)
	n += int64(k)
	return
}

func (sw *segwrap) WriteTo(w io.Writer) (n int64, err error) {
	b := uint8((uint(len(sw.strings))&7)<<5 | (sw.stralign&7)<<2)

	switch {
	case (sw.has_remote && sw.has_fork) || (!sw.has_fork && sw.has_stop):
		panic("corrupt block")
	case sw.has_remote:
		n, _ = sw.r.WriteTo(sw.wback) // back for remote
		b |= 1
	case sw.has_stop:
		b |= 3
	case sw.has_fork:
		b |= 2
	}
	_, err = w.Write([]byte{byte(b), byte(len(sw.strings) >> 3)})
	n = int64(2)
	k := int64(0)
	for _, s := range sw.strings {
		if k, err = s.WriteTo(w); err != nil {
			break
		}
		n += k
	}
	if err == nil && sw.has_fork {
		k, err = (&forkwrap{fork: sw.f, ptrwidth: sw.ptrwidth}).WriteTo(w)
		n += k
	}
	return
}

func (bw *blockwrap) WriteTo(w io.Writer) (n int64, err error) {
	nseg := uint(len(bw.block.seg))
	ptrwidth := uint(bits.Len(nseg))
	n, err = marshall_basic(uint16(nseg), w)
	for i := range bw.block.seg {
		var k int64

		sw := segwrap{wback: bw.wback, segment: bw.block.seg[i], ptrwidth: ptrwidth}
		if k, err = sw.WriteTo(w); err != nil {
			return
		}
		n += k
	}
	return
}

func marshall(b *block, buf *buff, compress bool) (int, error) {
	w := io.Writer(newwriter(buf))
	bw := blockwrap{wback: &w.(*writer).revwriter, block: *b}

	if compress {
		w = gzip.NewWriter(w)
		defer w.(io.WriteCloser).Close()
	}
	n, err := bw.WriteTo(w)
	return int(n), err
}

func demarshall_basic(x interface{}, r io.Reader) (int64, error) {
	switch v := x.(type) {
	case *uint16:
		var s [2]byte

		if _, err := r.Read(s[:]); err != nil {
			return 0, ErrCorrupt
		}
		*v = uint16(s[1])<<8 | uint16(s[0])
		return 2, nil
	case *uint32:
		var hi, lo uint16

		demarshall_basic(&lo, r)
		_, err := demarshall_basic(&hi, r)
		*v = uint32(hi)<<16 | uint32(lo)
		return 4, err
	case *uint64:
		var hi, lo uint32

		demarshall_basic(&lo, r)
		_, err := demarshall_basic(&hi, r)
		*v = uint64(hi)<<32 | uint64(lo)
		return 8, err
	case *bucket.Gen:
		var b uint64
		_, err := demarshall_basic(&b, r)
		*v = bucket.Gen(b)
		return 8, err
	case *bucket.Block:
		var b uint64
		_, err := demarshall_basic(&b, r)
		*v = bucket.Block(b)
		return 8, err
	default:
		panic(fmt.Sprintf("unknown type %T", v))
	}
}

func (rem *remote) ReadFrom(r io.ReadSeeker) (int64, error) {
	pos, _ := r.Seek(0, io.SeekCurrent)
	rem.pos = uint(pos)
	demarshall_basic(&rem.bn, r)
	_, err := demarshall_basic(&rem.gen, r)
	return 16, err
}

func (s *str) ReadFrom(r io.Reader) (int64, error) {
	b, err := r.(io.ByteReader).ReadByte()
	nread := int64(1)

	if err != nil {
		return 0, ErrCorrupt
	}
	bitlen := uint(b) >> 2
	if (b & 2) != 0 {
		if b, err := r.(io.ByteReader).ReadByte(); err != nil {
			return 0, ErrCorrupt
		} else {
			bitlen |= uint(b) << 8
			nread++
		}
	}

	s = &str{bits: make([]byte, (bitlen+s.align+7)/8), bitlen: bitlen, has_stop: ((b & 1) == 1)}
	if n, err := r.Read(s.bits); err != nil {
		return nread + int64(n), ErrCorrupt
	} else {
		return nread + int64(n), nil
	}
}

func (f *forkwrap) ReadFrom(r io.Reader) (int64, error) {
	prevbyte := byte(0)
	var err error

	var getbits func(from, length uint) (uint16, int)
	getbits = func(from, length uint) (uint16, int) {
		n := 0
		if (from & 7) == 0 {
			if prevbyte, err = r.(io.ByteReader).ReadByte(); err != nil {
				err = ErrCorrupt
				return 0, 1
			}
			n = 1
		}
		if i := from / 8; i == (from+length-1)/8 {
			return (uint16(prevbyte) >> (from & 7)) & ((1 << length) - 1), n
		}
		l := 8 - (from & 7)
		k, m := getbits(from, l)
		j, i := getbits(from+l, length-l)
		return k | j<<l, m + n + i
	}

	b, n := getbits(0, f.ptrwidth)
	nread := int64(n)
	i := 0

	for f.fe = make([]forkelem, uint(b)+2); i < len(f.fe) && err == nil; i++ {
		t, n := getbits(f.ptrwidth*uint(i+1), f.ptrwidth)
		f.fe[i].segidx = uint(t)
		nread += int64(n)
	}
	for j := 0; j < len(f.fe) && err == nil; j++ {
		t, n := getbits(f.ptrwidth*uint(i+1)+4*uint(j), 4)
		f.fe[j].shorthandmatch = uint(t)
		nread += int64(n)
	}

	return nread, err
}

func (seg *segwrap) ReadFrom(forw io.Reader) (int64, error) {
	var b [2]byte
	nread := int64(len(b))

	if _, err := forw.Read(b[:]); err != nil {
		return 0, ErrCorrupt
	}
	stralign := uint(b[0]>>2) & 7
	seg.stralign = stralign
	nstr := (uint(b[0]>>5) & 7) | uint(b[1]<<3)
	for seg.strings = make([]str, 0, nstr); nstr > 0; nstr-- {
		s := str{align: stralign}

		if n, err := s.ReadFrom(forw); err != nil {
			return 0, ErrCorrupt
		} else {
			nread += n
		}
		seg.strings = append(seg.strings, s)
		stralign = (stralign + s.bitlen) & 7
	}
	if seg.has_remote = ((b[0] & 3) == 1); seg.has_remote {
		n, err := seg.r.ReadFrom(seg.rback)
		return nread + n, err
	}
	seg.has_stop = ((b[0] & 3) == 3)
	if seg.has_fork = ((b[0] & 3) >= 2); seg.has_fork {
		fw := forkwrap{ptrwidth: seg.ptrwidth}
		n, err := fw.ReadFrom(forw)
		seg.f = fw.fork
		return nread + n, err
	}
	return nread, nil
}

func (bw *blockwrap) ReadFrom(forw io.Reader) (int64, error) {
	var nseg uint16
	nread := int64(0)

	demarshall_basic(&nseg, forw)
	bw.block = block{seg: make([]segment, 0, nseg)}
	bw.segidxbits = uint(bits.Len(uint(nseg)))
	seg := segwrap{rback: bw.rback, ptrwidth: bw.segidxbits}

	for ; nseg > 0; nseg-- {
		if n, err := seg.ReadFrom(forw); err != nil {
			return 0, err
		} else {
			nread += n
		}
		bw.seg = append(bw.seg, seg.segment)
	}
	return nread, nil
}

func demarshall(b *buff, compressed bool) (*block, error) {
	r := io.Reader(newreader(b, 0, 0))
	bw := blockwrap{rback: &r.(*reader).revreader}

	if compressed {
		err := error(nil)
		if r, err = gzip.NewReader(r); err != nil {
			return nil, err
		}
		defer r.(io.ReadCloser).Close()
	}

	_, err := bw.ReadFrom(r)
	return &bw.block, err
}
