# DockName

[![Docker Hub](https://img.shields.io/docker/v/kiwamizamurai/dockname?logo=docker)](https://hub.docker.com/r/kiwamizamurai/dockname)
[![License](https://img.shields.io/github/license/kiwamizamurai/dockname)](https://github.com/kiwamizamurai/dockname/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/kiwamizamurai/dockname)](https://goreportcard.com/report/github.com/kiwamizamurai/dockname)
[![Image Size](https://img.shields.io/badge/image%20size-10.9MB-blue)](https://hub.docker.com/r/kiwamizamurai/dockname)

DockName is a zero-config reverse proxy that makes container routing effortless in development environments. It automatically detects your containers and makes them accessible via `.localhost` domains based on their service names.

| Solution | Image Size | Relative Size |
|----------|------------|---------------|
| **DockName** | **10.9MB** | **1x (Base)** |
| Traefik | 185MB | 17x larger |
| Nginx Proxy Manager | 1.09GB | 100x larger |

## Features

- 🎯 Zero configuration required - just add the proxy service
- 🎯 Automatic service discovery and routing
- 🔄 Real-time container detection
- 🌐 Automatic `.localhost` domain generation
- 🛡️ Lightweight design optimized for development

## Quick Start

```yaml
services:
  proxy:
    image: kiwamizamurai/dockname
    ports:
      - "80:80"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    user: root
    restart: always

  web:
    image: nginx:latest
    ports:
      - "3000:80"
    # That's it! Automatically accessible at http://web.localhost
```

Launch:
```bash
docker compose up -d
```

Your services are instantly accessible:
- `http://web.localhost` -> nginx container
- etc...

## How It Works

1. **Automatic Service Discovery**:
   - DockName detects your running containers
   - Uses service names from Docker Compose
   - Automatically finds exposed ports

2. **Smart Domain Generation**:
   - Creates domains based on service names
   - Example: `web` service -> `web.localhost`
   - No `/etc/hosts` editing needed

3. **Port Detection**:
   - Automatically detects container ports
   - Uses first exposed port by default
   - Supports both published and internal ports

## License

MIT License - See [LICENSE](LICENSE) file for details.
