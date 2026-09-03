# Stellar Sentinel

[![CI](https://github.com/SmartCraftGroup/stellar-sentinel/actions/workflows/ci.yml/badge.svg)](https://github.com/SmartCraftGroup/stellar-sentinel/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)
[![Go](https://img.shields.io/badge/Go-v1.22%2B-00ADD8.svg)](https://go.dev/)
[![Drips Wave](https://img.shields.io/badge/Drips-Stellar%20Wave-blue.svg)](https://drips.network)

**Stellar Sentinel** is a lightweight, self-hosted streaming event monitor and notification relay for the Stellar blockchain. It monitors Stellar Horizon account streams (`/accounts/{id}/payments`) and dispatches authenticated webhooks, Discord messages, or Telegram alerts whenever transactions occur on tracked public keys.

---

## Why Stellar Sentinel Exists

Exchanges, merchant payment processors, and fintech apps need real-time alerts when XLM or SAC token deposits land on their Stellar addresses.

Subscribing to Horizon's Server-Sent Events (SSE) directly requires custom code to handle network reconnects, deduplicate event IDs, parse XDR operation details, and sign outgoing HTTP webhooks. `stellar-sentinel` packages this into a single zero-dependency Docker container with exponential backoff and a configuration Web UI.

---

## Key Features

- **Streaming Horizon Subscriptions:** Subscribes to real-time payment streams (`/payments`) with automatic reconnect and cursor persistence.
- **Multi-Channel Dispatch:** Dispatches notifications via signed HTTP POST webhooks, Slack, Discord, and Telegram.
- **HMAC Signature Security:** Outgoing webhooks are signed with an `X-Sentinel-Signature` (HMAC-SHA256) header to prevent spoofing.
- **Topic & Asset Filtering:** Filter notifications by asset code (e.g. `USDC`, `EURC`, `XLM`), amount threshold, or transaction operation type.
- **Docker Ready:** Runs as a lightweight container with minimal memory overhead (<30MB RAM).

---

## Architecture Overview

```
+---------------+
| Stellar Horizon|
| (SSE Payments)|
+-------+-------+
        |
        v (Server-Sent Events)
+-------+---------------+
|   stellar-sentinel    |
|   Event Consumer      |
+-------+---------------+
        |
        +-----------------------+-----------------------+
        |                       |                       |
        v                       v                       v
+---------------+       +---------------+       +---------------+
| Signed Webhook|       | Telegram Bot  |       | Discord Alert |
|  (HTTP POST)  |       | (Chat Message)|       | (Webhook URL) |
+---------------+       +---------------+       +---------------+
```

---

## Quickstart & Docker Setup

### 1. Using Docker Compose
Create a `docker-compose.yml`:

```yaml
version: '3.8'

services:
  sentinel:
    image: smartcraft/stellar-sentinel:latest
    environment:
      - HORIZON_URL=https://horizon-testnet.stellar.org
      - TRACKED_ADDRESSES=GA...1,GA...2
      - WEBHOOK_URL=https://my-app.com/api/stellar-webhook
      - WEBHOOK_SECRET=super-secret-key-123
    ports:
      - "8080:8080"
```

### 2. Run Container
```bash
docker compose up -d
```

---

## Environment Variables

| Variable | Required | Description | Default |
|---|---|---|---|
| `HORIZON_URL` | Required | Stellar Horizon endpoint URL | `https://horizon.stellar.org` |
| `TRACKED_ADDRESSES` | Required | Comma-separated list of Stellar public keys | — |
| `WEBHOOK_URL` | Optional | HTTP POST target for event payloads | — |
| `WEBHOOK_SECRET` | Optional | Shared secret for HMAC-SHA256 signature header | — |
| `DISCORD_WEBHOOK_URL` | Optional | Discord channel webhook URL | — |
| `TELEGRAM_BOT_TOKEN` | Optional | Telegram Bot token for alerts | — |

---

## Maintainers & Contact

| Maintainer | Role | Contact |
|---|---|---|
| **Abdulmalik Ojo** (`@tecmalik`) | Lead Maintainer | [abdulmalikojo2@gmail.com](mailto:abdulmalikojo2@gmail.com) |

---

## License

MIT — see [`LICENSE`](./LICENSE).
