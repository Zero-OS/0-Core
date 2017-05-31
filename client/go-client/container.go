package client

type NetworkConfig struct {
	Dhcp    bool     `json:"dhcp"`
	CIDR    string   `json:"cidr"`
	Gateway string   `json:"gateway"`
	DNS     []string `json:"dns"`
}

type Nic struct {
	Type      string        `json:"type"`
	ID        string        `json:"id"`
	HWAddress string        `json:"hwaddr"`
	Config    NetworkConfig `json:"config"`
}

type ContainerCreateAguments struct {
	Root        string            `json:"root"`         //Root plist
	Mount       map[string]string `json:"mount"`        //data disk mounts.
	HostNetwork bool              `json:"host_network"` //share host networking stack
	Nics        []Nic             `json:"nics"`         //network setup (only respected if HostNetwork is false)
	Port        map[int]int       `json:"port"`         //port forwards (only if default networking is enabled)
	Hostname    string            `json:"hostname"`     //hostname
	Storage     string            `json:"storage"`      //ardb storage needed for g8ufs mounts.
	Tags        []string          `json:"tags"`         //for searching containers
}

type ContainerInfo struct {
	Arguments ContainerCreateAguments `json:"arguments"`
	Pid       int                     `json:"pid"`
	Root      string                  `json:"root"`
}

type ContainerResult struct {
	Container ContainerInfo `json:"container"`
	CPU       int           `json:"cpu"`
	Swap      int           `json:"swap"`
	RSS       int           `json:"ress"`
}

type ContainerManager interface {
	Client(id int) Client
	List() (map[int16]ContainerResult, error)
}

type containerMgr struct {
	client Client
}

func Container(cl Client) ContainerManager {
	return &containerMgr{cl}
}

func (c *containerMgr) Client(id int) Client {
	return &containerClient{
		Client: c.client,
		id:     id,
	}
}

func (c *containerMgr) List() (map[int16]ContainerResult, error) {
	r, err := sync(c.client, "corex.list", A{})
	if err != nil {
		return nil, err
	}
	result := make(map[int16]ContainerResult)
	r.Json(&result)
	return result, nil
}

type containerClient struct {
	Client
	id int
}

func (cl *containerClient) Raw(command string, args A, opts ...Option) (JobId, error) {
	cmd := &Command{
		Command:   command,
		Arguments: args,
	}

	for _, opt := range opts {
		opt.apply(cmd)
	}

	result, err := sync(cl.Client, "corex.dispatch", A{
		"container": cl.id,
		"command":   cmd,
	})

	if err != nil {
		return JobId(""), err
	}

	var job JobId
	if err := result.Json(&job); err != nil {
		return job, err
	}

	return job, nil
}
