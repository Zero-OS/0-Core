# Zero-OS Bootstrap Service

The Zero-OS Bootstrap Service is available on [bootstrap.gig.tech](https://bootstrap.gig.tech).

From the Zero-OS Bootstrap Service you can easily download a build of the kernel, or ISO, USB, and iPXE boot files.

For instance to download an ISO image:
```shell
BRANCH="zero-os-master"
ZEROTIER_NETWORK="zerotier-network-ID"
curl -o zero-os.iso https://bootstrap.gig.tech/iso/$BRANCH/$ZEROTIER_NETWORK
```

See the Zero-OS Bootstrap Service documentation for more details: https://github.com/zero-os/0-bootstrap/tree/master/docs
