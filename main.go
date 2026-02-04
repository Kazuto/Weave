package main

import (
	"fmt"
	"os"

	"github.com/Kazuto/Weave/internal/version"
	"github.com/Kazuto/Weave/pkg/weave"
)

func main() {
	name := ""
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	fmt.Println(weave.Hello(name))
	fmt.Println("version:", version.Version)
}
