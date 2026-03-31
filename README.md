<img height=144px src="./frontend/public/Lighthouse.svg" alt="Lighthouse Logo"/>

# Lighthouse — Windows Native

A temporary file-receiving station hosted on the Tor network. Run it, share your `.onion` address, receive files, shut it down.

> This is the **Windows native** branch. No Docker required.
> For the Docker version (Linux/macOS/Windows), see the [master branch](https://github.com/neozmmv/Lighthouse/tree/master).

## Concept

Lighthouse removes the usual friction from receiving files from someone:

- No port forwarding
- No cloud storage accounts
- No server setup
- No file size limits
- No Docker

You run the installer, Lighthouse downloads and configures everything automatically. Tor creates a hidden service and gives you an `.onion` address. You share that address with whoever needs to send you files. They open it in Tor Browser, upload the file, you download it. Done. Shut it down.

## How it works

```
Sender (Tor Browser) --> .onion address --> Tor network --> Lighthouse (your machine)
```

Tor's hidden service acts as the networking layer, so your machine is reachable without a public IP or open ports.

On first run, Lighthouse automatically downloads and configures:

- **Tor** — creates the hidden service and exposes your `.onion` address
- **MinIO** — stores received files locally
- **Caddy** — reverse proxy routing traffic between services

Everything runs natively as background processes. No Docker, no virtual machines.

## Stack

- **Frontend** — React + TypeScript (Vite, TanStack Router, Tailwind CSS)
- **Backend** — Go (Gin)
- **Storage** — MinIO
- **Proxy** — Caddy
- **Transport** — Tor hidden service
- **CLI** — Go (Cobra)

## Installation

Download `LighthouseSetup.exe` from the [releases page](https://github.com/neozmmv/Lighthouse/releases/latest) and run it.

The installer will:

1. Copy `lighthouse.exe` to `%LOCALAPPDATA%\Lighthouse\`
2. Add it to your PATH
3. Start Lighthouse automatically

## Usage

**Start**

```
lighthouse up
```

On first run this will download Tor, MinIO and Caddy (~50MB), configure everything and start all services automatically.

**Get your `.onion` address**

```
lighthouse url
```

Share this address with whoever needs to send you files. They open it in Tor Browser and upload.

**Manage files (host interface)**

Open `http://localhost:4405` in your browser to view and download received files.

**List files via CLI**

```
lighthouse files
```

**Download a file via CLI**

```
lighthouse download 0
lighthouse download 0 --here
lighthouse download 0 --remove
```

- `--here` saves to the current directory instead of `~/Downloads`
- `--remove` deletes the file from the bucket after downloading

**Check status**

```
lighthouse status
```

**Show MinIO credentials**

```
lighthouse config
```

**Stop**

```
lighthouse down
```

**Update**

```
lighthouse update
```

Downloads and runs the latest installer automatically.

## Data storage

All data is stored in `%APPDATA%\Lighthouse\`:

```
%APPDATA%\Lighthouse\
├── config.json
├── initialized
├── Caddyfile
├── bin\
├── tor\
├── data\minio\
└── frontend\
```

## Project structure

```
lighthouse/
├── backend-go\
├── cli\
└── frontend\
```

## Building from source

Prerequisites: Go 1.23+, Node.js 20+

```powershell
cd frontend
npm install
npm run build
Copy-Item -Recurse dist ..\cli\cmd\frontend

cd ..\backend-go
go build -o ..\cli\cmd\backend.exe .

cd ..\cli
go build -o lighthouse.exe .
```
