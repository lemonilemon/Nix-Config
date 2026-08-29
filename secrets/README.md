# secrets/

sops-encrypted files, committed to this public repo as ciphertext only. Age
recipients live in [`.sops.yaml`](../.sops.yaml); the private key sits at
`~/.config/sops/age/keys.txt`, backed up in 1Password (`just key-backup` /
`just key-fetch`).

## Inventory

| Name (`just secrets <name>`) | File | Format | Contents | Consumed by |
|------------------------------|------|--------|----------|-------------|
| `ssh-hosts` | `ssh-hosts.conf` | binary (ssh_config) | SSH host aliases: `Host` blocks with `HostName`/`User`/`Port` for private servers | `modules/gui/home/apps/default.nix`, decrypted to `~/.ssh/config.inventory` |
| `syncthing` | `syncthing.yaml` | yaml | Per-host Syncthing device identities (`<host>/cert`, `<host>/key`). The matching device IDs are public and live in `modules/general/nixos/syncthing.nix` | `modules/general/nixos/syncthing.nix`, installed into `~/.config/syncthing/{cert,key}.pem` |

Add a row whenever a new secret lands. Descriptions are public metadata: say
what a file is for, never any value inside it.

## Usage

- `just secrets` — print this inventory.
- `just secrets <name>` — decrypt into `$EDITOR`, re-encrypt on save; rebuild
  to deploy.
- `sops secrets/<file>` — create a brand-new secret file, then declare it in
  the consuming module and add a row above.
- `sops updatekeys secrets/<file>` — re-wrap a file after adding a recipient
  to `.sops.yaml`.
