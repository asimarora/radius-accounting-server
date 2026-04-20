FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG BUILD_TARGET
RUN CGO_ENABLED=0 GOOS=linux go build -o /app-binary ./cmd/${BUILD_TARGET}

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
RUN mkdir -p /var/log
COPY --from=builder /app-binary /usr/local/bin/app-binary
ENV CONFIG_DIR=/etc/radius/configs
ENTRYPOINT ["app-binary"]
