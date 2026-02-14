# === Build Stage ===
FROM golang:1.24-alpine AS builder

# Build arguments for version information
ARG GIT_COMMIT
ARG BUILD_TIME
ARG VERSION

WORKDIR /marchat

# Copy dependency files first for better layer caching
COPY go.mod go.sum ./
COPY plugin/sdk/go.mod ./plugin/sdk/
RUN go mod download

# Copy source code (changes here won't invalidate the dependency cache)
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X github.com/Cod-e-Codes/marchat/shared.ClientVersion=${VERSION} -X github.com/Cod-e-Codes/marchat/shared.ServerVersion=${VERSION} -X github.com/Cod-e-Codes/marchat/shared.BuildTime='${BUILD_TIME}' -X github.com/Cod-e-Codes/marchat/shared.GitCommit=${GIT_COMMIT}" \
    -o marchat-server ./cmd/server

# === Runtime Stage ===
FROM alpine:3.22

# Update the package index and upgrade all installed packages
RUN apk update && apk upgrade --no-cache

# Build arguments for user/group ID
ARG USER_ID=1000
ARG GROUP_ID=1000

# Install necessary packages for user management
RUN apk add --no-cache shadow

# Create marchat user with specified UID/GID
RUN groupadd -g ${GROUP_ID} marchat && \
    useradd -u ${USER_ID} -g marchat -s /bin/sh -m marchat

# Create config directory with proper ownership
RUN mkdir -p /marchat/config && \
    chown -R marchat:marchat /marchat

# Switch to marchat user
USER marchat
WORKDIR /marchat

# Copy the binary from builder stage
COPY --from=builder /marchat/marchat-server .

# Copy entrypoint script
COPY --chown=marchat:marchat entrypoint.sh /marchat/entrypoint.sh
RUN chmod +x /marchat/entrypoint.sh

# Expose port 8080
EXPOSE 8080

ENTRYPOINT ["/marchat/entrypoint.sh"]
