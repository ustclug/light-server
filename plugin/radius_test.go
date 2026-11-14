package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2866"
	"layeh.com/radius/rfc2869"
)

func TestAccountingPacket(t *testing.T) {
	const input = uint64(2)<<32 | 5678
	const output = uint64(3)<<32 | 1234
	packet, err := accountingPacket([]byte("secret"), "squid", "ibug", "session", input, output, rfc2866.AcctStatusType_Value_Stop, time.Unix(123, 0))
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
	inputOctets, err := rfc2866.AcctInputOctets_Lookup(packet)
	require.NoError(t, err)
	assert.Equal(t, uint32(5678), uint32(inputOctets))
	inputGigawords, err := rfc2869.AcctInputGigawords_Lookup(packet)
	require.NoError(t, err)
	assert.Equal(t, uint32(2), uint32(inputGigawords))
	outputOctets, err := rfc2866.AcctOutputOctets_Lookup(packet)
	require.NoError(t, err)
	assert.Equal(t, uint32(1234), uint32(outputOctets))
	outputGigawords, err := rfc2869.AcctOutputGigawords_Lookup(packet)
	require.NoError(t, err)
	assert.Equal(t, uint32(3), uint32(outputGigawords))
}

func TestAccountingStartPacketUsesSessionStartTime(t *testing.T) {
	sessionStart := time.Unix(123, 0)
	packet, err := accountingPacket([]byte("secret"), "squid", "ibug", "session", 0, 0, rfc2866.AcctStatusType_Value_Start, sessionStart)
	require.NoError(t, err)
	eventTime, err := rfc2869.EventTimestamp_Lookup(packet)
	require.NoError(t, err)
	assert.Equal(t, sessionStart, eventTime)
}

func TestAccountingStopPacketUsesSessionEndTime(t *testing.T) {
	sessionEnd := time.Unix(456, 0)
	packet, err := accountingPacket([]byte("secret"), "squid", "ibug", "session", 1, 2, rfc2866.AcctStatusType_Value_Stop, sessionEnd)
	require.NoError(t, err)
	eventTime, err := rfc2869.EventTimestamp_Lookup(packet)
	require.NoError(t, err)
	assert.Equal(t, sessionEnd, eventTime)
}
