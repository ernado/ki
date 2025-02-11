package main

import (
	"fmt"
	"os"

	"github.com/ernado/ki/internal/install"
)

func main() {
	if err := install.Run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
		os.Exit(1)
	}
}
