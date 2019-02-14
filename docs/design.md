# Design Overview

Event Reactor is an event-driven container runnter. This runs any containers as a reaction of the event. The user can specify which container to run for which event. Event Reactor is similar to [GitHub Actions](https://github.com/features/actions) but works on any Kubernetes cluster.

## Concept

Many tasks related to software development and operation are based on some event. For example, many development teams are running the tests when their code committed to the repository. In such a case, the code commit is an event.

We think that if we can automatically run any containers when events occur, we can automate various tasks. Event Reactor is developing based on this idea.

## Architecture

Event Reactor consists of several components and custom resources of Kubernetes.

![Architecture](images/architecture.png)

### Custom Resources

Event Reactor uses the following Kubernetes custom resource.

- **Event** represents an event. It has metadata such as the type and the source, and the body of the event.
- **Pipeline** defines the processing to be run when an Event is created. It is an extension of Build resource of [Knative Build](https://github.com/knative/build/).
- **Action** represents the result of processing when receiving an event. It is created based on the contents of the Pipeline resource.

### Components

- **Controller** manages the state of custom resources. When a Event resource is created, controller reads Pipeline resources and creates Actions based on the content of Pipeline. It runs on their own namespace and manages custom resources in the all namespaces.
- **Event Receiver** receives [CloudEvents](https://cloudevents.io/) formatted event from external event sources and creates Event resource. It runs on any namespace and manages Event resources in the same namespace.
- **Resource Cleaner** deletes expired Event and Action resource. It will prevent Kubernetes resources from becoming bloated. Resource Cleaner runs as a regular job on any namespace and removes the resources in the same namespace.
- **reactorctl** is a command-line tools for managing Event Reactor's resouces. Users can use this to check the details of Event and the execution result of Action.
