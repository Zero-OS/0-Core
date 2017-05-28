# Booting G8OS

* [Booting from USB](usb.md)
* [Booting from PXE Boot Server](pxe.md)
* [Booting G8OS on a VM using QEMU](qemu.md)
* [Booting G8OS on VirtualBox](virtualbox.md)
* [Booting G8OS on Packet.net](ays.md)

# Boot Options
g8os handles the following kernel params 
* `debug` sets the log level to debug, it also sets `sync` flag on the log file so if the system crashed for an
 unknown reason we make sure that the crash logs is committed to the permanent storage
* `quiet` only log to the log file and don't print logs on the console.