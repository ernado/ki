package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-faster/errors"
	"gopkg.in/yaml.v3"
)

//go:embed main.tf
var mainTerraform string

func run() error {
	var arg struct {
		Token         string
		PublicKeyPath string
	}
	var defaultPublicKey string
	if home, err := os.UserHomeDir(); err == nil {
		defaultPublicKey = filepath.Join(home, ".ssh", "id_ed25519.pub")
	}
	flag.StringVar(&arg.Token, "token", os.Getenv("HETZNER_TOKEN"), "Hetzner token ($HETZNER_TOKEN)")
	flag.StringVar(&arg.PublicKeyPath, "pubkey", defaultPublicKey, "Host public key")
	flag.Parse()

	if arg.Token == "" {
		return errors.New("no token provided")
	}

	fmt.Println("> Preparing terraform in current directory")

	fmt.Println("> Writing main.tf")
	if err := os.WriteFile("main.tf", []byte(mainTerraform), 0600); err != nil {
		return errors.Wrap(err, "write main.tf")
	}

	fmt.Println("> Checking for shh keys")
	if _, stat := os.Stat("worker_ed25519"); stat != nil {
		fmt.Println("> Generating ed25519 key")
		cmd := exec.Command("ssh-keygen",
			"-t", "ed25519",
			"-f", "worker_ed25519",
			"-N", "",
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return errors.Wrap(err, "ssh-keygen")
		}
	} else {
		fmt.Println("> Key already exist")
	}

	workerPublicKey, err := os.ReadFile("worker_ed25519.pub")
	if err != nil {
		return errors.Wrap(err, "read worker public key")
	}
	workerPrivateKey, err := os.ReadFile("worker_ed25519")
	if err != nil {
		return errors.Wrap(err, "read worker private key")
	}

	hostPublicKey, err := os.ReadFile(arg.PublicKeyPath)
	if err != nil {
		return errors.Wrap(err, "read host public key")
	}

	fmt.Println("> Generating cloud init script for worker")
	kiLink := "https://github.com/ernado/ki/releases/download/v0.6.1/ki-linux-amd64.tar.gz"
	cloudInitWorkerConfig := CloudConfig{
		Packages: []string{"curl"},
		Users: []User{
			{
				Name:  "cluster",
				Sudo:  "ALL=(ALL) NOPASSWD:ALL",
				Shell: "/bin/bash",
				SSHAuthorizedKeys: []string{
					string(hostPublicKey),
				},
			},
		},
		WriteFiles: []File{
			{
				Path:        "/root/.ssh/id_ed25519",
				Content:     string(workerPrivateKey),
				Permissions: "0600",
			},
		},
		RunCmd: []string{
			"wget " + kiLink,
			"tar -xvf ki-linux-amd64.tar.gz",
			"mv ki /usr/local/bin/ki",
			"ki --install --join",
		},
	}
	cloudInitWorkerData, err := yaml.Marshal(cloudInitWorkerConfig)
	if err != nil {
		return errors.Wrap(err, "marshal")
	}
	if err := os.WriteFile("cloud-init-worker.yaml", cloudInitWorkerData, 0600); err != nil {
		return errors.Wrap(err, "cloud init worker write")
	}

	fmt.Println("> Generating cloud init script for control plane")
	cloudInitControlPlaneConfig := CloudConfig{
		Packages: []string{"curl"},
		Users: []User{
			{
				Name:  "cluster",
				Sudo:  "ALL=(ALL) NOPASSWD:ALL",
				Shell: "/bin/bash",
				SSHAuthorizedKeys: []string{
					string(hostPublicKey),
					string(workerPublicKey),
				},
			},
		},
		WriteFiles: []File{
			{
				Path:        "/root/.hcloud",
				Content:     arg.Token,
				Permissions: "0600",
			},
		},
		RunCmd: []string{
			"wget " + kiLink,
			"tar -xvf ki-linux-amd64.tar.gz",
			"mv ki /usr/local/bin/ki",
			"ki --install",
		},
	}
	cloudInitControlPlaneData, err := yaml.Marshal(cloudInitControlPlaneConfig)
	if err != nil {
		return errors.Wrap(err, "marshal")
	}
	if err := os.WriteFile("cloud-init.yaml", cloudInitControlPlaneData, 0600); err != nil {
		return errors.Wrap(err, "cloud init control plane write")
	}

	fmt.Println("> Write .tfvars")
	{
		var b strings.Builder
		b.WriteString(`hcloud_token = "`)
		b.WriteString(arg.Token)
		b.WriteRune('"')
		b.WriteRune('\n')
		if err := os.WriteFile(".tfvars", []byte(b.String()), 0600); err != nil {
			return errors.Wrap(err, "write tfvars")
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
