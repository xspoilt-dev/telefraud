# ---- build stage: compile a static Go binary ----

FROM golang:1.25-alpine AS build

WORKDIR /src

# Cache the module graph before copying source.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/telefraud ./cmd/telefraud

# ---- runtime stage: minimal image with CA certs (for TLS to Postgres) ----
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata postgresql16-client bash gzip \
    && addgroup -S app \
    && adduser -S -G app app \
    && mkdir -p /backups \
    && chown -R app:app /backups

COPY --from=build /out/telefraud /usr/local/bin/telefraud

USER app
ENTRYPOINT ["telefraud"]

