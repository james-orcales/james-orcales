package rsa

import (
	"crypto/internal/rand"
	"io"
	"local/james-orcales/shared/std/sim/prng"
)

// entropy returns rand.Reader for a nil generator, and a reader that advances generator
// otherwise.
func entropy(generator *prng.Xoshiro) io.Reader {
	if generator == nil {
		return rand.Reader
	}
	return &xoshiroReader{generator: generator}
}

type xoshiroReader struct {
	generator *prng.Xoshiro
}

func (r *xoshiroReader) Read(b []byte) (int, error) {
	for i := 0; i < len(b); i += 8 {
		word := prng.Xoshiro_Next(prng.Xoshiro_Pointer(r.generator))
		for j := 0; j < 8 && i+j < len(b); j++ {
			b[i+j] = byte(word >> (8 * j))
		}
	}
	return len(b), nil
}
