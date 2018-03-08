
ideas
=======
- see if we can use https://syncthing.net/ as sync backend for jsagent
- see if we can use security also for jsagent

syncing
=======
- define a ac master sync dir
  - subdir: jumpscripts -> will be synced to every agent to /opt/jsagent/jumpscripts (1WAY)
  - subdir per agent: agents/$agentid
- each agent has a directory e.g. /opt/jsagent/sync/
  - this sync dir will be sycned to all agent controllers to agents/$agentid
  
jsagent new cmds
================
@todo complete

- createSyncFolder($name,$path)
- enableSync...


