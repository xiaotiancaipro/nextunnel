package main

import "github.com/xiaotiancaipro/nextunnel/cmd"

var version = "v1.0.0-alpha"

func main() {
	_ = new(cmd.Root).New(version)
}
