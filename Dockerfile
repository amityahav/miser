# builder
FROM golang:1.19-alpine AS builder

RUN apk add --no-progress --no-cache gcc musl-dev

WORKDIR /app
COPY go.* ./
COPY . .
#RUN go test -tags musl
RUN GOOS=linux GOARCH=amd64 go build -a -o miser ./cmd/main.go

FROM alpine:edge
WORKDIR /miser
COPY --from=builder /app/miser .
ENTRYPOINT ["/miser/miser"]
