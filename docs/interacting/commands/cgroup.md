# Cgroup Commands

Available commands:

- [ensure](#ensure)
- [list](#list)
- [remove](#remove)
- [tasks](#tasks)
- [task-add](#task-add)
- [task-remove](#task-remove)
- [reset](#reset)
- [memory](#memory)
- [cpuset](#cpuset)


## ensure
Make sure that a cgroup exists, and create it if it does not. The ensure method does not change configuration of the group if it exists.

Arguments:
```javascript
{
    'subsystem': {subsystem},
    'name': {name},
}
```

Values:
- **{subsystem}**: the Cgroup subsystem currently only support (`memory`, and `cpuset`)
- **{name}**: name of the cgroup

## list

Lists all available cgroups on a host. It takes no arguments.

## remove

Removes a cgroup. Note that any process that is attached to this cgroups are moved to the default cgroup which has no limitation

Arguments:
```javascript
{
    'subsystem': {subsystem},
    'name': {name},
}
```

Values:
- **{subsystem}**: the Cgroup subsystem currently only support (`memory`, and `cpuset`)
- **{name}**: name of the cgroup

## tasks

List all tasks/processes that are added to this cgroup

Arguments:
```javascript
{
    'subsystem': {subsystem},
    'name': {name},
}
```

Values:
- **{subsystem}**: the Cgroup subsystem currently only support (`memory`, and `cpuset`)
- **{name}**: name of the cgroup


### task-add
Add a new task (process ID) to a cgroup

Arguments:
```javascript
{
    'subsystem': {subsystem},
    'name': {name},
    'pid': {pid},
}
```

Values:
- **{subsystem}**: the Cgroup subsystem currently only support (`memory`, and `cpuset`)
- **{name}**: name of the cgroup
- **{pid}**: PID to add


### task-remove
Remove a task/process (process ID) from a cgroup

Arguments:
```javascript
{
    'subsystem': {subsystem},
    'name': {name},
    'pid': {pid},
}
```

Values:
- **{subsystem}**: the Cgroup subsystem currently only support (`memory`, and `cpuset`)
- **{name}**: name of the cgroup
- **{pid}**: PID to remove

### reset
Resets all limitations on the cgroup to the default values

Arguments:
```javascript
{
    'subsystem': {subsystem},
    'name': {name},
}
```

Values:
- **{subsystem}**: the Cgroup subsystem currently only support (`memory`, and `cpuset`)
- **{name}**: name of the cgroup


### memory
Get/Set memory limits on a memory cgroup. A call to this method without a `mem` value will not change the current limitations.
A call to this method will always return the current values.

Arguments:
```javascript
{
    'name': {name},
    'mem': {mem},
    'swap': {swap},
}
```

Values:
- **{name}**: name of a memory cgroup
- **{mem}**: Set memory limit to the given value (in bytes), ignore if 0
- **{swap}**: Set swap limit to the given value (in bytes) (only if mem is not zero)


### cpuset
Get cpuset cgroup specification/limitation the call to this method will always GET the current set values for both cpus and mems If cpus, or mems is NOT NONE value it will be set as the spec for that attribute

Arguments:
```javascript
{
    'name': {name},
    'cpus': {cpus},
    'mems': {mems},
}
```

Values:
- **{name}**: name of a cpuset cgroup
- **{cpus}**: Set cpus affinity limit to the given value (0, 1, 0-10, etc...)
- **{mems}**: Set mems affinity limit to the given value (0, 1, 0-10, etc...)

