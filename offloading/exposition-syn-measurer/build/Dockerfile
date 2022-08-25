FROM golang:1.18 as builder
WORKDIR /tmp/builder

RUN apt update && apt install --yes libpcap-dev

COPY go.mod ./go.mod
COPY go.sum ./go.sum
RUN  go mod download

COPY . ./
RUN CGO_ENABLED=1 GOOS=linux GOARCH=$(go env GOARCH) go build -ldflags="-s -w" -o exposition-syn-measurer ./cmd/main.go


FROM debian:bullseye

RUN apt update && \
    apt install -y ca-certificates libpcap0.8 && \
    update-ca-certificates && \
    apt-get clean -y && \
    rm -rf /var/lib/apt/lists/*.*

ARG COMPONENT
COPY --from=builder /tmp/builder/exposition-syn-measurer /usr/bin/exposition-syn-measurer

ENTRYPOINT [ "/usr/bin/exposition-syn-measurer" ]
