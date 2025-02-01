<div>
<img src="https://github.com/kubernetes/kubernetes/raw/master/logo/logo.svg" align="left" width="144px" height="144px"/>

#### home-ops - a mono-repo for my homelab

_... automated via [Flux](https://fluxcd.io), [Renovate](https://github.com/renovatebot/renovate) and [GitHub Actions](https://github.com/features/actions)_ 🐟
</div>

<div align="center">

[![Discord](https://img.shields.io/discord/673534664354430999?style=for-the-badge&label&logo=discord&logoColor=white&color=blue)](https://discord.gg/home-operations)&nbsp;&nbsp;

</div>

---

## Overview

This is a monorepository for my home Kubernetes cluster.
I try to adhere to Infrastructure as Code (IaC) and GitOps practices using tools like [Ansible](https://www.ansible.com/),  [Kubernetes](https://kubernetes.io/), [Flux](https://github.com/fluxcd/flux2), [Renovate](https://github.com/renovatebot/renovate), and [GitHub Actions](https://github.com/features/actions).

The purpose here is to learn Kubernetes, while practicing GitOps. I have two longer-term goals:

1. migrate many of the services that I currently run on different raspberry pi's or servers around the house into a single, congruous k8s environment
2. Learn more about managing K8s clusters in prod
3. Get some decent NVME SSD's to use as boot drives
4. Set up ceph instead of using NFS for PVCs

---

## ⛵ Kubernetes

There is a template over at [onedr0p/flux-cluster-template](https://github.com/onedr0p/flux-cluster-template) if you want to try and follow along with some of the practices I use here.

### Installation

My cluster is built using [k3s](https://k3s.io/), provisioned on bare-metal debian Linux using the [Ansible](https://www.ansible.com/) galaxy role [ansible-role-k3s](https://github.com/PyratLabs/ansible-role-k3s). This is a hyper-converged cluster, workloads and block storage are sharing the same available resources on my nodes. I also have a separate NAS server with ZFS for NFS shares, bulk file storage and backups.

## Core Components

- [actions-runner-controller](https://github.com/actions/actions-runner-controller): self-hosted Github runners
- [cert-manager](https://cert-manager.io/docs/): Configured to create TLS certs for all ingress services automatically using LetsEncrypt.
- [external-dns](https://github.com/kubernetes-sigs/external-dns): monitors service and ingress resources, and automatically generates DNS updates for them. This lets me maintain DNS mappings and LetsEncrypt certificates without a cloudflare account or domain.
- [external-secrets](https://github.com/external-secrets/external-secrets/): managed Kubernetes secrets using [1Password](https://1password.com/).
- [ingress-nginx](https://github.com/kubernetes/ingress-nginx/): ingress controller for Kubernetes using NGINX as a reverse proxy and load balancer
- [sops](https://toolkit.fluxcd.io/guides/mozilla-sops/): managed secrets for Kubernetes, Ansible, and Terraform which are committed to Git

### GitOps

[Flux](https://github.com/fluxcd/flux2) watches the clusters in my [kubernetes](./kubernetes/) folder (see Directories below) and makes the changes to my clusters based on the state of my Git repository.

The way Flux works for me here is it will recursively search the `kubernetes/apps` folder until it finds the most top level `kustomization.yaml` per directory and then apply all the resources listed in it. That aforementioned `kustomization.yaml` will generally only have a namespace resource and one or many Flux kustomizations. Those Flux kustomizations will generally have a `HelmRelease` or other resources related to the application underneath it which will be applied.

[Renovate](https://github.com/renovatebot/renovate) watches my **entire** repository looking for dependency updates, when they are found a PR is automatically created. When some PRs are merged Flux applies the changes to my cluster.

### Directories

This Git repository contains the following directories under [Kubernetes](./kubernetes/).

```sh
📁 kubernetes
├── 📁 apps           # applications
├── 📁 bootstrap      # bootstrap procedures
└── 📁 components     # re-useable components
├── 📁 flux           # core flux configuration
```

---

## ☁️ Cloud Dependencies

While most of my infrastructure and workloads are self-hosted I do rely upon the cloud for certain key parts of my setup. This saves me from having to worry about two things. (1) Dealing with chicken/egg scenarios and (2) services I critically need whether my cluster is online or not.

The alternative solution to these two problems would be to host a Kubernetes cluster in the cloud and deploy applications like [HCVault](https://www.vaultproject.io/), [Vaultwarden](https://github.com/dani-garcia/vaultwarden), [ntfy](https://ntfy.sh/), and [Gatus](https://gatus.io/). However, maintaining another cluster and monitoring another group of workloads is a lot more time and effort than I am willing to put in.

| Service                                         | Use                                                               | Cost           |
|-------------------------------------------------|-------------------------------------------------------------------|----------------|
| [1Password](https://1password.com/)             | Secrets with [External Secrets](https://external-secrets.io/)     | ~$60/yr        |
| [Cloudflare](https://www.cloudflare.com/)       | Domain and S3                                                     | Free           |
| [GitHub](https://github.com/)                   | Hosting this repository and continuous integration/deployments    | Free           |
|                                                 |                                                                   | Total: ~$5/mo  |

---

## 🌐 DNS

### Home DNS

In cluster I run [blocky](https://github.com/0xERR0R/blocky) deployed. In my cluster `external-dns` is deployed with the `RFC2136` provider which syncs DNS records to. `blocky` is used by non-servers as ad-blocking and caching proxy.

### Public DNS

The `external-dns` instance mentioned above another instance is deployed in my cluster and configured to sync DNS records to [Cloudflare](https://www.cloudflare.com/). The only ingress this `external-dns` instance looks at to gather DNS records to put in `Cloudflare` are ones that have an ingress class name of `external` and contain an ingress annotation `external-dns.alpha.kubernetes.io/target`.

---

## 🔧 Hardware

### Main Kubernetes Cluster

| Node                | CPU               |  RAM | Storage    | Function             | OS         |
|---------------------|-------------------|------|------------|----------------------|------------|
| Beelink EQ13 | Intel N200   | 16GB | 512Gib SSD  | control-plane        | Talos |
| Beelink EQ13 | Intel N200   | 16GB | 512Gib SSD  | control-plane        | Talos |
| Beelink EQ13 | Intel N200   | 16GB | 512Gib SSD  | control-plane        | Talos |

### Bulk storage

| Node                | CPU               |  RAM | Storage    | Function             | OS         |
|---------------------|-------------------|------|------------|----------------------|------------|
| CM3588 | Quad-core ARM Cortex-A76 and quad-core Cortex-A55 CPU    | 16GB | 4* 4 Tib SSD  | Bulk storage        | Debian |

Big shout out to original [flux-cluster-template](https://github.com/onedr0p/flux-cluster-template), and the [Home Operations](https://discord.gg/home-operations) Discord community.

Be sure to check out [kubesearch.dev](https://kubesearch.dev/) for ideas on how to deploy applications or get ideas on what you may deploy.
