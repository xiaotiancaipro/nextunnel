package main

import (
	"os"

	"github.com/xiaotiancaipro/nextunnel"
)

func main() {
	if err := New(nextunnel.Version).Execute(); err != nil {
		os.Exit(1)
	}
}
