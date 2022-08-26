FROM golang:1.18 as builder
WORKDIR /tmp/builder

COPY go.mod ./go.mod
COPY go.sum ./go.sum
RUN  go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$(go env GOARCH) go build -ldflags="-s -w" -o storage-measurer ./cmd/main.go


FROM alpine:3.14

RUN apk update && \
    apk add --no-cache ca-certificates && \
    update-ca-certificates && \
    rm -rf /var/cache/apk/*

ARG COMPONENT
COPY --from=builder /tmp/builder/storage-measurer /usr/bin/storage-measurer

ENTRYPOINT [ "/usr/bin/storage-measurer" ]
