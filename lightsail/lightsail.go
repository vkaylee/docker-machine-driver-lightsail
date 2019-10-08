package lightsail

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lightsail"
	"net"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/engine"
	"github.com/docker/machine/libmachine/log"
	"github.com/docker/machine/libmachine/mcnflag"
	"github.com/docker/machine/libmachine/mcnutils"
	"github.com/docker/machine/libmachine/state"
)

type Driver struct {
	*drivers.BaseDriver
	awsCredentialsFactory func() awsCredentials
	EnginePort      int
	SSHKey          string
	AccessKey       string
	SecretKey       string
	SessionToken    string
}
const (
	defaultTimeout = 15 * time.Second
	defaultSSHUser = "admin"
	driverName = "lightsail"
	defaultAvailabilityZone = "a"
	defaultRegion = "ap-northeast-1"
	defaultBlueprintId = "debian_9_5"
	defaultBundleId = "small_2_0"
	defaultInstanceName = "server"
)
var (
	errorMissingCredentials              = errors.New("lightsail driver requires AWS credentials configured with the --lightsail-access-key and --lightsail-secret-key options, environment variables, ~/.aws/credentials, or an instance role")
)
// GetCreateFlags registers the flags this driver adds to
// "docker hosts create"
func (d *Driver) GetCreateFlags() []mcnflag.Flag {
	return []mcnflag.Flag{
		mcnflag.StringFlag{
			Name:   "lightsail-access-key",
			Usage:  "AWS Access Key",
			EnvVar: "AWS_ACCESS_KEY_ID",
		},
		mcnflag.StringFlag{
			Name:   "lightsail-secret-key",
			Usage:  "AWS Secret Key",
			EnvVar: "AWS_SECRET_ACCESS_KEY",
		},
		mcnflag.StringFlag{
			Name:   "lightsail-session-token",
			Usage:  "AWS Session Token",
			EnvVar: "AWS_SESSION_TOKEN",
		},
		mcnflag.IntFlag{
			Name:   "lightsail-engine-port",
			Usage:  "Docker engine port",
			Value:  engine.DefaultPort,
			EnvVar: "LIGHTSAIL_ENGINE_PORT",
		},
		mcnflag.StringFlag{
			Name:   "lightsail-ip-address",
			Usage:  "IP Address of machine",
			EnvVar: "LIGHTSAIL_IP_ADDRESS",
		},
		mcnflag.StringFlag{
			Name:   "lightsail-ssh-user",
			Usage:  "SSH user",
			Value:  defaultSSHUser,
			EnvVar: "LIGHTSAIL_SSH_USER",
		},
		mcnflag.StringFlag{
			Name:   "lightsail-ssh-key",
			Usage:  "SSH private key path (if not provided, default SSH key will be used)",
			Value:  "",
			EnvVar: "LIGHTSAIL_SSH_KEY",
		},
		mcnflag.IntFlag{
			Name:   "lightsail-ssh-port",
			Usage:  "SSH port",
			Value:  drivers.DefaultSSHPort,
			EnvVar: "LIGHTSAIL_SSH_PORT",
		},
	}
}

// NewDriver creates and returns a new instance of the driver
func NewDriver(hostName, storePath string) drivers.Driver {
	driver := &Driver{
		EnginePort: engine.DefaultPort,
		BaseDriver: &drivers.BaseDriver{
			MachineName: hostName,
			StorePath:   storePath,
		},
	}
	driver.awsCredentialsFactory = driver.buildCredentials
	return driver
}
func (d *Driver) buildCredentials() awsCredentials {
	return NewAWSCredentials(d.AccessKey, d.SecretKey, d.SessionToken)
}
// DriverName returns the name of the driver
func (d *Driver) DriverName() string {
	return driverName
}

func (d *Driver) GetSSHHostname() (string, error) {
	return d.GetIP()
}

func (d *Driver) GetSSHUsername() string {
	return d.SSHUser
}

func (d *Driver) GetSSHKeyPath() string {
	return d.SSHKeyPath
}

func (d *Driver) SetConfigFromFlags(flags drivers.DriverOptions) error {
	d.EnginePort = flags.Int("lightsail-engine-port")
	d.IPAddress = flags.String("lightsail-ip-address")
	d.SSHUser = flags.String("lightsail-ssh-user")
	d.SSHKey = flags.String("lightsail-ssh-key")
	d.SSHPort = flags.Int("lightsail-ssh-port")
	d.AccessKey = flags.String("lightsail-access-key")
	d.SecretKey = flags.String("lightsail-secret-key")
	d.SessionToken = flags.String("lightsail-session-token")

	//if d.IPAddress == "" {
	//	return errors.New("lightsail driver requires the --lightsail-ip-address option")
	//}
	_, err := d.awsCredentialsFactory().Credentials().Get()
	if err != nil {
		return errorMissingCredentials
	}
	return nil
}

func (d *Driver) PreCreateCheck() error {
	if d.SSHKey != "" {
		if _, err := os.Stat(d.SSHKey); os.IsNotExist(err) {
			return fmt.Errorf("SSH key does not exist: %q", d.SSHKey)
		}

		// TODO: validate the key is a valid key
	}

	return nil
}

func (d *Driver) Create() error {
	if d.SSHKey == "" {
		log.Info("No SSH key specified. Assuming an existing key at the default location.")
	} else {
		log.Info("Importing SSH key...")

		d.SSHKeyPath = d.ResolveStorePath(path.Base(d.SSHKey))
		if err := copySSHKey(d.SSHKey, d.SSHKeyPath); err != nil {
			return err
		}

		if err := copySSHKey(d.SSHKey+".pub", d.SSHKeyPath+".pub"); err != nil {
			log.Infof("Couldn't copy SSH public key : %s", err)
		}
	}

	log.Debugf("IP: %s", d.IPAddress)
	if err := d.innerCreate(); err != nil {
		// cleanup partially created resources
		//d.Remove()
		return err
	}
	return nil
}
func (d *Driver) innerCreate() error {
	log.Infof("Launching instance...")
	// Create Session with MaxRetries configuration to be shared by multiple
	// service clients.
	sess := session.Must(session.NewSession(aws.NewConfig().
		WithMaxRetries(3),
	))
	// Create lightsail service client with a specific Region.
	svc := lightsail.New(sess, aws.NewConfig().WithRegion(defaultRegion))
	// Create lightsail instance
	if err := d.createInstances(svc);err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			switch awsErr.Code() {
			case lightsail.ErrCodeInvalidInputException:
				fmt.Println("The instance existed!")
				if err := d.getInstance(svc);err != nil {
					return err
				}
				return nil
			}
		}
		return err
	}
	if err := d.getInstance(svc); err != nil {
		return err
	}
	return nil
}
func (d *Driver) createInstances(lightsailSess *lightsail.Lightsail) error {
	availabilityZone := fmt.Sprintf("%s%s", defaultRegion, defaultAvailabilityZone)
	instanceName := d.MachineName
	var instanceNames []*string
	instanceNames = append(instanceNames, &instanceName)
	var inputCreate lightsail.CreateInstancesInput
	inputCreate.AvailabilityZone = &availabilityZone
	inputCreate.SetBlueprintId(defaultBlueprintId)
	inputCreate.SetBundleId(defaultBundleId)
	inputCreate.SetInstanceNames(instanceNames)
	_, err := lightsailSess.CreateInstances(&inputCreate)
	return err
}
//func (d *Driver) instanceIsRunning() bool {
//	st, err := d.GetState()
//	if err != nil {
//		log.Debug(err)
//	}
//	if st == state.Running {
//		return true
//	}
//	return false
//}
//func (d *Driver) waitForInstance() error {
//	if err := mcnutils.WaitFor(d.instanceIsRunning); err != nil {
//		return err
//	}
//	return nil
//}
func (d *Driver) getInstance(lightsailSess *lightsail.Lightsail) error {
	instanceName := d.MachineName
	var instanceInput lightsail.GetInstanceInput
	instanceInput.SetInstanceName(instanceName)
	result, err := lightsailSess.GetInstance(&instanceInput)
	fmt.Println(result)
	return err
}
func (d *Driver) GetURL() (string, error) {
	if err := drivers.MustBeRunning(d); err != nil {
		return "", err
	}

	ip, err := d.GetIP()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("tcp://%s", net.JoinHostPort(ip, strconv.Itoa(d.EnginePort))), nil
}

func (d *Driver) GetState() (state.State, error) {
	address := net.JoinHostPort(d.IPAddress, strconv.Itoa(d.SSHPort))

	_, err := net.DialTimeout("tcp", address, defaultTimeout)
	if err != nil {
		return state.Stopped, nil
	}

	return state.Running, nil
}

func (d *Driver) Start() error {
	return errors.New("lightsail driver does not support start")
}

func (d *Driver) Stop() error {
	return errors.New("lightsail driver does not support stop")
}

func (d *Driver) Restart() error {
	_, err := drivers.RunSSHCommandFromDriver(d, "sudo shutdown -r now")
	return err
}

func (d *Driver) Kill() error {
	return errors.New("lightsail driver does not support kill")
}

func (d *Driver) Remove() error {
	return nil
}

func copySSHKey(src, dst string) error {
	if err := mcnutils.CopyFile(src, dst); err != nil {
		return fmt.Errorf("unable to copy ssh key: %s", err)
	}

	if err := os.Chmod(dst, 0600); err != nil {
		return fmt.Errorf("unable to set permissions on the ssh key: %s", err)
	}

	return nil
}