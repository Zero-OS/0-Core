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
* `debug` sets the log level to debug, it also sets the `sync` flag on the log file so if the system crashes for an unknown reason we make sure that the crash logs are committed to permanent storage
When debug is not set firewall rules will be applied so that the Redis port and ssh are only available via the ZeroTier network
* `organization=<org>` When set, Zero-OS will only accept ItsYou.online signed JWT tokens that have the `user:memberof:<org>` role set and are valid
If not provided, zero-os will not require a password
* `zerotier=<id>` Join this zerotier network on boot, this flag will also force zos to only accept client connections from this network.
