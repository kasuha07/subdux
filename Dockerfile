# Stage 1: Build frontend
FROM --platform=$BUILDPLATFORM oven/bun:1 AS frontend
WORKDIR /app/web
COPY web/package.json web/bun.lock ./
RUN bun install --frozen-lockfile
COPY web/ .
RUN bun run build

# Stage 2: Build Go binary
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS backend
WORKDIR /app

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
ARG TARGETOS=linux
ARG TARGETARCH

COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-$(go env GOARCH)} go build \
    -ldflags="-s -w -X github.com/shiroha/subdux/internal/version.Version=${VERSION} -X github.com/shiroha/subdux/internal/version.Commit=${COMMIT} -X github.com/shiroha/subdux/internal/version.BuildDate=${BUILD_DATE}" \
    -o /subdux ./cmd/server

# Stage 3: Minimal runtime
FROM gcr.io/distroless/static-debian12:nonroot
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
ARG REPOSITORY_URL=https://github.com/kasuha07/subdux
LABEL org.opencontainers.image.title="Subdux" \
      org.opencontainers.image.description="Self-hosted subscription tracker with notifications, multi-currency, and modern auth." \
      org.opencontainers.image.source="${REPOSITORY_URL}" \
      org.opencontainers.image.url="${REPOSITORY_URL}" \
      org.opencontainers.image.documentation="${REPOSITORY_URL}#readme" \
      org.opencontainers.image.licenses="GPL-3.0-or-later" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT}" \
      org.opencontainers.image.created="${BUILD_DATE}"
COPY --from=backend /subdux /subdux
EXPOSE 8080
ENTRYPOINT ["/subdux"]
