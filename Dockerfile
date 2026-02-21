# Stage 1: Build frontend
FROM oven/bun:1 AS frontend
WORKDIR /app/web
COPY web/package.json web/bun.lock ./
RUN bun install --frozen-lockfile
COPY web/ .
RUN bun run build

# Stage 2: Build Go binary
FROM golang:1.23-alpine AS backend
WORKDIR /app

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X github.com/shiroha/subdux/internal/version.Version=${VERSION} -X github.com/shiroha/subdux/internal/version.Commit=${COMMIT} -X github.com/shiroha/subdux/internal/version.BuildDate=${BUILD_DATE}" \
    -o /subdux ./cmd/server

# Stage 3: Minimal runtime
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=backend /subdux /subdux
EXPOSE 8080
ENTRYPOINT ["/subdux"]
