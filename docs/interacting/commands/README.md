# Commands

## 0-core

0-core is the master process for Zero-OS, replacing systemd, the init system for bootstrapping the user space and managing all processes.

Interacting with Zero-OS is done by sending commands to 0-core, allowing you to manage disks, set-up networks, and run both containers and virtual machines.

When Zero-OS starts, 0-core is the first process that starts. First it configures the networking, and then starts a local Redis instance through which the actual commands are received, and dispatches the commands to the other processes, e.g. the CoreX cores. CoreX is the master process of a container running on Zero-OS, the equivalent of 0-core in the containers.

You always certainly won't need to compose and push a command to the zero-os queue manually. All available commands are abstracted in the [python client](../../client/py-client) 0-core repo and JumpScale.

A command is a json serialized string of the following object structure

## Command structure

```javascript
{
	"id": "command-id",
	"command": "command-name",
	"arguments": {},
	"queue": "optional-queue",
	"stats_interval": 0,
	"max_time": 0,
	"max_restart": 0,
	"recurring_period": 0,
	"stream": false,
	"log_levels": [int]
}
```

- id: unique command id. This ID is generated or set by the client. If 2 commands are pushed to zero-os with the same ID. the second one will get **DROPPED**.
- command: the actual command name (ex: core.system)
- arguments: arguments of the command
- queue: Push the command to an internal queue for synchronization. Commands on queues are process sequentially.
- max_time: If command execution takes more that this given time in seconds the process is forced to stop.
- max_restart: How many times to restart the command if it exited with error.
- recurring_period: If set, the command execution is rescheduled to execute repeatedly, wating for `recurring_period` seconds between each excution.
- stream: Enable command output streaming
- log_levels: Which log levels are captured from command output.

> `arguments` structure totally depends on the command name. the py-client is promissed to always be up-to-date with the available commands and their arguments


Hereby:
- See [Streaming Process Output from Zero-OS](../streaming.md) for more details about the `stream` attribute.
- With the `log_levels` attribute you can filter which log levels will get passed to the loggers, if nothing specified all log levels will be passed. See [Logging](../../monitoring/logging.md) for more details.

0-core understands a very specific set of commands:
- [Core commands](core.md)
- [Info commands](info.md)
- [Container commands](container.md)
- [Bridge commands](bridge.md)
- [Disk commands](disk.md)
- [Btrfs commands](btrfs.md)
- [ZeroTier commands](zerotier.md)
- [KVM commands](kvm.md)
- [Job commands](job.md)
- [Process commands](process.md)
- [Filesystem commands](filesystem.md)