# Auto discovery

In a G8OS Grid Architecture, the G8OS node need to register themself against the AYS server that managed the node.
In order to do that, core0 need to send an event to the AYS server.

You can find the script which enabled this service here: [initamfs/discover.toml](https://github.com/g8os/initramfs/blob/0.12.0/config/g8os-conf/discover.toml)

Notice the `{ays}`. This string will be replace by the kernel parameters passed during boot.


## Weird header
The header of the script is a workaround for checking the existence of the parameter:
```bash
hexhost=$(echo \"{ays}\" | hexdump -e '\"%x\"')
if [ \"$hexhost\" == \"7379617ba7d\" ]; then
    echo \"No AYS host set, skipping discovery\"
    exit 0
fi
```

You can't compare is `"{ays}" == "{ays}"` because it will be replaced later and be always equals.

This trick compare the hex-value of the content to a well-know hex-value (`7379617ba7d` is `{ays}` hex-representation).

This trick avoid running the script if the argument is not defined.

# Example with qemu
You can run the kernel by appending argument with qemu directly:
```shell
qemu-system-x86_64 -kernel /tmp/vmlinuz.efi -m 2048 -enable-kvm -cpu host -net nic,model=e1000 -net bridge,br=lxc0  -nodefaults  -nographic -serial mon:stdio -append ays=192.168.5.10:5000
```
