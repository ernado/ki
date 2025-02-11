package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
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

func appendEnv(vars []string, key, value string) []string {
	return append(vars, key+"="+value)
}

func APTUpdate() error {
	fmt.Println("> apt-get update")
	cmd := exec.Command("apt-get", "update")
	cmd.Env = appendEnv(os.Environ(), "DEBIAN_FRONTEND", "noninteractive")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "apt update")
	}
	return nil
}

func CheckTCPPortIsFree(n int) error {
	// nc 127.0.0.1 6443 -v
	// ^ should fail, but in go.
	fmt.Printf("> Checking port %d\n", n)
	tcpAddr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: n}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		fmt.Printf("> Port %d is free\n", n)
		return nil
	}
	_ = conn.Close()
	return errors.Errorf("port %d is in use", n)
}

func run() error {
	if err := DisableSwap(); err != nil {
		return errors.Wrap(err, "disable swap")
	}
	if err := APTUpdate(); err != nil {
		return errors.Wrap(err, "apt update")
	}
	if err := CheckTCPPortIsFree(6443); err != nil {
		return errors.Wrap(err, "check k8s port")
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
		os.Exit(1)
	}
}
