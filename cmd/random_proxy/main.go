package main

import (
	"bufio"
	"flag"
	"log"
	"net"
	"os"

	"github.com/WGH-/random-source-ip"
	"github.com/armon/go-socks5"
)

func main() {
	var (
		bind  = flag.String("bind", "[::1]:1080", "Port to bind the server to")
		debug = flag.Bool("debug", false, "Enable some debug output")
	)

	flag.Parse()

	ipChanger := newIpchaner()
	ipChanger.Randomize()

	chooser := func(destination net.IP) (net.IP, error) {
		return ipChanger.GetSourceIP(destination)
	}

	dialer := randomsourceip.NewDialer(chooser, *debug)

	server, err := socks5.New(&socks5.Config{
		Dial:     dialer.DialContext,
		Resolver: emptyResolver{},
	})
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		log.Fatal(server.ListenAndServe("tcp", *bind))
	}()

	rd := bufio.NewReader(os.Stdin)

	for {
		b, err := rd.ReadByte()
		if err != nil {
			log.Fatal(err)
		}
		if b == '\n' {
			ipChanger.Randomize()
		}
	}
}
