package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/go-faster/errors"
)

func KubectlApply(file string) error {
	fmt.Println("> kubectl apply -f", file)
	cmd := exec.Command("kubectl", "apply", "-f", file)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "kubectl apply")
	}
	return nil
}
