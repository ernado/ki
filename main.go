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
		// Update /etc/fstab.
		fileName := "/etc/fstab"
		data, err := os.ReadFile(fileName)
		if err != nil {
			return errors.Wrap(err, "read")
		}
		// Replace line with / swap to # Swap disabled.
		scanner := bufio.NewScanner(bytes.NewReader(data))
		targetString := []byte(" swap ")
		if !bytes.Contains(data, targetString) {
			fmt.Println("> Swap is not enabled")
			return nil
		}
		fmt.Println("> Updating /etc/fstab")
		var out []byte
		for scanner.Scan() {
			line := scanner.Text()
			if len(line) > 0 && bytes.Contains(scanner.Bytes(), targetString) {
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
	{
		// Disable swap.
		fmt.Println("> Disabling swap")
		cmd := exec.Command("swapoff", "-a")
		if err := cmd.Run(); err != nil {
			return errors.Wrap(err, "run")
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
