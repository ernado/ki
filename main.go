package main

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/go-faster/errors"
)

func appendEnv(vars []string, key, value string) []string {
	return append(vars, key+"="+value)
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

func Systemctl(action, service string) error {
	fmt.Printf("> systemctl %s %s\n", action, service)
	cmd := exec.Command("systemctl", action, service)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "systemctl")
	}
	return nil
}

//go:embed cilium.yml.tmpl
var ciliumConfigTemplate string

type CiliumConfig struct {
	K8sServiceHost string // 1.1.1.1
}

type CiliumInstallOptions struct {
	Version        string
	K8sServiceHost string
}

func CiliumInstall(opt CiliumInstallOptions) error {
	// Should be installed via helm.
	// helm upgrade --version 1.13.2 --install --create-namespace --namespace "cilium" cilium cilium/cilium --values cilium.yml
	// 1. Render template.
	tmpl, err := template.New("cilium.yml").Parse(ciliumConfigTemplate)
	if err != nil {
		return errors.Wrap(err, "parse template")
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, CiliumConfig{
		K8sServiceHost: opt.K8sServiceHost,
	}); err != nil {
		return errors.Wrap(err, "execute template")
	}

	// Write to file.
	fileName := "cilium.yml"
	fmt.Printf("> Writing %s\n", fileName)
	if err := os.WriteFile(fileName, buf.Bytes(), 0600); err != nil {
		return errors.Wrap(err, "write")
	}

	if err := HelmUpgrade(HelmUpgradeOptions{
		Version:         opt.Version,
		Name:            "cilium",
		Install:         true,
		Namespace:       "cilium",
		CreateNamespace: true,
		Chart:           "cilium/cilium",
		Values:          fileName,
	}); err != nil {
		return errors.Wrap(err, "helm upgrade")
	}

	return nil
}

func ConfigureContainerd() error {
	// 1. Get default config.
	cmd := exec.Command("containerd", "config", "default")
	out, err := cmd.Output()
	if err != nil {
		return errors.Wrap(err, "containerd config default")
	}
	// 2. Update config.
	// sudo sed -i 's/SystemdCgroup = false/SystemdCgroup = true/g' /etc/containerd/config.toml
	out = bytes.ReplaceAll(out, []byte("SystemdCgroup = false"), []byte("SystemdCgroup = true"))
	// Write back.
	fileName := "/etc/containerd/config.toml"
	fmt.Printf("> Writing %s\n", fileName)
	if err := os.WriteFile(fileName, out, 0600); err != nil {
		return errors.Wrap(err, "write")
	}
	// 3. Restart containerd.
	if err := Systemctl("restart", "containerd"); err != nil {
		return errors.Wrap(err, "restart containerd")
	}
	// 4. Enable containerd.
	if err := Systemctl("enable", "containerd"); err != nil {
		return errors.Wrap(err, "enable containerd")
	}
	fmt.Println("> Configured, restarted and enabled containerd")
	return nil
}

const serviceMonitorCRD = "https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/" +
	"main/example/prometheus-operator-crd/monitoring.coreos.com_servicemonitors.yaml"

func SetupKubeconfig() error {
	const kubeConfig = "/etc/kubernetes/admin.conf"
	fmt.Println("> Setting up kubeconfig")
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return errors.Wrap(err, "user home dir")
	}
	kubeDir := filepath.Join(homeDir, ".kube")
	if err := os.MkdirAll(kubeDir, 0750); err != nil {
		return errors.Wrap(err, "mkdir")
	}
	data, err := os.ReadFile(kubeConfig)
	if err != nil {
		return errors.Wrap(err, "read")
	}
	if err := os.WriteFile(filepath.Join(kubeDir, "config"), data, 0600); err != nil {
		return errors.Wrap(err, "write")
	}
	fmt.Println("> Kubeconfig is ready")
	return nil
}

func run() error {
	var arg struct {
		Version                string
		Join                   bool
		CiliumVersion          string
		CiliumCliVersion       string
		CiliumCliSHA256        string
		HelmVersion            string
		HelmSHA256             string
		ControlPlaneInternalIP string
	}
	flag.StringVar(&arg.Version, "version", "v1.31", "kubernetes version")
	flag.StringVar(&arg.HelmVersion, "helm-version", "v3.17.0", "helm version")
	flag.StringVar(&arg.HelmSHA256, "helm-sha256", "fb5d12662fde6eeff36ac4ccacbf3abed96b0ee2de07afdde4edb14e613aee24", "helm sha256")
	flag.StringVar(&arg.CiliumVersion, "cilium-version", "1.17.0", "cilium version")
	flag.StringVar(&arg.CiliumCliVersion, "cilium-cli-version", "v0.16.24", "cilium cli version")
	flag.StringVar(&arg.CiliumCliSHA256, "cilium-cli-sha256", "019c9c765222b3db5786f7b3a0bff2cd62944a8ce32681acfb47808330f405a7", "cilium cli sha256")
	flag.BoolVar(&arg.Join, "join", false, "join cluster")
	flag.StringVar(&arg.ControlPlaneInternalIP, "control-plane-internal-ip", "10.0.1.1", "control plane internal ip")
	flag.Parse()

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
	defaultGateway, err := GetDefaultGatewayIP()
	if err != nil {
		return errors.Wrap(err, "get default gateway")
	}
	fmt.Println("> Default gateway:", defaultGateway)
	// Check required ports
	if err := CheckTCPPortIsFree(6443); err != nil {
		return errors.Wrap(err, "check k8s port")
	}
	if err := InstallBinary(Binary{
		Name:   "helm",
		URL:    "https://get.helm.sh/helm-" + arg.HelmVersion + "-linux-amd64.tar.gz",
		SHA256: arg.HelmSHA256,
	}); err != nil {
		return errors.Wrap(err, "install helm")
	}

	// https://github.com/cilium/cilium-cli/releases/
	if err := InstallBinary(Binary{
		Name:   "cilium",
		URL:    "https://github.com/cilium/cilium-cli/releases/download/" + arg.CiliumCliVersion + "/cilium-linux-amd64.tar.gz",
		SHA256: arg.CiliumCliSHA256,
	}); err != nil {
		return errors.Wrap(err, "install cilium")
	}

	// Swap configuration
	if err := DisableSwap(); err != nil {
		return errors.Wrap(err, "disable swap")
	}
	//  Update apt cache
	if err := APTUpdate(); err != nil {
		return errors.Wrap(err, "apt update")
	}
	//  Upgrade packages
	if err := APTUpgrade(); err != nil {
		return errors.Wrap(err, "apt upgrade")
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
	if err := APTAddRepo(APTAddRepoOptions{
		Name:       "docker",
		URL:        "https://download.docker.com/linux/ubuntu",
		SignedBy:   "/etc/apt/keyrings/docker.gpg",
		Arch:       []string{"amd64"},
		Components: []string{release, "stable"},
	}); err != nil {
		return errors.Wrap(err, "add docker repo")
	}
	if err := APTUpdate(); err != nil {
		return errors.Wrap(err, "apt update")
	}
	if err := APTInstall("curl", "containerd.io"); err != nil {
		return errors.Wrap(err, "install containerd")
	}
	if err := ConfigureContainerd(); err != nil {
		return errors.Wrap(err, "configure containerd")
	}
	// Install k8s
	fmt.Println("> Installing k8s")
	if err := APTKey("k8s", "https://pkgs.k8s.io/core:/stable:/"+arg.Version+"/deb/Release.key"); err != nil {
		return errors.Wrap(err, "add k8s key")
	}
	if err := APTAddRepo(APTAddRepoOptions{
		Name:       "k8s",
		URL:        "https://pkgs.k8s.io/core:/stable:/" + arg.Version + "/deb/",
		SignedBy:   "/etc/apt/keyrings/k8s.gpg",
		Components: []string{"/"},
	}); err != nil {
		return errors.Wrap(err, "add k8s repo")
	}
	if err := APTUpdate(); err != nil {
		return errors.Wrap(err, "apt update")
	}
	if err := APTInstall("kubeadm", "kubelet", "kubectl"); err != nil {
		return errors.Wrap(err, "install k8s")
	}
	if err := APTHold("kubeadm", "kubelet", "kubectl"); err != nil {
		return errors.Wrap(err, "hold k8s")
	}
	// Enable and start kubelet
	fmt.Println("> Starting kubelet")
	if err := Systemctl("enable", "kubelet"); err != nil {
		return errors.Wrap(err, "enable kubelet")
	}
	if err := Systemctl("start", "kubelet"); err != nil {
		return errors.Wrap(err, "start kubelet")
	}
	// Initialize k8s
	fmt.Println("> Initializing k8s")
	if arg.Join {
		if err := KubeadmJoin(arg.ControlPlaneInternalIP); err != nil {
			return errors.Wrap(err, "kubeadm join")
		}
		fmt.Println("> Joined")
		return nil
	}
	if err := KubeadmInit(KubeadmInitOptions{
		SkipPhases:           []string{"addon/kube-proxy"},
		PodNetworkCIDR:       "10.244.0.0/16",
		ServiceCIDR:          "10.96.0.0/16",
		ControlPlaneEndpoint: defaultGateway,
		ExtraSans:            []string{arg.ControlPlaneInternalIP},
	}); err != nil {
		return errors.Wrap(err, "kubeadm init")
	}
	if err := SetupKubeconfig(); err != nil {
		return errors.Wrap(err, "setup kubeconfig")
	}
	// Install cilium.
	if err := KubectlApply(KubectlApplyOptions{
		File: serviceMonitorCRD,
	}); err != nil {
		return errors.Wrap(err, "kubectl apply service monitor CRD")
	}
	fmt.Println("> Installing cilium")
	if err := HelmAddRepo("cilium", "https://helm.cilium.io"); err != nil {
		return errors.Wrap(err, "helm add repo")
	}
	if err := CiliumInstall(CiliumInstallOptions{
		Version:        arg.CiliumVersion,
		K8sServiceHost: defaultGateway,
	}); err != nil {
		return errors.Wrap(err, "cilium install")
	}
	fmt.Println("> Done")
	return nil
}

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
		os.Exit(1)
	}
}
