package main

import (
	"fmt"
	"melato.org/export/password"
	"os"
	"strconv"
)

func generate() error {
	size := 20
	var err error
	if len(os.Args) == 2 {
		size, err = strconv.Atoi(os.Args[1])
		if err != nil {
			return err
		}
	}
	s, err := password.Generate(size)
	if err != nil {
		return err
	} else {
		fmt.Println(s)
		return nil
	}

}
func main() {
	err := generate()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
