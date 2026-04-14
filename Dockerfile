# ── Stage 1: Build ────────────────────────────────────────────────────────────
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Cache dependencies separately from source
COPY go.mod go.sum ./
RUN go mod tidy

# Copy the rest of the source
COPY . .

# Build the binary from the correct entry-point package
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/gp_software_dev_project

# ── Stage 2: Run ──────────────────────────────────────────────────────────────
FROM alpine:3.19

WORKDIR /app

# ca-certificates for TLS; tzdata for correct time zone handling
RUN apk add --no-cache ca-certificates tzdata

# Copy the compiled binary from the builder stage
COPY --from=builder /app/server .

EXPOSE 42069

ENTRYPOINT ["./server"]
