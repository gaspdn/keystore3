package main

import (
	"fmt"
	"math/bits"
	"os"
	"strconv"
)

func forkfan(bs int, compressed bool) (int, int, int) { // return max fork fanout, bit distance (including stops) and ptrsize
	p3, p2 := 1, 1

	bs -= 2 + 1 // block header + fork header
	if compressed {
		bs -= 18 // gzip header
	}
	maxsegbits := bits.Len(uint(bs/3 - 1)) //  3 is minimum size of a segment

	for bd := 1; ; bd++ {
		w := 2*p3 + p2
		s := (w*bits.Len(uint(bd+1)) + 7) / 8 // bits for shorthands
		if 18*w+s+(maxsegbits*w+7)/8 > bs {   // 18 is size of a segment holding 1 remote ptr
			return 2*p3/3 + p2/2, bd - 1, maxsegbits
		}
		p3 *= 3
		p2 *= 2
	}
}

func main() {
	for _, arg := range os.Args[1:] {
		bs, _ := strconv.Atoi(arg)
		f, bd, ps := forkfan(bs, false)
		fmt.Printf("blocksize %v fanout %v bitdistance %v ptrsize %v\n", bs, f, bd, ps)
	}
}
