package record

import (
	"bytes"
	"testing"
)

func FuzzCodecReadIsBounded(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0, 0, 1, 0})
	f.Add(bytes.Repeat([]byte{0xff}, 512))
	f.Fuzz(func(t *testing.T, data []byte) {
		opts := Options{MaxPlaintext: 1024, Buckets: []int{256, 512, 1024}, MaxRecordsPerKey: 8}
		_, server := codecPair(t, opts)
		_, _ = server.Read(bytes.NewReader(data))
	})
}
