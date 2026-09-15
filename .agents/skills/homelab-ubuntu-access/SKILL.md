---
name: homelab-ubuntu-access
description: Access and operate the local Homelab from this Ubuntu workstation, including SSH, Ansible playbooks, VM routing, and safe application deployments. Use for Homelab inspection, deployment, or VM operations; not for unrelated application development.
---

# Homelab Ubuntu Access

The Homelab checkout is `/home/stefan/workspace/homelab`; run Ansible commands from
`/home/stefan/workspace/homelab/ansible`. Its inventory is
`inventories/homelab` and `ansible.cfg` selects it automatically.

## SSH from this Ubuntu workstation

- For Docker VMs, use the existing `~/.ssh/homelab_from_ubuntu` private key.
  `~/.ssh/config` defines `vm-*` aliases, the user `stefan`, and this identity.
  Prefer aliases, for example `ssh vm-infra-dev`.
- The inventory still names `~/.ssh/PVE_id_ed25519`, which is **not present on
  this machine**. Do not edit the inventory just to deploy from this workstation.
  Pass the working key as a one-run Ansible override:

  ```bash
  ansible-playbook playbooks/<playbook>.yml \
    -e "ansible_ssh_private_key_file=/home/stefan/.ssh/homelab_from_ubuntu"
  ```

- The Proxmox host is `pve` at `192.168.188.200` and its inventory user is
  `root`. Treat host-level changes as separate, high-impact work; verify the
  intended account/key and obtain explicit authorization before mutating it.
- Do not print private keys, SOPS-decrypted `.env` files, tokens, credentials,
  or the inventory's health-check URLs in output.

## VM map

| Alias / Ansible host | IP | Role |
| --- | --- | --- |
| `pve` | `192.168.188.200` | Proxmox hypervisor |
| `vm-edge` | `192.168.188.201` | edge services |
| `vm-infra-core` | `192.168.188.208` | core infrastructure |
| `vm-infra-apps` | `192.168.188.206` | shared application infrastructure |
| `vm-infra-dev` | `192.168.188.207` | development application stacks; hosts Kita |
| `vm-documents` | `192.168.188.203` | Nextcloud, Joplin, document/scanning services |
| `vm-photos` | `192.168.188.204` | Immich/photo services |
| `vm-media` | `192.168.188.205` | media services |

All Docker VMs use stacks under `/srv/homelab/stacks/<group>` and persistent
data under `/srv/homelab-data`. Inspect first with `ssh vm-… 'sudo docker ps'`.

## Common operations

Use read-only inspection by default. Get explicit authorization before a command
that deploys, changes configuration, restarts containers, writes data, or
changes infrastructure.

```bash
# Connectivity and basic facts
ansible all -m ping \
  -e "ansible_ssh_private_key_file=/home/stefan/.ssh/homelab_from_ubuntu"

# Deploy a single application after its image/build is known to be available
ansible-playbook playbooks/deploy-app.yml -e "app=<name>" \
  -e "ansible_ssh_private_key_file=/home/stefan/.ssh/homelab_from_ubuntu"

# Deploy one complete VM stack
ansible-playbook playbooks/deploy-stacks.yml --tags <edge|infra-core|infra-apps|infra-dev|documents|photos|media> \
  -e "ansible_ssh_private_key_file=/home/stefan/.ssh/homelab_from_ubuntu"
```

`deploy-app.yml` copies the selected app files, handles encrypted environment
files when applicable, and runs Docker Compose. Read its `app_config` mapping
before using an unfamiliar app name. For Kita, use `app=kita`; it targets
`vm-infra-dev`, deploys `infra-dev/apps/kita`, and controls the database,
management/fees backends, and banking-sync services.

For a normal code deployment: confirm the relevant CI image build is successful,
run the one-app playbook with the key override, then verify affected containers
and their documented health endpoints. If a playbook fails due to the missing
`PVE_id_ed25519`, rerun with the override above; do not create or substitute a
key at that legacy path without the user's direction.

Other playbooks are purpose-specific: `bootstrap.yml`, `deploy-vm.yml`,
`proxmox-host.yml`, networking/DNS playbooks, backup installers, and SOPS
encrypt/decrypt playbooks. Read the playbook header and the relevant Homelab
documentation before executing them; they can make broad or security-sensitive
changes.
