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
- **Backend** — Python (FastAPI), proxied at `/api/`
- **Transport** — Tor hidden service

## Installation

### Linux / macOS

```bash
curl -fsSL https://github.com/neozmmv/Lighthouse/releases/latest/download/install.sh | sh
```

### Windows

Download the latest `lighthouse-windows-amd64.exe` from the [releases page](https://github.com/neozmmv/Lighthouse/releases/latest), rename it to `lighthouse.exe` and move it to a folder in your PATH (e.g. `C:\Windows\System32`), or add its folder to PATH via **System Properties → Environment Variables**.

### Manual (any platform)

Download the binary for your platform from the [releases page](https://github.com/neozmmv/Lighthouse/releases/latest), make it executable and move it to your PATH:

```bash
chmod +x lighthouse-linux-amd64
sudo mv lighthouse-linux-amd64 /usr/local/bin/lighthouse
```

## Usage

> Prerequisites: Docker and Docker Compose.

**Start**

```bash
lighthouse up
```

**Get your `.onion` address**

```bash
lighthouse url
```

**Share** the `.onion` address with the sender and wait for the file to arrive.

Access **localhost** on your browser to be able to go to the `/files` route and download your file.

You can access the **MinIO** Panel on `http://localhost:9001`.

By default, the access is **"lighthouse"** and password **"lighthouse_secret"**. You can change these values by creating a `.env` file on the project root with these values:

```
MINIO_ROOT_USER=yourUser
MINIO_ROOT_PASSWORD=yourPassword
```

This route is inaccessible outside your localhost, so only you have access to your files.

**Stop**

```bash
lighthouse down
```

## Uninstall

```bash
curl -fsSL https://github.com/neozmmv/Lighthouse/releases/latest/download/uninstall.sh | sh
```

## Project structure

```
lighthouse/
├── backend/      # Python API
├── cli/          # Go CLI
└── frontend/     # React app
```

## For development

For getting development dependencies:

```bash
sudo docker compose -f docker-compose.dev.yml up -d
```

or

```bash
sudo docker compose -f docker-compose.dev.yml up -d --build
```

Front-end:

```bash
cd frontend
npm install
npm run dev
```

Back-end:

```bash
cd backend/app
python -m venv env
source ./env/bin/activate
pip install -r requirements.txt
fastapi dev main.py --host 0.0.0.0
```
