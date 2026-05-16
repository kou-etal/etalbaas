# kubespray - Kubernetes Cluster Provisioning

## Prerequisites

- 3 VPS nodes with SSH access (root)
- Ubuntu 22.04 or 24.04 LTS
- Python 3.10+ on control machine
- SSH key registered on all nodes

## Setup

```bash
# Clone kubespray (version pinned)
cd deploy/kubespray
git clone -b v2.25.0 https://github.com/kubernetes-sigs/kubespray.git kubespray
cd kubespray
pip install -r requirements.txt
```

## Configuration

1. Update `inventory/production/hosts.yaml` with actual node IPs
2. Update `kube_vip_address` in `group_vars/k8s_cluster/k8s-cluster.yml`
3. Update `supplementary_addresses_in_ssl_keys` with the VIP

## Cluster Provisioning

```bash
cd deploy/kubespray/kubespray
ansible-playbook -i ../inventory/production/hosts.yaml cluster.yml
```

Estimated time: 15-20 minutes.

## Post-Install

After kubespray completes, run the post-install bootstrap:

```bash
cd deploy/post-install
ansible-galaxy collection install -r requirements.yml
ansible-playbook playbooks/bootstrap.yml
```

## Node Topology

| Node | Spec | Role |
|------|------|------|
| node-01 | 8GB+ RAM, 6 vCPU | Control Plane + Worker |
| node-02 | 8GB+ RAM, 6 vCPU | Control Plane + Worker |
| node-03 | 2GB+ RAM, 2 vCPU | Control Plane only (etcd voter, NoSchedule) |

## VPS Provider Compatibility (kube-vip ARP mode)

| Provider | L2 Support | Notes |
|----------|------------|-------|
| X Server VPS | OK | Same network, ARP works |
| Hetzner Cloud | OK | Private Network enables L2 |
| DigitalOcean | Partial | VPC + Reserved IP required |
| Vultr | Partial | Reserved IP required |
| AWS Lightsail | No | L2 unavailable, use MetalLB BGP |

## Phase 2 Scale Path

- Add worker nodes (node-04, node-05) to `kube_node` group
- Move node-01/02/03 to CP-only (remove from `kube_node`)
- Consider MetalLB BGP for multi-provider or L2-incompatible environments

## Disaster Recovery

1. Provision new VPS (update IPs in hosts.yaml)
2. Run `kubespray cluster.yml` (15-20 min)
3. Run `post-install bootstrap.yml` (5 min)
4. ArgoCD restores from Git (5-10 min)
5. CloudNativePG restores DB from R2 WAL backup

Total recovery: ~30-40 minutes.
