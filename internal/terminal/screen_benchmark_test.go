package terminal

import (
	"fmt"
	"strings"
	"testing"
)

func BenchmarkScreenLongLines(b *testing.B) {
	for _, symbol := range []string{"x", "界"} {
		for _, width := range []int{256, 1024, 2048, 4096, 8192} {
			b.Run(fmt.Sprintf("%s/%d", symbol, width), func(b *testing.B) {
				input := []byte(strings.Repeat(symbol, width) + "\n")
				b.ReportAllocs()
				b.SetBytes(int64(len(input)))
				for b.Loop() {
					var screen Screen
					screen.Write(input)
					if len(screen.Lines()) != 2 {
						b.Fatal("lost output")
					}
				}
			})
		}
	}
}

func BenchmarkScreenShortLinesAndRedraw(b *testing.B) {
	input := []byte(strings.Repeat("界 café progress\r\x1b[Kupdated\n", 100))
	b.ReportAllocs()
	b.SetBytes(int64(len(input)))
	for b.Loop() {
		var screen Screen
		screen.Write(input)
		screen.Lines()
	}
}
