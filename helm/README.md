# Helm Chart

The Helm repository where this chart is located is `https://dayflower.github.io/backup-vaultwarden`,
and the name of the artifact within this repository is `backup-vaultwarden`.

## Installation Instructions

1.  Add the Helm repository:

        helm repo add backup-vaultwarden https://dayflower.github.io/backup-vaultwarden

2.  Update the Helm repositories:

        helm repo update

3.  Install the `backup-vaultwarden` chart. Replace `<my-release>` with your desired release name:

        helm install <my-release> backup-vaultwarden/backup-vaultwarden

    If you have a custom configuration file (`values.yaml`), you can specify it using the `-f` option:

        helm install <my-release> -f values.yaml backup-vaultwarden/backup-vaultwarden

## CAVEATS

When using `rclone` for backups with this Helm chart, it is essential to configure a Persistent Volume.

This is because the `rclone.conf` file, which contains configuration details for `rclone`, will be updated during operation.
Furthermore, as the `rclone.conf` file may store sensitive credentials necessary for accessing your backup storage,
it is of utmost importance to ensure that the configured Volume is adequately secured with appropriate access controls to prevent unauthorized access.
