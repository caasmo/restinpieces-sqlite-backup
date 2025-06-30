# restinpieces-sqlite-backup

`restinpieces-sqlite-backup` provides backup functionality for SQLite databases within the `restinpieces` framework. It uses a push-pull design, consisting of a server-side job to create local backups and a client-side binary to retrieve them.

This handler supports two distinct backup strategies (`vacuum` and `online`) to accommodate different production workloads. See the "Backup Strategies" section for a detailed comparison to help you choose the right one for your needs.

## Architecture

The solution is composed of two main components:

1.  **The Backup Job (Server-side)**: This component runs as a job within a `restinpieces` application. It periodically creates a compressed backup of a specified SQLite database and stores it locally on the server. It also maintains a `latest.txt` manifest file, which always points to the most recent backup.

2.  **The Pull Client (Client-side)**: This is a standalone command-line binary that can be run on any client machine. It connects to the server via SFTP, reads the `latest.txt` manifest to find the newest backup, downloads it, and verifies its integrity.

This design decouples backup creation from retrieval, allowing backups to be pulled from a central server to any number of client machines.

## Features

-   **Flexible Backup Strategies**: Choose between a fast, locking `vacuum` strategy or a non-locking `online` strategy.
-   **Push-Pull Design**: Decouples backup creation (server-side) from retrieval (client-side).
-   **Gzip Compression**: Compresses backup files.
-   **Descriptive Filenames**: Embeds the database name, timestamp, and strategy into filenames (e.g., `app-2025-07-01T10-30-00Z-vacuum.bck.gz`).
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

1.  **Configure the Backup Job**: First, you must add the backup configuration (in TOML format) to the `restinpieces` database using its secure `ConfigStore`. The configuration should be stored under the scope `sqlite_backup`. See the "Backup Strategies" section for details on which `strategy` to choose.
    ```toml
    # Example for the 'online' strategy
    source_path = "/path/to/your/database.db"
    backup_dir = "/path/to/your/backups"
    strategy = "online"
    pages_per_step = 200
    sleep_interval = "20ms"
    progress_log_interval = "30s"
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

## Backup Strategies

Choosing the correct backup strategy is critical for ensuring your application remains performant.

### `vacuum` (Default Strategy)

This strategy uses the `VACUUM INTO` command to create the backup.

-   **Pros:**
    -   **Fast:** It's the quickest way to get a backup.
    -   **Defragmented:** The resulting backup file is clean and unfragmented, making it slightly smaller and faster to restore.
-   **Cons:**
    -   **Locks Writers:** This is the major drawback. It places a read lock on the source database, **blocking all write operations** for the entire duration of the backup.
-   **When to use it:**
    -   Databases with low write activity.
    -   During scheduled maintenance or predictable off-peak hours where a brief write-pause is acceptable.
    -   When you need a defragmented copy for analytical purposes.

### `online`

This strategy uses SQLite's built-in Online Backup API.

-   **Pros:**
    -   **Non-Locking:** It copies the database page-by-page, yielding between steps. It does **not** block application writers for any significant length of time, making it safe for 24/7, high-traffic applications.
-   **Cons:**
    -   **Slower:** The total backup time is longer than `vacuum` due to the overhead of the incremental copy.
    -   **Direct Copy:** The resulting file is a direct copy of the source, including any fragmentation.
-   **When to use it:**
    -   **Recommended for most production systems.**
    -   Databases with high, unpredictable, or 24/7 write workloads.
    -   When application availability is more important than raw backup speed.

### Configuration Parameters

The `online` strategy can be tuned with the following parameters in your TOML config:

-   `pages_per_step` (integer, default: `100`): How many pages to copy in a single step. A smaller value is "politer" to other connections but increases overhead.
-   `sleep_interval` (duration, default: `"10ms"`): How long to pause between steps to yield system resources. A value of `"0s"` will run the backup as fast as possible, while a higher value will reduce its CPU/IO impact.
-   `progress_log_interval` (duration, default: `"15s"`): How often to log backup progress.

## Tools and Examples

This repository contains several `cmd` utilities that serve as tools and examples.

-   **[cmd/example](https://github.com/caasmo/restinpieces-sqlite-backup/tree/master/cmd/example)**: A fully working example of a `restinpieces` server that registers and runs the backup handler. This is the primary reference for integrating the handler into your own application.

-   **[cmd/generate-blueprint-config](https://github.com/caasmo/restinpieces-sqlite-backup/tree/master/cmd/generate-blueprint-config)**: A simple tool that prints a template TOML configuration file to the console. This is useful for getting started with the configuration.
    ```bash
go run ./cmd/generate-blueprint-config
    ```

-   **[cmd/insert-job](https://github.com/caasmo/restinpieces-sqlite-backup/tree/master/cmd/insert-job)**: The command-line tool used to insert the recurrent backup job into the database. See the "Deployment Workflow" section for usage details.

-   **[cmd/client](https://github.com/caasmo/restinpieces-sqlite-backup/tree/master/cmd/client)**: An example of a client-side binary that connects to the server via SFTP to pull the latest backup. This can be adapted to your specific needs for retrieving backups.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.