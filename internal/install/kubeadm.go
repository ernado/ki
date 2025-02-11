package install

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/go-faster/errors"
)

type KubeadmInitOptions struct {
	SkipPhases           []string
	PodNetworkCIDR       string
	ServiceCIDR          string
	ControlPlaneEndpoint string
	ExtraSans            []string
}

type InitParams struct {
	Endpoint string `json:"endpoint"` // 1.2.3.4:6443
	Token    string `json:"token"`
	Hash     string `json:"hash"`
}

const initParamsPath = "/etc/kubeadm-init.json"

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
	for _, san := range opts.ExtraSans {
		args = append(args, "--apiserver-cert-extra-sans="+san)
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

	data, err := json.Marshal(InitParams{
		Endpoint: net.JoinHostPort(opts.ControlPlaneEndpoint, "6443"),
		Token:    token,
		Hash:     hash,
	})
	if err != nil {
		return errors.Wrap(err, "marshal")
	}
	if err := os.WriteFile(initParamsPath, data, 0600); err != nil {
		return errors.Wrap(err, "write")
	}

	return nil
}

func KubeadmJoin(controlPlaneNodeInternalIP string) error {
	// Wait for 6443 port on control plane node.
	{
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		ticker := time.NewTicker(time.Second)
		fmt.Println("> Waiting for control plane node")
		defer ticker.Stop()
		for range ticker.C {
			if ctx.Err() != nil {
				return errors.New("timeout waiting for control plane to listen on 6443")
			}
			err := func() error {
				conn, err := net.Dial("tcp", net.JoinHostPort(controlPlaneNodeInternalIP, "6443"))
				if err != nil {
					fmt.Println(err)
					return err
				}
				_ = conn.Close()
				return nil
			}()
			if err == nil {
				break
			}
		}
	}

	fmt.Println("> kubeadm join")

	var params InitParams
	{
		bo := backoff.NewConstantBackOff(time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := backoff.RetryNotify(func() error {
			cmd := exec.Command("ssh", "-o", "StrictHostKeyChecking=accept-new", "cluster@"+controlPlaneNodeInternalIP, "sudo", "cat", initParamsPath)
			output, err := cmd.Output()
			if err != nil {
				return errors.Wrap(err, "ssh")
			}
			if err := json.Unmarshal(output, &params); err != nil {
				return errors.Wrap(err, "unmarshal")
			}
			return nil
		}, backoff.WithContext(bo, ctx), func(err error, d time.Duration) {}); err != nil {
			return errors.Wrap(err, "retrieve config")
		}
	}
	if params.Hash == "" || params.Token == "" || params.Endpoint == "" {
		return errors.Errorf("invalid params from %s", initParamsPath)
	}
	fmt.Printf("Got params: %+v\n", params)
	arg := []string{
		"join", params.Endpoint, "--token", params.Token, "--discovery-token-ca-cert-hash", params.Hash,
	}
	cmd := exec.Command("kubeadm", arg...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "kubeadm join")
	}

	return nil
}
