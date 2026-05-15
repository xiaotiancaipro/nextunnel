package main

import "github.com/xiaotiancaipro/nextunnel-server/cmd"

var version = "v0.1.4"

func main() {
	_ = cmd.New(version).Execute()
}
