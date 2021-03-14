FROM golang:1.15.7 as builder

WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go

FROM ubuntu:20.04
WORKDIR /
COPY --from=builder /workspace/manager .

ENTRYPOINT ["/manager"]
