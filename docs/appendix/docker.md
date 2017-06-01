## Running Core0 on a Docker Container

Core0 is the `systemd` replacement for Zero-OS.

The below steps will guide you through the process to running Core0 on a Docker container running Ubuntu.

Steps:

- [Starting a Docker container with Core0](#start-container)
- [Access Core0 from Python](#python-client)

<a id="start-container"></a>
## Starting a Docker container with Zero-OS

```
docker run --privileged -d --name core -p 6379:6379 g8os/g8os-dev:1.0
```

To follow the container logs do:

```bash
docker logs -f core
```

<a id="python-client"></a>
## Access Zero-OS from Python

Use `docker inspect core` to find out on which IP address Core0 can be reached.

See [Python Client](../interacting/python.md).
