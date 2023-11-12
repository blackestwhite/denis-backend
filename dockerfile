FROM golang:1.19.0-alpine3.16 as builder

WORKDIR /app

# Copy only the dependency files and download them separately.
COPY go.* ./
RUN go mod download

# Copy the entire application code.
COPY . .

# Build the Go application with optimizations.
RUN CGO_ENABLED=0 go build -o /app/app -ldflags="-s -w"

# Use a minimalistic Alpine Linux as the final base image.
FROM alpine:3.14

# Install necessary packages (e.g., timezone data).
RUN apk add --no-cache tzdata

# Set the timezone
ENV TZ=Asia/Tehran

# Create a non-root user for running the application for better security.
RUN adduser -D -u 1001 myuser
USER myuser

# Set the working directory.
WORKDIR /app

# Copy the binary from the builder stage.
COPY --from=builder --chown=myuser /app/app .

# Expose the port your application listens on.
EXPOSE 8080

CMD ["/app/app"]