# JumpScale Client

A quick and easy way to get JumpScale on your machine is creating a Docker container with JumpScale preinstalled, achieved by following the steps in https://github.com/Jumpscale/developer.

Once the JS9 Docker container is installed you can use the interactive JumpScale shell inside the container and try the examples documented in [Examples](examples/README.md).

The Zero-OS client can be found under `j.clients.zero_os`:
```python
# omit data if you want to use the js config manager
data = {
  "host": "ip-of-0-os-node",
  "port": 6379,
}

cl = j.clients.zero_os.get(data=data)

cl.ping()
```
