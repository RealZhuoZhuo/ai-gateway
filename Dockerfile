FROM golang:1.23-alpine AS builder

WORKDIR /src
ENV GO111MODULE=on \
	GOPROXY=https://goproxy.cn,direct

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/ai-gateway ./cmd/gateway

FROM alpine:3.20

RUN addgroup -S app && adduser -S -G app app \
	&& apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /out/ai-gateway /usr/local/bin/ai-gateway

USER app
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/ai-gateway"]
