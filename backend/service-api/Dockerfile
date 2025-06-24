# build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# copy go.mod and go.sum
COPY go.mod go.sum ./

# download dependencies
RUN go mod download

# copy source code
COPY . .

# build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./service-api/cmd

# final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# copy the binary from builder stage
COPY --from=builder /app/main .

# expose port
EXPOSE 8080

# run the binary
CMD ["./main"]
