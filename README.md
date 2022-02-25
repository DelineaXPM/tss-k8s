# Delinea Secret Server Kubernetes Secret Injector

![Docker](https://github.com/thycotic/tss-k8s/workflows/Docker/badge.svg)
![GitHub Package Registry](https://github.com/thycotic/tss-k8s/workflows/GitHub%20Package%20Registry/badge.svg)
![Red Hat Quay](https://github.com/thycotic/tss-k8s/workflows/Red%20Hat%20Quay/badge.svg)

A [Kubernetes](https://kubernetes.io/)
[Mutating Webhook](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#admission-webhooks)
that injects Secret data from Delinea Secret Server (TSS) into Kubernetes Secrets.
The webhook can be hosted as a pod or as a stand-alone service.

The webhook works by intercepting `CREATE` and `UPDATE` Secret admissions and mutating the Secret with data from tss.
The webhook configuration consists of one or more _role_ to Client Credential and Server mappings.
The webhook updates Kubernetes Secrets based on annotations on the Secret itself when it is created or updated.

The webhook uses the [Golang SDK](https://github.com/thycotic/tss-sdk-go) to communicate with the tss API.

It was tested with [Minikube](https://minikube.sigs.k8s.io/) and [Minishift](https://docs.okd.io/3.11/minishift/index.html).

## Configure

The webhook requires a JSON formatted list of _role_ to Client Credential and Tenant mappings.
The _role_ is a simple name that does not relate to Kubernetes Roles.
Declaring the role annotation selects which credentials to use to get the Secret from Secret Server.

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

That does not prescribe a runtime but, typically, the injector is run as a POD in the Kubernetes cluster that uses it.

## Build

Building the `tss-injector` image requires [Docker](https://www.docker.com/) or [Podman](https://podman.io/).
To build it, run:

```sh
make image
```

NOTE: The image has been published so building it is optional

### Minikube and Minishift

Remember to run `eval $(minikube docker-env)` in the shell to push the image to Minikube's Docker daemon.
Likewise for Minishift except its `eval $(minishift docker-env)`.

### Install

Installation requires [Helm](https://helm.sh).

The `Makefile` demonstrates a typical installation via the [Helm](https://helm.sh/) chart.
It imports `roles.json` as a file that it templates as a Kubernetes Secret for the injector.
It also provides a (Helm-generated) self-signed certificate to the injector as a Kubernetes Secret.

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

However, the `Makefile` contains an `install-image` target that configures Helm
to use the image built with `make image`:

```sh
make install-image
```

`make uninstall` uninstalls the Helm Chart.

`make clean` removes the Docker image.

### Helm and Make

The use of `make` is optional. Running `helm install` works too:

```sh
helm install --set-file rolesJson=configs/roles.json tss-injector charts/tss-injector
```

## Use

Once the `tss-injector` is available in the Kubernetes cluster, and the
[MutatingAdmissionWebhook](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#mutatingadmissionwebhook)
is in place, any appropriately annotated k8s Secrets are modified on create and update.

The four annotations that affect the behavior of the webhook are:

```golang
const(
    roleAnnotation   = "tss.thycotic.com/role"
    setAnnotation    = "tss.thycotic.com/set-secret"
    addNotation      = "tss.thycotic.com/add-to-secret"
    updateAnnotation = "tss.thycotic.com/update-secret"
)
```

`roleAnnotation` selects the credentials that the injector uses to retrieve the tss Secret.
If it is present then the role must exist in the role to Client Credential and Tenant mappings.
If it is absent then the _default_ mapping is used.

The `setAnnotation`, `addAnnotation` and `updateAnnotation` contain the path to
the tss Secret that the injector will use to mutate the submitted Kubernetes Secret.

* `addAnnotation` adds missing fields without overwriting or removing existing fields.
* `updateAnnotation` adds and overwrites existing fields but does not remove fields.
* `setAnnotation` overwrites fields and removes fields that do not exist in the tss Secret.

A Kubernetes Secret should specify only one of these, however, if the Secret specifies more
than one then, the order of precedence is `setAnnotation` then
`addAnnotation` then `updateAnnotation`.

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

The above example specifies a Role so a mapping for that role must exist in the
current webhook configuration. It uses the `setAnnotation` so the data in the
secret will be overwritten; if the Secret with ID 1 contains a `username` and
`password` but no `domain` then the secret would contain the `username` and
`password` from the TSS Secret Data and the `domain` field will be removed.

There are more examples in the `examples` directory. Each one will show
how each annotation works when run against an example with a username and
private-key but no domain in it.
