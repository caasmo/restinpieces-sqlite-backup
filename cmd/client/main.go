
package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"zombiezen.com/go/sqlite"
)

// Config holds the configuration for the pullfile client.
// In a real application, this would be populated from command-line flags or a config file.
type Config struct {
	SSHUser           string
	SSHHost           string
	SSHPort           string
	SSHPrivateKeyPath string
	RemoteBackupDir   string // The directory on Machine B where backups are stored.
	LocalBackupDir    string // The directory on Machine A to save the backups.
}

func main() {
	// Basic configuration. Replace with your actual data.
	cfg := Config{
		SSHUser:           "user",
		SSHHost:           "machine-b.example.com",
		SSHPort:           "22",
		SSHPrivateKeyPath: "/home/user/.ssh/id_rsa",
		RemoteBackupDir:   "/var/caasmo/backups",
		LocalBackupDir:    "/home/lipo/backups",
	}

	ctx := context.Background()
	slog.Info("Starting pullfile client")

	// 1. Connect via SSH and set up SFTP client
	sftpClient, err := setupSftpClient(cfg)
	if err != nil {
		slog.Error("Failed to set up SFTP client", "error", err)
		os.Exit(1)
	}
	defer sftpClient.Close()

	// 2. Get the latest backup filename from the manifest
	latestBackupFilename, err := getLatestBackupFilename(sftpClient, cfg.RemoteBackupDir)
	if err != nil {
		slog.Error("Failed to get latest backup filename", "error", err)
		os.Exit(1)
	}
	slog.Info("Found latest backup file to fetch", "filename", latestBackupFilename)

	// 3. Download the backup file
	localPath, err := downloadBackup(sftpClient, cfg.RemoteBackupDir, latestBackupFilename, cfg.LocalBackupDir)
	if err != nil {
		slog.Error("Failed to download backup", "error", err)
		os.Exit(1)
	}
	slog.Info("Successfully downloaded backup", "path", localPath)

	// 4. Decompress and verify the backup
	if err := verifyBackup(ctx, localPath); err != nil {
		slog.Error("Backup verification failed", "error", err)
		os.Exit(1)
	}

	slog.Info("Backup verification successful! The backup is valid.", "path", localPath)
}

func setupSftpClient(cfg Config) (*sftp.Client, error) {
	key, err := os.ReadFile(cfg.SSHPrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read private key: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("unable to parse private key: %w", err)
	}

	sshConfig := &ssh.ClientConfig{
		User: cfg.SSHUser,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // In production, use a proper host key callback
		Timeout:         15 * time.Second,
	}

	addr := fmt.Sprintf("%s:%s", cfg.SSHHost, cfg.SSHPort)
	conn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to dial ssh: %w", err)
	}

	client, err := sftp.NewClient(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to create sftp client: %w", err)
	}

	return client, nil
}

func getLatestBackupFilename(client *sftp.Client, remoteDir string) (string, error) {
	manifestPath := filepath.Join(remoteDir, "latest.txt")
	f, err := client.Open(manifestPath)
	if err != nil {
		return "", fmt.Errorf("could not open remote manifest file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		return scanner.Text(), nil
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading manifest file: %w", err)
	}
	return "", fmt.Errorf("manifest file is empty")
}

func downloadBackup(client *sftp.Client, remoteDir, filename, localDir string) (string, error) {
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return "", fmt.Errorf("could not create local backup directory: %w", err)
	}

	remotePath := filepath.Join(remoteDir, filename)
	localPath := filepath.Join(localDir, filename)

	srcFile, err := client.Open(remotePath)
	if err != nil {
		return "", fmt.Errorf("could not open remote backup file: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(localPath)
	if err != nil {
		return "", fmt.Errorf("could not create local backup file: %w", err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return "", fmt.Errorf("failed to copy backup file: %w", err)
	}

	return localPath, nil
}

func verifyBackup(ctx context.Context, gzippedBackupPath string) error {
	// Decompress to a temporary file
	tempDBPath := filepath.Join(os.TempDir(), fmt.Sprintf("verified-%d.db", time.Now().UnixNano()))
	if err := decompressFile(gzippedBackupPath, tempDBPath); err != nil {
		return fmt.Errorf("failed to decompress for verification: %w", err)
	}
	defer os.Remove(tempDBPath)

	slog.Info("Decompressed backup for verification", "path", tempDBPath)

	// Open the database with the zombiezen driver
	conn, err := sqlite.OpenConn(tempDBPath, sqlite.OpenReadOnly)
	if err != nil {
		return fmt.Errorf("failed to open decompressed database: %w", err)
	}
	defer conn.Close()

	// Run the integrity check
	stmt, err := conn.Prepare("PRAGMA integrity_check;")
	if err != nil {
		return fmt.Errorf("failed to prepare integrity_check statement: %w", err)
	}
	defer stmt.Close()

	row, err := stmt.Step()
	if err != nil {
		return fmt.Errorf("failed to execute integrity_check: %w", err)
	}
	if !row {
		return fmt.Errorf("integrity_check returned no rows")
	}

	result := stmt.ColumnText(0)
	if result != "ok" {
		return fmt.Errorf("integrity_check failed, result was: %s", result)
	}

	return nil
}

func decompressFile(sourcePath, destPath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file for decompression: %w", err)
	}
	defer sourceFile.Close()

	gzipReader, err := gzip.NewReader(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file for decompression: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, gzipReader); err != nil {
		return fmt.Errorf("failed to copy and decompress data: %w", err)
	}

	return nil
}
