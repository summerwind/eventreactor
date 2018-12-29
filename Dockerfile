FROM golang:1.11 as builder

WORKDIR /go/src/github.com/summerwind/eventreactor

COPY pkg/    pkg/
COPY cmd/    cmd/
COPY vendor/ vendor/

RUN CGO_ENABLED=0 go build -o bin/manager    github.com/summerwind/eventreactor/cmd/manager
RUN CGO_ENABLED=0 go build -o bin/apiserver  github.com/summerwind/eventreactor/cmd/apiserver
RUN CGO_ENABLED=0 go build -o bin/event-init github.com/summerwind/eventreactor/cmd/event-init

#################################################

FROM scratch AS manager

COPY --from=builder /go/src/github.com/summerwind/eventreactor/bin/manager /bin/manager

ENTRYPOINT ["/bin/manager"]

#################################################

FROM scratch AS apiserver

COPY --from=builder /go/src/github.com/summerwind/eventreactor/bin/apiserver /bin/apiserver

ENTRYPOINT ["/bin/apiserver"]

#################################################

FROM scratch AS event-init

COPY --from=builder /go/src/github.com/summerwind/eventreactor/bin/event-init /bin/event-init

ENTRYPOINT ["/bin/event-init"]
