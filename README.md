# Driver Amazon AWS Lightsail for docker-machine
## The repo has been archived, Docker has ended active development and support for Docker Machine
https://www.terraform.io/docs/providers/aws/r/lightsail_instance.html
## Install
Access release tab and download it.
### For MacOS (darwin)
- untar it.
- copy docker-machine-driver-lightsail to /usr/local/bin/
- Make it can be excuted by "chmod +x /usr/local/bin/docker-machine-driver-lightsail"

### For Linux
- untar it.
- copy docker-machine-driver-lightsail to /usr/local/bin/
- Make it can be excuted by "chmod +x /usr/local/bin/docker-machine-driver-lightsail"
## Let's start
```
docker-machine create -d lightsail machine_name
```
It will create a machine "machine_name" and a instance with a name like "machine_machine_name_ubuntu_18_04_7e9018ea03ed1668b5fdf6d3aeb43c6d"
## Options
- The path of your ssh key: default is the driver will generate the new SSH key.
```
--lightsail-ssh-key
```
- The AWS access key: default is AWS SDK config
```
--lightsail-aws-access-key
```
- The AWS secret key: default is AWS SDK config
```
--lightsail-aws-secret-key
```
- The region: default is "ap-northeast-1"
```
--lightsail-region
```
- The zone: default is "a"
```
--lightsail-availability-zone
```
- The OS of your instance: default is "ubuntu_18_04"
```
--lightsail-blueprint-id
```
- The instance plan of your instance: default is "small_2_0"
```
--lightsail-bundle-id
```
- The prefix name of your instance: default is "machine_"
```
--lightsail-instance-prefix
```
## For Rancher users
Copy the link of the release.
For examble:
```
https://github.com/vleedev/docker-machine-driver-lightsail/releases/download/1.1.0/docker-machine-driver-lightsail-linux-amd64.tar
```
Add to "Download URL" field of your "Add Node Driver" form.
Keep blank for other fields.
