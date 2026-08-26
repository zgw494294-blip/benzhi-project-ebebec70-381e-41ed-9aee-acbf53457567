package main

import (
	"flag"
	"os"
)

func listenAddress(addr string) string {
	if p := os.Getenv("PORT"); p != "" && addr == "127.0.0.1:19081" {
		return "127.0.0.1:" + p
	}
	return addr
}

var _ = flag.ErrHelp
