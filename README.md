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

## Usage

> Prerequisites: Docker and Docker Compose.

**Start**

```bash
sudo docker compose up -d
```

**Get your `.onion` address**
```bash
sudo cat ./docker/tor-data/hidden_service/hostname
```

Lighthouse will spin up the frontend, backend, storage, and Tor hidden service.

**Share** the `.onion` address with the sender and wait for the file to arrive.

**Stop**

```bash
docker compose down
```

## Project structure

```
lighthouse/
├── backend/      # Python API
└── frontend/     # React app
```
