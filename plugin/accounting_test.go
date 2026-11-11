package main

import (
	"os"
	"strings"
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

func TestAggregateIgnoresUnrelatedAndZeroByteLines(t *testing.T) {
	log := strings.Join([]string{
		"not JSON",
		"{\"level\":\"info\",\"inputBytes\":100,\"outputBytes\":200}",
		"{\"user\":\"alice\"}",
		"{\"user\":\"alice\",\"inputBytes\":0,\"outputBytes\":0}",
		"{\"user\":\"alice\",\"inputBytes\":7}",
		"{\"user\":\"alice\",\"outputBytes\":11}",
		"{\"user\":\"bob\",\"clientID\":\"different\",\"inputBytes\":13,\"outputBytes\":17}",
	}, "\n")

	totals, err := aggregate(strings.NewReader(log))
	if err != nil {
		t.Fatal(err)
	}
	if got := totals["alice"]; got != 18 {
		t.Fatalf("alice total = %d, want 18", got)
	}
	if got := totals["bob"]; got != 30 {
		t.Fatalf("bob total = %d, want 30", got)
	}
	if len(totals) != 2 {
		t.Fatalf("totals = %#v", totals)
	}
}
