# Build the manager binary
FROM golang:1.11 as builder

# Copy in the go src
WORKDIR /go/src/github.com/summerwind/eventreactor
COPY pkg/    pkg/
COPY cmd/    cmd/
COPY vendor/ vendor/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager   github.com/summerwind/eventreactor/cmd/manager
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o apiserver github.com/eventreactor/eventreactor/cmd/apiserver

#################################################

FROM scratch

COPY --from=builder /go/src/github.com/summerwind/eventreactor/manager /bin/manager
COPY --from=builder /go/src/github.com/summerwind/eventreactor/manager /bin/apiserver

CMD ["/bin/manager"]
