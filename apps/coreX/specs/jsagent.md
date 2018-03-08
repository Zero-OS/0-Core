Agent Specs
===========

Defs
----

### AgentController (AC)
-   Golang written webservice who works 100% against redis (ledis) as backend

### Agent (A)
-   Installed on windows or linux node
-   Talks to AC only using HTTPS/REST (longpolling for incoming messages)

### Logger (L)
-   Coroutine in agent which is responsible for processing all messages coming in.
-   Send message to L using a Queue in the golang agent
-   L will parse the raw message ( is textual coming from e.g. subprocess)
-   L will see what to do with message
    -   Log in sqlite
    -   Batch & send to AC

### ProcessManager (PM)
-   Coroutine in agent which manages & launches all subprocesses
-   A subprocess can be a lua or python script or it can be a daemon alike subprocess (cmdline) or an internal method.
-   Process manager monitors the behavior of the subprocess & will kill/restart if required.

Agent process manager
---------------------
-   Each jumpscript or long running daemon gets launched managed by the agent in a subprocess.
-   Each of the processes is monitored per 30 sec
    -   for cpu usage, mem usage, network?, …
    -   also for the subprocesses, just aggregate them all
-   Send as statsd messages to logger (using queues internal to Golang agent)

Stats
-----
-   send all as statsd format on level 10 (see below)
-   aggregated in memory by embedded statsd aggregator
-   statsd aggregated values are send to central agentcontroller batched over rest
-   Aggregation is done once per X sec (see config) and aggregated values are sent to AC.

Logging Messages
----------------
-   All messages are send to stout or stderr, nothing goes to logserver, …
-   Process manager captures stderr & stdout & will store in a local sqlite
-   Levels are marked as
    -   $level:: at beginning of line (2x ‘:’)
    -   exception 1 & 2: they are just stdout & stderr
-   messages are multiline if
    -   $level::: has 3 x ‘:’
    -   end of multiline when ::: found at beginning of line
-   levels
    -   1: stdout
    -   2: stderr
    -   3: message for endusers / public message
    -   4: message for operator / internal message
    -   5: log msg (unstructured = level5, cat=unknown)
    -   6: log msg structured
    -   7: warning message
    -   8: ops error
    -   9: critical error
    -   10: statsd message(s)
    -   20: result message, json
    -   21: result message, yaml
    -   22: result message, toml
    -   23: result message, hrd
    -   30: job, json (full result of a job)

-   sqlite store
    -   each message gets unique id
    -   all in 1 table
    -   1 db cannot grow beyond 100 MB
    -   if 100 MB then start new DB and rename old one as $currentepoch.db
        -   new db name: current.db

### Log Message Format
-   fields
    -   jobdomain : each running process in agent is for specific domain & cmdname
    -   jobname
    -   epoch : time when log message came
    -   level
    -   id: create unique incremental id (is done per agent, restart after 4 bytes)
    -   data: is the textual representation of what came from stdout
-   stored in sqlite
    -   most of logging info will not be send to AC e.g. stdout, logs, … but can be queried by AC when required
-   when forwarded to AC do in list of dicts & send as json

identification
--------------
-   each agent has unique id
-   each agent is part of unique grid id (gid)

agent config
------------

[Agent Configuration](https://github.com/Jumpscale/jsagent/wiki/agent-configuration)

Processes
---------
-   each process is identified by
    -   domain
    -   name
-   when more than 1 process with same name & domain then the stats are aggregated for both of them before sending to AC
-   this makes that at AC name we see each process & subprocesses identified by a unique domain/name
-   the domain/name can refer to a long running jumpscript or a jumpscript which runned once
-   this aggregation on this level will allow us to aggregate on rack, datacenter, region level

Cmds
----
Agentcontroller sends messages to Agent.
Agent connects to AC using long polling over rest
### Cmd Message format (result of executing a cmd is a job)
-   JSON
-   Dict
    -   id:$id unique id given by requester (is id for the job)
    -   gid:grid id is unique
    -   nid:node id, only 1 agent per node, is unique per grid
    -   cmd:$name of command (for list of cmds see below)
    -   args:$dict with arguments relevant for cmd
    -   data:$value depending cmd means something different
-   return is again a json dict
    -   id:$id unique id given by requester
    -   gid:grid id is unique
    -   nid:node id, only 1 agent per node, is unique per grid
    -   cmd:$name of command
    -   args:$dict with arguments for cmd
    -   data:return data
    -   state: OK, ERROR, TIMEOUT, KILLED
    -   time: time it took in msec
    -   starttime: when job started on agent (msec)

### Cmds
-   execute_js_lua (implemented)
    -   args
        -   max_time: in seconds how long maximum script can run (-1 for long runing processes -will be remembered during restart- 0 means can take as long as it needs, will not be remembered and otherwise it's the timeout value)
        -   max_restart: max time that PM will try to restart (if fails < 5 min we count times started, if max reaced send critical alert)
        -   domain: domain of jumpscript (to do categorization)
        -   name: name of jumpscript or cmd to execute (will give all a name)
        -   loglevels: * or 1,2,3 or 1 or 1-5 #defines which msglevels are processed (rest is ignored)
        -   loglevels_db: * or 1,2,3 or 1 or 1-5 #defines which msglevels are send to db
        -   loglevels_ac: * or 1,2,3 or 1 or 1-5 #defines which msglevels send to ac (this overrules the default config)
        -   recurring_period: 0 or 100
            -   seconds between recurring execute this cmd
            -   0 is default means not recurring, so only once
        -   stats_interval:120 means we overrule the default for this process and only gets stats each 120 sec
    -   data:
        -   is data passed to jumpscript over stdin when starting, can be empty
        -   if data is required then done as json by default
-   execute_js_py (implemented)
    -   see execute_js_lua
-   execute (implemented)
    -   execute a cmd
    -   see execute_js_lua
    -   additional arguments
        -   cmd: command to execute with full path
            -   $agentroot, $root, $tmp are template vars which will be replaced by PM
        -   workingdir: working dir for cmd to execute in

-   get_msgs (not implemented)
    - Only return the first 1000 match.
    - data:
        -   jobid (optional)
        -   timefrom,timeto (optional)
        -   levels (2 or 1,2,3 or 1-6, or *)

-   del_msgs (not implemented)
    -   same as get_msgs but to delete them

-   restart (implemented)
    -   restart agent
> restart will just force the agent to exit. A deamon manager should be responsible for bringing it up again.

-   killall (implemented)
    -   killall processes managed by agent
        -   the recurring ones are remembered & will be scheduled again
        -   the long running ones are also remembered and will be restarted
-   kill (implemented)
    -   kill a process managed by agent (with cmd-id)
    -   data:
        -   id
-   get_processes_stats (implemneted)
    -   data
        -   domain (optional)
        -   name (optional)
    -   gets cpu, mem, … from processes known to PM running at this point
-   get_process_stats (implemneted)
    -   data
        -   id (mandatory)
    -   gets cpu, mem, … from process with cmd-id
-   get_nic_info (implemented)
    -   gets basic info about nics, macaddresses, ip addresses
    -   in a nice structured obj (check what has been done in current agent)
-   get_mem_info (implemented)
-   get_disk_info (implemented)
    -   disks & partitions found
    -   need to work on windows & linux (try to be generic)
-   get_cpu_info (implemented)
    -   cpu/cores
    -   need to work on windows & linux (try to be generic)
    -   ?
-   get_os_info (implemented)
    -   hostname
    -   ostype
    -   need to work on windows & linux (try to be generic)
    -   ?
    
AC daemon
---------
-   a golang process with co-routines
-   runs webservice & only webservice on certain port
-   all data is stored in L/Redis
-   cmds also come from Redis
-   this is done to get to ultimate scalability
-   start as
    -   jsacc –p 4000 –r localhost:3456
        -   \-p is port to use to expose webservice
        -   \-r is redis to connect to
-   cmds (from 2 queues)
    -   ac reads from cmds_queue in redis
    -   result stored in cmds_queue_$jobid in redis
    -   this makes it very easy for any client to send a cmd and wait for return by polling the cmds_queue_$jobid
-   logs/messages in hlist?
    -   in log.$domain.$name

Python Client
-------------

-   need python client which talks to redis and makes it easy to execute commands as specified above
