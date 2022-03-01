# Delinea Secret Server Kubernetes Secret Injector

![Docker](https://github.com/thycotic/tss-k8s/workflows/Docker/badge.svg)
![GitHub Package Registry](https://github.com/thycotic/tss-k8s/workflows/GitHub%20Package%20Registry/badge.svg)
![Red Hat Quay](https://github.com/thycotic/tss-k8s/workflows/Red%20Hat%20Quay/badge.svg)

A [Kubernetes](https://kubernetes.io/)
[Mutating Webhook](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#admission-webhooks)
that injects Secret data from Delinea Secret Server (TSS) into Kubernetes Secrets.
The webhook can be hosted as a pod or as a stand-alone service.

The webhook works by intercepting `CREATE` and `UPDATE` Secret admissions and mutating the Secret with data from Secret Server.
The webhook configuration is a set of _role_ to Client Credential and Server mappings.
It updates Kubernetes Secrets based on annotations on the Secret itself.

The webhook uses the [Golang SDK](https://github.com/thycotic/tss-sdk-go) to communicate with the Secret Server API.

It was tested with [Minikube](https://minikube.sigs.k8s.io/) and [Minishift](https://docs.okd.io/3.11/minishift/index.html).

## Configure

The webhook requires a JSON formatted list of _role_ to Client Credential and Tenant mappings.
The _role_ is a simple name that does not relate to Kubernetes Roles.
It simply selects which credentials to use to get the Secret from Secret Server.

```json
{
    "my-role": {
        "credentials": {
            "username": "appuser1",
            "password": "Password1!"
        },
        "tenant": "corpname"
    },
    "default": {
        "credentials": {
            "username": "appuser2",
            "password": "Password2!"
        },
        "ServerURL": "https://hostname/SecretServer"
    }
}
```

NOTE: the injector uses the _default_ role when it mutates a Kubernetes Secret that does not have a _roleAnnotation_.
[See below](#use).

## Run

The injector is a Golang executable that runs a built-in HTTPS server hosting the Kubernetes Mutating Webhook Webservice.

```bash
$ /usr/bin/tss-injector-svc -?
flag provided but not defined: -?
Usage of ./tss-injector-svc:
  -cert string
        the path of the certificate file in PEM format (default "injector.pem")
  -hostport string
        the host:port e.g. localhost:8080 (default ":18543")
  -key string
        the path of the certificate key file in PEM format (default "injector.key")
  -roles string
        the path of JSON formatted roles file (default "roles.json")
```

Thus the injector can run "anywhere," but, typically, the injector runs as a POD in the Kubernetes cluster that uses it.

## Build

_NOTE: Building the `tss-injector` image is not required to install it as it is available on multiple public registries._

Building the injector requires [Docker](https://www.docker.com/) or [Podman](https://podman.io/). To build it, run:

```sh
make image
```

### Minikube and Minishift

Remember to run `eval $(minikube docker-env)` in the shell to push the image to Minikube's Docker daemon.💡
Likewise for Minishift except its `eval $(minishift docker-env)`.

To publish the image to the Minikube (or Minishift) registry, enable it:

```sh
minikube addons enable registry
```

Then start Minikube's [tunnel](https://minikube.sigs.k8s.io/docs/commands/tunnel/) in a separate terminal to make the service available on the host.

```sh
minikube tunnel
```

_It will run continuously. Stopping it will render the registry inaccessible._

### Publish

> NOTE: Publishing is _not_ required unless the cluster cannot download the image from the internet.

To publish, set `$(REGISTRY)` to the target registry, e.g., registry.example.com/_myusername_:

```sh
make release REGISTRY=registry.example.com/me
```

The `Makefile` sets it using `kubectl`:

```sh
kubectl get -n kube-system service registry -o jsonpath="{.spec.clusterIP}{':'}{.spec.ports[0].port}"
```

Thus `make release` without setting `$(REGISTRY)` will assume that the cluster hosts a registry and will push the image there.

### Install

Installation requires [Helm](https://helm.sh).

The `Makefile` demonstrates a typical installation via the [Helm](https://helm.sh/) chart.
It imports `roles.json` as a file that it templates as a Kubernetes Secret for the injector.

The Helm `values.yaml` file `image.repository` is `thycotic/tss-injector`:

```yaml
image:
  repository: thycotic/tss-injector
  pullPolicy: IfNotPresent
  # Overrides the image tag whose default is the chart appVersion.
  tag: ""
```

That means, by default, `make install` will pull from Docker, GitHub, or Quay.

```sh
make install
```

However, the `Makefile` contains an `install-image` target that configures Helm to use the image built with `make image`:

```sh
make install-image
```

`make uninstall` uninstalls the Helm Chart.

`make clean` removes the Docker image.

### Helm and Make

__The use of `make` is optional.__ Running `helm install` works too:

```sh
helm install --set-file rolesJson=configs/roles.json tss-injector charts/tss-injector
```

## Use

Once the injector is available in the Kubernetes cluster, and the
[MutatingAdmissionWebhook](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#mutatingadmissionwebhook)
is in place, any appropriately annotated Kubernetes Secrets are modified on create and update.

The four annotations that affect the behavior of the webhook are:

```golang
const(
    roleAnnotation   = "tss.thycotic.com/role"
    setAnnotation    = "tss.thycotic.com/set-secret"
    addNotation      = "tss.thycotic.com/add-to-secret"
    updateAnnotation = "tss.thycotic.com/update-secret"
)
```

`roleAnnotation` selects the credentials that the injector uses to retrieve the Secret Server Secret.
If the role is present, it must map to Client Credential and Tenant mapping.
If the role is absent, the injector will use the _default_ Credential and Tenant a mapping.

The `setAnnotation`, `addAnnotation` and `updateAnnotation` specify the numeric ID of
the Secret Server Secret that the injector will use to mutate the Kubernetes Secret.

* `addAnnotation` adds missing fields without overwriting or removing existing fields.
* `updateAnnotation` adds and overwrites existing fields but does not remove fields.
* `setAnnotation` overwrites fields and removes fields that do not exist in the Secret Server Secret.

NOTE: A Kubernetes Secret should specify only one of the "add," "update," or "set" annotations.
The order of precedence is `setAnnotation`, then `addAnnotation`, then `updateAnnotation` when multiple are present.

### Examples

```yaml
---
apiVersion: v1
kind: Secret
metadata:
  name: example-secret
  annotations:
    tss.thycotic.com/role: my-role
    tss.thycotic.com/set-secret: "1"
type: Opaque
data:
  username: dW5tb2RpZmllZC11c2VybmFtZQ==
  private-key: b21pdHRlZAo=
```

The above example specifies a Role, so a mapping for that role must exist in the
current webhook configuration. It uses the `setAnnotation` so the data in the
injector will overwrite the existing contents of the Kubernetes Secret;
if the Secret with ID 1 contains a `username` and `password` but no `domain`, then the Kubernetes Secret would get the `username` and
`password` from the Secret Server Secret but, the injector will remove the `domain` field.

There are more examples in the `examples` directory. Each one will show
how each annotation works when run against an example with a username and
private-key but no domain in it.
