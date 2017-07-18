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