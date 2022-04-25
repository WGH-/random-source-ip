This package contains a custom dialer that select random source IPv6
address for every `Dial`.

If you're inside typical /64 IPv6 network with SLAAC or something similar,
where you can technically assign any /64 address arbitrarily, you can
use the included `./add_extra_ips.sh` script to add some IPv6 address
to your interface.
