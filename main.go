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

	cli.Dispatch()
}
