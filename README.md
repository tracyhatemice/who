# who

[![Build and Push Container Image](https://github.com/tracyhatemice/who/actions/workflows/docker-build.yml/badge.svg?branch=master)](https://github.com/tracyhatemice/who/actions/workflows/docker-build.yml)

Small Go webserver that returns client IPs and stores simple name→IP mappings. Optionally performs DDNS updates when an IP changes.

Intended to run in parallel with Traefik — place the container on the same Docker network and configure Traefik router labels (see example below).

## Deployment with Traefik

Assume that Traefik is already set up with a Docker provider and is running on a Docker network named `traefik`.

Copy the `config.example.json` to `config.json`, change as needed, and mount it into the container.

```yml
services:
  who:
    image: ghcr.io/tracyhatemice/who:latest
    container_name: 'who'
    networks:
      - traefik
    volumes:
      - ./config.json:/config.json
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
       - --config=/config.json
```

Command line flags:

| Flag      | Description                                     |
|-----------|-------------------------------------------------|
| `port`    | Port number to listen on (default: `80`)        |
| `verbose` | Enable verbose logging                          |
| `config`  | Path to config file (optional) |

## Usage

```console
# Get your IP (requires X-Real-Ip header, typically set by reverse proxy)
$ curl -H "X-Real-Ip: 203.0.113.50" http://localhost:8080/whoami
203.0.113.50

# Register a name with your IP
$ curl -H "X-Real-Ip: 203.0.113.50" http://localhost:8080/iam/alice
203.0.113.50

# Register a name with a specific IP
$ curl http://localhost:8080/iam/bob/192.168.1.100
192.168.1.100

# Look up a registered name
$ curl http://localhost:8080/whois/alice
203.0.113.50

# Looking up an unregistered name returns 404
$ curl -v http://localhost:8080/whois/unknown
< HTTP/1.1 404 Not Found
```

### Endpoints

#### `GET /whoami`

Returns the client's IP address using fallback chain: X-Forwarded-For (last IP) → X-Real-Ip → RemoteAddr.

**Response:**
- Returns the detected IP address
- Returns empty response if no valid IP can be extracted

#### `GET /iam/{name}`

Registers a name with the client's IP address. Stores the mapping of `name` to the client's real IP.

**Request:**
- `{name}` - Path parameter for the name to register
- Client IP extracted from X-Forwarded-For → X-Real-Ip → RemoteAddr (fallback chain)

**Response:**
- Returns the stored IP address
- Returns `400 Bad Request` if no valid IP can be extracted from any source

#### `GET /iam/{name}/{ip}`

Registers a name with a manually specified IP address.

**Request:**
- `{name}` - Path parameter for the name to register
- `{ip}` - Path parameter for the IP address (IPv4 or IPv6)

**Response:**
- If `{ip}` is a valid IPv4/IPv6: stores and returns the provided IP
- If `{ip}` is invalid: falls back to `GET /iam/{name}` behavior

#### `GET /whois/{name}`

Looks up a previously registered name and returns the associated IP address.

**Request:**
- `{name}` - Path parameter for the name to look up

**Response:**
- Returns the IP address associated with the name
- Returns `404 Not Found` if the name is not registered

## Features

### 1. Persistent IP Storage

By default, the service stores name-to-IP mappings only in memory, which are lost on restart. The service can persist name-to-IP mappings across restarts by storing them in the `config.json` file.

#### Configuration

To use this feature, create a `config.json` file with a `who` section (see `config.example.json` for reference).

The `who` config section in `config.json` pre-loads name-to-IP mappings on startup and writes changes back to `config.json` when updated via `/iam/{name}`. This means IPs survive container restarts without needing to re-register.

Predefine addresses by including both `iam` and `ip` in the config so the service can respond to `/whois/{name}` immediately. Alternatively, include only `iam` entries and allow clients to register and update addresses dynamically via the `/iam/{name}` endpoints.

```json
{
  "who": [
    { "iam": "juliav4", "ip": "111.111.111.111" },
    { "iam": "juliav6" }
  ]
}
```

- On startup: entries with `ip` set are loaded into the store, immediately available via `/whois/{name}`
- On IP change: if `{name}` is in the `who` config, the new IP is written back to `config.json`
- Names not in the `who` config are stored in memory only (lost on restart)

### 2. DDNS

The DDNS feature allows automatic DNS updates when a name is registered or updated via `/iam/{name}`. When an IP address changes, the configured DNS provider is updated asynchronously.

Currently, only AWS Route53 is supported.

#### Configuration

To use this feature, create a `config.json` file with a `ddns` section (see `config.example.json` for reference):

```json
{
  "ddns": [
    {
      "provider": "route53",
      "domain": "julia.ddns.example.com",
      "ip_version": "ipv4",
      "access_key": "AKIAIOSFODNN7EXAMPLE",
      "secret_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
      "zone_id": "Z3M3LMPEXAMPLE",
      "ttl": 300,
      "iam": "juliav4"
    }
  ]
}
```

| Field        | Description                                                                 |
|--------------|-----------------------------------------------------------------------------|
| `provider`   | DNS provider (currently only `route53` is supported)                        |
| `domain`     | Domain to update (e.g., `sub.example.com`, `example.com`, `*.example.com`)  |
| `ip_version` | `ipv4` for A records, `ipv6` for AAAA records                               |
| `access_key` | AWS Access Key ID                                                           |
| `secret_key` | AWS Secret Access Key                                                       |
| `zone_id`    | Route53 Hosted Zone ID                                                      |
| `ttl`        | DNS record TTL in seconds (default: 300)                                    |
| `iam`        | Name that triggers this DDNS update (matches `{name}` in `/iam/{name}`)     |

#### How It Works

1. A client calls `/iam/{name}` with an IP address
2. The IP is stored in memory and returned immediately
3. If `{name}` matches an `iam` field in the DDNS config, and the IP changed, a background update is triggered
4. The DNS update runs asynchronously and does not block the API response
5. DDNS failures are logged but do not affect the `/whois/{name}` lookup

### 3. Webhook Notifications

The webhook feature sends HTTP notifications to external services when a name's IP address changes. This is useful for triggering firewall allowlist reloads, cache invalidations, or other automation tasks.

#### Configuration

Add `webhooks` entries to your `config.json`:

```json
{
  "webhooks": [
    {
      "iam": "juliav4",
      "url": "https://example.com/reload/allowlist",
      "method": "POST",
      "headers": {
        "Authorization": "Bearer token123"
      }
    }
  ]
}
```

#### Webhook Configuration Fields

| Field     | Description                                                         |
|-----------|---------------------------------------------------------------------|
| `iam`     | Name that triggers this webhook (matches `{name}` in `/iam/{name}`) |
| `url`     | Target URL to send the HTTP request                                 |
| `method`  | HTTP method (default: `POST`)                                       |
| `headers` | Custom headers (e.g., authentication tokens)                        |

#### Webhook Payload

Webhooks send a fixed JSON payload with `Content-Type: application/json`:

```json
{
  "iam": "juliav4",
  "ip": "203.0.113.50",
  "timestamp": "2026-01-28T12:34:56Z"
}
```

#### How It Works

1. A client calls `/iam/{name}` and the IP changes
2. If `{name}` matches an `iam` field in the webhook config, a notification is triggered
3. The webhook sends the request asynchronously
4. Multiple webhooks can be configured for the same `iam` name

#### Example Use Case

Configure a webhook to reload a firewall allowlist when a client's IP changes:

```json
{
  "webhooks": [
    {
      "iam": "office-vpn",
      "url": "https://firewall.example.com/api/reload",
      "method": "POST",
      "headers": {
        "Authorization": "Bearer secret-token",
        "X-Action": "reload-allowlist"
      }
    }
  ]
}
```

When a client calls `/iam/office-vpn`, the firewall receives the new IP and can update its allowlist automatically.

### 4. Aliases

The alias feature allows grouping multiple IAM names together. When querying an alias via `/whois/{alias}`, the service returns all associated IP addresses, one per line.

Aliases are read-only and cannot be updated via `/iam/{alias}`. They don't trigger DDNS updates or webhook notifications. IPs are resolved dynamically from the current store values.

#### Configuration

Add alias entries to the `who` section in your `config.json`:

```json
{
  "who": [
    {
      "iam": "juliav4",
      "ip": "111.111.111.111"
    },
    {
      "iam": "juliav6",
      "ip": "2001:db8::1"
    },
    {
      "iam": "julia",
      "alias": ["juliav4", "juliav6"]
    }
  ]
}
```

In this example:
- `juliav4` and `juliav6` are regular IAM entries with IP addresses
- `julia` is an alias that references both `juliav4` and `juliav6`

#### How It Works

1. Define an alias entry with the `alias` field containing an array of IAM names
2. Query the alias via `/whois/{alias}` to get all associated IPs
3. The service resolves each IAM name in the alias list and returns their current IP addresses
4. If an aliased name has no IP, it's omitted from the response
5. If none of the aliased names have IPs, the request returns `404 Not Found`

#### Usage Example

```console
# Query an alias - returns multiple IPs, one per line
$ curl http://localhost:8080/whois/julia
111.111.111.111
2001:db8::1

# Attempting to update an alias is rejected
$ curl http://localhost:8080/iam/julia
cannot update alias
```

#### Notes

- Aliases are resolved at query time, so they always reflect current IP values
- Circular aliases are not validated - avoid creating them
- Aliases cannot reference other aliases (only regular IAM names work)

