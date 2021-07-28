package main

import (
	"context"
	"net"
)

type emptyResolver struct{}

func (emptyResolver) Resolve(ctx context.Context, name string) (context.Context, net.IP, error) {
	// return nil to force SOCKS pass the domain name as is to Dialer
	return ctx, nil, nil
}
