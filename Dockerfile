FROM golang:1.12 as builder

WORKDIR /go/src/github.com/summerwind/eventreactor

COPY vendor/  vendor/
COPY pkg/     pkg/
COPY cmd/     cmd/

RUN CGO_ENABLED=0 go build -o bin/manager github.com/summerwind/eventreactor/cmd/manager
RUN CGO_ENABLED=0 go build -o bin/event-receiver github.com/summerwind/eventreactor/cmd/event-receiver
RUN CGO_ENABLED=0 go build -o bin/resource-cleaner github.com/summerwind/eventreactor/cmd/resource-cleaner
RUN CGO_ENABLED=0 go build -o bin/event-init github.com/summerwind/eventreactor/cmd/event-init

#################################################

FROM scratch as eventreactor

COPY --from=builder /go/src/github.com/summerwind/eventreactor/bin/* /bin/

ENTRYPOINT ["/bin/manager"]

#################################################

FROM scratch as event-init

COPY --from=builder /go/src/github.com/summerwind/eventreactor/bin/event-init /bin/

ENTRYPOINT ["/bin/event-init"]
