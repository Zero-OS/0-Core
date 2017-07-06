# Commands

## 0-core

0-core is the master process for Zero-OS, replacing systemd, the init system for bootstrapping the user space and managing all processes.

Interacting with Zero-OS is done by sending commands to 0-core, allowing you to manage disks, set-up networks, and run both containers and virtual machines.

When Zero-OS starts, 0-core is the first process that starts. First it configures the networking, and then starts a local Redis instance through which the actual commands are received, and dispatches the commands to the other processes, e.g. the CoreX cores. CoreX is the master process of a container running on Zero-OS, the equivalent of 0-core in the containers.

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

## Check wether Redis is listening

A basic test to check if your Zero-OS is ready to receive commands, is using the `redis-cli` Redis command line tool:
```shell
ZEROTIER_NETWORK="..."
REDIS_PORT="6379"
redis-cli -h $ZEROTIER_NETWORK -p $REDIS_PORT ping
```
