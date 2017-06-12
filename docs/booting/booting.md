# Booting Zero-OS

* [Booting from USB](usb.md)
* [Booting from PXE Boot Server](pxe.md)
* [Booting Zero-OS on a VM using QEMU](qemu.md)
* [Booting Zero-OS on VirtualBox](virtualbox.md)
* [Booting Zero-OS on Packet.net](packet.md)
* [Create a Bootable Zero-OS ISO File](iso.md)

## Boot Options

Zero-OS handles the following kernel params:
* `debug` sets the log level to debug, it also sets `sync` flag on the log file so if the system crashed for an unknown reason we make sure that the crash logs is committed to the permanent storage.
When debug is not set firewall rules will be applied so that the redis port and ssh are only available via the zerotier network.
* `organization=<org>` When set, zero-os will only accept itsyouonline signed JWT tokens that has the user:memberOf:<org> role set and valid.
If not provided, zero-os will not require a password.
* `quiet` only log to the log file and don't print logs on the console
