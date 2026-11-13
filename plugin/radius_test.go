package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2866"
	"layeh.com/radius/rfc2869"
)

func TestAccountingPacket(t *testing.T) {
	const total = uint64(3)<<32 | 1234
	packet, err := accountingPacket([]byte("secret"), "squid", "ibug", "session", total, rfc2866.AcctStatusType_Value_Stop)
	require.NoError(t, err)
	userName, err := rfc2865.UserName_LookupString(packet)
	require.NoError(t, err)
	assert.Equal(t, "ibug", userName)
	nasIdentifier, err := rfc2865.NASIdentifier_LookupString(packet)
	require.NoError(t, err)
	assert.Equal(t, "squid", nasIdentifier)
	sessionID, err := rfc2866.AcctSessionID_LookupString(packet)
	require.NoError(t, err)
	assert.Equal(t, "session", sessionID)
	outputOctets, err := rfc2866.AcctOutputOctets_Lookup(packet)
	require.NoError(t, err)
	assert.Equal(t, uint32(1234), uint32(outputOctets))
	outputGigawords, err := rfc2869.AcctOutputGigawords_Lookup(packet)
	require.NoError(t, err)
	assert.Equal(t, uint32(3), uint32(outputGigawords))
}
