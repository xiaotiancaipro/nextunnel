package main

import "github.com/xiaotiancaipro/nextunnel/cmd/server/root"

var version = "v0.0.0"

func main() {
	_ = new(root.Root).New(version)
}
