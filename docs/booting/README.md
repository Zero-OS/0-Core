# Booting Zero-OS

* [Booting from USB](usb.md)
* [Booting from PXE Boot Server](pxe.md)
* [Booting Zero-OS on a VM using QEMU](qemu.md)
* [Booting Zero-OS on VirtualBox](virtualbox.md)
* [Booting Zero-OS on Packet.net](packet.md)
* [Booting Zero-OS on OVH](ovh.md)
* [Create a Bootable Zero-OS ISO File](iso.md)

## Boot Options

Zero-OS handles the following kernel params:
* `debug` sets the initial log level to debug, can be changed later via the client.
* `organization=<org>` When set, Zero-OS will only accept ItsYou.online signed JWT tokens that have the `user:memberof:<org>` role set and are valid
If not provided, zero-os will not require a password
* `zerotier=<id>` Join this zerotier network on boot
* `development` If set, start the redis-proxy allow direct client connections, also opening the required client ports. If not set, no direct client connections
will be allowed
