FROM golang:1.24.4 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/gcpalert2slack ./cmd/gcpalert2slack

FROM gcr.io/distroless/static-debian12

ENV PORT=8080

COPY --from=builder /out/gcpalert2slack /gcpalert2slack

EXPOSE 8080

ENTRYPOINT ["/gcpalert2slack"]
