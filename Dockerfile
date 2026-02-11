# Use the official Golang image for building
FROM golang:1.25.1 as builder

WORKDIR /app

# Copy Go modules manifests
COPY go.mod go.sum ./
RUN go mod download

# Copy all source files
COPY . .

# Build the Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Use a small image for runtime
FROM gcr.io/distroless/base-debian11

WORKDIR /app
COPY --from=builder /app/main .
COPY templates ./templates
COPY static ./static

# Expose port
ENV PORT=8080
EXPOSE 8080

# Run the binary
CMD ["./main"]
