## Statistics

Statistics are reported as log messages with `log level` 10, as discussed in [log levels](logging.md#log-levels).

The statistics message format is as follows:

```
10::<key>:<value float>|<OP>[|<tags>]
```

Hereby:

- `10::` (const) is the message prefix and must be of this value. it tells `core0/X` that this is a stats message
- `key` (string) is the metric key reported by the process.
- `value` (float) is the metric value at the time of the reporting
- `{OP}` (string) How to aggregate the reported values
  - `A` Average the values reported at the end of the current aggregator period
  - `D` Differential values (used usually for incremental counters like number of packets over network card) (delta to previous D value)
- `tags` (string optional) user defined tags attached to the metric (currently not used)


<a id="stats-sending"></a>
## Where do the statistics go anyway?
The metrics will be pushed to our Ledis (every 300 seconds and every 3600 seconds) to specific Ledis queues.
Later own, a 3rd party software can pull the aggregated metrics and push it to a graph-able database like `influxdb` for visuals.

The 2 queues to hold the aggregated metrics are:

- **statistics:300** for the 5 min aggregation  
- **statistics:3600** for the 1 hour aggregation

Each object in the queue is a json object that is formatted as following:

```javascript
{
 'key': 'metric.key', // reported metric key
 'tags': '', //reported metric tags
 'avg': 1605.370703125, //avergae value of the metric over the defined period (300 second, or 3600 seconds according to queue)
 'count': 10, //how many samples reported during this period
 'max': 1605.48828125, //max reported sample during this period
 'start': 1498033200, //start time of the period
 'total': 16053.70703125 //total of the reported values
}
```
