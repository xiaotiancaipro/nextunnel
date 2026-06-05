package main

import "github.com/xiaotiancaipro/nextunnel-server/cmd"

var version = "v0.5.0"

func main() {
	_ = new(cmd.Root).New(version).Execute()
}
