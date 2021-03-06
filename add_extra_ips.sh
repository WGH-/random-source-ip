#!/bin/bash

INTERFACE="$(ip -json route get :: | jq -r '.[0].dev')"
ADDR="$(ip -json address show dev ${INTERFACE} | jq -r '.[0].addr_info[] | select(.prefixlen == 64 and .mngtmpaddr and .family == "inet6").local')"
VALIDITY=$(( 3600 * 24 * 365 ))
NUM_ADDRESSES=128

gen_addresses() {
    python3 -c '
import sys
import ipaddress
import random
net = ipaddress.IPv6Network((sys.argv[1], 64), strict=False)
while True:
    i = random.randint(2, net.num_addresses-1)
    addr = net[i]
    print(addr)
' "$1"
}

gen_addresses "$ADDR" | head -n "$NUM_ADDRESSES" | while read addr; do
    echo "Adding $addr..." >&2
    ip addr add "$addr/64" dev "$INTERFACE" valid_lft "$VALIDITY" preferred_lft 0
done
