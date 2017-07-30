# Network Configuration

Zero-OS applies the network configuration as defined in the `/etc/zero-od/network.toml`, structured as follows:

```toml
[network]
auto = true

[interface.<ifname>]
protocol = "<protocol>"

[<protocol>.<ifname>]
#protocol specific configuration
#for that interface
```

The configuration is applied as follows:

- It tries to reach at least one of the configured `controllers`
- If it can't reach any, it tries `dhcp` on all interfaces, one at a time and tries to reach any of the controllers again
- If it still can't reach a controller, it will fall back to a random static IP address in the `10.254.254.0/24` range and tries to reach a controller on `10.254.254.254`

The `[network]` section currently only has the `auto` flag, which tells core to auto configure all found interfaces using the DHCP method if there is no specific `interface` section for that interface. If `auto` is set to false, the core will just ignore any interface that is not specifically configured in the `network.toml` file.


## Supported network protocols

Currently only the following network protocols are supported:

- **none**: Only brings the interface up, it doesn't set an IP address on that interface, it also excludes it from the networking fallback plan.
- **dhcp**: Doesn't require any further configurations, specifying `protocl = "dhcp"` is enough
- **static**: Requires a `[static.<if>]` section, see the below example

Below two examples:

<a id="static"></a>
### Static Example

```toml
[interface.eth0]
protocol = "static"

[static.eth0]
ip = "10.20.30.40/24"
gateway = "10.20.30.1"
```

```toml
[interface.lo]
protocol = "static"

[static.lo]
ip = "127.0.0.1/8"

[interface.eth0]
protocol = "dhcp"

[interface.eth1]
protocol = "none"
```