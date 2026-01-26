# kscaler

kscaler is a simple tool for performing resource scale testing against your kubernetes cluster. 
Think of it as a gas pedal; push as much as you want.

## design

Why use kscaler?

I started with various forms of "simple" scale testing for k8s clusters; bash scripts, one-off manifests, kubectl copy-pasta, and terraform modules.

Ultimately, I wanted a single solution that works on *all* of the platforms I develop on: Windows, Mac, and Linux. Enter kscaler.

## default kubeconfig paths

Windows: `$HOME\.kube\config`  
Linux/Mac: `$HOME/.kube/config`

## building

### Using make targets

```shell
make build 
```

### Using mage

NB. mage will perform multi-platform and multi-arch builds by default.
- darwin/arm64
- darwin/amd64
- linux/arm64
- linux/amd64
- windows/arm64
- windows/amd64

```shell
go install github.com/magefile/mage@latest
mage

# 
```

## usage

`kscaler resource <count> --kubeconfig /path/to/kube/config`

**Note:** kscaler will attempt read the local `KUBECONFIG` variable if the `--kubeconfig` parameter is not set.

Some resources have additional parameters, which are noted below:

pod: `image`

- set a custom container image. The default is busybox. 

secret: `size`

- set a custom secret size in kilobytes. The default is a random size in the range of 1Kb to 1500Kb for each secret kscaler creates.

## examples

I test using a local (m2 pro, 32gb), single node kind v1.32.2 cluster with client config: qps 250, burst 500 using colima + podman desktop with a 4cpu/8.29gb podman machine.

```console
# Create 1000 namespaces
% kscaler namespace 1000

time="2026-01-25T20:32:26-05:00" level=info msg="kscaler action took 2.036540083s"
time="2026-01-25T20:32:26-05:00" level=info msg="1000/1000 jobs were successful with 0 retries required"
time="2026-01-25T20:32:26-05:00" level=info msg="finished creating 1000 resources of kind (*v1.Namespace)"

# Create 5000 namespaces, without cleanup
% kscaler namespace 5000

time="2026-01-25T20:35:14-05:00" level=info msg="kscaler action took 18.384376875s"
time="2026-01-25T20:35:14-05:00" level=info msg="5000/5000 jobs were successful with 0 retries required"
time="2026-01-25T20:35:14-05:00" level=info msg="finished creating 5000 resources of kind (*v1.Namespace)"

# Create another 5000 namespaces (total of 10k)
% kscaler namespace 5000

time="2026-01-25T20:35:47-05:00" level=info msg="kscaler action took 18.024904792s"
time="2026-01-25T20:35:47-05:00" level=info msg="5000/5000 jobs were successful with 30 retries required"
time="2026-01-25T20:35:47-05:00" level=info msg="finished creating 5000 resources of kind (*v1.Namespace)"

% kubectl get ns --no-headers | wc -l
   10006
```


```console
# Create 5000 secrets with a static size of 750Kb
kscaler secret -n 5000 -s 750

# first run, with cleanup
% kscaler secret -n 5000 -s 750
time="2026-01-25T20:26:04-05:00" level=info msg="kscaler action took 18.239497917s"
time="2026-01-25T20:26:04-05:00" level=info msg="5000/5000 jobs were successful with 32 retries required"
time="2026-01-25T20:26:04-05:00" level=info msg="finished creating 5000 resources of kind (*v1.Secret)"

# second run, without cleanup
% kscaler secret -n 5000 -s 750
time="2026-01-25T20:30:05-05:00" level=info msg="kscaler action took 18.455027875s"
time="2026-01-25T20:30:05-05:00" level=info msg="5000/5000 jobs were successful with 80 retries required"

# third run for total of 10k secrets
% kscaler secret -n 5000 -s 750
time="2026-01-25T20:31:05-05:00" level=info msg="kscaler action took 17.902367583s"
time="2026-01-25T20:31:05-05:00" level=info msg="5000/5000 jobs were successful with 0 retries required"

% kubectl get secrets -A --no-headers | wc -l
   10003
```

```console
# Create 5000 secrets with auto-generated size ranging from 1Kb to 1500Kb
# The default behavior for kscaler when performing secret generation is to randomize the size of each secret
kscaler secret -n 5000

# Create 875 pods using the default busybox container image
% kscaler pod -n 1000

time="2026-01-25T20:39:31-05:00" level=info msg="kscaler action took 2.59118025s"
time="2026-01-25T20:39:31-05:00" level=info msg="999/1000 jobs were successful with 37 retries required"
time="2026-01-25T20:39:31-05:00" level=info msg="finished creating 999 resources of kind (*v1.Pod)"

% kubectl get pods -A --no-headers | wc -l
    1013

# this broke my kind node with 111 pods schedule onto it, kube-scheduler died
# running a kscaler cleanup revived it :) 

# Create 875 pods using a custom container image
kscaler pod -n 875 --image cptrosskirk/busybox-scaler
```

## garbage collection

- Does kscaler cleanup after itself?

Not unless you tell it to. kscaler was intended to be a "scale until it breaks" tool. With that design in mind, I opted to make cleanup a manual task as the easier solution would be to tear down and build a new kubernetes cluster.

- How can I tell kscaler to cleanup the resources it creates?

`kscaler cleanup` will attempt to perform a deletion on every resource it created by looking for the `created-by: kscaler` label on each resource in the specified k8s cluster. 
kscaler will prompt for confirmation prior to starting the cleanup unless `--force` is also passed.

## Should I run this in production?

No. 

If you really want to run kscaler in prod, I have an ocean-view condo in Arizona available too.
