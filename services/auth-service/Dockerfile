FROM golang:1.22-alpine AS builder
WORKDIR /app
ENV GOTOOLCHAIN=auto
RUN apk add --no-cache build-base
COPY . .
RUN CGO_ENABLED=1 go mod tidy && CGO_ENABLED=1 go build -o auth-service .

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/auth-service .
EXPOSE 8080
CMD ["./auth-service"]
