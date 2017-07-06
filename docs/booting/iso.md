# Create a Bootable Zero-OS ISO File

You can get bootable Zero-OS ISO file from the Zero-OS Bootstrap Service, as documented in [Zero-OS Bootstrap Service](../bootstrap/README.md).

Alternatively you can of course create one yourself, after having build your own Zero-OS kernel based on the documentation in [Building your own Zero-OS Boot Image](../building/README.md).

This is how:
```shell
dd if=/dev/zero of=zero-os.iso bs=1M count=90
mkfs.vfat zero-os.iso
mount zero-os.iso /mnt
mkdir -p /mnt/EFI/BOOT
cp staging/vmlinuz.efi /mnt/EFI/BOOT/BOOTX64.EFI
umount /mnt
```
