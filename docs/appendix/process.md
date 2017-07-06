# Zero-OS Boot Process Flow

![Sequence Diagram](http://www.plantuml.com/plantuml/img/NL9DZnCn3BtdL_W84imFMAb8PSMA5Mn158bBBsxYpaJDs2F7NJJyUfpfTa1xYkAyZxoNMBP2i9D4y574gYbEK-O-X32XMevvGZQu5pQLKaW1AyHNPqfjYY5iDl38sJAM_0Sj2yDc4qAlSfbWH_PRz0p7PkCk0U7z1y0x-2gOWCawax44B0O_TOOep1IRHWCwCjwzceF9WUDwiK2b4ZnWBfJyw0PSRHev3N42Ps8f1yvif7p2IDLd1CUvBJUPKeuOpojxMslk6HGvoGZLF5s4n-_W-uFVO8O1DKMlCVaq42Vti6LTqeUkwvPwFZkX3dYcrimYxhds3VUqlG_nnUqJHvsd9UGNcWEB4SXpApyyoSKxfol0tHxsSAdjmMmWK3BDzEpZCyrriM_SrUW7lRHovK2jvOfS73JtWuLVc0rEebxWEBRRmaazClR45hQBeevOG2RIvOrhzy-enVIKkolasmtImluVOc_-Vy0_HK8QNG7Ur3gy0xBe0czNkRy0)

- Bootstrap OS booted from USB or VM (template in cloud)
    - Bootstrap OS v0.9:
        - Kernel
        - 0-core
        - SSH daemon
        - Networking tools (VXLAN, Open vSwitch, ip)
        - Docker / KVM
        - Midnight Commander (mc)
    - Can start from minimal OS distribution (Arch or other)

- 0-core gets started and does the following:
    - Start network
        - Configure using `net.toml`
            - Use `/etc/g8os/net.toml`
            - Check if we can reach at least 1 of the agent controllers (AC's)
        - If none of the AC's can be reached
            - Use DHCP on each interface (do 1 by 1) and check DHCP ok and connection to AC's, stop when connection found
            - Keep other interfaces configured as specified in `net.toml`
        - If still cannot connect to an AC after trying DHCP
            -  Go over each interface configure as
                - 10.254.254.X  X is random chosen until no conflict (address conflict, could be because other node was in same fallback plan)
                - Check AC connection on 10.254.254.254
                - Stop when found
        - If AC still not found, keep on repeating process above (for ever)
    - Mount encrypted filesystem (for config info)
        - Connect to AC over HTTPS
            - Post MAC address
            - Will retrieve an unlock key from AC
        - Use this unlock key to mount
            - `/etc/g8os/private/`
            - Unlock key is encryption key to mount this filesystem (EncFS)
            - Mount as `/mnt/etc`
    - Re-establish connection to AC over SSL
        - From now on use SSL keys to connect to AC
        - Check AC connection still ok with SSL keys
            - If not ok keep on retrying process
    - Start SSH daemon
        - Use SSH keys in `/mnt/etc/ssh/`
        - Now a root can access using authorization info as specified in `/mnt/etc/ssh/`
    - Start g8os_fs
        - Mount sandbox for
            - OS tools
            - JumpScale
            - G8OS binaries
            - OVS

- AYS robot will now manage grid
    - AYS repository created/checked out on controller
    - In there nodes are added
    - Apps are linked to nodes
    - AYS manages installation, process management and monitoring, in fact full application lifecycle management is done by AYS
