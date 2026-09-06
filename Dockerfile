FROM golang:alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o voidsink .

FROM scratch
COPY --from=builder /app/voidsink /voidsink
EXPOSE 23 6379 8080
ENTRYPOINT ["/voidsink"]
