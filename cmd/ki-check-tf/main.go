package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"time"

	"github.com/go-faster/errors"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

func run() (rerr error) {
	var arg struct {
		Cleanup bool
		Timeout time.Duration
	}
	flag.BoolVar(&arg.Cleanup, "cleanup", false, "destroy terraform resources")
	flag.DurationVar(&arg.Timeout, "timeout", 10*time.Minute, "timeout for checking the load balancer")
	flag.Parse()

	if arg.Cleanup {
		defer func() {
			cmd := exec.Command("terraform", "destroy", "-auto-approve", "-var-file", ".tfvars")
			cmd.Stderr = os.Stderr
			cmd.Stdout = os.Stdout
			if err := cmd.Run(); err != nil {
				rerr = errors.Wrap(err, "terraform destroy")
			}
		}()
	}

	start := time.Now()

	client := hcloud.NewClient(hcloud.WithToken(os.Getenv("HETZNER_TOKEN")))
	ctx, cancel := context.WithTimeout(context.Background(), arg.Timeout)
	defer cancel()

	loadBalancers, err := client.LoadBalancer.All(ctx)
	if err != nil {
		return errors.Wrap(err, "get load balancers")
	}

	var pingURL string
	for _, lb := range loadBalancers {
		ipAddr := lb.PublicNet.IPv4.IP.String()
		u := &url.URL{
			Scheme: "http",
			Host:   net.JoinHostPort(ipAddr, "80"),
		}
		pingURL = u.String()
		break
	}
	if pingURL == "" {
		return errors.New("no load balancer found")
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if ctx.Err() != nil {
			fmt.Println("> Timed out")
			return ctx.Err()
		}
		fmt.Println("pinging", pingURL)
		resp, err := http.Get(pingURL)
		if err != nil {
			fmt.Println("ping failed:", err)
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			fmt.Println("ping succeeded", time.Since(start))
			return nil
		} else {
			fmt.Println("ping failed:", resp.Status)
		}
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %+v\n", err)
		os.Exit(1)
	}
}
