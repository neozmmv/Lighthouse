<img height=144px src="./frontend/public/Lighthouse.svg" alt="Lighthouse Logo"/>

# Lighthouse

A temporary file-receiving station hosted on the Tor network. Run it, share your `.onion` address, receive files, shut it down.

## Concept

Lighthouse removes the usual friction from receiving files from someone:

- No port forwarding
- No cloud storage accounts
- No server setup
- No file size limits

You spin it up, Tor creates a hidden service and gives you an `.onion` address. You share that address with whoever needs to send you files. They open it in Tor Browser, upload the file, you download it. Done. Shut it down.

## How it works

```
Sender (Tor Browser) --> .onion address --> Tor network --> Lighthouse (your machine)
```

Tor's hidden service acts as the networking layer, so your machine is reachable without a public IP or open ports.

## Stack

- **Frontend** — React + TypeScript (Vite, TanStack Router, Tailwind CSS)
- **Backend** — Go (Gin), proxied at `/api/`
- **Storage** — MinIO (S3-compatible object storage)
- **Transport** — Tor hidden service

On **Linux and macOS**, everything runs in Docker. On **Windows**, all components run natively — no Docker required.

## Installation

### Linux / macOS

Requires Docker and Docker Compose.

```bash
curl -fsSL https://github.com/neozmmv/Lighthouse/releases/latest/download/install.sh | sh
```

### Windows

Download and run `LighthouseSetup.exe` from the [releases page](https://github.com/neozmmv/Lighthouse/releases/latest). The installer adds `lighthouse` to your PATH automatically.

### Manual (Linux / macOS)

Download the binary for your platform from the [releases page](https://github.com/neozmmv/Lighthouse/releases/latest), make it executable and move it to your PATH:

```bash
chmod +x lighthouse-linux-amd64
sudo mv lighthouse-linux-amd64 /usr/local/bin/lighthouse
```

## Usage

### Start

```bash
lighthouse up
```

On first run, Lighthouse sets itself up automatically — no configuration needed.

If port 80 is already in use, specify a different one:

```bash
lighthouse up --port 8080
```

### Get your `.onion` address

```bash
lighthouse url
```

Share this address with whoever needs to send you files. They open it in Tor Browser.

### Check status

```bash
lighthouse status
```

### List received files

```bash
lighthouse files
```

### Download a file

```bash
lighthouse download 0
lighthouse download 0 --here      # save to current directory instead of ~/Downloads
lighthouse download 0 --remove    # delete from bucket after downloading
```

### Stop

```bash
lighthouse down
```

## Accessing the web interface

|                     | Linux / macOS           | Windows                 |
| ------------------- | ----------------------- | ----------------------- |
| **Main interface**  | `http://localhost`      | `http://localhost:8080` |
| **File management** | `http://localhost:4405` | `http://localhost:4405` |
| **MinIO console**   | `http://localhost:9001` | `http://localhost:9001` |

On **Linux/macOS**, MinIO credentials default to `lighthouse` / `lighthouse_secret`. You can override them by creating a `.env` file in `~/.lighthouse/` before first run:

```
MINIO_ROOT_USER=yourUser
MINIO_ROOT_PASSWORD=yourPassword
```

On **Windows**, credentials are generated automatically on first run and stored in `%APPDATA%\lighthouse\config.json`.

## Updating

**Linux / macOS:**

```bash
lighthouse update
```

Updates the CLI binary, `docker-compose.yml`, `Caddyfile`, and pulls the latest Docker images. If Lighthouse is running, restart it to apply the changes.

**Windows:**

Download and run the latest `LighthouseSetup.exe` from the [releases page](https://github.com/neozmmv/Lighthouse/releases/latest).

## Uninstall

**Linux / macOS:**

```bash
curl -fsSL https://github.com/neozmmv/Lighthouse/releases/latest/download/uninstall.sh | sh
```

**Windows:**

Use **Add or remove programs** in Windows Settings.

## Project structure

```
lighthouse/
├── backend-go/   # Go API (Gin + MinIO)
├── cli/          # Go CLI
└── frontend/     # React app
```

## Development

**Dependencies (Linux / macOS):**

```bash
docker compose -f docker-compose.dev.yml up -d
```

**Frontend:**

```bash
cd frontend
npm install
npm run dev
```

**Backend:**

```bash
cd backend-go
go run .
```
