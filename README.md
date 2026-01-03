# SQLite (or any file actually) to S3 Backup Tool

A simple Go application to backup SQLite database files to S3-compatible object storage. Designed for use with cron jobs.

## Features

- Uploads SQLite database files to S3-compatible storage
- Timestamped backups
- Automatic cleanup of old backups (configurable retention)
- Works with any S3-compatible storage (AWS S3, MinIO, Wasabi, etc.)

## Prerequisites

- Go 1.21 or higher
- S3-compatible object storage

## Installation

1. Clone the repository:
```bash
cd /root/Projects/backuper
```

2. Install dependencies:
```bash
go mod tidy
```

3. Build the application:
```bash
go build -o backuper main.go
```

## Configuration

Set the following environment variables:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `BACKUP_FILE` | Yes | - | Path to the SQLite database file |
| `S3_BUCKET` | Yes | - | S3 bucket name |
| `S3_ENDPOINT` | Yes | - | S3-compatible storage endpoint |
| `S3_ACCESS_KEY` | Yes | - | S3 access key |
| `S3_SECRET_KEY` | Yes | - | S3 secret key |
| `S3_REGION` | No | `us-east-1` | S3 region |
| `BACKUP_PREFIX` | No | `backups` | Directory inside bucket for backup objects |
| `KEEP_BACKUPS` | No | `7` | Number of backups to retain |

See `.env.example` for an example configuration.

## Usage

### Manual Backup

Run the application with environment variables set:

```bash
export BACKUP_FILE=/path/to/database.db
export S3_BUCKET=my-backup-bucket
export S3_ENDPOINT=https://s3.example.com
export S3_ACCESS_KEY=your-access-key
export S3_SECRET_KEY=your-secret-key
export S3_REGION=us-east-1

./backuper
```

Or load from a `.env` file:

### Cron Job Setup

Add the following to your crontab (`crontab -e`):

```cron
# Daily backup at 2 AM
0 2 * * * cd /root/Projects/backuper && /usr/bin/env bash -c 'source .env' && ./backuper >> backup.log 2>&1
```

Or using a shell script wrapper:

1. Create a wrapper script:
```bash
#!/bin/bash
cd /root/Projects/backuper
export $(cat .env | xargs)
./backuper
```

2. Make it executable:
```bash
chmod +x backup.sh
```

3. Add to cron:
```cron
0 2 * * * /root/Projects/backuper/backup.sh >> /root/Projects/backuper/backup.log 2>&1
```

### Cron Examples

```cron
# Every 6 hours
0 */6 * * * /path/to/backup.sh

# Every Sunday at 3 AM
0 3 * * 0 /path/to/backup.sh

# Every day at midnight, keep only last backup
0 0 * * * KEEP_BACKUPS=1 /path/to/backup.sh
```

## Backup Naming

Backups are stored with the following naming convention:
```
{PREFIX}/{TIMESTAMP}-{FILENAME}
```

Example: `backups/20260103-143022-database.db`

## Storage Compatibility

Compatible with any S3-compatible storage including:
- AWS S3
- MinIO
- Wasabi
- DigitalOcean Spaces
- Scaleway Object Storage
- Linode Object Storage

## Logging

The application logs to stdout/stderr. Redirect to a file for persistent logging:
```bash
./backuper >> backup.log 2>&1
```

## Building for Production

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o backuper main.go

# Linux ARM (for Raspberry Pi)
GOOS=linux GOARCH=arm64 go build -o backuper main.go

# macOS
GOOS=darwin GOARCH=amd64 go build -o backuper main.go
```

## License

MIT
