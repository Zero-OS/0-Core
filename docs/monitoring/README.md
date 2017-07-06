# Monitoring

All running processes can feedback logs about their operations using the same logging mechanism.

In [Logging](logging.md) you learn about how the output of processes is processed as log messages.

One of the processes that comes preinstalled with Zero-OS is the Zero-OS monitoring process.

See below for:
- [Monitoring metrics](#monitoring-metrics) - an overview of all metrics that are monitored by the Zero-OS monitoring process
- [Configuring monitoring](#configuring-monitoring)  - discussing how to configure the Zero-OS monitoring process

The output messages of the Zero-OS monitoring process are all prefixed as level 10 log messages. Or in other words: statistics are reported as level 10 log messages.

Here's an example of a statistics log message:
```
10::{monitoring-metric}:23.12|A
```

This example level 10 (statistics) log message reports that the metric `{monitoring-metric}` is `23.12` and that the reported values should be averaged over the defined aggregator period (usually 5 minutes), as indicated by the `A`; see [Statistics Log Message Format](stats-msg-format.md) for more details about this statistics log message format.


## Monitoring metrics

The built-in Zero-OS monitoring process logs statistics for following metrics:

```
disk.iops.read@phys.sda
disk.iops.read@phys.sda1
disk.iops.write@phys.sda
disk.iops.write@phys.sda1
disk.throughput.read@phys.sda
disk.throughput.read@phys.sda1

machine.CPU.contextswitch@phys
machine.CPU.interrupts@phys

machine.CPU.percent@pyhs.0
machine.CPU.percent@pyhs.1
machine.CPU.percent@pyhs.2
machine.CPU.percent@pyhs.3
machine.CPU.percent@pyhs.4
machine.CPU.percent@pyhs.5
machine.CPU.percent@pyhs.6
machine.CPU.percent@pyhs.7

machine.CPU.utilisation@pyhs.0
machine.CPU.utilisation@pyhs.1
machine.CPU.utilisation@pyhs.2
machine.CPU.utilisation@pyhs.3
machine.CPU.utilisation@pyhs.4
machine.CPU.utilisation@pyhs.5
machine.CPU.utilisation@pyhs.6
machine.CPU.utilisation@pyhs.7

machine.memory.ram.available@phys
machine.memory.swap.left@phys
machine.memory.swap.used@phys

network.packets.rx@phys.core-0
network.packets.rx@phys.eth0
network.packets.rx@phys.lo
network.packets.rx@phys.zt0
network.packets.tx@phys.core-0
network.packets.tx@phys.eth0
network.packets.tx@phys.lo
network.packets.tx@phys.zt0

network.throughput.incoming@phys.core-0
network.throughput.incoming@phys.eth0
network.throughput.incoming@phys.lo
network.throughput.incoming@phys.zt0
network.throughput.outgoing@phys.core-0
network.throughput.outgoing@phys.eth0
network.throughput.outgoing@phys.lo
network.throughput.outgoing@phys.zt0
```


## Configuring Monitoring

The built-in Zero-OS monitoring process is scheduled to run automatically as defined in the `conf/monitor.toml` configuration file:

```
[startup."monitor.cpu"]
name = "monitor"
recurring_period = 30 #seconds

[startup."monitor.cpu".args]
domain = "cpu"

[startup."monitor.memory"]
name = "monitor"
recurring_period = 30 #seconds

[startup."monitor.memory".args]
domain = "memory"

[startup."monitor.disk"]
name = "monitor"
recurring_period = 30 #seconds

[startup."monitor.disk".args]
domain = "disk"

[startup."monitor.network"]
name = "monitor"
recurring_period = 30 #seconds

[startup."monitor.network".args]
domain = "network"
```
