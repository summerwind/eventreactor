FROM golang:1.11 as builder

WORKDIR /go/src/github.com/summerwind/eventreactor

COPY pkg/    pkg/
COPY cmd/    cmd/
COPY vendor/ vendor/

RUN CGO_ENABLED=0 go build -a -o apiserver github.com/summerwind/eventreactor/cmd/apiserver

#################################################

FROM scratch

COPY --from=builder /go/src/github.com/summerwind/eventreactor/apiserver /bin/apiserver

ENTRYPOINT ["/bin/apiserver"]
