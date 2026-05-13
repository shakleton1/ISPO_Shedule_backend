# syntax=docker/dockerfile:1

FROM golang:1.25.10-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

RUN CGO_ENABLED=0 go install github.com/pressly/goose/v3/cmd/goose@v3.26.0


FROM debian:bookworm-slim AS runtime

RUN apt-get update \
  && apt-get install -y --no-install-recommends \
     ca-certificates \
     tzdata \
     curl \
     postgresql-client \
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
COPY scripts/entrypoint.sh /usr/local/bin/entrypoint.sh

# Needed at runtime: OpenAPI spec endpoint serves docs/openapi.yaml
COPY docs /app/docs

# Useful for one-off migration containers: goose reads these files.
COPY db/migrations /app/db/migrations

# Optional reference; runtime should mount a real config.yaml
COPY configs/config.example.yaml /app/configs/config.example.yaml

RUN sed -i 's/\r$//' /usr/local/bin/entrypoint.sh \
  && chown -R app:app /app \
  && chmod +x /usr/local/bin/entrypoint.sh

USER app

ENV ISPO_SERVER_ADDR=0.0.0.0:8080 \
    ISPO_PDF_CHROME_EXECUTABLE_PATH=/usr/bin/chromium \
    ISPO_CONFIG_PATH=/app/config.yaml

EXPOSE 8080

# Use entrypoint script that applies migrations before starting API
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
