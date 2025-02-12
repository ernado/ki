package main

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	hcl "github.com/alecthomas/hcl/v2"
	"github.com/go-faster/errors"
	"gopkg.in/yaml.v3"
)

func marshal(v interface{}) ([]byte, error) {
	b := new(bytes.Buffer)
	b.WriteString("#cloud-config\n")
	e := yaml.NewEncoder(b)
	e.SetIndent(2)
	if err := e.Encode(v); err != nil {
		return nil, errors.Wrap(err, "encode")
	}
	return b.Bytes(), nil
}

//go:embed main.tf
var mainTerraform string

func run() error {
	var arg struct {
		Token                string
		PublicKeyPath        string
		WorkerNodeType       string
		WorkerNodeCount      int
		SSHKeyName           string
		ControlPlaneNodeType string
		Location             string
	}
	var defaultPublicKey string
	if home, err := os.UserHomeDir(); err == nil {
		defaultPublicKey = filepath.Join(home, ".ssh", "id_ed25519.pub")
	}
	flag.StringVar(&arg.Token, "token", os.Getenv("HETZNER_TOKEN"), "Hetzner token ($HETZNER_TOKEN)")
	flag.StringVar(&arg.PublicKeyPath, "pubkey", defaultPublicKey, "Host public key")
	flag.StringVar(&arg.WorkerNodeType, "worker-type", "cpx11", "Worker node type")
	flag.IntVar(&arg.WorkerNodeCount, "worker-count", 1, "Worker node count")
	flag.StringVar(&arg.ControlPlaneNodeType, "control-plane-type", "cpx11", "Control plane node type")
	flag.StringVar(&arg.Location, "location", "hel1", "Location")
	flag.StringVar(&arg.SSHKeyName, "ssh-key-name", "ki", "SSH key name")
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
	kiLink := "https://github.com/ernado/ki/releases/download/v0.8.0/ki-linux-amd64.tar.gz"
	cloudInitWorkerConfig := CloudConfig{
		Packages: []string{"curl", "wget"},
		Users: []User{
			{
				Name:  "cluster",
				Sudo:  "ALL=(ALL) NOPASSWD:ALL",
				Shell: "/bin/bash",
				SSHAuthorizedKeys: []string{
					strings.TrimSpace(string(hostPublicKey)),
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
	cloudInitWorkerData, err := marshal(cloudInitWorkerConfig)
	if err != nil {
		return errors.Wrap(err, "marshal")
	}
	if err := os.WriteFile("cloud-init-worker.yaml", cloudInitWorkerData, 0600); err != nil {
		return errors.Wrap(err, "cloud init worker write")
	}

	fmt.Println("> Generating cloud init script for control plane")
	cloudInitControlPlaneConfig := CloudConfig{
		Packages: []string{"curl", "wget"},
		Users: []User{
			{
				Name:  "cluster",
				Sudo:  "ALL=(ALL) NOPASSWD:ALL",
				Shell: "/bin/bash",
				SSHAuthorizedKeys: []string{
					strings.TrimSpace(string(hostPublicKey)),
					strings.TrimSpace(string(workerPublicKey)),
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
	cloudInitControlPlaneData, err := marshal(cloudInitControlPlaneConfig)
	if err != nil {
		return errors.Wrap(err, "marshal")
	}
	if err := os.WriteFile("cloud-init.yaml", cloudInitControlPlaneData, 0600); err != nil {
		return errors.Wrap(err, "cloud init control plane write")
	}

	fmt.Println("> Write .tfvars")
	{
		type Config struct {
			Location         string `hcl:"location"`
			WorkerType       string `hcl:"worker_type"`
			WorkerCount      int    `hcl:"worker_count"`
			SSHKeyName       string `hcl:"ssh_key_name"`
			ControlPlaneType string `hcl:"control_plane_type"`
			Token            string `hcl:"hcloud_token"`
		}
		data, err := hcl.Marshal(&Config{
			Location:         arg.Location,
			WorkerType:       arg.WorkerNodeType,
			WorkerCount:      arg.WorkerNodeCount,
			ControlPlaneType: arg.ControlPlaneNodeType,
			SSHKeyName:       arg.SSHKeyName,
			Token:            arg.Token,
		})
		if err != nil {
			return errors.Wrap(err, "marshal tfvars")
		}
		if err := os.WriteFile(".tfvars", data, 0600); err != nil {
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
