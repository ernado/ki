package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

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

func APTInstall(packages ...string) error {
	fmt.Println("> apt-get install", packages)
	cmd := exec.Command("apt-get", append([]string{"install", "-y"}, packages...)...)
	cmd.Env = appendEnv(os.Environ(), "DEBIAN_FRONTEND", "noninteractive")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "apt install")
	}
	return nil
}

func APTKey(keyName, keyURL string) error {
	fmt.Printf("> Adding GPG key %s\n", keyName)
	fileName := filepath.Join("/etc/apt/trusted.gpg.d/", keyName+".gpg")
	if _, err := os.Stat(fileName); err == nil {
		fmt.Printf("> GPG key %s already exists\n", fileName)
		return nil
	}
	fmt.Println("Downloading key", keyURL)
	res, err := http.Get(keyURL)
	if err != nil {
		return errors.Wrap(err, "get key")
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode != http.StatusOK {
		return errors.Errorf("bad status: %s", res.Status)
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return errors.Wrap(err, "read key")
	}
	fmt.Printf("> Writing %s\n", fileName)
	cmd := exec.Command("gpg", "--dearmour", "-o", fileName)
	cmd.Stdin = bytes.NewReader(data)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "gpg")
	}
	return nil
}

func lsbRelease() (string, error) {
	cmd := exec.Command("lsb_release", "-cs")
	out, err := cmd.Output()
	if err != nil {
		return "", errors.Wrap(err, "lsb_release")
	}
	return string(bytes.TrimSpace(out)), nil
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

func LoadKernelModules(name string, modules ...string) error {
	for _, module := range modules {
		fmt.Println("> Loading module", module)
		cmd := exec.Command("modprobe", module)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			return errors.Wrapf(err, "modprobe %s", module)
		}
	}
	// Persist in named configuration.
	fileName := filepath.Join("/etc/modules-load.d/", name+".conf")
	fmt.Printf("> Writing %s\n", fileName)
	var out []byte
	for _, module := range modules {
		out = append(out, module...)
		out = append(out, '\n')
	}
	if err := os.WriteFile(fileName, out, 0644); err != nil {
		return errors.Wrap(err, "write")
	}
	return nil
}

func ConfigureKernelParameters(name string, params map[string]any) error {
	fmt.Printf("> Configuring kernel parameters for %s\n", name)
	fileName := filepath.Join("/etc/sysctl.d/", name+".conf")
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fmt.Printf("> Writing %s\n", fileName)
	var out []byte
	for _, key := range keys {
		out = append(out, key...)
		out = append(out, '=')
		out = append(out, fmt.Sprintf("%v", params[key])...)
		out = append(out, '\n')
	}
	if err := os.WriteFile(fileName, out, 0644); err != nil {
		return errors.Wrap(err, "write")
	}
	// Reload.
	cmd := exec.Command("sysctl", "--system")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "sysctl --system")
	}
	return nil
}

func run() error {
	// 0. Check OS.
	release, err := lsbRelease()
	if err != nil {
		return errors.Wrap(err, "lsb_release")
	}
	fmt.Println("> OS release:", release)
	supported := map[string]struct{}{
		"noble": {},
	}
	if _, ok := supported[release]; !ok {
		return errors.Errorf("unsupported OS: %s", release)
	}
	// 1. Check required ports
	if err := CheckTCPPortIsFree(6443); err != nil {
		return errors.Wrap(err, "check k8s port")
	}
	// 2. Swap configuration
	if err := DisableSwap(); err != nil {
		return errors.Wrap(err, "disable swap")
	}
	// 3. Update apt cache
	if err := APTUpdate(); err != nil {
		return errors.Wrap(err, "apt update")
	}
	// Installing a container runtime.
	if err := LoadKernelModules("containerd", "overlay", "br_netfilter"); err != nil {
		return errors.Wrap(err, "load kernel modules")
	}
	if err := ConfigureKernelParameters("kubernetes", map[string]any{
		"net.bridge.bridge-nf-call-ip6tables": 1,
		"net.bridge.bridge-nf-call-iptables":  1,
		"net.ipv4.ip_forward":                 1,
	}); err != nil {
		return errors.Wrap(err, "configure kernel parameters")
	}
	fmt.Println("> Installing containerd")
	if err := APTInstall("curl", "gnupg2", "software-properties-common", "apt-transport-https", "ca-certificates"); err != nil {
		return errors.Wrap(err, "install containerd dependencies")
	}
	if err := APTKey("docker", "https://download.docker.com/linux/ubuntu/gpg"); err != nil {
		return errors.Wrap(err, "add docker key")
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
		os.Exit(1)
	}
}
