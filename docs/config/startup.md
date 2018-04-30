# Startup Services

A startup service is any of the available commands and extensions, that get started automatically.

When Core0 boots, it will start all startup services defined in the TOML files specified in the `[include]` section of the main configuration.

Startup services are defined as follows in a `[startup.{service-id}]` section:

```toml
[startup.{service-id}]
name = "command.name"
condition = "condition expression"
after = ["dep-1", "dep-2"]
running_delay = 0
running_match = ""
recurring_period = 30
max_restart = 10

[startup."service id".args]
key1 = "value"
key2 = 100
```

- **{service-id}**: Unique identification tag for referencing the startup service in other configuration files

- **name**: Name the Core0 command to execute, e.g. `core.system`, can also be an extension
- **condition**: (optional) condition expression that must be true to auto start the service. default to a "true" expression. Check [condition syntax](#condition-syntax) below
- **after**: Lists the services identified by their {service-id} that must be considered running before Core0 attempts to start this service, these services can be any of the services defined in the other TOML files, or one of the Core0 built-in services `init, net, and boot`:
  - **init**: Service must be started as fast as possible, even before Core0 attempts to setup the networking, services like that are needed for the hardware operation (e.g. loading modules or starting udev), when the init run is complete Core0 will attempt to setup networking
  - **net**: Service must run as fast as possible once networking is up, this can include joining a ZeroTier network, or registering itself with an AYS service, once those services are up, Core0 will move on starting the other services which have `after` = ['boot']
  - **boot**: The default dependency of any service that doesn't define an `after`

- **running_delay**: By default a service is considered running if it started and did not exit within 2 seconds, this value can be adjusted here, you can set the parameter to a negative value (e.g. -1) which means you can assume that the service has ran unless it exits successfully, this is usually used by startup scripts that needs to prepare something (crate directories, or clean up files), before other services starts, so it needs to exit before u can start subsequent services

- **running_match**: This has higher presence than the `running_delay`, so if both are defined `running_delay` will be ignored, `running_match` is a regular expression that will flag the service as `running` if the service outputs a line that matches this expression, so simply you assume the service is running if it prints something like `service is up` for instance

- **recurring_period**: Run this job every time specified number of seconds

- **max_restart**: If service exited with an error, restart it, but only max number of trials before giving up

- **args**: Arguments needed to start this service, this depends totally on the command to execute, for example, if the name is `core.system` the arguments (as defined by core.system) are:
  ```
  name = "executable"
  dir = "cwd to run command"
  args = ["command", "arguements"]
  env = {"ENV1": "VALUE1", "ENV2": "VALUE2"}
  stdin = "data to pass to executable over stdin"
  ```

Bash for example requires only `script` argument also accepts `stdin`

> Startup `args` section also supports the `{variable}` name substitution. But it substitute the keys
with values passed to the kernel cmdline.

## Startup `args` conditions
Sometimes you want to apply different arguments based on a condition. Similar to the condition parameter above which control the entire service startup, an argument
can prefixed with a condition string. If the condition is not evaluated to true, the argument is dropped from the args block.

```
condition|argname = value
```

A real life example use of this option can be found [here](../../apps/core0/conf/redis.toml) where redis port opens on zt0 interface only if `zerotier` is set

## Condition Syntax
```
expression := and(expression, expression, ...)
expression := or(expression, expression, ...)
expression := not(expression)
expression := kernel-param
```

### Examples
```
condition = "development"
```
will evaluate to true if the kerenl has `development` as one of the kerenl params


```
condition = "not(development)"
```
true if `development` is NOT set


```
condition = "and(development, debug)"
```
condition is `true` if both `development` AND `debug` are set


more exmpales
```
condition = "or(development, debug)"
condition = "and(or(cond1, cond2), not(cond3))"
```

Note: empty expression is evaluated to true, which is the default behavior to start a service if a condition is not set