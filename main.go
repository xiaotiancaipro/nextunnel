package main

import "github.com/xiaotiancaipro/nextunnel-server/cmd"

var version = "v0.3.0"

func main() {
	_ = cmd.New(version).Execute()
}
