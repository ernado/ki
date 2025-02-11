package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/go-faster/errors"
)

type KubeadmInitOptions struct {
	SkipPhases           []string
	PodNetworkCIDR       string
	ServiceCIDR          string
	ControlPlaneEndpoint string
}

func KubeadmInit(opts KubeadmInitOptions) error {
	var args []string
	if len(opts.SkipPhases) > 0 {
		args = append(args, "--skip-phases="+strings.Join(opts.SkipPhases, ","))
	}
	if opts.PodNetworkCIDR != "" {
		args = append(args, "--pod-network-cidr="+opts.PodNetworkCIDR)
	}
	if opts.ServiceCIDR != "" {
		args = append(args, "--service-cidr="+opts.ServiceCIDR)
	}
	if opts.ControlPlaneEndpoint != "" {
		args = append(args, "--control-plane-endpoint="+opts.ControlPlaneEndpoint)
	}
	fmt.Println("> kubeadm init", args)
	cmd := exec.Command("kubeadm", append([]string{"init"}, args...)...)
	output := bytes.NewBuffer(nil)
	cmd.Stderr = io.MultiWriter(os.Stderr, output)
	cmd.Stdout = io.MultiWriter(os.Stdout, output)
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "kubeadm init")
	}
	reToken := regexp.MustCompile(`--token (\S+)`)
	reHash := regexp.MustCompile(`--discovery-token-ca-cert-hash (\S+)`)
	var token, hash string
	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		line := scanner.Text()
		if m := reToken.FindStringSubmatch(line); m != nil {
			token = m[1]
		}
		if m := reHash.FindStringSubmatch(line); m != nil {
			hash = m[1]
		}
	}
	if token == "" || hash == "" {
		return errors.New("token or hash not found")
	}
	fmt.Printf("> Token: %s\nHash: %s\n", token, hash)
	return nil
}
