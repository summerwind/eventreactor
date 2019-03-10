# Event Reactor

***This is under development. Please note that backwards incompatible changes may occur frequently.***

Event Reactor is an event-driven container runner for Kubernetes. This extends Kubernetes and allows you to run any container on Kubernetes when receiving events. Event Reactor is similar to GitHub Actions but works on any Kubernetes cluster.

Event Reactor is developed to deal with the following use cases:

- Run unit test when pushed to Git repository.
- Build container image when a tag is created in the Git repository.
- Create a GitHub issue when received alert from monitoring system.
- Create a Pull Request to update Dockerfile when it's base image is updated.

## Documentation

Follow the links below to learn more about Event Reactor.

- [Design Overview](docs/design.md)
- [Getting Started](docs/getting-started.md)
