FROM golang:1.22-alpine AS builder
WORKDIR /app
ENV GOTOOLCHAIN=auto
COPY . .
RUN go mod tidy && go build -o gateway .

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/gateway .
EXPOSE 8080
CMD ["./gateway"]
