# ki

WIP: Opinionated Kubernetes Installer.

This can be done in Ansible, but I'm doing it myself in Go.

## Links

- https://v1-31.docs.kubernetes.io/docs/setup/production-environment/tools/kubeadm/install-kubeadm/
- https://www.linkedin.com/pulse/step-by-step-guide-installing-kubernetes-ubuntu-2404-lts-jayaraman-okozc

## What it should do

- [x] Disable swap
- [x] Configure kernel modules and parameters
- [x] Install conteinerd
- [x] Install kubelet, kubeadm, kubectl
- [ ] Initialize the cluster
- [ ] Install a CNI
