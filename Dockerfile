FROM golang:1.12 AS builder

ENV KUBEBUILDER_VERSION=1.0.8 \
    KUSTOMIZE_VERSION=2.0.1

ENV GO111MODULE=on \
    GOPROXY=https://proxy.golang.org

RUN curl -L -O https://github.com/kubernetes-sigs/kubebuilder/releases/download/v${KUBEBUILDER_VERSION}/kubebuilder_${KUBEBUILDER_VERSION}_linux_amd64.tar.gz \
  && tar zxvf kubebuilder_${KUBEBUILDER_VERSION}_linux_amd64.tar.gz \
  && mv kubebuilder_${KUBEBUILDER_VERSION}_linux_amd64 /usr/local/kubebuilder

RUN curl -L -O https://github.com/kubernetes-sigs/kustomize/releases/download/v${KUSTOMIZE_VERSION}/kustomize_${KUSTOMIZE_VERSION}_linux_amd64 \
  && install -o root -g root -m 755 kustomize_${KUSTOMIZE_VERSION}_linux_amd64 /usr/local/bin/kustomize

WORKDIR /go/src/github.com/summerwind/eventreactor
COPY go.mod go.sum .
RUN go mod download

RUN go get sigs.k8s.io/controller-tools/cmd/controller-gen \
  && install -o root -g root -m 755 ${GOPATH}/bin/controller-gen /usr/local/bin/controller-gen

RUN go get k8s.io/code-generator/cmd/deepcopy-gen \
  && install -o root -g root -m 755 ${GOPATH}/bin/deepcopy-gen /usr/local/bin/deepcopy-gen

COPY . /workspace
WORKDIR /workspace

RUN make binary

#################################################

FROM scratch as eventreactor

COPY --from=builder /workspace/bin/* /bin/

ENTRYPOINT ["/bin/manager"]

#################################################

FROM scratch as event-init

COPY --from=builder /workspace/bin/event-init /bin/

ENTRYPOINT ["/bin/event-init"]
