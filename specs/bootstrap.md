
## boot process

- all is using iPXE to boot the g8OS core (kernel + core0)
    - only 1 parameter required to the iPXE: zerotier network id (16 char's)
- the core0 will try to create network using multiple ways how to get to internet & connect to the zerotier network
    - it will restart as many times as required untill the zerotier network is up & running (ip addr given)
    - @TODO define all possible network connection ways
- the developer or a service now uses the api of zerotier to 
    - enable the node on the zt network (needs to be enabled otherwise no ip addr is give)
    - query all other nodes known to the zt network, check if their redis is accessible, this allows further configuration in step2


## how to boot iPXE

- virtualbox/physical/kvm/... node : use iso with iPXE 
    - there is a service ???/g8os.iso?netid=32423kj4ghi23jk4h
        - result is an iso with ipxe which has configuration params inside for the netid
        - this iso can now be attached to any hypervisor or physical node
        - the iso = ipxe image which will then further download the right g8os.ipxe image to boot from (using http=ipxe)
- packet.net
    - specify as ipxe an url like ```https://stor.jumpscale.org/public/ipxe/g8os.ipxe?netid=23423423423423```
    - the webservice will put netid directly into the ipxe config file
- xhyve osx
    - @TODO to be defined
    
## phase 2 (optional): use ays for further configuration of G8OS

- any AYS development or production environment will do 
    - use https://github.com/Jumpscale/builder to start with, this allows you to develop locally
- connect this env to the right zerotier network
- start ays blueprint
- first AYS = zerotier g8os mgmt network
    - this one detects over api all nodes on the network & try to make connection over redis, if new node found then ays for the node is created
    - monitoring is done that new nodes are auto detected every 5 min (and enabled if required)

remarks:

- there should be no dependencies between G8OS & AYS !!!
