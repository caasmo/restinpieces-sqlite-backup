# restinpieces-sqlite-backup

`restinpieces-sqlite-backup` is a simple, open-source solution for backing up SQLite databases within the `restinpieces` framework. It uses a push-pull design, consisting of a server-side job to create local backups and a client-side binary to retrieve them.

## Architecture

The solution is composed of two main components:

1.  **The Backup Job (Server-side)**: This component runs as a job within a `restinpieces` application. It periodically creates a compressed backup of a specified SQLite database and stores it locally on the server. It also maintains a `latest.txt` manifest file, which always points to the most recent backup, simplifying retrieval.

2.  **The Pull Client (Client-side)**: This is a standalone command-line binary that can be run on any client machine. It connects to the server (where the backups are stored) via SFTP, reads the `latest.txt` manifest to find the newest backup, downloads it, and verifies its integrity.

This design decouples the backup creation from the retrieval, allowing for a simple and secure way to pull backups from a central server to any number of client machines.

## Features

-   **Simple Push-Pull Design**: A straightforward and robust architecture for managing backups.
-   **SQLite Backup**: Performs a backup using the `VACUUM INTO` command for a clean, minimal copy of the database.
-   **Gzip Compression**: Compresses backup files to save space.
-   **Manifest File**: A `latest.txt` file makes it easy to identify and retrieve the latest backup.
-   **SFTP Client**: A secure and simple client for pulling backups from a remote server.
-   **Backup Verification**: The client verifies the integrity of downloaded backups using `PRAGMA integrity_check`.

## Installation

To use this package, you need a working Go environment (Go 1.24.2 or later). You can add it to your project using `go get`:

```bash
go get github.com/caasmo/restinpieces-sqlite-backup
```

## Usage

### Server-Side Setup (The Backup Job)

The backup job is configured via the `restinpieces` `ConfigStore`. You need to add a configuration with the scope `db_backup_config` to your `restinpieces` database. The configuration should be in TOML format and contain the following fields:

```toml
source_path = "/path/to/your/database.db"
backup_dir = "/path/to/your/backups"
```

The `cmd/example/main.go` file provides a working example of how to initialize the `restinpieces` framework and register the backup job handler.

### Client-Side Setup (The Pull Client)

The `cmd/client/main.go` file contains the client that can be used to pull the latest backup from a remote server. The client is configured via a `Config` struct in the `main` function. You will need to update the configuration with your SSH user, host, private key path, and the remote and local backup directories.

You can then build and run the client from any machine that needs a copy of the backup.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
