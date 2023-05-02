## ArtificialInc mods

For each version we need, make a branch called `artificialinc/cluster-autoscaler-version`. This should be the one we build/deploy from. We can make PRs to it

## Building

```bash
DOCKER_CONTEXT=default make build-in-docker
DOCKER_CONTEXT=default docker build . -f Dockerfile.amd64 -t ghcr.io/artificialinc/cluster-autoscaler:v1.23.1-artificial
```
