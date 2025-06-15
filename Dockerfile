# syntax=docker/dockerfile:1.4

# Stage 1: Build
FROM --platform=$BUILDPLATFORM golang:1.23 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build static binary for target platform
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$(go env GOARCH) go build -o push_app

# Stage 2: Minimal final image
FROM --platform=$TARGETPLATFORM scratch
WORKDIR /app
COPY --from=builder /app/push_app .

EXPOSE 10026
ENTRYPOINT ["./push_app"]