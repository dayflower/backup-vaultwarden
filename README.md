# backup-vaultwarden

A zero-dependency tool to backup your [Vaultwarden](https://github.com/dani-garcia/vaultwarden) instance, offering optional cloud storage upload via [rclone](https://rclone.org/) and backup file encryption.

## Features

- Zero Dependency
- Optionally Back up to the Cloud, powered by [rclone](https://rclone.org/)
- Can encrypt backup files

## Usage

    Usage:
      backup-vaultwarden [OPTIONS] source-dir

    Application Options:
      -o, --output=             Output file (default: backup.tar.gz;
                                backup.tar.gz.enc if --encrypt is set)
      -t, --targets=            Backup targets (default: default)
      -e, --encrypt             Encrypt backup file. The passphrase can be set via
                                BACKUP_PASSPHRASE environment variable, or
                                interactively
      -b, --archive-base-dir=   Base directory in archive (default: data/)
      -r, --rclone-destination= Rclone destination path.  If set, the backup file
                                will be uploaded to the remote.
      -c, --rclone-config-file= Rclone config file. This option is required when using cloud storage upload.
      -k, --preserve-archive    Preserve archive file after uploading to rclone
                                destination
      -l, --loglevel=           Log level (default: info)

    Help Options:
      -h, --help                Show this help message

    Arguments:
      source-dir:               Source directory

### Backup Targets

The `--targets` option allows you to specify which parts of your Vaultwarden data to backup. You can specify multiple targets separated by commas. For more details on each target, see the [official wiki](https://github.com/dani-garcia/vaultwarden/wiki/Backing-up-your-vault).

| Target        | Description                                                                                                              |
| :------------ | :----------------------------------------------------------------------------------------------------------------------- |
| `db`          | [The SQLite database files](https://github.com/dani-garcia/vaultwarden/wiki/Backing-up-your-vault#sqlite-database-files) |
| `attachments` | [The attachments directory](https://github.com/dani-garcia/vaultwarden/wiki/Backing-up-your-vault#the-attachments-dir)   |
| `config`      | [The `config.json` file](https://github.com/dani-garcia/vaultwarden/wiki/Backing-up-your-vault#the-configjson-file)      |
| `key`         | [The `rsa_key` files](https://github.com/dani-garcia/vaultwarden/wiki/Backing-up-your-vault#the-rsa_key-files)           |
| `sends`       | [The sends directory](https://github.com/dani-garcia/vaultwarden/wiki/Backing-up-your-vault#the-sends-dir)               |
| `icon_cache`  | [The icon cache directory](https://github.com/dani-garcia/vaultwarden/wiki/Backing-up-your-vault#the-icon_cache-dir)     |

#### Simplified Target Specifiers

You can also use the following specifiers to backup common sets of data:

| Specifier   | db  | attachments | key | config | sends | icon_cache | Description                                                                                           |
| :---------- | :-: | :---------: | :-: | :----: | :---: | :--------: | :---------------------------------------------------------------------------------------------------- |
| `every`     | ✅  |     ✅      | ✅  |   ✅   |  ✅   |     ✅     | Backs up everything found within the Vaultwarden data directory.                                      |
| `all`       | ✅  |     ✅      | ✅  |   ✅   |  ✅   |     ✅     | Backs up all core data components.                                                                    |
| `recommend` | ✅  |     ✅      | ✅  |   ✅   |       |            | Backs up the components marked as "Backup required" and "Backup recommended" in the Vaultwarden wiki. |
| `default`   | ✅  |     ✅      |     |   ✅   |       |            | The default set of targets.                                                                           |

If you do not specify the `--targets` option, the `default` target specifier will be used.

## Encryption

The backup file can be encrypted using the `-e` option.

The passphrase for encryption can be provided in two ways:

- By setting the `BACKUP_PASSPHRASE` environment variable. If this variable is set, its value will be used as the passphrase.
- If the environment variable is not set, you will be prompted to enter a passphrase interactively when the backup process starts.

To decrypt your encrypted backup file (for example, if it was named `backup.tar.gz.enc`), you can use the following `openssl` command:

    openssl enc -d -aes256 -pbkdf2 -md sha-256 -in backup.tar.gz.enc -out backup.tar.gz

## Cloud Storage Upload

By default, this tool only creates a local backup archive and does not upload it to the cloud.

To upload the backup file to cloud storage, you need to specify the destination using the `-r` or `--rclone-destination` option. This option takes an [rclone](https://rclone.org/) destination path. For example:

    backup-vaultwarden -r gdrive:/vaultwarden -c /path/to/rclone.conf /path/to/vaultwarden/data

In this example, `gdrive:/vaultwarden` refers to a remote named `gdrive` configured in your rclone configuration, and the `/vaultwarden` part specifies the directory within that remote where the backup will be uploaded. **You must also specify the path to your rclone configuration file using the `-c` or `--rclone-config-file` option.**

## Important Security Note

When uploading backup files to cloud storage, it is **crucial** to ensure that the uploaded files are not publicly accessible. Leaking your Vaultwarden backup could expose sensitive information.

**We are not responsible for any data leaks that occur due to misconfigured cloud storage or publicly shared backup files.** Please exercise extreme caution when configuring your cloud storage and ensure appropriate privacy settings are in place.

## Questions

### Is there a feature for generational backups?

No, this tool does not offer built-in support for generational backups and there are no plans to add this feature in the future. You will need to manage backup versions yourself. If you are using cloud storage, many providers offer features like file versioning or history that you can utilize for this purpose.

## License

This software is released under the MIT License. See the [LICENSE.txt](./LICENSE.txt) file for more information.

This software depends on various modules, and in particular, incorporates functionality from rclone. The license for rclone can be found in [licenses/3rdparty/rclone/COPYING](./licenses/3rdparty/rclone/COPYING).
