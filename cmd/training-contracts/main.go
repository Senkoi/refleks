package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"

	"aimmeow/internal/training/contracts"
)

func main() {
	path := flag.String("output", "frontend/src/features/training/contracts.generated.ts", "TypeScript output")
	check := flag.Bool("check", false, "Fail if generated contracts are stale")
	flag.Parse()
	expected := contracts.Generate()
	if *check {
		actual, err := os.ReadFile(*path)
		if err != nil || !bytes.Equal(actual, expected) {
			fmt.Fprintln(os.Stderr, "training DTO contracts are stale; run go run ./cmd/training-contracts")
			os.Exit(1)
		}
		return
	}
	if err := os.WriteFile(*path, expected, 0644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
