# Systemd service

This folder gathers utilities to install `xdprobe` as a systemd service.
In addition it also provides some test files.

## Installation

The [`install.sh`](./install.sh) is the one-liner installer that:

- downloads the `xdprobe` binary from latest release
- downloads the [`xdprobe.service`](./xdprobe.service) from the the master branch
- generates a configuration to `/etc/sysconfig/xdprobe` (env variables read by the systemd service)
- creates a dedicated user (`xdprobe`)
- sets the right permissions to files and directories
- enables and starts the service

## Testing

Two test suites live in this directory:
- [`test_install.sh`](./test_install.sh) checks that the installation process worked well
- [`test_api.sh`](./test_api.sh) tests the application
  
The [`Vagrantfile`](./Vagrantfile) is for local testing (need to build `xdprobe` locally first).

To start, provision and run the tests before halting the box:
```shell
vagrant up --provision && vagrant halt
```