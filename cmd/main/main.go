package main

import (
	"io"
	"net/http"
	"os"

	"github.com/WGH-/random-source-ip"
)

func main() {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = randomsourceip.NewRandomChoiceDialer(nil, true).DialContext
	transport.DisableKeepAlives = true

	client := &http.Client{
		Transport: transport,
	}

	for i := 0; i < 16; i++ {
		func() {
			resp, err := client.Get("https://icanhazip.com")
			if err != nil {
				panic(err)
			}
			defer resp.Body.Close()
			io.Copy(os.Stdout, resp.Body)
		}()
	}
}
