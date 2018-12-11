FROM golang:1.11 as builder

WORKDIR /go/src/github.com/summerwind/eventreactor

COPY pkg/    pkg/
COPY cmd/    cmd/
COPY vendor/ vendor/

RUN CGO_ENABLED=0 go build -a -o manager github.com/summerwind/eventreactor/cmd/manager

#################################################

FROM scratch

COPY --from=builder /go/src/github.com/summerwind/eventreactor/manager /bin/manager

CMD ["/bin/manager"]
