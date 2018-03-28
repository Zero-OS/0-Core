# Configuration
Once the kernel image is built, there is no way to customize the config file. Config file
controlls the zos (initial) state, once the machine is booted, you can change the state
of the operating system using the API (zos client).

To change the initial state of the system, you will need to change the configuration _BEFORE_ you
build the kernel image.

## Configuration files

* [Main Configuration](main.md)
* [Network Configuration](network.md)
* [Other Configurations](other.md)
* [Startup Services](startup.md)
