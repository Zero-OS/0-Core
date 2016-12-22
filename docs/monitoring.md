# Monitoring
## Introduction
Monitoring, is the process of collecting metrics about the system and the health of the services.
Any running process can report statistics about it's operations to its core (core0 or coreX) using
the same [logging](logging.md) mechanism.

One of the reserved log message levels is log level `10` which is reserved for stats reporting. It basically
means that a process can output a message like
```
10::some.metric.key:23.12|A
```
To report that the value of `some.metric.key` at time `now` is `32.12` and the reported values should
be averaged over the defined aggregator period (usually `5 minutes`)

## Stats message format
Syntax of status message is as follows

```
10::<key string>:<value float>|OP[|tags]
```
- `10::` (const) is the message prefix and must be of this value. it tells `core0/X` that this is a stats message
- `key` (string) is the metric key reported by the process.
- `value` (float) is the metric value at the time of the reporting
- `OP` (string) How to aggregate the reported values
  - `A` Average the values reported at the end of the current aggregator period
  - `D` Differential values (used usually for incremental counters like number of packets over network card)
- `tags` (string optional) user defined tags attached to the metric (currently not used)

## Built in monitoring
Core0 has built in monitoring command that when called reports the following metrics

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

It's not automated to run by default, but by including this [config](../core0/conf/monitor.toml) it will be 
scheduled to run automatically and report system metrics
 
## Where do the metrics go anyway?
I am glad u asked. 
The metrics will be pushed to our redis stored procedure that do the aggregation of the metrics and then push all
the aggregated metrics (every 5 minutes and every 1 hour) to specific redis queues. Later own, a 3rd party
software can pull the aggregated metrics and push it to a graph-able database like `influxdb` for visuals.

The 2 queues to hold the aggregated metrics are
- queues:stats:min
- queues:stats:hour

Each object in the queue is a string that is formatted as following:

```
node|key|epoch|last reported value|calcluated avg|max reported value|total
```