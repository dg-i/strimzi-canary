# Developing Canary

## Build

This project is developed by using [Golang](https://golang.org/).
The repository provides a `Makefile` with different make targets for building the project.

### Building the binary and running the application

In order to build the application binary, use the `go_build` make target.

```shell
make go_build
```

The binary is always built for Linux, by default for the local architecture (use `GOARCH=amd64` or `GOARCH=arm64` to build for a different one).
It will be available in the `cmd/target/linux/<arch>` folder named as `strimzi-canary` that you can just run locally (on Linux) from the root project folder.

```shell
./cmd/target/linux/amd64/strimzi-canary
```

### Building a container image

To build the container image, use the `docker_build` make target (after building the binary for the same architecture) together with the `docker_push` one if you want to push it to a registry as well.

```shell
make go_build docker_build docker_push
```

The image is built for the same architecture as the binary, which defaults to the local one.
When building on a different architecture than the one of the Kubernetes nodes (for example on an Apple Silicon Mac for `amd64` nodes), set `GOARCH` for both targets:

```shell
GOARCH=amd64 make go_build docker_build
```

The CI (GitHub Actions) builds a multi-arch image for `linux/amd64` and `linux/arm64` instead, using `docker buildx`.

It is possible to customize the container image build by using the following environment variables:

* `DOCKER_CMD` : the Docker binary to use for all container commands (default is `docker`).
* `DOCKER_REGISTRY`: the Docker registry where the image will be pushed (default is `ghcr.io`).
* `DOCKER_ORG`: the Docker organization for tagging/pushing the image (default is `dg-i`).
* `DOCKER_TAG`: the Docker tag (default is `latest`).
* `DOCKER_REPO`: the Docker repository where the image will be pushed (default is `strimzi-canary`).