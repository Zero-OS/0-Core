# Logging

Discussed here:

- [Log mechanism](#log-mechanism)
- [Log format](#log-format)
- [Log levels](#log-levels)
- [Where logs are send to](#log-sending)


<a id="log-mechanism"></a>
## Logging Mechanism

In Core0 terminology, logging means capturing the output of the running processes and store or forward it to loggers.

A logger can decide to print the output of the command to the console, and/or push it to Ledis.

You can control for each logger which messages should be logged. This is achieved  with the `levels` setting in the [main configuration file](../config/main.md):

```
[logging.file]
levels = [1, 2, 4, 7, 8, 9]

[logging.ledis]
levels = [1, 2, 4, 7, 8, 9]
size = 10000 # how many log lines to keep in ledis

[stats]
enabled = true
```

As you can see, in the (above) default configuration template of Core0, all log messages of levels `[1, 2, 4, 7, 8, 9]` are passed to both to the log file and Ledis.

When issuing a command you can override this configuration using the `log_levels` attribute of the command to force all loggers to capture and process specific log levels for that command.

CoreX logging is not configurable, it simply forwards all logs to Core0 logging. Which means Core0 logging configuration applies for both Core0 and CoreX domains.

See `/0-core/core0/logger/` for the implementation of the loggers are implemented, both written in Go.


<a id="log-format"></a>
## Logging Messages

When running any process on Core0 or CoreX the output of the processes are captured and processed as log messages. By default messages that are output on `stdout` stream are considered of level `1`, messages that are output on `stderr` stream are defaulted to level `2` messages.

The running process can leverage on the ability of Core0 to process and handle different log messages by prefixing the output with the desired level as:

```
8::Some message text goes here
```

Or for multi-line output bulk:

```
20:::
{
    "description": "A structured json output from the process",
    "data": {
        "key1": 100,
    }
}
:::
```

Using specific levels, you can pipe your messages through a different path based on your nodes.

Also all `result` levels will make your return data captured and set in the `data` attribute of your job result object.


<a id="log-levels"></a>
## Log Levels

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
- 20: result message, json
- 21: result message, yaml
- 22: result message, toml
- 23: result message, hrd
- 30: job, json (full result of a job)


<a id="log-sending"></a>
## Where do the logs go anyway?

Two places
- file `/var/log/core.log`
- ledis queue `core:logs`
