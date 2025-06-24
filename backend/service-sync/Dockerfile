# build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# copy go mod files
COPY ../../go.mod ../../go.sum ./

# download dependencies
RUN go mod download

# copy source code
COPY . .

# build the service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/sync-service cmd/main.go

# runtime stage
FROM alpine:latest

# install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# copy the binary from builder stage
COPY --from=builder /app/bin/sync-service .

# expose port
EXPOSE 8081

# run the service
CMD ["./sync-service"]
