package randomsourceip

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"syscall"

	"golang.org/x/sys/unix"

	"github.com/vishvananda/netlink"
)

func ensureIP4NotMappedIP6(ip net.IP) net.IP {
	// IPv4 address might be in IPv4-mapped IPv6 form.
	// Convert it to "canonical" 4-byte form.
	if ip4 := ip.To4(); ip4 != nil {
		ip = ip4
	}
	return ip
}

func getRandomIntefaceAddress(linkIndex int, randint func() int) (net.IP, error) {
	link, err := netlink.LinkByIndex(linkIndex)
	if err != nil {
		return nil, err
	}

	addrs, err := netlink.AddrList(link, netlink.FAMILY_V6)
	if err != nil {
		return nil, err
	}
	var candidates []net.IP

	for _, addr := range addrs {
		ip := ensureIP4NotMappedIP6(addr.IP)
		if len(ip) == net.IPv6len && ip.IsGlobalUnicast() {
			candidates = append(candidates, ip)
		}
	}

	return candidates[randint()%len(candidates)], nil
}

func getInterfaceForDestination(destination net.IP) (int, error) {
	r, err := netlink.RouteGet(destination)
	if err != nil {
		return 0, err
	}

	if len(r) != 1 {
		return 0, fmt.Errorf("RouteGet returned %d routes", len(r))
	}

	return r[0].LinkIndex, nil
}

func NewDialer(randomSource rand.Source, debug bool) *net.Dialer {
	if randomSource == nil {
		var b [8]byte
		if _, err := cryptorand.Read(b[:]); err != nil {
			panic(err)
		}
		randomSource = rand.NewSource(int64(binary.LittleEndian.Uint64(b[:])))
	}
	random := rand.New(randomSource)
	var m sync.Mutex
	randint := func() int {
		m.Lock()
		defer m.Unlock()
		return random.Int()
	}

	return &net.Dialer{
		Control: func(network, address string, c syscall.RawConn) error {
			if debug {
				log.Printf("Control: network=%q address=%q", network, address)
			}
			if network == "tcp6" || network == "udp6" {
				host, _, err := net.SplitHostPort(address)
				if err != nil {
					log.Printf("%#v", err)
					return err
				}

				destinationIp := net.ParseIP(host)
				if destinationIp == nil {
					log.Printf("failed to parse IP %q", host)
					return fmt.Errorf("bad IP %q", host)
				}

				iface, err := getInterfaceForDestination(destinationIp)
				if err != nil {
					log.Printf("%#v", err)
					return err
				}
				localAddr, err := getRandomIntefaceAddress(iface, randint)
				if err != nil {
					log.Printf("%#v", err)
					return err
				}
				if debug {
					log.Printf("selected addr %q", localAddr)
				}
				c.Control(func(fd uintptr) {
					sa := unix.SockaddrInet6{}
					copy(sa.Addr[:], localAddr)
					err = unix.Bind(int(fd), &sa)
				})
				return err
			}
			return nil
		},
	}
}
