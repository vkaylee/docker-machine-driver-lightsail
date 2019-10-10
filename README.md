# docker-machine-driver-lightsail
## Let's start
```
docker-machine create -d lightsail machine_name
```
### Option
- Docker engine port: default is 2376
```
--lightsail-engine-port
```
- The path of your ssh key: default is the driver will generate the new SSH key.
```
--lightsail-ssh-key
```
- The SSH Port: default is 22
```
--lightsail-ssh-port
```
- The AWS access key: default is AWS SDK config
```
--lightsail-access-key
```
- The AWS secret key: default is AWS SDK config
```
--lightsail-secret-key
```
- The region: default is "ap-northeast-1"
```
--lightsail-region
```
- The zone: default is "a"
```
--lightsail-availability-zone
```
- The OS of your instance: default is "debian_9_5"
```
--lightsail-blueprint-id
```
- The instance plan of your instance: default is "small_2_0"
```
--lightsail-bundle-id
```