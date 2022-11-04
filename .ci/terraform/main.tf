terraform {
  backend "gcs" {
    bucket = "sap-fgl-gcve-pub-preview-terraform"
    prefix = "peering/state"
  }
  required_providers {
    nsxt = {
      source = "vmware/nsxt"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
  zone    = var.zone
}

# VPC
data "google_compute_network" "vpc" {
  name = "default"
}

# Subnet
data "google_compute_subnetwork" "subnet" {
  name = var.region
}

data "google_client_config" "current" {}

# GKE cluster
resource "google_container_cluster" "cluster" {
  name     = "${var.project_id}-gke"
  project  = var.project_id
  location = var.region

  # We can't create a cluster with no node pool defined, but we want to only use
  # separately managed node pools. So we create the smallest possible default
  # node pool and immediately delete it.
  remove_default_node_pool = true
  initial_node_count       = 1


  network    = data.google_compute_network.vpc.name
  subnetwork = data.google_compute_subnetwork.subnet.name

  node_locations = [
    var.zone,
  ]
}

# Separately Managed Node Pool
resource "google_container_node_pool" "primary_nodes" {
  name       = "${google_container_cluster.cluster.name}-node-pool"
  project    = var.project_id
  location   = var.region
  cluster    = google_container_cluster.cluster.name
  node_count = var.gke_num_nodes

  node_config {
    oauth_scopes = [
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring",
    ]

    labels = {
      env = var.project_id
    }

    machine_type = "n1-standard-4"
    tags         = ["gke-node", "${var.project_id}-gke"]
    metadata     = {
      disable-legacy-endpoints = "true"
    }
  }
}

data "google_compute_instance_group" "node_ig" {
  name = regex("(?:.*/)+(.*)", google_container_node_pool.primary_nodes.instance_group_urls.0).0
}

module "gke_auth" {
  source = "terraform-google-modules/kubernetes-engine/google//modules/auth"

  project_id           = var.project_id
  cluster_name         = google_container_cluster.cluster.name
  location             = var.region
  use_private_endpoint = false
}

resource "local_file" "kubeconfig" {
  content  = module.gke_auth.kubeconfig_raw
  filename = "/tmp/kubeconfig"
}

resource "google_storage_bucket" "tm-storage" {
  name          = format("%s-tm-storage", var.project_id)
  force_destroy = true
  location      = var.region
}

resource "google_storage_bucket_iam_binding" "binding" {
  bucket  = google_storage_bucket.tm-storage.name
  role    = "roles/storage.admin"
  members = [
    "serviceAccount:${var.sa_email}",
  ]
}

provider "kubernetes" {
  host                   = "https://${google_container_cluster.cluster.endpoint}"
  cluster_ca_certificate = base64decode(google_container_cluster.cluster.master_auth.0.cluster_ca_certificate)
  token                  = data.google_client_config.current.access_token
}

resource "google_storage_hmac_key" "key" {
  service_account_email = var.sa_email
}

provider "helm" {
  kubernetes {
    host                   = "https://${google_container_cluster.cluster.endpoint}"
    cluster_ca_certificate = base64decode(google_container_cluster.cluster.master_auth.0.cluster_ca_certificate)
    token                  = data.google_client_config.current.access_token
  }
}

resource "helm_release" "tm-secrets" {
  name       = "testmachinery-secrets"
  chart      = "charts/testmachinery-secrets"
  depends_on = [google_container_node_pool.primary_nodes, google_container_cluster.cluster]

  values = [
    file(var.privatecloud_cred)
  ]
  set {
    name  = "sa_email"
    value = var.sa_email
  }
}

resource "helm_release" "tm" {
  name       = "testmachinery"
  chart      = "${var.tm_repo_path}/charts/testmachinery"
  depends_on = [helm_release.tm-secrets]

  values = [
    file("${path.module}/local-values.yaml")
  ]
  set {
    name  = "testmachinery.local"
    value = "true"
  }
  set {
    name  = "testmachinery.insecure"
    value = "true"
  }
  set {
    name  = "controller.hostPath"
    value = "/tmp/tm"
  }
  set {
    name  = "controller.serviceAccountName"
    value = "vsphere-tm-sa"
  }
  set {
    name  = "global.s3Configuration.accessKey"
    value = google_storage_hmac_key.key.access_id
  }
  set {
    name  = "global.s3Configuration.secretKey"
    value = google_storage_hmac_key.key.secret
  }
  set {
    name  = "global.s3Configuration.bucketName"
    value = google_storage_bucket.tm-storage.name
  }
  set {
    name  = "global.s3Configuration.server.endpoint"
    value = "storage.googleapis.com"
  }
}

locals {
  privatecloud_cred_obj = yamldecode(file(var.privatecloud_cred))
}

data "google_client_openid_userinfo" "me" {
}

data "google_service_account" "me" {
  account_id = data.google_client_openid_userinfo.me.id
}

resource "local_file" "shell" {
  content = templatefile("shell.tftpl", {
    zone     = var.zone
    project  = var.project_id
    username = "sa_${data.google_service_account.me.unique_id}"
    bastion  = tolist(regex("(?:.*/)+(.*)", tolist(data.google_compute_instance_group.node_ig.instances).0)).0
    target   = local.privatecloud_cred_obj["privateCloud"]["nsx"]["internalip"]
  })
  filename = "/tmp/shell.sh"
}

provider "nsxt" {
  host                 = "127.0.0.1:8443"
  username             = local.privatecloud_cred_obj["nsxCredentials"]["username"]
  password             = local.privatecloud_cred_obj["nsxCredentials"]["password"]
  allow_unverified_ssl = true
  max_retries          = 2
}

resource "nsxt_policy_ip_block" "block1" {
  display_name = "ip-block1"
  cidr         = cidrsubnet(local.privatecloud_cred_obj["privateCloud"]["networkconfig"]["managementcidr"], 5, 2)
  depends_on   = [google_container_node_pool.primary_nodes]
}

resource "nsxt_policy_ip_pool" "pool1" {
  display_name = "snat-ippool" # this is used by .ci/terraform/charts/testmachinery-secrets/templates/secrets.yaml
  depends_on   = [google_container_node_pool.primary_nodes]
}

resource "nsxt_policy_ip_pool_block_subnet" "block_subnet1" {
  display_name        = "block-subnet1"
  pool_path           = nsxt_policy_ip_pool.pool1.path
  block_path          = nsxt_policy_ip_block.block1.path
  size                = 8
  auto_assign_gateway = false
  depends_on          = [google_container_node_pool.primary_nodes]
}
