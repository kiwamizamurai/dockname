FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dockname ./cmd/dockname

FROM gcr.io/distroless/static-debian11:nonroot

COPY --from=builder /app/dockname /dockname
USER nonroot:nonroot

ENTRYPOINT ["/dockname"]