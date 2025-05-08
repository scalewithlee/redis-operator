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
