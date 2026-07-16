package main

import "github.com/xiaotiancaipro/nextunnel/internal/client/cli"

var version = "v0.0.0"

func main() {
	_ = cli.New(version).Execute()
}
