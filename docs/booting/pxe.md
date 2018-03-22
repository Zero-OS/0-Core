# Booting from PXE Boot Server

We assume you already have a working PXE Boot Environment (DHCP and TFTP server).

In order to boot the Zero-OS kernel via PXE, you can use the popular `PXELINUX` tools.

## Configuration

- On your root TFTP directory, create a new directory called `zos` and put the `vmlinuz.efi` on it
- Configure the bootfile as well:

  ```
  DEFAULT 1
  TIMEOUT 100
  PROMPT  1

  LABEL 1
    kernel zos/vmlinuz.efi
  ```

- Save that config under `pxelinux.cfg/default` to run all your devices under Zero-OS

## Per device PXE boot

If you want to boot only some devices, you can symlink a special config for that MAC Address. Let assume you've put the config above on a `pxelinux.cfg/zos`, on the `pxelinux.cfg` directory, run:

```
ln -s zos 01-2c-88-88-cd-2a-01
```
