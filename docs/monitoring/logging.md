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
