# redis-operator
A Kubernetes Operator for Redis

## Development Notes
- This operator was developed using the `operator-sdk` ([installation guide](https://sdk.operatorframework.io/docs/installation/))
- I had issues getting `bin/kustomize` to work on my mac, so I just ran `brew install kustomize` and set `KUSTOMIZE ?= $(shell which kustomize)` in the `Makefile`.

## Build and Push the Image
I'm working on a mac (arm64), and my Kubernetes cluster uses amd64, so I run this command to build and push both archs go my GCP repository in `us-central1`:
```
GCP_PROJECT=foo
REPOSITORY_ID=bar
docker buildx build --platform linux/amd64,linux/arm64 \
  -t us-central1-docker.pkg.dev/$GCP_PROJECT/$REPOSITORY_ID/redis-operator:v0.1.0 \
  --push .
```

# Install the CRDs into the cluster (apply resources)
```
make install
```

# Deploy the manager deployment:
```
GCP_PROJECT=foo
REPOSITORY_ID=bar
REGION=baz
make deploy IMG=$REGION-docker.pkg.dev/$GCP_PROJECT/$REPOSITORY_ID/redis-operator:v0.1.0
```

# Undeploy current deployment
```
make undeploy
```

# Iterating
After making changes to the controller,
```
# 1. Rebuild and push the image:
PLATFORMS=linux/amd64,linux/arm64 make docker-buildx IMG=us-central1-docker.pkg.dev/scalewithlee/dev-ml-containers/redis-operator:v0.1.1
```

```
# 2. Update the deployment to use the new image:
make deploy IMG=us-central1-docker.pkg.dev/scalewithlee/dev-ml-containers/redis-operator:v0.1.1
```

**Note:** be sure to update the image tags to release new versions


## Initializing the Redis cluster
Exec into the first pod of the stateful set and run:
```
# Replace the IPs with the IPs of redis-sample-0, 1, and 2. (assuming size = 3)
redis-cli --cluster create 10.16.1.15:6379 10.16.0.3:6379 10.16.1.18:6379 --cluster-replicas 0
```

## Adding replicas to the masters
```
# For example...
# Add replica 3 to master 0 (redis-sample-0)
redis-cli --cluster add-node <pod-3-ip>:6379 10.16.1.15:6379 --cluster-slave --cluster-master-id fb32a3f3b980c56c636f86301fb63ffbcbcf9964

# Add replica 4 to master 1 (redis-sample-1)
redis-cli --cluster add-node <pod-4-ip>:6379 10.16.0.3:6379 --cluster-slave --cluster-master-id 246d2bda39bf12f9d4ee10686a7b4470752d8127

# Add replica 5 to master 2 (redis-sample-2)
redis-cli --cluster add-node <pod-5-ip>:6379 10.16.1.18:6379 --cluster-slave --cluster-master-id 4506ce928ea34ec11d69e0826723a63b5398e7e0
```