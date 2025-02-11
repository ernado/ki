package install

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/go-faster/errors"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

func HetznerCloudInstall() error {
	// https://community.hetzner.com/tutorials/kubernetes-on-hetzner-with-crio-flannel-and-hetzner-balancer#step-7---install-hetzner-cloud-controller
	// https://github.com/hetznercloud/csi-driver/blob/main/docs/kubernetes/README.md#kubernetes-hetzner-cloud-csi-driver

	fmt.Println("> Getting token")
	token, err := os.ReadFile("/root/.hcloud")
	if err != nil {
		return errors.New("read hetzner cloud token")
	}

	fmt.Println("> Getting network ID")
	tokenStr := strings.TrimSpace(string(token))
	client := hcloud.NewClient(hcloud.WithToken(tokenStr))
	var networkID int64
	{
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		networks, err := client.Network.All(ctx)
		if err != nil {
			return errors.Wrap(err, "get networks")
		}
		for _, network := range networks {
			if network.Name == "kubernetes-cluster" {
				networkID = network.ID
				break
			}
		}
	}
	if networkID == 0 {
		return errors.New("network not found")
	}

	fmt.Println("> Creating namespace and secret for Hetzner constrollers")

	const namespace = "hcloud"
	{
		// Create namespace.
		cmd := exec.Command("kubectl", "create", "namespace", namespace)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			return errors.Wrap(err, "kubectl create namespace")
		}
	}
	{
		cmd := exec.Command("kubectl", "create", "secret", "generic", "hcloud",
			"-n", namespace,
			"--from-literal=token="+tokenStr,
			"--from-literal=network="+fmt.Sprintf("%d", networkID),
		)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			return errors.Wrap(err, "kubectl create secret")
		}
	}
	fmt.Println("> Installing Hetzner controllers")
	if err := HelmAddRepo("hcloud", "https://charts.hetzner.cloud"); err != nil {
		return errors.Wrap(err, "helm repo add")
	}
	fmt.Println("> Installing Hetzner cloud controller manager")
	if err := HelmUpgrade(HelmUpgradeOptions{
		Chart:     "hcloud/hcloud-cloud-controller-manager",
		Install:   true,
		Name:      "hccm",
		Namespace: namespace,
	}); err != nil {
		return errors.Wrap(err, "helm upgrade hccm")
	}
	fmt.Println("> Installing Hetzner cloud csi driver")
	if err := HelmUpgrade(HelmUpgradeOptions{
		Chart:     "hcloud/hcloud-csi",
		Install:   true,
		Namespace: namespace,
		Name:      "hcsi",
	}); err != nil {
		return errors.Wrap(err, "helm upgrade")
	}

	fmt.Println("> Hetzner support installed")

	return nil
}
