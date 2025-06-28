# restinpieces-sqlite-backup

`restinpieces-sqlite-backup` is a Go package that provides a job handler for the `restinpieces` framework to perform backups of SQLite databases. It is designed to be a simple, reliable, and easy-to-integrate solution for backing up your SQLite databases.

This package is part of the `restinpieces` ecosystem and is intended to be used as a plugin for the main `restinpieces` application.

## Features

-   **SQLite Backup**: Performs a backup of a SQLite database using the `VACUUM INTO` command, which creates a clean, minimal copy of the database.
-   **Gzip Compression**: Compresses the backup file using gzip to save space.
-   **Manifest File**: Creates a `latest.txt` manifest file that contains the name of the latest backup file, making it easy to find and restore the latest backup.
-   **SFTP Client**: Includes a client to pull the latest backup from a remote server via SFTP.
-   **Verification**: The client verifies the integrity of the downloaded backup using `PRAGMA integrity_check`.

## Installation

To use this package, you need to have a working Go environment (Go 1.24.2 or later). You can add it to your project using `go get`:

```bash
go get github.com/caasmo/restinpieces-sqlite-backup
```

## Usage

The `restinpieces-sqlite-backup` package provides a `Handler` that can be registered with a `restinpieces` server. The handler requires a configuration that specifies the source database path and the backup directory.

### Server-Side (Backup Job)

The backup job is configured via the `restinpieces` `ConfigStore`. You need to add a configuration with the scope `db_backup_config` to your `restinpieces` database. The configuration should be in TOML format and contain the following fields:

```toml
source_path = "/path/to/your/database.db"
backup_dir = "/path/to/your/backups"
```

The `cmd/example/main.go` file provides a working example of how to initialize the `restinpieces` framework and register the backup job handler.

### Client-Side (Pulling Backups)

The `cmd/client/main.go` file contains a client that can be used to pull the latest backup from a remote server via SFTP. The client is configured via a `Config` struct in the `main` function. You will need to update the configuration with your SSH user, host, private key path, and the remote and local backup directories.

## How it Works

The backup process consists of the following steps:

1.  **VACUUM INTO**: The `VACUUM INTO` command is used to create a clean copy of the database in a temporary file. This is a safe and efficient way to create a backup of a live SQLite database.
2.  **Gzip Compression**: The temporary backup file is compressed using gzip to reduce its size.
3.  **Manifest File**: A `latest.txt` file is created in the backup directory. This file contains the name of the latest backup file.
4.  **SFTP Transfer**: The client connects to the server via SFTP, reads the `latest.txt` file to get the name of the latest backup, and downloads the backup file.
5.  **Verification**: The client decompresses the backup file and runs `PRAGMA integrity_check` to ensure that the backup is not corrupt.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.