# whoami

[![Docker Pulls](https://img.shields.io/docker/pulls/traefik/whoami.svg)](https://hub.docker.com/r/traefik/whoami/)
[![Build Status](https://github.com/traefik/whoami/workflows/Main/badge.svg?branch=master)](https://github.com/traefik/whoami/actions)

Tiny Go webserver that returns client IP information and provides a simple name-to-IP mapping store.

## Usage

### Endpoints

#### `GET /whoami`

Returns the client's real IP address from the `X-Real-Ip` header.

**Request:**
- Requires `X-Real-Ip` header to be set (typically by a reverse proxy)

**Response:**
- Returns the IP address from the `X-Real-Ip` header
- Returns empty response if header is not present

#### `GET /iam/{name}`

Registers a name with the client's IP address. Stores the mapping of `name` to the client's real IP.

**Request:**
- `{name}` - Path parameter for the name to register
- Requires `X-Real-Ip` header to be set

**Response:**
- Returns the stored IP address
- Returns `400 Bad Request` if `X-Real-Ip` header is missing

#### `GET /whois/{name}`

Looks up a previously registered name and returns the associated IP address.

**Request:**
- `{name}` - Path parameter for the name to look up

**Response:**
- Returns the IP address associated with the name
- Returns `404 Not Found` if the name is not registered

### Flags

| Flag      | Env var              | Description                            |
|-----------|----------------------|----------------------------------------|
| `port`    | `WHOAMI_PORT_NUMBER` | Port number to listen on (default: `80`) |
| `verbose` |                      | Enable verbose logging                 |

## Examples

```console
$ docker run -d -p 8080:80 --name iamfoo traefik/whoami

# Get your IP (requires X-Real-Ip header, typically set by reverse proxy)
$ curl -H "X-Real-Ip: 203.0.113.50" http://localhost:8080/whoami
203.0.113.50

# Register a name with your IP
$ curl -H "X-Real-Ip: 203.0.113.50" http://localhost:8080/iam/alice
203.0.113.50

# Look up a registered name
$ curl http://localhost:8080/whois/alice
203.0.113.50

# Looking up an unregistered name returns 404
$ curl -v http://localhost:8080/whois/unknown
< HTTP/1.1 404 Not Found
```

```yml
version: '3.9'

services:
  whoami:
    image: traefik/whoami
    command:
       # It tells whoami to start listening on 2001 instead of 80
       - --port=2001
       - --verbose
```