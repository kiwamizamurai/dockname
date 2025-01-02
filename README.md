# dockname

[![Docker Hub](https://img.shields.io/docker/v/kiwamizamurai/dockname?logo=docker)](https://hub.docker.com/r/kiwamizamurai/dockname)
[![License](https://img.shields.io/github/license/kiwamizamurai/dockname)](https://github.com/kiwamizamurai/dockname/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/kiwamizamurai/dockname)](https://goreportcard.com/report/github.com/kiwamizamurai/dockname)
[![Image Size](https://img.shields.io/badge/image%20size-10.9MB-blue)](https://hub.docker.com/r/kiwamizamurai/dockname)

dockname is a simple label-based reverse proxy that makes container routing effortless in development environments. It serves as a lightweight alternative to other reverse proxies, offering simpler configuration with a tiny footprint.

| Solution | Image Size | Relative Size |
|----------|------------|---------------|
| **dockname** | **10.9MB** | **1x (Base)** |
| Traefik | 185MB | 17x larger |
| Nginx Proxy Manager | 1.09GB | 100x larger |

## Features

- 🎯 Simple label-based configuration
- 🔄 Automatic container discovery and configuration
- 🚀 Easy setup with `.localhosthost` domains (no `/etc/hosts` editing required)
- 🛡️ Lightweight design optimized for development environments

## Quick Start

```yaml
services:
  proxy:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "80:80"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    user: root
    restart: always

  web:
    image: nginx:latest
    labels:
      - "dockname.domain=web.localhost"  # Access domain
      - "dockname.port=80"               # Container port
```

Launch:
```bash
docker compose up -d
```

Simply visit http://web.localhost in your browser!

## How It Works

1. DNS Level:
   - `.localhost` domains automatically resolve to `127.0.0.1`
   - No need to edit `/etc/hosts`

2. dockname Proxy:
   - Monitors Docker containers
   - Label-based routing configuration
   - Forwards requests to appropriate containers

## Label Configuration

| Label | Description | Example |
|--------|------------|---------|
| `dockname.domain` | Access domain | `web.localhosthost` |
| `dockname.port` | Container port (default: 80) | `80` |

## License

MIT License - See [LICENSE](LICENSE) file for details.
