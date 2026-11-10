// micsmoke reads the default input level, nudges it, and restores it.
package main

import (
	"fmt"
	"os"

	"pixel-control-center/internal/audio"
)

func main() {
	b, err := audio.NewInput()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	before, err := b.Level()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !before.Adjustable {
		fmt.Printf("skip: not adjustable before=%+v\n", before)
		return
	}
	// A muted device reads as 0, so restoring that reading would wipe the
	// real gain. Leave it alone.
	if before.Muted {
		fmt.Printf("skip: muted before=%+v\n", before)
		return
	}
	target := before.Percent + 3
	if target > 100 {
		target = before.Percent - 3
	}
	if err := b.Set(target); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
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
	fmt.Printf("ok before=%d target=%d restored=%d muted=%v\n", before.Percent, target, before.Percent, before.Muted)
}
