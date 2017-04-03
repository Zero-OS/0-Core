## Building plugins for containers
Containers support automatic loading of plugins from the container image. When a container starts
it searches the plugin search path `/var/lib/corex/plugins` for valid `.so` plugins

Plugins adds capabilities to the container coreX by plug-in handlers to function calls

### Plugin structure
To create a plugin create a new go package as follows

```go
package main

import (
	"github.com/g8os/core0/base/plugin"
	"github.com/g8os/core0/base/pm/core"
)

func ping(cmd *core.Command) (interface{}, error) {
	return "pong", nil
}

var Manifest = plugin.Manifest{
	Domain:  "test",
	Version: plugin.Version_1,
}

var Plugin = plugin.Commands{
	"ping": ping,
}

```

> It must be a `main` package

- *Manifest* sets the function domains, so the `ping` method will be callable as
`test.ping` if the `Domain` is not set, the name of the `.so` file will be used instead.
- *Version* sets the plugin interface version, so coreX can support older plugin formats.
- *Plugin* sets the map of commands

## Building a plugin
```bash
go build -buildmode=plugin -o test.so
```

By placing the output `.so` file under the plugin search path as defined above (inside the container image of-course)
once you start the container, you can call the `test.ping` method as follows

```python
cl = g8core.Client("hostname")
id = cl.container.create('url to image that has the plugin')
container = cl.container.client(id)

container.raw("test.ping", {}) ### returns "pong"
```
