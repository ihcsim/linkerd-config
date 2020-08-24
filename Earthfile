FROM golang:1.13

deps:
  WORKDIR /workspace
  COPY go.mod go.mod
  COPY go.sum go.sum
  RUN go mod download
  SAVE IMAGE

build:
  FROM +deps
  COPY main.go main.go
  COPY api/ api/
  COPY controllers/ controllers/
  COPY proto/ proto/
  RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go
  SAVE ARTIFACT manager AS LOCAL build/manager

image:
  FROM gcr.io/distroless/static:nonroot
  WORKDIR /
  USER nonroot:nonroot
  COPY +build/manager .
  ENTRYPOINT ["/manager"]
  SAVE IMAGE linkerd-config:earthbuild
