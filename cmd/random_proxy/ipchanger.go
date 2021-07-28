package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"golang.org/x/sys/unix"

	"github.com/vishvananda/netlink"
)

type trackedAddressInfo struct {
	link netlink.Link
	addr *netlink.Addr
}

type ipChanger struct {
	m      sync.Mutex
	suffix [8]byte

	// for removing old ones
	trackedAddresses []trackedAddressInfo
}

func newIpchaner() *ipChanger {
	res := &ipChanger{}
	go res.runRefresher()
	return res
}
func (ipChanger *ipChanger) randomizeInterface(linkIndex int) (net.IP, error) {
	link, err := netlink.LinkByIndex(linkIndex)
	if err != nil {
		return nil, err
	}

	addrs, err := netlink.AddrList(link, netlink.FAMILY_V6)
	if err != nil {
		return nil, err
	}

	// find network prefix
	var prefix net.IP
	for _, addr := range addrs {
		if addr.IP.IsGlobalUnicast() {
			if ones, _ := addr.IPNet.Mask.Size(); ones != 64 {
				return nil, fmt.Errorf("unexpected mask size for %s", addr.IPNet)
			}
			prefix = addr.IP
		}
	}

	ipChanger.m.Lock()
	defer ipChanger.m.Unlock()

	result := net.IPNet{
		IP:   make(net.IP, net.IPv6len),
		Mask: net.CIDRMask(64, 128),
	}
	copy(result.IP[:], prefix)
	copy(result.IP[8:], ipChanger.suffix[:])

	// figure out if the address is already configured
	for _, addr := range addrs {
		if addr.IP.Equal(result.IP) {
			return addr.IP, nil
		}
	}

	addr := &netlink.Addr{
		IPNet:       &result,
		ValidLft:    90,
		PreferedLft: 0,
		Flags:       unix.IFA_F_NODAD,
	}

	// have to configure the address
	err = netlink.AddrAdd(link, addr)
	if err != nil {
		log.Printf("failed to configure addr: %v", err)
		return nil, err
	}

	log.Printf("assigned new address %q", addr)

	ipChanger.trackedAddresses = append(ipChanger.trackedAddresses, trackedAddressInfo{link, addr})

	return result.IP, nil
}

func (ipChanger *ipChanger) GetSourceIP(destination net.IP) (net.IP, error) {
	routes, err := netlink.RouteGet(destination)
	if err != nil {
		return nil, err
	}
	if len(routes) != 1 {
		return nil, fmt.Errorf("got unexpected number of routes: %#v", routes)
	}
	route := routes[0]

	addr, err := ipChanger.randomizeInterface(route.LinkIndex)
	if err != nil {
		return nil, err
	}

	return addr, nil
}

func (ipChanger *ipChanger) Randomize() {
	ipChanger.m.Lock()
	defer ipChanger.m.Unlock()

	if _, err := rand.Read(ipChanger.suffix[:]); err != nil {
		panic(err)
	}

	for _, info := range ipChanger.trackedAddresses {
		log.Printf("deleting address %q", info.addr)
		if err := netlink.AddrDel(info.link, info.addr); err != nil {
			panic(err)
		}
	}
	ipChanger.trackedAddresses = nil
}

func (ipChanger *ipChanger) runRefresher() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		func() {
			ipChanger.m.Lock()
			defer ipChanger.m.Unlock()
			for _, info := range ipChanger.trackedAddresses {
				if err := netlink.AddrReplace(info.link, info.addr); err != nil {
					panic(err)
				}
			}
		}()
	}
}
