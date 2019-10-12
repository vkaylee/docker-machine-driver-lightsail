#!/usr/bin/env bash
if [ "$(command -v go)" ]; then
  mkdir release
  # build app for darwin amd64
  env GOOS=darwin GOARCH=amd64 go build
  # tar it
  tar -cjf release/docker-machine-driver-lightsail-darwin-amd64.tar docker-machine-driver-lightsail
  # copy to local
  cp docker-machine-driver-lightsail /usr/local/bin/
  # Make it can be excuted
  chmod +x /usr/local/bin/docker-machine-driver-lightsail
  # Remove it
  rm docker-machine-driver-lightsail

  # Build app for linux amd64
  env GOOS=linux GOARCH=amd64 go build
  # Tar it
  tar -cjf release/docker-machine-driver-lightsail-linux-amd64.tar docker-machine-driver-lightsail
  # Remove it
  rm docker-machine-driver-lightsail
fi

