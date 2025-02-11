# ki

WIP: Opinionated Kubernetes Installer.

This can be done in Ansible, but I'm doing it myself in Go.

## Links

- https://v1-31.docs.kubernetes.io/docs/setup/production-environment/tools/kubeadm/install-kubeadm/
- https://www.linkedin.com/pulse/step-by-step-guide-installing-kubernetes-ubuntu-2404-lts-jayaraman-okozc
- https://community.hetzner.com/tutorials/setup-your-own-scalable-kubernetes-cluster

## What it should do

- [x] Disable swap
- [x] Configure kernel modules and parameters
- [x] Install conteinerd
- [x] Install kubelet, kubeadm, kubectl
- [x] Install helm, cilium
- [x] Initialize the cluster
- [x] Install a CNI (Cilium)
- [x] Join cluster

### Usage

#### Create cluster

```yaml
runcmd:
  - wget https://github.com/ernado/ki/releases/download/v0.1.1/ki-linux-amd64.tar.gz
  - tar -xvf ki-linux-amd64.tar.gz
  - mv ki /usr/local/bin/ki
  - ki --install
```

Control node should have `10.0.1.1` as internal IP.
You can specify it by `-control-plane-internal-ip` flag.

#### Join cluster

```yaml
runcmd:
  - wget https://github.com/ernado/ki/releases/download/v0.1.1/ki-linux-amd64.tar.gz
  - tar -xvf ki-linux-amd64.tar.gz
  - mv ki /usr/local/bin/ki
  - ki --join --install
```

## TODO

```bash
W0211 15:38:05.996784   19812 checks.go:846] detected that the sandbox image "registry.k8s.io/pause:3.8" of the container runtime is inconsistent with that used by kubeadm.It is recommended to use "registry.k8s.io/pause:3.10" as the CRI sandbox image.
```
