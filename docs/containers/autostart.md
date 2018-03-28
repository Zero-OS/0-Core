# Starting containers on boot
It is possible to start containers on boot via config files by putting
a similar toml config in zero-os config directory

```toml
[startup.ovs]
name = "corex.create-sync"
tags = ["ovs"] #container tags
protected = true #set protected or not

[startup.ovs.args]
root = "http://path/to/image.flist" #
name = "name"
```

The `args` section supports all the args that the client supports.

# Auto start services on container creation
A container flist can define one or more tasks that should started on container creation. This is accomplished by defining a file names `/.startup.toml`
in your flist.

The `.startup.toml` file must exist at the `/` (root) of the flist and it has the same exact structure as the main zos [start up files](../config/startup.md)

An example [startup file](https://github.com/zero-os/openvswitch-plugin/blob/master/startup.toml) here is used to auto start ovs services once the container
is created.

# Container Plugins
A container flist can also define a more complex feature called `plugins` (optional) where the flist creator defines an `extension` to the known container commands. They
work as wrapper arround some binaries that are built in the flist

Plugins are defined in a file named `/.plugins.toml` that must exist at the root of the flist

```toml
[plugin.<name>]
path = "/path/to/binary"
queue = true|false
exports = ["list", "of", "functions", "names"]
```

The plugins works as follow, let's assume our plugin is called `test`, that exports a function called `test1` and `test2`
```toml
[plugin.test]
path = "/path/to/binary"
exports = ["test1", "test2"]
```

after container creation a user can do a call like
```python
container.sync("test.test1", {"some": "arguments"})
```

and this will become translated to a
```bash
/path/to/binary test1 '{"some": "argument"}'
```

The point is, your binary can handle the input and then output (on stdout) a valid json message as defined in [loggin message format](../monitoring/logging.md#message-format)
this way, a very complex commands can really abstracted as simple as

```python
result = container.json("test.test1", {"some": "arguments"})
```

> The optional `queue` flag which means all calls to any function on this plugin are gonna be queued, which means no 2 functions are gonna be executed at the same time

Example [plugins file](https://github.com/zero-os/openvswitch-plugin/blob/master/plugin.toml)