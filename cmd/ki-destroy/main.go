package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-faster/errors"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

func run() error {
	client := hcloud.NewClient(hcloud.WithToken(os.Getenv("HETZNER_TOKEN")))
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()

	fmt.Println("> Removing all servers")
	servers, err := client.Server.All(ctx)
	if err != nil {
		return errors.Wrap(err, "get servers")
	}
	for _, server := range servers {
		fmt.Println("Removing server", server.Name)
		res, _, err := client.Server.DeleteWithResult(ctx, server)
		if err != nil {
			return errors.Wrap(err, "delete server")
		}
		// Wait until action processed.
		if err := client.Action.WaitForFunc(ctx, func(update *hcloud.Action) error {
			fmt.Printf("Server(%s): %s\n", server.Name, update.Command)
			return nil
		}, res.Action); err != nil {
			return errors.Wrap(err, "wait for action")
		}
		fmt.Println("Server removed")
	}

	fmt.Println("> Removing all load balancers")
	lbs, err := client.LoadBalancer.All(ctx)
	if err != nil {
		return errors.Wrap(err, "get load balancers")
	}
	for _, lb := range lbs {
		fmt.Println("Removing load balancer", lb.Name)
		if _, err := client.LoadBalancer.Delete(ctx, lb); err != nil {
			return errors.Wrap(err, "delete load balancer")
		}
		fmt.Println("Load balancer removed")
	}

	fmt.Println("> Removing all networks")
	networks, err := client.Network.All(ctx)
	if err != nil {
		return errors.Wrap(err, "get networks")
	}
	for _, network := range networks {
		fmt.Println("Removing network", network.Name)
		if _, err := client.Network.Delete(ctx, network); err != nil {
			return errors.Wrap(err, "delete network")
		}
		fmt.Println("Network removed")
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
	}
}
