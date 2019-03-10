# Getting Started

This guide shows you how to deploy an pipeline using Event Reactor.

## Before you begin

You need:

- A Kubernetes cluster.
- Permissions to create Namespace and Custom Resource Definition on Kubernetes.

## Install resources

Event Reactor uses Knative Build to run containers. So you need to install Kanative Build first.

```
$ kubectl apply -f https://github.com/knative/build/releases/download/v0.4.0/build.yaml
```

Next, install Event Reactor as follows. Please note that the Event Reactor will be installed in the *eventreactor* namespace.

```
$ kubectl apply -f https://github.com/summerwind/eventreactor/releases/latest/download/eventreactor.yaml
```

Finally install the addons to automate the management of custom resources. Addons can be installed on any namespace. In this document we assume that you installed in the *default* namespace.

```
$ kubectl apply -n ${NAMESPACE} -f https://github.com/summerwind/eventreactor/releases/latest/download/eventreactor-addons.yaml
```

## Install command-line tool

`reactorctl` is a dedicated command-line tool used to manage custom resources. `reactorctl` is installed as follows.

**For macOS**

```
$ curl -L -O https://github.com/summerwind/eventreactor/releases/latest/download/reactorctl-darwin-amd64.tar.gz
$ tar zxvf reactorctl-darwin-amd64.tar.gz
$ mv reactorctl /usr/local/bin/reactorctl
```

**For Linux**

```
$ curl -L -O https://github.com/summerwind/eventreactor/releases/latest/download/reactorctl-linux-amd64.tar.gz
$ tar zxvf reactorctl-linux-amd64.tar.gz
$ mv reactorctl /usr/local/bin/reactorctl
```

## Verify the installation

The installation process created components for Knative Build and Event Reactor.

```
$ kubectl get pods -n knative-build
NAME                                READY     STATUS    RESTARTS   AGE
build-controller-68dfb74954-ktzxr   1/1       Running   0          1m
build-webhook-866fd64885-jsv7z      1/1       Running   0          1m

$ kubectl get pods -n eventreactor
NAME                                       READY     STATUS    RESTARTS   AGE
eventreactor-controller-774495548f-j9lm6   1/1       Running   0          1m
```

Addons are running in the specified namespace.

```
$ kubectl get pods -n default
NAME                                READY     STATUS       RESTARTS   AGE
event-receiver-9ff46fc8f-2mp4h      1/1       Running      0          1m

$ kubectl get cronjobs -n default
NAME               SCHEDULE     SUSPEND   ACTIVE    LAST SCHEDULE   AGE
resource-cleaner   30 0 * * *   False     0         <none>          1m
```

`reactorctl` should display 'No resources found.' for now.

```
$ reactorctl actions list
No resources found.
```

You are ready to create your first pipeline!

## Create your first pipeline

Pipeline resource defines what to do when receiving an event.

The following pipeline definition executes a container that outputs `hello world!' message when it receives an event called *test.hello*.

```
$ vim hello.yaml
```

```
apiVersion: eventreactor.summerwind.github.io/v1alpha1
kind: Pipeline
metadata:
  name: hello
spec:
  trigger:
    event:
      type: test.hello
      sourcePattern: .+
  steps:
  - name: hello
    image: ubuntu:18.04
    command: ["echo"]
    args: ["hello world!"]
```

Apply this pipline to your cluster.

```
$ kubectl apply -f hello.yaml
pipeline.eventreactor.summerwind.github.io "hello" created
```

Your first pipeline is created. Let's send the event to start the pipeline.

## Publish event

Send an event of type *test.hello* to run your pipeline. To send the event, use the `reactorctl events publish` command. The following command sends the *test.hello* event.

```
$ reactorctl events publish -t test.hello -s /test/hello -d '{"message":"hello"}'
01d5hczh1cb71xp22pd6qkbptm
```

You can check the list of sent events with `reactorctl events list` command. You can confirm that the *test.hello* event has been sent.

```
$ reactorctl events list
NAME                         TYPE         DATE
01d5hczh1cb71xp22pd6qkbptm   test.hello   2019-03-09 23:16:17
```

Let's check the execution result of the pipeline.

## Inspect action

Action is a resource for managing execution results of pipeline.

Check the Action resource with `reactorctl events list` command. you can confirm that the action was created.

```
$ reactorctl actions list
NAME                         PIPELINE   EVENT                        STATUS      DATE
01d5hczh2etmb9txj04jmjyycg   hello      01d5hczh1cb71xp22pd6qkbptm   Succeeded   2019-03-09 23:16:17
```

Use the `reactorctl actions get` command to check the details of the action. You can confirm that "Hello World!" is output as a pipeline execution result.

```
$ reactorctl actions get 01d5hczh2etmb9txj04jmjyycg
Name:     01d5hczh2etmb9txj04jmjyycg
Status:   Succeeded
Date:     2019-03-09 23:16:17 +0900 JST
Event:    01d5hczh1cb71xp22pd6qkbptm
Pipeline: hello

[ hello ]
Started At: 2019-03-09 23:16:36 +0900 JST
Finished At: 2019-03-09 23:16:36 +0900 JST
Exit Code: 0
-------------------
hello world!
```

You can use the `reactorctl actions logs` command if you want to see only the output log of the pipeline.

```
$ reactorctl actions logs 01d5hczh2etmb9txj04jmjyycg
hello world!
```
