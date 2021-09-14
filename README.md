# Brigade ACR Gateway

[![codecov](https://codecov.io/gh/brigadecore/brigade-acr-gateway/branch/main/graph/badge.svg?token=9J6ZWN52PI)](https://codecov.io/gh/brigadecore/brigade-acr-gateway)
[![Go Report Card](https://goreportcard.com/badge/github.com/brigadecore/brigade-acr-gateway)](https://goreportcard.com/report/github.com/brigadecore/brigade-acr-gateway)

This is a work-in-progress
[Brigade 2](https://github.com/brigadecore/brigade/tree/v2)
compatible gateway that receives events (webhooks) from Azure Container Registry
and propagates them into Brigade 2's event bus.

## Installation

Prerequisites:

* A Kubernetes cluster:
    * For which you have the `admin` cluster role
    * That is already running Brigade 2
    * Capable of provisioning a _public IP address_ for a service of type
      `LoadBalancer`. (This means you won't have much luck running the gateway
      locally in the likes of kind or minikube unless you're able and willing to
      mess with port forwarding settings on your router, which we won't be
      covering here.)

* `kubectl`, `helm` (commands below assume Helm 3), and `brig` (the Brigade 2
  CLI)

### 1. Create a Service Account for the Gateway

__Note:__ To proceed beyond this point, you'll need to be logged into Brigade 2
as the "root" user (not recommended) or (preferably) as a user with the `ADMIN`
role. Further discussion of this is beyond the scope of this documentation.
Please refer to Brigade's own documentation.

Using Brigade 2's `brig` CLI, create a service account for the gateway to use:

```console
$ brig service-account create \
    --id brigade-acr-gateway \
    --description brigade-acr-gateway
```

Make note of the __token__ returned. This value will be used in another step.
_It is your only opportunity to access this value, as Brigade does not save it._

Authorize this service account to create new events:

```console
$ brig role grant EVENT_CREATOR \
    --service-account brigade-acr-gateway \
    --source brigade.sh/acr
```

__Note:__ The `--source brigade.sh/acr` option specifies that
this service account can be used _only_ to create events having a value of
`brigade.sh/acr` in the event's `source` field. _This is a
security measure that prevents the gateway from using this token for
impersonating other gateways._

### 2. Install the ACR Gateway

For now, we're using the [GitHub Container Registry](https://ghcr.io) (which is
an [OCI registry](https://helm.sh/docs/topics/registries/)) to host our Helm
chart. Helm 3.7 has _experimental_ support for OCI registries. In the event that
the Helm 3.7 dependency proves troublesome for users, or in the event that this
experimental feature goes away, or isn't working like we'd hope, we will revisit
this choice before going GA.

First, be sure you are using
[Helm 3.7.0-rc.1](https://github.com/helm/helm/releases/tag/v3.7.0-rc.1) and
enable experimental OCI support:

```console
$ export HELM_EXPERIMENTAL_OCI=1
```

As this chart requires custom configuration as described above to function
properly, we'll need to create a chart values file with said config.

Use the following command to extract the full set of configuration options into
a file you can modify:

```console
$ helm inspect values oci://ghcr.io/brigadecore/brigade-acr-gateway \
    --version v0.2.1 > ~/brigade-acr-gateway-values.yaml
```

Edit `~/brigade-acr-gateway-values.yaml`, making the following changes:

* `host`: Set this to the host name where you'd like the gateway to be
  accessible.

* `brigade.apiAddress`: Address of the Brigade API server, beginning with
  `https://`

* `brigade.apiToken`: Service account token from step 2

* `tokens`: This field should define tokens that can be used by clients to send
  events (webhooks) to this gateway. Note that keys are completely ignored by
  the gateway and only the values (tokens) matter. The keys only serve as
  recognizable token identifiers for human operators.

Save your changes to `~/brigade-acr-gateway-values.yaml` and use the following command to install
the gateway using the above customizations:

```console
$ helm install brigade-acr-gateway
    oci://ghcr.io/brigadecore/brigade-acr-gateway \
    --version v0.2.1 \
    --create-namespace \
    --namespace brigade-acr-gateway \
    --values ~/brigade-acr-gateway-values.yaml
```

### 3. (RECOMMENDED) Create a DNS Entry

If you installed the gateway without enabling support for an ingress controller,
this command should help you find the gateway's public IP address:

```console
$ kubectl get svc brigade-acr-gateway \
    --namespace brigade-acr-gateway \
    --output jsonpath='{.status.loadBalancer.ingress[0].ip}'
```

If you overrode defaults and enabled support for an ingress controller, you
probably know what you're doing well enough to track down the correct IP without
our help. 😉

With this public IP in hand, edit your name servers and add an `A` record
pointing your domain to the public IP.

### 4. Create Webhooks

In your browser, visit the Azure Portal and navigate to the Azure Container
Registry for which you'd like to send webhooks to this gateway. From the
__Services__ section, select __Webhooks__ and click __Add__.

Here, you can add webhooks for the entire registry or for specific repositories
within the registry.

* In the __Webhook name__ field, add a meaningful name for the webhook.

* If your registry is replicated across regions, select the applicable
  __Location__.

* In the __Service URI__ field, use a value of the form 
  `https://<DNS hostname or publicIP>/events`.

* In the __Custom headers__ field, add `Authorization: Bearer <token>`, where
  `<token>` is any of the tokens that were specified in `my-values.yaml` at the
  time of gateway installation. This will enable authentication to this gateway.

* In the __Actions__ field, select the actions that should trigger a webhook.
  Note that `chart_push` and `chart_delete` are not supported by this gateway.

* The __Status__ field can be used to enable or disable the webhook. It is
  enabled by default.

* The __Scope__ field can be used to make this webhook be triggered only by the
  selected action (__Actions__ field) on a _specific repository within the
  registry_. If __Scope__ is left blank, selected actions on _any repository
  within the registry_ will trigger the webhook.

* Click __Create__

### 5. Add a Brigade Project

You can create any number of Brigade projects (or modify an existing one) to
listen for events that were sent from an ACR repository to your gateway and, in
turn, emitted into Brigade's event bus. You can subscribe to all event types
emitted by the gateway, or just specific ones.

In the example project definition below, we subscribe to all events emitted by
the gateway, provided they've originated from the fictitious
`example.azurecr.io` ACR registry and the fictitious `example-repo` repository
(see the `registry` qualifier and `repo` label).

```yaml
apiVersion: brigade.sh/v2-beta
kind: Project
metadata:
  id: acr-demo
description: A project that demonstrates integration with ACR
spec:
  eventSubscriptions:
  - source: brigade.sh/acr
    types:
    - *
    qualifiers:
      registry: example.azurecr.io
    labels:
      repo: example-repo
  workerTemplate:
    defaultConfigFiles:
      brigade.js: |-
        const { events } = require("@brigadecore/brigadier");

        events.on("brigade.sh/acr", "push", () => {
          console.log("Someone pushed an image to example.azurecr.io/example-repo!");
        });

        events.process();
```

In the alternative example below, we subscribe _only_ to `push` events:

```yaml
apiVersion: brigade.sh/v2-beta
kind: Project
metadata:
  id: acr-demo
description: A project that demonstrates integration with ACR
spec:
  eventSubscriptions:
  - source: brigade.sh/acr
    types:
    - push
    qualifiers:
      registry: example.azurecr.io
    labels:
      repo: example-repo
  workerTemplate:
    defaultConfigFiles:
      brigade.js: |-
        const { events } = require("@brigadecore/brigadier");

        events.on("brigade.sh/acr", "push", () => {
          console.log("Someone pushed an image to example.azurecr.io/example-repo!");
        });

        events.process();
```

Assuming this file were named `project.yaml`, you can create the project like
so:

```console
$ brig project create --file project.yaml
```

Push an image to the ACR repo for which you configured webhooks to send an event
(webhook) to your gateway. The gateway, in turn, will emit the event into
Brigade's event bus. Brigade should initialize a worker (containerized event
handler) for every project that has subscribed to the event, and the worker
should execute the `brigade.js` script that was embedded in the example project
definition.

List the events for the `acr-demo` project to confirm this:

```console
$ brig event list --project acr-demo
```

Full coverage of `brig` commands is beyond the scope of this documentation, but
at this point, additional `brig` commands can be applied to monitor the event's
status and view logs produced in the course of handling the event.

## Events Received and Emitted by this Gateway

_A subset of events_ received by this gateway from ACR are, in turn, emitted
into Brigade's event bus.

ACR supports the following events:

* `push`
* `delete`
* `chart_push`
* `chart_delete`

According to ACR documentation, `push` and `delete` are triggered when OCI
images are, respectively, pushed to or deleted from a repository in ACR. Again,
according to ACR documentation, `chart_push` and `chart_delete` are triggered
when Helm charts are, respectively, pushed to or delete from a repository in
ACR. _However_ `chart_push` and `chart_delete` are only triggered when using the
Azure CLI's `az acr` commands to push and delete charts from ACR and these
commands are _deprecated_ in favor of Helm 3's built in support. When pushing or
deleting via `helm` commands, regular `push` and `delete` webhooks are
triggered.

Moreover, `chart_push` and `chart_delete` webhook payloads lack critical
information (namely the affected registry) that is present in the `push` and
`delete` webhook payloads. This makes it impossible for this gateway to
effectively qualify and label events emitted into Brigade in a manner that is
compatible with Brigade's event subscription model.

__Due to these constraints, only `push` and `delete` webhooks are supported.__

All `push` and `delete` events emitted into Brigade's event bus will be
qualified by the registry of origin (`registry` qualifier) and labeled with the
repository of origin (`repo` label). Subscribers _must_ match on the `registry`
qualifier to receive events originating from a given registry and _may
optionally_ match on the `repo` label to narrow their subscription to only
originating from a specific repository. Refer back to examples in previous
sections to see this in action.

## Examples Projects

See `examples/` for complete Brigade projects that demonstrate various
scenarios.

## Contributing

The Brigade project accepts contributions via GitHub pull requests. The
[Contributing](CONTRIBUTING.md) document outlines the process to help get your
contribution accepted.

## Support & Feedback

We have a slack channel!
[Kubernetes/#brigade](https://kubernetes.slack.com/messages/C87MF1RFD) Feel free
to join for any support questions or feedback, we are happy to help. To report
an issue or to request a feature open an issue
[here](https://github.com/brigadecore/brigade-acr-gateway/issues)
