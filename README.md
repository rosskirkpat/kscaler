# kscaler

kscaler is a simple tool for performing resource scale testing against your kubernetes cluster. 
Think of it as a gas pedal; push as much as you want.

## design

Why use kscaler?

I started with various forms of "simple" scale testing for k8s clusters; bash scripts, one-off manifests, kubectl copy-pasta, and terraform modules.

Ultimately, I wanted a single solution that works on *all* the platforms I develop on: Windows, Mac, and Linux. Enter revvy.

## default kubeconfig paths

Windows: `$HOME\.kube\config`  
Linux/Mac: `$HOME/.kube/config`

## usage

`kscaler resource count --kubeconfig /path/to/kube/config`

**Note:** kscaler will attempt read the local `KUBECONFIG` variable if the `--kubeconfig` parameter is not set.

Some resources have additional parameters, which are noted below:

pod: `image`

- set a custom container image. The default is busybox. 

secret: `size`

- set a custom secret size in kilobytes. The default is a random size in the range of 1Kb to 1500Kb for each secret revvy creates.

## examples

```console
# Create 500 namespaces
kscaler namespace 500

# Create 5000 secrets with a static size of 750Kb
kscaler secret 5000 --size 750 

# Create 5000 secrets with auto-generated size ranging from 1Kb to 1500Kb
# The default behavior for revvy when performing secret generation is to randomize the size of each secret
kscaler secret 5000

# Create 875 pods using the default busybox container image
kscaler pod 875

# Create 875 pods using a custom container image
kscaler pod 875 --image cptrosskirk/busybox-scaler
```

## garbage collection

- Does kscaler cleanup after itself?

Not unless you tell it to. kscaler was intended to be a "scale until it breaks" tool. With that design in mind, I opted to make cleanup a manual task as the easier solution would be to tear down and build a new kubernetes cluster.

- How can I tell revvy to cleanup the resources it creates?

`kscaler cleanup` will attempt to perform a deletion on every resource it created by looking for the `revvy-made-this: true` label on each resource in the specified k8s cluster. revvy will prompt for confirmation prior to starting the cleanup unless `--approve` is also passed.

## Should I run this in production?

No. 

If you really want to run kscaler in prod, I have an ocean-view condo in Arizona available too.
