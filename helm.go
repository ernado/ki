package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/go-faster/errors"
)

func HelmAddRepo(name, url string) error {
	fmt.Printf("> helm repo add %s %s\n", name, url)
	cmd := exec.Command("helm", "repo", "add", name, url)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "helm repo add")
	}
	return nil
}

type HelmUpgradeOptions struct {
	Version         string
	Name            string
	Install         bool
	Namespace       string
	CreateNamespace bool
	Chart           string
	Values          string
	Repo            string
	KubeConfig      string
}

func HelmUpgrade(opt HelmUpgradeOptions) error {
	fmt.Println("> helm: installing", opt.Name, opt.Chart)
	args := []string{
		"upgrade",
	}
	if opt.Install {
		args = append(args, "--install")
	}
	if opt.Repo != "" {
		args = append(args, "--repo", opt.Repo)
	}
	if opt.Values != "" {
		args = append(args, "--values", opt.Values)
	}
	if opt.CreateNamespace {
		args = append(args, "--create-namespace")
	}
	if opt.Namespace != "" {
		args = append(args, "--namespace", opt.Namespace)
	}
	if opt.Version != "" {
		args = append(args, "--version", opt.Version)
	}
	args = append(args, opt.Name, opt.Chart)
	fmt.Println("> helm upgrade", args)
	cmd := exec.Command("helm", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if opt.KubeConfig != "" {
		cmd.Env = appendEnv(os.Environ(), "KUBECONFIG", opt.KubeConfig)
	}
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "helm upgrade")
	}
	return nil
}
