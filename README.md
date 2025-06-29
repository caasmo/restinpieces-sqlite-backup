# restinpieces-sqlite-backup

`restinpieces-sqlite-backup` provides backup functionality for SQLite databases within the `restinpieces` framework. It uses a push-pull design, consisting of a server-side job to create local backups and a client-side binary to retrieve them.

## Architecture

The solution is composed of two main components:

1.  **The Backup Job (Server-side)**: This component runs as a job within a `restinpieces` application. It periodically creates a compressed backup of a specified SQLite database and stores it locally on the server. It also maintains a `latest.txt` manifest file, which always points to the most recent backup.

2.  **The Pull Client (Client-side)**: This is a standalone command-line binary that can be run on any client machine. It connects to the server via SFTP, reads the `latest.txt` manifest to find the newest backup, downloads it, and verifies its integrity.

This design decouples backup creation from retrieval, allowing backups to be pulled from a central server to any number of client machines.

## Features

-   **Push-Pull Design**: Decouples backup creation (server-side) from retrieval (client-side).
-   **SQLite Backup**: Performs a backup using the `VACUUM INTO` command.
-   **Gzip Compression**: Compresses backup files.
-   **Manifest File**: A `latest.txt` file points to the latest backup for easy retrieval.
-   **SFTP Client**: A client is provided to pull backups from a remote server.
-   **Backup Verification**: The client verifies the integrity of downloaded backups using `PRAGMA integrity_check`.

## Installation

To use this package, you need a working Go environment (Go 1.24.2 or later). You can add it to your project using `go get`:

```bash
go get github.com/caasmo/restinpieces-sqlite-backup
```

## Deployment Workflow

Deploying this add-on involves three main steps: configuring the job, inserting the recurrent job into the database, and running the main application.

1.  **Configure the Backup Job**: First, you must add the backup configuration (in TOML format) to the `restinpieces` database using its secure `ConfigStore`. The configuration should be stored under the scope `sqlite_backup`.
    ```toml
    source_path = "/path/to/your/database.db"
    backup_dir = "/path/to/your/backups"
    ```

2.  **Insert the Recurrent Job**: Use the `insert-job` tool provided in this repository to create the recurrent job entry in the database. This only needs to be done once.
    ```bash
    go build ./cmd/insert-job
    ./insert-job \
      -dbpath /path/to/restinpieces.db \
      -interval 24h \
      -scheduled 2025-07-01T10:00:00Z
    ```

3.  **Run the Application**: Start your main `restinpieces` application. It will load the configuration, register the backup handler, and automatically start executing the backup job at its scheduled time.

## The `insert-job` Tool

This repository includes a small command-line tool located at `cmd/insert-job/main.go` specifically for activating the recurrent backup job. Because `restinpieces-sqlite-backup` is an optional add-on, the core application is not aware of it, so this tool is required to insert the job into the database without modifying the core application's code.

### Usage

The tool must be run once to create the job. It requires three flags:

-   `-dbpath`: The path to the main `restinpieces` SQLite database file.
-   `-interval`: The interval for the recurrent backup job (e.g., `24h`, `1h30m`).
-   `-scheduled`: The start time for the job in RFC3339 format (e.g., `2025-07-01T10:00:00Z`). This determines when the first job will run.
```

## Usage

### Server-Side Setup (The Backup Job)

The backup job itself is configured via the `restinpieces` `ConfigStore`. The job's configuration, including the source database path and backup directory, is stored securely in the `restinpieces` database, not passed via command-line flags in a production environment.

The configuration should be stored under the scope `sqlite_backup` in TOML format:

```toml
source_path = "/path/to/your/database.db"
backup_dir = "/path/to/your/backups"
```

The example binary in `cmd/example/main.go` demonstrates how to initialize the `restinpieces` framework and register the backup job handler. For convenience, the example uses the `-dbpath` flag to specify the path to the `restinpieces` database itself.

### Client-Side Setup (The Pull Client)

The `cmd/client/main.go` file contains the client that can be used to pull the latest backup from a remote server. The client is configured via a `Config` struct in the `main` function. You will need to update the configuration with your SSH user, host, private key path, and the remote and local backup directories.

You can then build and run the client from any machine that needs a copy of the backup.


## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.