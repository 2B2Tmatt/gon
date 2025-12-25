package main

import (
	"fmt"
	"gon/cli"
)

func main() {
	cli, err := cli.LoadCli()
	if err != nil {
		fmt.Println("issue loading cli: ", err)
		return
	}

	err = cli.Dispatch()
	if err != nil {
		fmt.Println("dispatch failed, error:", err)
	}
}
