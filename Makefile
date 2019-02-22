VERSION = 0.1.0

all: test binary

# Run tests
test: generate fmt vet manifests
	go test ./pkg/... ./cmd/... -coverprofile cover.out

# Build binary
binary: generate fmt vet
	CGO_ENABLED=0 go build -o bin/manager github.com/summerwind/eventreactor/cmd/manager
	CGO_ENABLED=0 go build -o bin/event-receiver github.com/summerwind/eventreactor/cmd/event-receiver
	CGO_ENABLED=0 go build -o bin/resource-cleaner github.com/summerwind/eventreactor/cmd/resource-cleaner
	CGO_ENABLED=0 go build -o bin/event-init github.com/summerwind/eventreactor/cmd/event-init
	CGO_ENABLED=0 go build -o bin/reactorctl github.com/summerwind/eventreactor/cmd/reactorctl

# Run manager against the configured Kubernetes cluster in ~/.kube/config
manager: generate fmt vet
	go run ./cmd/manager/main.go

# Run apiserver against the configured Kubernetes cluster in ~/.kube/config
event-receiver: generate fmt vet
	go run ./cmd/event-receiver/main.go

# Install CRDs into a cluster
install: manifests
	kubectl apply -f config/crds

# Deploy manifests in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	kustomize build config/default | kubectl apply -f -
	kustomize build config/apps | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests:
	go run vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go crd
	go run vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go rbac --name eventreactor-controller

# Run go fmt against code
fmt:
	go fmt ./pkg/... ./cmd/...

# Run go vet against code
vet:
	go vet ./pkg/... ./cmd/...

# Generate code
generate:
	go generate ./pkg/... ./cmd/...

# Build the docker image
docker-build: test
	docker build -t summerwind/eventreactor:latest -t summerwind/eventreactor:$(VERSION) .

# Push the docker image
docker-push:
	docker push summerwind/eventreactor:latest

# Build release assets
release:
	hack/release.sh

# Cleanup
clean:
	rm -rf release/
	rm -rf bin/*
	rm -rf cover.out
