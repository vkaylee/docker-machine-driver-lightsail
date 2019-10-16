# Driver Amazon AWS Lightsail for docker-machine
https://www.terraform.io/docs/providers/aws/r/lightsail_instance.html
## Let's start
```
docker-machine create -d lightsail machine_name
```
### Options
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