package main

import (
	"testing"

	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2866"
	"layeh.com/radius/rfc2869"
)

func TestAccountingPacket(t *testing.T) {
	const total = uint64(3)<<32 | 1234
	packet, err := accountingPacket([]byte("secret"), "squid", "ibug", "session", total, rfc2866.AcctStatusType_Value_Stop)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := rfc2865.UserName_LookupString(packet); got != "ibug" {
		t.Fatalf("User-Name = %q", got)
	}
	if got, _ := rfc2865.NASIdentifier_LookupString(packet); got != "squid" {
		t.Fatalf("NAS-Identifier = %q", got)
	}
	if got, _ := rfc2866.AcctSessionID_LookupString(packet); got != "session" {
		t.Fatalf("Acct-Session-Id = %q", got)
	}
	if got, _ := rfc2866.AcctOutputOctets_Lookup(packet); uint32(got) != 1234 {
		t.Fatalf("Acct-Output-Octets = %d", got)
	}
	if got, _ := rfc2869.AcctOutputGigawords_Lookup(packet); uint32(got) != 3 {
		t.Fatalf("Acct-Output-Gigawords = %d", got)
	}
}
