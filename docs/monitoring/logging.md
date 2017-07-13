# Logging

Discussed here:

- [Logging mechanism](#logging-mechanism)
- [Message format](#message-format)
- [Log levels](#log-levels)


## Logging mechanism

In Zero-OS 0-core captures the output of all running processes as "log messages" and forwards them to loggers.

Currently there are two loggers available, both implemented in Go:
- [File logger](/core0/logger/logger.go) writes log messages to `/var/log/core.log`
- [Ledis logger](/core0/logger/ledis.go) writes log messages to the LedisDB queue `core:logs`

In the `zero-os.toml` configuration file, as documented in [Main Configuration](../config/main.md), you specify for each logger which categories of log messages it should process. Log messages are categorized by `levels`:

```
[logging.file]
levels = [1, 2, 4, 7, 8, 9]

[logging.ledis]
levels = [1, 2, 4, 7, 8, 9]
size = 10000 # how many log lines to keep in LedisDB

[stats]
enabled = true
```

As you can see, in the above default configuration template, all log messages of levels (categories) 1, 2, 4, 7, 8 and 9 are passed to both to the file logger and the Ledis logger. For defintion of each log level see below in [Log levels](#log-levels).

Processes can leverage this mechanism by prefixing their output with a specific level, as shown below in [Message format](#logging-format).

When issuing a command, as discussed in [Commands](../interacting/commands/README.md), using the command's attribute `log_levels` you can filter which output of the command gets passed to the loggers. Setting for instance the value of `log_levels` to `[2,9]` will only pass `(2) stderr` and `(9) critical error` output to the loggers that been configured to process log messages of level 2 and/or 9.

Logging in the containers is not configurable, it simply forwards all logs to 0-core. Which means that logging configuration applies for both 0-core processes and the container processes.


## Message format

The running process can leverage on the ability of 0-core to process and handle different log messages by prefixing the output with the desired level as:
```
8::Some message text goes here
```

Or for multi-line output:
```
20:::
{
    "description": "A structured JSON output from the process",
    "data": {
        "key1": 100,
    }
}
:::
```


## Log levels

By default messages that are output on `stdout` stream are considered level `1`, messages that are output on `stderr` stream are considered level `2`.

- 1: stdout
- 2: stderr
- 3: message for endusers / public message
- 4: message for operator / internal message
- 5: log msg (unstructured = level5, cat=unknown)
- 6: log msg structured
- 7: warning message
- 8: ops error
- 9: critical error
- 10: statistics/monitoring message(s)
- 20: result message, JSON
- 21: result message, yaml
- 22: result message, toml
- 23: result message, hrd
- 30: job, json (full result of a job)


## Streams
The Ledis logger aggregate all logs from all process to a single queue so a system like `logstash` will be
able to pull the logs from _all_ the jobs running on the system. There is another way to read streams 
of a single process in runtime.

Once `stream` flag is set on a command, zero-os will make sure this command logs are sent to a separate queue for that
job (also to the aggregated queue). The client already exposes a stream method to read the output stream of a job

```python
job = client.system('ping google.com', stream=True)

job.stream() # this will start printing ping output in real time on screen. Check stream docstr
```

Check [Streaming docs](../interacting/streaming.md) for more details
 
## Subscribers
Although streams is useful in most cases, sometimes we need to process the output stream of a job
by multiple `subscribers`, streams support described above has the following cons:

- Only one receiver can listen to job output stream
- Enabling streams has to be planned a head starting the job with the stream flags, once started
  the state of the flag can't be changed.

Subscribers on the other hand, allows anyone (also any number of the subscribers) to hook to the streams of any job

```python
job = client.system('ping google.com')

subscriber = job.subscribe()

subscriber.stream() # this again, will print the ping output in real time on screen. Check stream docstr
```

In another process/thread u can safely start another subscriber on the same job

```python
job = client.response_for('job id')

subscriber = job.subscribe()
subscriber.stream() # this again, will print the ping output in real time on screen. Check stream docstr
```

> Currently there is noway to un-subscribe from a job stream, subscriber job will terminate automatically
once the watched job exits. Also killing a subscriber job won't stop it or affect the watched job by any means.