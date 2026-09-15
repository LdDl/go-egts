# syntax=docker/dockerfile:1
FROM docker.io/golang:1.25.9-alpine3.23 AS builder

WORKDIR /build_app

ENV GOTOOLCHAIN=local
ENV GOPROXY=https://proxy.golang.org

# Download dependencies before copying application sources.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY ./crc ./crc
COPY ./egts ./egts
COPY ./gateway ./gateway
COPY ./cmd ./cmd

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -mod=readonly -buildvcs=false -trimpath -ldflags="-s -w" -o egts_gateway ./cmd/egts_gateway

FROM scratch

WORKDIR /app

COPY --from=builder /build_app/egts_gateway ./

EXPOSE 8081

ENTRYPOINT ["./egts_gateway"]
