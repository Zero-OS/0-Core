# Configuring Core0

- Main configuration
- Adding extensions
- Starting up services on boot
  - Start up dependencies

## Main core0 configuration
defaults to `/etc/g8os/g8os.toml`

```toml
# Main configuration section to manage the process manager behaviour
[main]
# max_jobs sets max number of concurrent jobs. Once this limit is reached
# core0 will not pull for any new jobs from his dedicated redis
# queue until it has at least one free job slot to fill.
max_jobs = 200

# `include` more configuration from the specific locations. core0 on boot will try to load
# all `.toml` files from the given locations, where each toml can define one or more extension
# to core0 commands, and/or start up services
include = ["/root/conf"]

# `log_level` sets the logging level of core0 logs (what prints on console)
log_level = "debug"

# `network` points to the network configuration file.
network = "./network.toml"

# Each sink is defined in each own section as `[sink.<name>]` where name can be anything.
# A sink is a source of commands to run jobs, theoritically we can support more than sink type.
# Currently the only supported sink type is `redis`
[sink.main]
# `url` redis source url as shown below
url = "redis://127.0.0.1:6379"
# `password` optional redis password.
password = ""

#Logging section defines how core0 process logs from running jobs.
[logging]
#each logger is defined in its own section,
    [logging.console]
    # define the logger type, we have support for `console` (prints on stdout of core0)
    type = "console"
    # define which log levers are logged to this logger
    levels = [1, 2, 4, 7, 8, 9]

	[logging.redis]
	type = "redis"
	levels = [1, 2, 4, 7, 8, 9]
	address = "127.0.0.1:6379"
	# batch_size (wronglly named) is how many log messages are kept in the queue
	# if the redis queue length reached this limit, the queue will start to be trimmed
	# so older log messages will be dropped
	batch_size = 1000


[stats]
# `interval` is deprecated and should be removed from the config files.
interval = 60000 # milliseconds (1 min)

# redis stats aggregator, use the redis lua script to aggregate statistcs outed by jobs.
[stats.redis]
# `enabled`
enabled = true
flush_interval = 10 # seconds
address = "127.0.0.1:6379"

# global config available for built in modules.
[globals]
# `storage` default Zero-OS file system to use when not passed by the container.create command.
storage = ""
```

## Extensions
An extension is simply a new command or functionality to extend what core0 can do (check built in supported commands).
This allow adding new functionality and commands to core0 without actually changing it's code.

An extension works as a wrapper around the `core.system` command by wrapping the actual command
call in a more abstract way.

### Example extension
Lets say that we want to add a user management extension to add and remove users and also change
 their passwords
```toml
[extension."user.add"]
binary = "useradd"
args = ["-m", "{username}"]

[extension."user.delete"]
binary = "userdel"
args = ["-f", "-r", "{username}"]

[extension."user.chpasswd"]
binary = "sh"
args = ["-c", "echo '{username}:{password}' | chpasswd"]
```

Writing the above into a toml file under one of the included paths defined by `include` in the main
 core0 toml will add the following commands to core0

- user.add
  - Args: `{"username": "user name to add"}`
- user.delete
  - Args: `{"username": "user name to remove"}`
- user.chpasswd
  - Args: `{"username": "user", "password": "password to set"}`

Then u can simply call the extension from the python client as follows
```python
client.raw("user.add", {"username": "testuser"})
client.raw("user.chpasswd", {"username": "testuser", "password": "new-password"})
```

> Core0 take care of substituting the `{key}` notation in the extension arguments with
the ones passed from the client.

Extension also supports the following attributes

```toml
[extension.test]
binary = "binary"
args = ["args", "list", "to", "binary"]
# cwd of the binary
cwd = "/path"

#env variables will be available during command execution
[extension.test.env]
env1 = "value-1"
env2 = "value-2"
```

## Startup services
Core0 on boot, will start all services defined in all the `toml` files included as defined by the `include`
attributes in the main config file.

A startup service is an automatic calling of ANY of the available commands, even the ones that
are defined via extensions.

A service can define a list of dependencies where the service will only run `after` all the its dependencies
has `ran` successfully. A service is considered `running` according to certain configurable criteria.  

### Structure of startup service
```toml
[startup."service id"]
name = "command.name"
after = ["dep-1", "dep-2"]
running_delay = 0
running_match = ""
recurring_period = 30
max_restart = 10

[startup."service id".args]
key = "value"
key2 = 100
```

#### `name` attribute
Define what core0 command to execute like `core.system` or others (can be an extension doesn't have to be built in)

#### `after` attribute
After defines a list of services that must be considered running, before core0 attempt to start
this service. The dependencies names can be one of the defined services ids defined in other toml
files, or one of the phony services names built in core0 `init, net, and boot`
- `init` means that this service must be started as fast as possible even before core0 attempt to
  setup the networking. Services like that are needed for the hardware operation (ex: loading modules or starting udev)
  When the init run is complete core0 will attempt to setup networking.
- `net` phony service means this service must run as fast as possible once networking is up. This can
include joining `zerotear` network, or registering itself with an ays service. Once those services
are up, core0 will move on starting the next slice of services which has after = ['boot']
- `boot` that's the default dependencies of any service that doesn't define an `after`

### `running_delay` attribute
By default, a service is considered running if it started and did not exit for 2 seconds. This value can be
adjusted by setting the `running_delay` to the required value.
You can set `running_delay` to a value less than zero (-1 for example) which means you can assume that
the service has ran unless it exits successfully. This is usually used by startup scripts that needs
to prepare something (crate directories, or clean up files), before other services starts, so it needs to
exit before u can start subsequent services.

### `running_match` attribute
This has higher presence than the `running_delay` attribute, so if both are defined `running_delay` will be
ignored. `running_match` is a regural expression that will flags the service as `running` if the service
output a line that matches this expression. So simply u assume the service is running if it prints something
like `service is up` or whatever.

### `recurring_period` attribute
Run this job every this defined number of seconds

### `max_restart` attribute
If service exited with an error, restart it this max number of trials before giving up.

### `args` section
Arguments needed to start this service, this depends totally on the command name you going to execute

For example, if the name is `core.system` so args (as defined by core.system) are
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
