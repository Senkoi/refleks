package main

import (
	"bytes"
	"testing"

	"aimmeow/internal/training/contracts"
)

func TestContractsMatchAcrossCheckouts(t *testing.T) {
	expected := contracts.Generate()
	for _, tc := range []struct {
		name   string
		actual []byte
		match  bool
	}{
		{"LF checkout", expected, true},
		{"CRLF checkout", bytes.ReplaceAll(expected, []byte("\n"), []byte("\r\n")), true},
		{"changed contract", bytes.Replace(expected, []byte("export type"), []byte("export interface"), 1), false},
		{"extra declaration", append(bytes.Clone(expected), []byte("export type Stale = {};\n")...), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := contractsMatch(tc.actual, expected); got != tc.match {
				t.Fatalf("contractsMatch = %v, want %v", got, tc.match)
			}
		})
	}
}
