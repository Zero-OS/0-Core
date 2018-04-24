## Zero-Robot
By default, when the system starts it auto start a container that auto run zrobot.
The robot listens on `6600` by default but you can control which interface it can
accept connections from by controlling the boot params.

By default zrobot will listen on ALL interfaces, unless a `zerotier=<network>` flag is
passes as boot params. In that case it will only accept connection from zerotier interface.

> Please check [Booting](../booting/README.md) for more details about the boot params

The robot starts with the templates from [0-templates](http://github.com/zero-os/0-templates). For full
documentation of how to use 0-robot please refer to the [official documentation](https://github.com/zero-os/0-robot/tree/master/docs)