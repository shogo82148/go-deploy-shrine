package main

import (
	"testing"

	"github.com/tkuchiki/parsetime"
)

func BenchmarkQreki(b *testing.B) {
	t, _ := parsetime.Parse("2015-12-10")
	for i := 0; i < b.N; i++ {
		Time2Qreki(t)
	}
}
