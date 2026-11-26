# Static Go binary: no runtime, no node_modules on the tiny VPS.
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# BuildKit cache mounts keep the module + compile caches across rebuilds:
# without them every deploy recompiles stdlib and deps from scratch.
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -o /fatsecret-mcp ./cmd/fatsecret-mcp

FROM alpine:3.21
RUN apk add --no-cache tzdata ca-certificates && mkdir -p /home/app/.fatsecret-mcp
ENV HOME=/home/app
WORKDIR /app
COPY --from=build /fatsecret-mcp /app/fatsecret-mcp
# Profile tokens live in ~/.fatsecret-mcp/config.json; mount it as a volume
VOLUME /home/app/.fatsecret-mcp
ENV PORT=8000
EXPOSE 8000
ENTRYPOINT ["/app/fatsecret-mcp"]
