# go-egts [![GoDoc](https://godoc.org/github.com/LdDl/go-egts?status.svg)](https://godoc.org/github.com/LdDl/go-egts) [![Sourcegraph](https://sourcegraph.com/github.com/LdDl/go-egts/-/badge.svg)](https://sourcegraph.com/github.com/LdDl/go-egts?badge) [![Go Report Card](https://goreportcard.com/badge/github.com/LdDl/go-egts)](https://goreportcard.com/report/github.com/LdDl/go-egts) [![GitHub tag](https://img.shields.io/github/tag/LdDl/go-egts.svg)](https://github.com/LdDl/go-egts/releases) [![Build Status](https://travis-ci.com/LdDl/go-egts.svg?branch=master)](https://travis-ci.com/LdDl/go-egts)

[Russian version (версия на русском)](README_RU.md)

Go library for decoding and encoding EGTS (Era Glonass Telematics Standard) packets. The repository also includes `egts_gateway`, a TCP server built on this library for receiving, storing and forwarding telemetry.

## Table of Contents

- [About](#about)
- [Installation](#installation)
- [Library examples](#library-examples)
- [Gateway](#gateway)
- [Docker](#docker)
- [Tests](#tests)
- [Support](#support)
- [License](#license)

## About

EGTS stands for Era Glonass Telematics Standard. The library decodes and encodes EGTS packets, records and subrecords. See the [protocol description in Russian](https://docs.cntd.ru/document/1200095098) and [packet walkthroughs](docs_rus).

`egts_gateway` accepts EGTS connections with optional authentication and delivers packets to stdout, rotating files, other EGTS servers, RabbitMQ and Valkey/Redis Streams. Multiple destinations can be enabled together.

## Installation

### Go library

```bash
go get github.com/LdDl/go-egts
```

### Gateway from Go

```bash
go install github.com/LdDl/go-egts/cmd/egts_gateway@latest
```

### Build from source

Run this command from the repository root to build a binary for the current platform:

```bash
go build -o egts_gateway ./cmd/egts_gateway
```

### Pre-built binaries

Download the archive for your platform from the [latest release](https://github.com/LdDl/go-egts/releases/latest):

| Platform | Archive |
| --- | --- |
| Linux amd64 | [linux-amd64-egts_gateway.tar.gz](https://github.com/LdDl/go-egts/releases/latest/download/linux-amd64-egts_gateway.tar.gz) |
| Linux arm64 | [linux-arm64-egts_gateway.tar.gz](https://github.com/LdDl/go-egts/releases/latest/download/linux-arm64-egts_gateway.tar.gz) |
| macOS amd64 | [darwin-amd64-egts_gateway.tar.gz](https://github.com/LdDl/go-egts/releases/latest/download/darwin-amd64-egts_gateway.tar.gz) |
| macOS arm64 | [darwin-arm64-egts_gateway.tar.gz](https://github.com/LdDl/go-egts/releases/latest/download/darwin-arm64-egts_gateway.tar.gz) |
| Windows amd64 | [windows-amd64-egts_gateway.zip](https://github.com/LdDl/go-egts/releases/latest/download/windows-amd64-egts_gateway.zip) |

Each archive contains `egts_gateway` or `egts_gateway.exe`. Extract it into a directory in your `PATH`.

Quick installation on Linux amd64:

```bash
curl -fsSL https://github.com/LdDl/go-egts/releases/latest/download/linux-amd64-egts_gateway.tar.gz \
  | sudo tar -xz -C /usr/local/bin egts_gateway
```

For Linux arm64, replace `linux-amd64` with `linux-arm64`. On macOS, use `darwin-amd64` for Intel or `darwin-arm64` for Apple Silicon.

## Library examples

The [egts_server](cmd/egts_server) and [egts_client](cmd/egts_client) commands demonstrate how to use the library. Start them in separate terminals:

- Start server

    ```shell
    go run cmd/egts_server/main.go
    ```

- Start client

    ```shell
    go run cmd/egts_client/main.go
    ```

After you start both server and client, you should see something like this:

- Client side

    ```shell
    go run cmd/egts_client/main.go
    2021/12/16 21:18:15 Response code: {0 0      0 0 0 0 0 0 0 0 0 <nil> 0 0}
    2021/12/16 21:18:15 Packet: {1 0 00 11 0 00 0 11 0 16 1 0 0 0 0 245 0xc0000040c0 4587 0}
    ```

- Server side

    ```shell
    go run cmd/egts_server/main.go
    2021/12/16 21:18:08 Accept connection on port 8081
    2021/12/16 21:18:15 Calling handleConnection for remote address: [::1]:50840
    2021/12/16 21:18:15 PosData is:
            OID: 825791382 | Longitude: 48.362186 | Latitude: 54.287315 | Time: 2021-12-16 21:18:15.1412741 +0300 MSK m=+7.036938801
    2021/12/16 21:18:15 Result code has been sent to '[::1]:50840'
    ```

`Packet.Encode()` returns `([]byte, error)`. Check the error before sending the encoded bytes. The encoder validates packet flags and lengths and returns errors from nested records and subrecords.

## Gateway

Start with the defaults:

```bash
egts_gateway
```

The server listens on `0.0.0.0:8081`, accepts connections without authentication, writes packets to stdout and application logs to stderr. Stop it with `Ctrl+C`.

### TOML configuration

Download the [example configuration](egts_gateway.toml), edit it and pass its path explicitly:

```bash
curl -fsSL https://raw.githubusercontent.com/LdDl/go-egts/master/egts_gateway.toml -o egts_gateway.toml
egts_gateway -conf egts_gateway.toml
```

With `-conf`, configuration comes from TOML and defaults. Environment variables and `.env` are ignored. Relative data paths are resolved from the process working directory.

To require incoming authentication, set `auth_cfg.enabled = true` and configure `auth_cfg.password`. Authentication is optional for outgoing EGTS relays as well.

### Environment configuration

Without `-conf`, the gateway reads environment variables. If `.env` exists in the working directory, its values override the process environment. The [.env.example](.env.example) file lists the available variables.

For example, save packets to rotating files:

```bash
EGTS_SERVER_PORT=8081 \
EGTS_PACKETS_STDOUT=false \
EGTS_PACKETS_FILE_ENABLED=true \
egts_gateway
```

Multiple destinations use a comma-separated ID list and variable names with the ID at the end. For example, these entries in `.env` enable two EGTS relays:

```dotenv
EGTS_RELAY_IDS=monitoring,backup
EGTS_RELAY_ENABLED_MONITORING=true
EGTS_RELAY_HOST_MONITORING=127.0.0.1
EGTS_RELAY_PORT_MONITORING=8082
EGTS_RELAY_ENABLED_BACKUP=true
EGTS_RELAY_HOST_BACKUP=127.0.0.1
EGTS_RELAY_PORT_BACKUP=8083
```

RabbitMQ and Valkey use `EGTS_RABBITMQ_IDS` and `EGTS_VALKEY_IDS` with the same suffix convention, such as `EGTS_RABBITMQ_PASSWORD_BACKUP` and `EGTS_VALKEY_PASSWORD_BACKUP`. In TOML, add one `[[destinations_cfg.egts]]`, `[[destinations_cfg.rabbitmq]]` or `[[destinations_cfg.valkey]]` entry per destination.

### Packet destinations

| Destination | Behavior |
| --- | --- |
| stdout | One JSON packet envelope per line. |
| File | NDJSON in `./data/packets/packets.ndjson`, with rotation by size, archive count, age and total size. |
| EGTS | Forward packets to one or more servers and wait for their EGTS responses. |
| RabbitMQ | Publish persistent JSON messages to configured durable queues and wait for publisher confirmations. |
| Valkey/Redis | Append entries to configured Streams with `message_id` and `payload` fields. |

Application logging is configured separately through `logs_cfg`. File logs go to `./data/logs/application.log` by default. The gateway rejects configurations where application logs, packet files or queue dumps would share their output location. File logs and packet files have separate rotation settings.

The JSON envelope contains `record.received_at`, `record.source`, the decoded `record.packet` and `record.raw`. The `raw` field contains the original EGTS frame encoded as Base64. RabbitMQ message IDs and Valkey/Redis `message_id` fields remain stable across delivery retries and dump restoration.

For Valkey/Redis, leave `username` empty to use `AUTH password`, or set both `username` and `password` to use `AUTH username password`. Leave both empty for a server without authentication. Stream trimming is controlled by the consumer or server administration.

### Delivery and recovery

`delivery_cfg.ack_mode` controls when the terminal receives a successful acknowledgement:

| Mode | Acknowledgement |
| --- | --- |
| `queued` | After the packet enters the memory queue. This is the default. |
| `delivered` | After every enabled destination accepts the packet. |

The queue holds up to `queue_capacity` packets, defaulting to 1024. When full, the gateway rejects new packets until space becomes available. Destinations retry independently, so one unavailable recipient does not block delivery to the others.

On shutdown, pending deliveries are saved under `dump_directory`, which defaults to `./data/queue`. If a destination keeps failing for `dump_after_seconds`, defaulting to 60, the gateway saves the pending queue and exits with an error. The next startup restores it and retries the remaining destinations. Keep their IDs enabled in the configuration when restoring a dump.

The `queued` mode acknowledges data held in memory, so a forced process termination or host failure before a dump can lose queued packets. Retries can produce duplicates if a destination accepted a packet but its confirmation was lost; consumers can use the stable message ID for deduplication.

Gateway examples include configurations and packet delivery checks for [multiple EGTS relays](gateway/examples/relay), [RabbitMQ](gateway/examples/rabbitmq) and [Valkey/Redis](gateway/examples/valkey).

## Docker

The image is published as [dimahkiin/egts_gateway](https://hub.docker.com/r/dimahkiin/egts_gateway). Start the latest image in the foreground:

```bash
docker run --rm --name egts_gateway --stop-timeout 30 \
  -p 127.0.0.1:8081:8081 \
  -v egts_gateway_data:/app/data \
  docker.io/dimahkiin/egts_gateway:latest
```

The named volume preserves packet files, file logs and queue dumps. The default configuration writes packets to stdout. To accept connections from other hosts, change the port mapping to `8081:8081`. Stop the container with `Ctrl+C`.

To use the TOML file downloaded above:

```bash
docker run --rm --name egts_gateway --stop-timeout 30 \
  -p 127.0.0.1:8081:8081 \
  --mount type=bind,source="$(pwd)/egts_gateway.toml",target=/app/egts_gateway.toml,readonly \
  -v egts_gateway_data:/app/data \
  docker.io/dimahkiin/egts_gateway:latest -conf /app/egts_gateway.toml
```

For ENV configuration, add `--mount type=bind,source="$(pwd)/.env",target=/app/.env,readonly` before the image name in the first command. The gateway loads this file itself. Keep the server listening on `0.0.0.0:8081` inside the container and store its data under `/app/data`.

### Docker Compose

From the repository root, use the published image with the included TOML configuration:

```bash
EGTS_GATEWAY_IMAGE=dimahkiin/egts_gateway:latest \
docker compose -f gateway/compose.yaml up --no-build --pull always
```

For ENV configuration, create `.env` from `.env.example`, edit it, then run:

```bash
EGTS_GATEWAY_IMAGE=dimahkiin/egts_gateway:latest \
docker compose -f gateway/compose.yaml -f gateway/compose.env.yaml up --no-build --pull always
```

`EGTS_GATEWAY_PORT` changes the published host port, `EGTS_GATEWAY_BIND_HOST` changes the host bind address and `EGTS_GATEWAY_CONFIG` selects a TOML file. Use an absolute path for a custom configuration file. `EGTS_GATEWAY_ENV_FILE` selects an ENV file for the ENV variant. The `gateway_data` volume survives container recreation.

The [infrastructure compose](gateway/infrastructure/compose.yaml) provides two RabbitMQ instances and two Valkey instances for testing. To run them in the same Compose network as the gateway, enable the desired destinations in `egts_gateway.toml` and set their hosts to `rabbitmq_monitoring`, `rabbitmq_backup`, `valkey_monitoring` or `valkey_backup`. Use port `5672` for either RabbitMQ instance and `6379` for either Valkey instance inside that network.

```bash
EGTS_GATEWAY_IMAGE=dimahkiin/egts_gateway:latest \
docker compose -f gateway/compose.yaml -f gateway/infrastructure/compose.yaml up --no-build --pull always
```

## Tests

Run all unit tests:

```bash
go test ./...
```

Integration tests use the example infrastructure and are opt-in. Start the required RabbitMQ or Valkey services in one terminal:

```bash
docker compose -f gateway/infrastructure/compose.yaml up
```

With the default example credentials and ports, run the integration tests in another terminal:

```bash
GO_EGTS_RABBITMQ_INTEGRATION=1 GO_EGTS_VALKEY_INTEGRATION=1 go test ./gateway/...
```

## Support
If you have troubles or questions please [open an issue](https://github.com/LdDl/go-egts/issues/new/choose).

Pull requests are welcome.

## License
It's MIT. You can check it [here](https://github.com/LdDl/go-egts/blob/master/LICENSE.md)
