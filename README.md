# who

[![Build and Push Container Image](https://github.com/tracyhatemice/who/actions/workflows/docker-build.yml/badge.svg?branch=master)](https://github.com/tracyhatemice/who/actions/workflows/docker-build.yml)

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

| Flag      | Description                            |
|-----------|----------------------------------------|
| `port`    | Port number to listen on (default: `80`) |
| `verbose` | Enable verbose logging                 |

## Examples

```console
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
services:
  who:
    image: ghcr.io/tracyhatemice/who:latest
    container_name: 'who'
    networks:
      - traefik
    labels:
      traefik.enable: true
      traefik.docker.network: traefik
      traefik.http.routers.who.entrypoints: https
      traefik.http.routers.who.tls: true
      traefik.http.routers.who.rule: HostRegexp(`^((ipv4|ipv6)\.)*example\.org$`) && ( PathPrefix(`/whoami`) || PathPrefix(`/whois`) || PathPrefix(`/iam`) )
      traefik.http.routers.who.tls.certresolver: le
    restart: 'unless-stopped'
    command:
       - --port=80
       - --verbose
```
