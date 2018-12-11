all: test binary

# Run tests
test: generate fmt vet manifests
	go test ./pkg/... ./cmd/... -coverprofile cover.out

# Build binary
binary: generate fmt vet
	go build -o bin/manager github.com/summerwind/eventreactor/cmd/manager
	go build -o bin/apiserver github.com/summerwind/eventreactor/cmd/apiserver
	go build -o bin/event-init github.com/summerwind/eventreactor/cmd/event-init

# Run manager against the configured Kubernetes cluster in ~/.kube/config
manager: generate fmt vet
	go run ./cmd/manager/main.go

# Run apiserver against the configured Kubernetes cluster in ~/.kube/config
apiserver: generate fmt vet
	go run ./cmd/apiserver/main.go

# Install CRDs into a cluster
install: manifests
	kubectl apply -f config/crds

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	kubectl apply -f config/crds
	kustomize build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests:
	go run vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go all

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
	docker build -t summerwind/eventreactor-controller:latest -f container/manager/Dockerfile .
	docker build -t summerwind/eventreactor-apiserver:latest -f container/apiserver/Dockerfile .
	docker build -t summerwind/event-init:latest -f container/event-init/Dockerfile .

# Push the docker image
docker-push:
	docker build -t summerwind/eventreactor-controller:latest
	docker build -t summerwind/eventreactor-apiserver:latest
	docker build -t summerwind/event-init:latest
