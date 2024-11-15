package keystore

import (
	"errors"
	"io"
	"bucket"
)

type revreader struct {
	b    []byte
	off  uint
	forw *reader
}

type reader struct {
	revreader
	off uint
}

func newreader(b *buff, off, roff uint) *reader {
	r := &reader{revreader{(*bucket.Buf)(b).Bytes(), roff, nil}, off}
	r.forw = r
	return r
}

type revwriter struct {
	b    []byte
	off  uint
	forw *writer
}

type writer struct {
	revwriter
	off uint
}

func newwriter(b *buff) *writer {
	w := &writer{revwriter{(*bucket.Buf)(b).Bytes(), 0, nil}, 0}
	w.forw = w
	return w
}

func (r *reader) Read(b []byte) (int, error) {
	left := len(r.b) - int(r.off)
	if left < len(b) || (int(r.off+r.revreader.off)+len(b)) >= len(r.b) {
		return 0, io.ErrUnexpectedEOF // short read is an error for this reader
	}
	copy(b, r.b[int(r.off):int(r.off)+len(b)])
	r.off += uint(len(b))
	return len(b), nil
}

func (r *reader) ReadByte() (byte, error) { // needed so some compression libs don't read too much
	var b [1]byte
	_, err := r.Read(b[:])
	return b[0], err
}

func (r *revreader) Read(b []byte) (int, error) {
	left := len(r.b) - int(r.off)
	if left < len(b) || (int(r.off+r.forw.off)+len(b)) >= len(r.b) {
		return 0, io.ErrUnexpectedEOF // short read is an error for this reader
	}
	copy(b, r.b[left-len(b):left])
	r.off += uint(len(b))
	return len(b), nil
}

var ErrInvalid = errors.New("invalid argument")

func (r *revreader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekEnd:
		offset += int64(len(r.b))
		fallthrough
	case io.SeekStart:
		if offset < 0 {
			break
		}
		r.off = uint(offset)
		return offset, nil
	case io.SeekCurrent:
		if offset > int64(r.off) {
			break
		}
		r.off += uint(offset)
		return offset, nil
	}
	return 0, ErrInvalid
}

func (w *writer) Write(b []byte) (int, error) {
	left := len(w.b) - int(w.off)
	if left < len(b) || (int(w.off+w.revwriter.off)+len(b)) >= len(w.b) {
		return 0, io.ErrShortWrite
	}
	copy(w.b[int(w.off):int(w.off)+len(b)], b)
	w.off += uint(len(b))
	return len(b), nil
}

func (w *revwriter) Write(b []byte) (int, error) {
	left := len(w.b) - int(w.off)
	if left < len(b) || (int(w.off+w.forw.off)+len(b)) >= len(w.b) {
		return 0, io.ErrShortWrite
	}
	copy(w.b[left-len(b):left], b)
	w.off += uint(len(b))
	return len(b), nil
}
