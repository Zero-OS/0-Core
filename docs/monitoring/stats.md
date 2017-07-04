# Statistics Log Message Format

Statistics are reported as level 10 log messages. See [Logging](logging.md#log-levels) for an overview of all log levels.

The statistics message format is as follows:
```
10::<key>:<value float>|<OP>[|<tags>]
```

Hereby:
- `10::` (const) is the message prefix that tells 0-core that it is a statistics message
- `key` (string) is the metric key reported by the process
- `value` (float) is the metric value at the time of the reporting
- `{OP}` (string) specifies how to aggregate the reported values
  - `A` Averages the values reported at the end of the current aggregation period
  - `D` Differentiates the values (used usually for incremental counters, e.g. the number of packets send over network card; delta to previous D value
- `tags` (string optional) user defined tags attached to the metric formatted as _key=value,..._ (ex: device=eth0,type=physical)

<a id="stats-sending"></a>
## Where do the statistics go anyway?

By default level 10 log messages are pushed (every 300 seconds and every 3600 seconds) to following LedisDB queues:

- **statistics:300** for the 5 minutes aggregation  
- **statistics:3600** for the 1 hour aggregation

Each object in the queue is a JSON object that is formatted as following:

```javascript
{
 'key': 'metric.key', // reported metric key
 'tags': {key: value, ...}, //reported metric tags
 'avg': 1605.370703125, //average value of the metric over the defined period (300 second, or 3600 seconds according to queue)
 'count': 10, //how many samples reported during this period
 'max': 1605.48828125, //max reported sample during this period
 'start': 1498033200, //start time of the period
 'total': 16053.70703125 //total of the reported values
}
```

You can use a 3rd-party software package to pull the aggregated metrics from the LedisDB queues and push then into a graphable database, e.g. InfluxDB.
