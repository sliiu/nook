package main

import (
	"fmt"
	"os"
	"time"

	"pixel-control-center/internal/brightness"
)

func main() {
	b, err := brightness.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !b.Available() {
		fmt.Println("skip: brightness unavailable (no controllable built-in display)")
		return
	}
	before, err := b.Level()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	target := before.Percent + 3
	if target > 100 {
		target = before.Percent - 3
	}
	if err := b.Set(target); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	time.Sleep(150 * time.Millisecond)
	after, err := b.Level()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := b.Set(before.Percent); err != nil {
		fmt.Fprintln(os.Stderr, "restore:", err)
		os.Exit(1)
	}
	if after.Percent != target {
		fmt.Fprintf(os.Stderr, "roundtrip fail before=%d target=%d after=%d\n", before.Percent, target, after.Percent)
		os.Exit(1)
	}
	fmt.Printf("ok before=%d target=%d restored=%d\n", before.Percent, target, before.Percent)
}
