# Stage 1: Build the Go application
FROM golang:1.22-alpine3.19 AS builder

WORKDIR /app

COPY . .

# Build the executable with no debug information
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o nfna

# Stage 2: Create a minimal image
FROM scratch

WORKDIR /app

# Copy only the built executable from the previous stage
COPY --from=builder /app/secrets /run/secrets
COPY --from=builder /app/nfna .

# Set the entrypoint to run the executable
ENTRYPOINT ["./nfna"]

# Allow the container to access anything on a shared Docker network
# This assumes the shared network is named "my-network"
# Modify the network name if necessary
CMD ["--network", "no-fun-neal"]
