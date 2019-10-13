package lightsail

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lightsail"
	"github.com/docker/machine/libmachine/ssh"
	"io/ioutil"
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
	clientFactory         func() *lightsail.Lightsail
	awsCredentialsFactory func() awsCredentials
	EnginePort          int
	SSHPrivateKey       string
	KeyPairName         string
	AwsAccessKey        string
	AwsSecretKey        string
	AwsSessionToken     string
	Region              string
	BundleId            string
	BlueprintId         string
	AvailabilityZone    string
}
const (
	defaultTimeout              =   15 * time.Second
	driverName                  =   "lightsail"
	defaultAvailabilityZone     =   "a"
	defaultRegion               =   "ap-northeast-1"
	defaultBlueprintId          =   "ubuntu_18_04"
	defaultBundleId             =   "small_2_0"
)
var (
	dockerPort  int64   = 2376
	swarmPort   int64   = 3376
	errorMissingCredentials = errors.New("lightsail driver requires AWS credentials configured with the --lightsail-access-key and --lightsail-secret-key options, environment variables, ~/.aws/credentials, or an instance role")
	errorZoneNameUnavailable = errors.New("Current zone is not available, please choose an another zone ")
	errorBundleIdIsUnavailable = errors.New("Your bundleId is unactive or wrong, please check the --lightsail-bundle-id")
	errorBlueprintIsUnavailable = errors.New("Your blueprintId is unactive or wrong, please check the --lightsail-blueprint-id")
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
			Name:   "lightsail-ssh-key",
			Usage:  "SSH private key path (if not provided, default SSH key will be used)",
			Value:  "",
			EnvVar: "LIGHTSAIL_SSH_KEY",
		},
		mcnflag.StringFlag{
			Name:   "lightsail-region",
			Usage:  "Lightsail Region",
			Value:  defaultRegion,
			EnvVar: "LIGHTSAIL_REGION",
		},
		mcnflag.StringFlag{
			Name:   "lightsail-availability-zone",
			Usage:  "Lightsail AvailabilityZone",
			Value:  defaultAvailabilityZone,
			EnvVar: "LIGHTSAIL_AVAILABILITY_ZONE",
		},
		mcnflag.StringFlag{
			Name:   "lightsail-blueprint-id",
			Usage:  "Lightsail BlueprintId",
			Value:  defaultBlueprintId,
			EnvVar: "LIGHTSAIL_BLUEPRINT_ID",
		},
		mcnflag.StringFlag{
			Name:   "lightsail-bundle-id",
			Usage:  "Lightsail BundleId",
			Value:  defaultBundleId,
			EnvVar: "LIGHTSAIL_BUNDLE_ID",
		},
	}
}
func (d *Driver) SetConfigFromFlags(flags drivers.DriverOptions) error {
	d.SSHPrivateKey = flags.String("lightsail-ssh-key")
	d.AwsAccessKey = flags.String("lightsail-access-key")
	d.AwsSecretKey = flags.String("lightsail-secret-key")
	d.Region = flags.String("lightsail-region")
	d.AvailabilityZone = flags.String("lightsail-availability-zone")
	d.BlueprintId = flags.String("lightsail-blueprint-id")
	d.BundleId = flags.String("lightsail-bundle-id")

	if _, err := d.awsCredentialsFactory().Credentials().Get();err != nil {
		return errorMissingCredentials
	}
	// Check lightsail-region and lightsail-availability-zone input
	var regionsInput lightsail.GetRegionsInput
	regionsInput.SetIncludeAvailabilityZones(true)
	regionsOutput, err := d.getClient().GetRegions(&regionsInput)
	if err != nil {
		return err
	}
	var regionZonePass bool = false
	for _, v1 := range regionsOutput.Regions {
		for _, v2 := range v1.AvailabilityZones {
			if fmt.Sprintf("%s%s", d.Region, d.AvailabilityZone) == *v2.ZoneName && "available" == *v2.State {
				regionZonePass = true
			}
		}
	}
	if regionZonePass == false {
		return errorZoneNameUnavailable
	}
	// Check the valid of lightsail-bundle-id
	bundlesOutput, err := d.getClient().GetBundles(&lightsail.GetBundlesInput{})
	if err != nil {
		return err
	}
	var bundleIdPass bool = false
	for _, v := range bundlesOutput.Bundles {
		if d.BundleId == *v.BundleId && true == *v.IsActive {
			bundleIdPass = true
		}
	}
	if bundleIdPass == false {
		return errorBundleIdIsUnavailable
	}
	// Check the valid of lightsail-blueprint-id
	blueprintsOutput, err := d.getClient().GetBlueprints(&lightsail.GetBlueprintsInput{})
	if err != nil {
		return err
	}
	var blueprintPass bool = false
	for _, v := range blueprintsOutput.Blueprints {
		if d.BlueprintId == *v.BlueprintId && true == *v.IsActive {
			blueprintPass = true
		}
	}
	if blueprintPass == false {
		return errorBlueprintIsUnavailable
	}
	return nil
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
	driver.clientFactory = driver.buildClient
	driver.awsCredentialsFactory = driver.buildCredentials
	return driver
}
func (d *Driver) buildClient() *lightsail.Lightsail {
	config := aws.NewConfig()
	config = config.WithRegion(d.Region)
	config = config.WithCredentials(d.awsCredentialsFactory().Credentials())
	config = config.WithMaxRetries(3)
	return lightsail.New(session.Must(session.NewSession(config)))
}
func (d *Driver) buildCredentials() awsCredentials {
	return NewAWSCredentials(d.AwsAccessKey, d.AwsSecretKey, d.AwsSessionToken)
}
func (d *Driver) getClient() *lightsail.Lightsail {
	return d.clientFactory()
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

func (d *Driver) GetSSHPrivateKeyPath() string {
	return d.SSHKeyPath
}

func (d *Driver) PreCreateCheck() error {
	if d.SSHPrivateKey != "" {
		if _, err := os.Stat(d.SSHPrivateKey); os.IsNotExist(err) {
			return fmt.Errorf("SSH key does not exist: %q", d.SSHPrivateKey)
		}
		// TODO: validate the key is a valid key
	}
	return nil
}

func (d *Driver) Create() error {
	// Process SSH Key first
	if err := d.processSSHKey(); err != nil {
		return err
	}
	// Import key pair to lightsail
	if err := d.importKeyPairToLightsail(); err != nil {
		return err
	}
	log.Debugf("IP: %s", d.IPAddress)
	if err := d.innerCreate();err != nil {
		// cleanup partially created resources
		d.Remove()
		return err
	}
	return nil
}
func (d *Driver) importKeyPairToLightsail() error {
	// Set KeyPairName
	d.KeyPairName = "docker_machine_" + d.MachineName
	// Check KeyPairName in lightsail
	var keyPairInput lightsail.GetKeyPairInput
	keyPairInput.SetKeyPairName(d.KeyPairName)
	currentKeyPair, err := d.getClient().GetKeyPair(&keyPairInput)
	var removeKeyPairBool bool = true
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == lightsail.ErrCodeNotFoundException {
				removeKeyPairBool = false
			} else {
				return err
			}
		}
	}
	if removeKeyPairBool == true {
		if *currentKeyPair.KeyPair.Name == d.KeyPairName && *currentKeyPair.KeyPair.Location.RegionName == d.Region {
			// Remove lightsail keypair
			if err := d.removeLightsailKeyPair(&d.KeyPairName); err != nil {
				return err
			}
		}
	}

	publicKey, err := ioutil.ReadFile(d.SSHKeyPath + ".pub")
	if err != nil {
		return err
	}
	var input lightsail.ImportKeyPairInput
	input.SetKeyPairName(d.KeyPairName)
	input.SetPublicKeyBase64(string(publicKey))
	if _, err := d.getClient().ImportKeyPair(&input);err != nil {
		return err
	}
	return nil
}
func (d *Driver) processSSHKey() error {
	if d.SSHPrivateKey == "" {
		d.SSHKeyPath = d.GetSSHKeyPath()
		log.Info("No SSH key specified. Creating new SSH Key")
		if err := ssh.GenerateSSHKey(d.SSHKeyPath); err != nil {
			return err
		}
	} else {
		log.Info("Importing SSH key in argv to system key...")
		d.SSHKeyPath = d.ResolveStorePath(path.Base(d.SSHPrivateKey))
		if err := copySSHPrivateKey(d.SSHPrivateKey, d.SSHKeyPath); err != nil {
			return err
		}
		if err := copySSHPrivateKey(d.SSHPrivateKey+".pub", d.SSHKeyPath+".pub"); err != nil {
			log.Infof("Couldn't copy SSH public key : %s", err)
			return err
		}
	}
	return nil
}
func (d *Driver) innerCreate() error {
	// Create lightsail instance
	if err := d.createInstances();err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			switch awsErr.Code() {
			case lightsail.ErrCodeInvalidInputException:
				fmt.Println("The instance existed!")
				if _, err := d.getLightsailInstanceInfo();err != nil {
					return err
				}
				return nil
			}
		}
		return err
	}
	// Wait for the instance has a status "running"
	if err := d.waitForLightsailInstance(); err != nil {
		return err
	}
	// Get the info of instance
	result, err := d.getLightsailInstanceInfo()
	if err != nil {
		return err
	}
	// Set SSHUser
	d.SSHUser = *result.Instance.Username
	// Set IPAddress
	d.IPAddress = *result.Instance.PublicIpAddress
	// Open ports in lightsail instance
	if err := d.openPortsInLightsailInstance(); err != nil {
		return err
	}
	return nil
}
func (d *Driver) openPortsInLightsailInstance() error {
	log.Infof("Opening port in lightsail instance...")
	var openInstancePublicPorts lightsail.OpenInstancePublicPortsInput
	openInstancePublicPorts.SetInstanceName(d.MachineName)
	var portInfo lightsail.PortInfo
	portInfo.SetFromPort(0)
	portInfo.SetToPort(65535)
	protocol := "tcp" // tcp, udp, all
	portInfo.SetProtocol(protocol)
	openInstancePublicPorts.SetPortInfo(&portInfo)
	_, err := d.getClient().OpenInstancePublicPorts(&openInstancePublicPorts)
	return err
}
func (d *Driver) createInstances() error {
	log.Infof("Launching lightsail instance...")
	availabilityZone := fmt.Sprintf("%s%s", d.Region, d.AvailabilityZone)
	instanceName := d.MachineName
	var instanceNames []*string
	instanceNames = append(instanceNames, &instanceName)
	var inputCreate lightsail.CreateInstancesInput
	inputCreate.AvailabilityZone = &availabilityZone
	inputCreate.SetBlueprintId(d.BlueprintId)
	inputCreate.SetBundleId(d.BundleId)
	inputCreate.SetInstanceNames(instanceNames)
	inputCreate.SetKeyPairName(d.KeyPairName)
	_, err := d.getClient().CreateInstances(&inputCreate)
	return err
}
func (d *Driver) checkLightsailInstanceIsRunning() bool {
	// Call AWS SDK
	result, err := d.getInstanceState()
	if err != nil {
		log.Debug(err)
		return false
	}
	// the instance is running if the state code == 16
	if *result.State.Code == 16 {
		return true
	}
	return false
}
func (d *Driver) waitForLightsailInstance() error {
	if err := mcnutils.WaitFor(d.checkLightsailInstanceIsRunning); err != nil {
		return err
	}
	return nil
}
func (d *Driver) getInstanceState() (*lightsail.GetInstanceStateOutput, error) {
	instanceName := d.MachineName
	var instanceInput lightsail.GetInstanceStateInput
	instanceInput.SetInstanceName(instanceName)
	result, err := d.getClient().GetInstanceState(&instanceInput)
	return result, err
}
func (d *Driver) getLightsailInstanceInfo() (*lightsail.GetInstanceOutput, error) {
	log.Infof("Getting the info of lightsail instance...")
	instanceName := d.MachineName
	var instanceInput lightsail.GetInstanceInput
	instanceInput.SetInstanceName(instanceName)
	result, err := d.getClient().GetInstance(&instanceInput)
	return result, err
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
	var startInstanceInput lightsail.StartInstanceInput
	startInstanceInput.SetInstanceName(d.MachineName)
	_, err := d.getClient().StartInstance(&startInstanceInput)
	if err != nil {
		return err
	}
	return d.waitForLightsailInstance()
}

func (d *Driver) Stop() error {
	var stopInstanceInput lightsail.StopInstanceInput
	stopInstanceInput.SetInstanceName(d.MachineName)
	_, err := d.getClient().StopInstance(&stopInstanceInput)
	return err
}

func (d *Driver) Restart() error {
	var rebootInstanceInput lightsail.RebootInstanceInput
	rebootInstanceInput.SetInstanceName(d.MachineName)
	_, err := d.getClient().RebootInstance(&rebootInstanceInput)
	return err
}

func (d *Driver) Kill() error {
	return errors.New("lightsail driver does not support kill")
}

func (d *Driver) Remove() error {
	// Get info of current instance
	currentInstance, err := d.getLightsailInstanceInfo();
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == lightsail.ErrCodeNotFoundException {
				return nil
			}
		}
		return err
	}
	// Delete lightsail instance
	if err := d.deleteLightsailInstance(); err != nil {
		return err
	}
	d.KeyPairName = *currentInstance.Instance.SshKeyName
	// Remove lightsail keypair
	if err := d.removeLightsailKeyPair(&d.KeyPairName); err != nil {
		return err
	}
	return nil
}
func (d *Driver) removeLightsailKeyPair(name *string) error {
	var input lightsail.DeleteKeyPairInput
	input.SetKeyPairName(*name)
	_, err := d.getClient().DeleteKeyPair(&input)
	if err != nil {
		log.Infof("We got an error when deleting the lightsail key pair.")
		return err
	}
	return nil
}
func (d *Driver) deleteLightsailInstance() error {
	var input lightsail.DeleteInstanceInput
	input.SetForceDeleteAddOns(true)
	input.SetInstanceName(d.MachineName)
	_, err := d.getClient().DeleteInstance(&input)
	if err != nil {
		log.Infof("We got an error when deleting the lightsail instance.")
		return err
	}
	return nil
}

func copySSHPrivateKey(src, dst string) error {
	if err := mcnutils.CopyFile(src, dst); err != nil {
		return fmt.Errorf("unable to copy ssh key: %s", err)
	}

	if err := os.Chmod(dst, 0600); err != nil {
		return fmt.Errorf("unable to set permissions on the ssh key: %s", err)
	}

	return nil
}
func (d *Driver) publicSSHKeyPath() string {
	return d.SSHKeyPath + ".pub"
}