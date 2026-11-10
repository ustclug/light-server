package main

import (
	"os"
	"testing"
)

func TestAggregateProvidedGOSTLog(t *testing.T) {
	file, err := os.Open("../inbox/gost.log")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	totals, err := aggregate(file)
	if err != nil {
		t.Fatal(err)
	}
	if got := totals["ibug"]; got != 178136 {
		t.Fatalf("inputBytes + outputBytes = %d, want 178136", got)
	}
}
