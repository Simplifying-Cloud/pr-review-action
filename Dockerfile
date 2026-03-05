FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/review .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/review /bin/review
ENTRYPOINT ["/bin/review"]
