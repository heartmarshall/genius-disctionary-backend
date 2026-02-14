# ---- Build stage ----
FROM golang:1.23 AS builder

WORKDIR /src

# Copy dependency manifests first for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Build a statically-linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/server ./cmd/server

# ---- Runtime stage ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates \
    && addgroup -S nonroot && adduser -S nonroot -G nonroot

COPY --from=builder /bin/server /bin/server
COPY config.yaml /etc/myenglish/config.yaml

USER nonroot

EXPOSE 8080

ENTRYPOINT ["/bin/server"]
