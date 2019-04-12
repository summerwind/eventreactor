VERSION = 0.2.1

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
	controller-gen all

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
docker-build:
	docker build --target eventreactor -t summerwind/eventreactor:latest -t summerwind/eventreactor:$(VERSION) .
	docker build --target event-init -t summerwind/event-init:latest -t summerwind/event-init:$(VERSION) .

# Push the latest image
docker-push:
	docker push summerwind/eventreactor:latest
	docker push summerwind/event-init:latest

# Push the release image
docker-push-release:
	docker push summerwind/eventreactor:$(VERSION)
	docker push summerwind/event-init:$(VERSION)

# Build release assets
release:
	hack/release.sh $(VERSION)

# Cleanup
clean:
	rm -rf release/
	rm -rf bin/*
	rm -rf cover.out
