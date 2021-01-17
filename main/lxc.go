package main

import (
	"fmt"
	"os"
)

func main() {
	var pargs []interface{}
	pargs = append(pargs, "lxd")
	for _, arg := range os.Args[1:] {
		pargs = append(pargs, arg)
	}
	fmt.Println(pargs...)
}
