VERSION = 0.1.0

NAME_CONTROLLER := summerwind/eventreactor-controller
NAME_APISERVER  := summerwind/eventreactor-apiserver
NAME_EVENT_INIT := summerwind/event-init

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
	docker build -t $(NAME_CONTROLLER):latest -t $(NAME_CONTROLLER):$(VERSION) -f container/manager/Dockerfile .
	docker build -t $(NAME_APISERVER):latest  -t $(NAME_APISERVER):$(VERSION)  -f container/apiserver/Dockerfile .
	docker build -t $(NAME_EVENT_INIT):latest -t $(NAME_EVENT_INIT):$(VERSION) -f container/event-init/Dockerfile .

# Push the docker image
docker-push:
	docker push $(NAME_CONTROLLER):latest
	docker push $(NAME_CONTROLLER):$(VERSION)
	docker push $(NAME_APISERVER):latest
	docker push $(NAME_APISERVER):$(VERSION)
	docker push $(NAME_EVENT_INIT):latest
	docker push $(NAME_EVENT_INIT):$(VERSION)

