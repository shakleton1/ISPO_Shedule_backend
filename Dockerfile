# syntax=docker/dockerfile:1

FROM golang:1.24.0-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

RUN CGO_ENABLED=0 go install github.com/pressly/goose/v3/cmd/goose@v3.24.1


FROM debian:bookworm-slim AS runtime

RUN apt-get update \
  && apt-get install -y --no-install-recommends \
     ca-certificates \
     tzdata \
     curl \
     chromium \
     fonts-dejavu-core \
     fonts-liberation \
     fonts-noto-core \
     fonts-noto-color-emoji \
  && rm -rf /var/lib/apt/lists/*

RUN useradd -r -u 10001 -m app

WORKDIR /app

COPY --from=build /out/api /app/api
COPY --from=build /go/bin/goose /usr/local/bin/goose

# Needed at runtime: OpenAPI spec endpoint serves docs/openapi.yaml
COPY docs /app/docs

# Useful for one-off migration containers: goose reads these files.
COPY db/migrations /app/db/migrations

# Optional reference; runtime should mount a real config.yaml
COPY configs/config.example.yaml /app/configs/config.example.yaml

RUN chown -R app:app /app
USER app

ENV ISPO_SERVER_ADDR=0.0.0.0:8080 \
    ISPO_PDF_CHROME_EXECUTABLE_PATH=/usr/bin/chromium \
    ISPO_CONFIG_PATH=/app/config.yaml

EXPOSE 8080

# Liveness: API process responds.
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=3 \
  CMD curl -fsS http://127.0.0.1:8080/api/v1/health || exit 1

CMD ["/app/api"]
