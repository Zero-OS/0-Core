## Building plugins for containers
Containers support automatic loading of plugins from the container image. When a container starts
it searches the plugin search path `/var/lib/corex/plugins` for valid `.so` plugins

Plugins adds capabilities to the container coreX by plug-in handlers to function calls

### Plugin structure
To create a plugin create a new go package as follows

```go
package main

import (
	"encoding/json"
	"github.com/g8os/core0/base/plugin"
)

const (
	Version = "1.0alpha"
)

func version(input json.RawMessage) (interface{}, error) {
	return Version, nil
}

var (
	Plugin = plugin.Commands{
		"version": version,
	}
)

func main() {
	plugin.Plugin(Plugin)
}
```

## Building a plugin
Nothing special!
```bash
go build
```

Place the outputed executable binary file in your container image.
Then on the root of your image place a `.plugin.toml` file with the following content
```toml
[plugin.ovs]
path = "/var/lib/corex/plugins/ovs-plugin"
exports = ["version"]
```

> the exports, tells the container which methods are available for execution (in our example `version`)

> `path` is the path to the plugin binary under your container image (absolute from the container image root)

Calling the plugin method.
```python
cl = Client("hostname")
id = cl.container.create('url to image that has the plugin').get()
container = cl.container.client(id)

container.raw("ovs.version", {}) ### returns "1.0alpha"
```
