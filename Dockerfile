# Start with a minimal Go environment
FROM golang:1.22.5 as builder

# Set the working directory
WORKDIR /app

# Copy the Go source code to the container
COPY . .

# Build the Go program
RUN GOOS=linux GOARCH=arm64 go build -o prometheus_exporter .

# Use a minimal base image for the final container
FROM arm64v8/alpine:latest

# Set working directory in container
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/prometheus_exporter /app/prometheus_exporter

# Expose the port the app runs on
EXPOSE 8080

# Run the Prometheus exporter
CMD ["./prometheus_exporter"]