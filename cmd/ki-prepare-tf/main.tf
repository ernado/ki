# Tell Terraform to include the hcloud provider
terraform {
  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      # Here we use version 1.48.0, this may change in the future
      version = "1.48.0"
    }
  }
}

# Declare the hcloud_token variable from .tfvars
variable "hcloud_token" {
  sensitive = true # Requires terraform >= 0.14
}

variable "location" {
  description = "Location for the Hetzner Cloud resources"
  default = "hel1"
}

variable "worker_type" {
  description = "Type of server for the worker nodes"
  default = "cx22"
}

variable "master_type" {
  description = "Type of server for the master node"
  default = "cx22"
}

variable "ssh_key_name" {
  description = "Name of the SSH key to use"
  default = "nexus"
}

variable "worker_count" {
  description = "Number of worker nodes to create"
  default = 3
}

# Configure the Hetzner Cloud Provider with your token
provider "hcloud" {
  token = var.hcloud_token
}

resource "hcloud_network" "private_network" {
  name     = "kubernetes-cluster"
  ip_range = "10.0.0.0/16"
}

resource "hcloud_network_subnet" "private_network_subnet" {
  type         = "cloud"
  network_id   = hcloud_network.private_network.id
  network_zone = "eu-central"
  ip_range     = "10.0.1.0/24"
}

resource "hcloud_server" "master-node" {
  name        = "master-node"
  image       = "ubuntu-24.04"
  server_type = var.master_type
  location    = var.location
  public_net {
    ipv4_enabled = true
    ipv6_enabled = true
  }
  network {
    network_id = hcloud_network.private_network.id
    # IP Used by the master node, needs to be static
    # Here the worker nodes will use 10.0.1.1 to communicate with the master node
    ip         = "10.0.1.1"
  }
  user_data = file("${path.module}/cloud-init.yaml")
  ssh_keys = [ var.ssh_key_name ]

  # If we don't specify this, Terraform will create the resources in parallel
  # We want this node to be created after the private network is created
  depends_on = [hcloud_network_subnet.private_network_subnet]
}

resource "hcloud_server" "worker-nodes" {
  count = var.worker_count

  # The name will be worker-node-0, worker-node-1, worker-node-2...
  name        = "worker-node-${count.index}"
  image       = "ubuntu-24.04"
  server_type = var.worker_type
  location    = var.location
  public_net {
    ipv4_enabled = true
    ipv6_enabled = true
  }
  network {
    network_id = hcloud_network.private_network.id
  }
  user_data = file("${path.module}/cloud-init-worker.yaml")
  ssh_keys = [ var.ssh_key_name ]

  depends_on = [hcloud_network_subnet.private_network_subnet, hcloud_server.master-node]
}

resource "hcloud_load_balancer" "load_balancer" {
  name               = "kubernetes-load-balancer"
  load_balancer_type = "lb11"
  location           = var.location

  depends_on = [hcloud_network_subnet.private_network_subnet, hcloud_server.master-node]
}


resource "hcloud_load_balancer_service" "load_balancer_service" {
  load_balancer_id = hcloud_load_balancer.load_balancer.id
  protocol         = "http"
  listen_port      = 80
  destination_port = 8080

  depends_on = [hcloud_load_balancer.load_balancer]
}

resource "hcloud_load_balancer_network" "lb_network" {
  load_balancer_id = hcloud_load_balancer.load_balancer.id
  network_id       = hcloud_network.private_network.id

  depends_on = [hcloud_network_subnet.private_network_subnet, hcloud_load_balancer.load_balancer]
}

resource "hcloud_load_balancer_target" "load_balancer_target_worker" {
  for_each = { for idx, server in hcloud_server.worker-nodes : idx => server }

  type             = "server"
  load_balancer_id = hcloud_load_balancer.load_balancer.id
  server_id        = each.value.id
  use_private_ip   = true

  depends_on = [hcloud_network_subnet.private_network_subnet, hcloud_load_balancer_network.lb_network, hcloud_server.worker-nodes]
}
