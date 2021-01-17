package main

import (
	"fmt"
	"os"
)

/** lxc -- dummy lxc for developing on non LXD machines */
func main() {
	args := os.Args[1:]

	var pargs []interface{}
	pargs = append(pargs, "lxc")
	for _, arg := range args {
		pargs = append(pargs, arg)
	}

	if args[0] == "file" && args[1] == "pull" && args[3] == "-" {
		fmt.Println(os.Stderr, "Error: not found")
		os.Exit(1)
	}
	fmt.Println(pargs...)
}
