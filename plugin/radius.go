package main

import (
	"context"
	"fmt"
	"time"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2866"
	"layeh.com/radius/rfc2869"
)

type radiusBackend struct {
	authServer, accountingServer string
	secret, nas                  string
	timeout                      time.Duration
	client                       *radius.Client
}

func newRadiusBackend(c Config) *radiusBackend {
	return &radiusBackend{
		authServer: c.RadiusAuthServer, accountingServer: c.RadiusAccountingServer,
		secret: c.RadiusSecret, nas: c.NASIdentifier, timeout: c.RadiusTimeout,
		client: &radius.Client{Retry: time.Second, MaxPacketErrors: 10},
	}
}

func (b *radiusBackend) exchange(ctx context.Context, packet *radius.Packet, address string) (*radius.Packet, error) {
	ctx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()
	return b.client.Exchange(ctx, packet, address)
}

func (b *radiusBackend) Authenticate(ctx context.Context, username, password string) (string, bool, error) {
	packet := radius.New(radius.CodeAccessRequest, []byte(b.secret))
	if err := rfc2865.UserName_SetString(packet, username); err != nil {
		return "", false, err
	}
	if err := rfc2865.UserPassword_SetString(packet, password); err != nil {
		return "", false, err
	}
	if err := rfc2865.NASIdentifier_SetString(packet, b.nas); err != nil {
		return "", false, err
	}
	reply, err := b.exchange(ctx, packet, b.authServer)
	if err != nil {
		return "", false, err
	}
	if reply.Code != radius.CodeAccessAccept {
		return "", false, nil
	}
	id, err := rfc2865.UserName_LookupString(reply)
	if err != nil {
		id = username
	}
	return id, true, nil
}

func (b *radiusBackend) Account(ctx context.Context, username string, bytes uint64, session string) error {
	for _, status := range []rfc2866.AcctStatusType{rfc2866.AcctStatusType_Value_Start, rfc2866.AcctStatusType_Value_Stop} {
		packet, err := accountingPacket([]byte(b.secret), b.nas, username, session, bytes, status)
		if err != nil {
			return err
		}
		reply, err := b.exchange(ctx, packet, b.accountingServer)
		if err != nil {
			return err
		}
		if reply.Code != radius.CodeAccountingResponse {
			return fmt.Errorf("unexpected RADIUS response: %s", reply.Code)
		}
	}
	return nil
}

func accountingPacket(secret []byte, nas, username, session string, bytes uint64, status rfc2866.AcctStatusType) (*radius.Packet, error) {
	packet := radius.New(radius.CodeAccountingRequest, secret)
	if err := rfc2865.UserName_SetString(packet, username); err != nil {
		return nil, err
	}
	if err := rfc2865.NASIdentifier_SetString(packet, nas); err != nil {
		return nil, err
	}
	if err := rfc2866.AcctSessionID_SetString(packet, session); err != nil {
		return nil, err
	}
	if err := rfc2866.AcctStatusType_Set(packet, status); err != nil {
		return nil, err
	}
	if status == rfc2866.AcctStatusType_Value_Stop {
		if err := rfc2866.AcctOutputOctets_Set(packet, rfc2866.AcctOutputOctets(uint32(bytes))); err != nil {
			return nil, err
		}
		if err := rfc2869.AcctOutputGigawords_Set(packet, rfc2869.AcctOutputGigawords(uint32(bytes>>32))); err != nil {
			return nil, err
		}
	}
	return packet, nil
}
