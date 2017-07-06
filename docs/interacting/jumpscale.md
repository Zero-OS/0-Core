# JumpScale Client

A quick and easy way to get JumpScale on your machine is creating a Docker container with JumpScale preinstalled, achieved by following the steps in https://github.com/Jumpscale/developer.

Once the JS9 Docker container is installed you can use the interactive JumpScale shell inside the container and try the examples documented in [Examples](examples/README.md).

Just remember that the Zero-OS client is under `j.clients.g8core`, you'll need to use it as follows:
```python
cl = j.clients.g8core.get("<IP-address-of-the-Zero-OS-node>", port=6379)
```
