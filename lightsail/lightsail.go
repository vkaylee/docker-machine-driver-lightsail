package lightsail

import "github.com/docker/machine/libmachine/drivers"

// Driver is the implementation of BaseDriver interface
type Driver struct {
	*drivers.BaseDriver

	APIToken         string
	UserAgentPrefix  string
	IPAddress        string
	PrivateIPAddress string
	CreatePrivateIP  bool
	DockerPort       int

	InstanceID    int
	InstanceLabel string

	Region          string
	InstanceType    string
	RootPassword    string
	AuthorizedUsers string
	SSHPort         int
	InstanceImage   string
	SwapSize        int

	StackScriptID    int
	StackScriptUser  string
	StackScriptLabel string
	StackScriptData  map[string]string

	Tags string
}

const (
	defaultSSHPort       = 22
	defaultSSHUser       = "root"
	defaultInstanceImage = "linode/ubuntu18.04"
	defaultRegion        = "us-east"
	defaultInstanceType  = "g6-standard-4"
	defaultSwapSize      = 512
	defaultDockerPort    = 2376

	defaultContainerLinuxSSHUser = "core"
)
// NewDriver creates and returns a new instance of the Linode driver
func NewDriver(hostName, storePath string) *Driver {
	return &Driver{
		InstanceImage: defaultInstanceImage,
		InstanceType:  defaultInstanceType,
		Region:        defaultRegion,
		SwapSize:      defaultSwapSize,
		BaseDriver: &drivers.BaseDriver{
			MachineName: hostName,
			StorePath:   storePath,
		},
	}
}
