package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/go-faster/errors"
)

// DisableSwap disables swap on node.
func DisableSwap() error {
	{
		// Disable swap.
		fmt.Println("> Disabling swap")
		cmd := exec.Command("swapoff", "-a")
		if err := cmd.Run(); err != nil {
			return errors.Wrap(err, "run")
		}
	}
	{
		// Update /etc/fstab.
		fmt.Println("> Updating /etc/fstab")
		fileName := "/etc/fstab"
		data, err := os.ReadFile(fileName)
		if err != nil {
			return errors.Wrap(err, "read")
		}
		// Replace line with / swap to # Swap disabled.
		var out []byte
		scanner := bufio.NewScanner(bytes.NewReader(data))
		for scanner.Scan() {
			line := scanner.Text()
			if len(line) > 0 && bytes.Contains([]byte(line), []byte(" swap ")) {
				out = append(out, '#')
			}
			out = append(out, line...)
			out = append(out, '\n')
		}
		// Write back.
		if err := os.WriteFile("/etc/fstab", out, 0644); err != nil {
			return errors.Wrap(err, "write")
		}
	}
	return nil
}

func run() error {
	if err := DisableSwap(); err != nil {
		return errors.Wrap(err, "disable swap")
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
		os.Exit(1)
	}
}
