FROM golang:1.11 as builder

WORKDIR /go/src/github.com/summerwind/eventreactor

COPY pkg/    pkg/
COPY cmd/    cmd/
COPY vendor/ vendor/

RUN CGO_ENABLED=0 go build -a -o event-init github.com/summerwind/eventreactor/cmd/event-init

#################################################

FROM scratch

COPY --from=builder /go/src/github.com/summerwind/eventreactor/event-init /bin/event-init

CMD ["/bin/event-init"]
