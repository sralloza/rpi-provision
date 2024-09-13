# Raspberry Provisioner

_Setup your Raspberry Pi with Raspbian without a screen, keyboard or ethernet connection._

**Best features:**

- Setup your Raspberry Pi without a screen, keyboard or ethernet connection (WiFi connection required).
- The router will assign a random IP address to the Raspberry Pi (DHCP protocol). You don't have to use nmap or open your router's admin panel to find the IP address, this command will find it.
- Improve the security of your Raspberry Pi by creating a new user, disabling the default user and forcing the use of SSH keys.
- Manage the authorized_keys file in your Raspberry Pi. You can use a local file, a file in S3 or a URL.
- Setup a static IP address for your Raspberry Pi.
- Install zsh and oh-my-zsh with some useful plugins.
- Install tailscale to access your raspberry pi from anywhere.
- Install docker and docker-compose to facilitate the deployment of your applications.

Tested with:

- [`2023-12-11-raspios-bookworm-armhf-lite.img.xz`](https://downloads.raspberrypi.com/raspios_lite_armhf/images/raspios_lite_armhf-2023-12-11/2023-12-11-raspios-bookworm-armhf-lite.img.xz)
- [`2024-07-04-raspios-bookworm-arm64-lite`](https://downloads.raspberrypi.com/raspios_lite_arm64/images/raspios_lite_arm64-2024-07-04/2024-07-04-raspios-bookworm-arm64-lite.img.xz)

**Index:**

- [Raspberry Provisioner](#raspberry-provisioner)
  - [Install](#install)
  - [Quick start](#quick-start)
  - [SSH Access \& Tailscale](#ssh-access--tailscale)
    - [Use tilescale only as VPN](#use-tilescale-only-as-vpn)
    - [Use tilescale as VPN and SSH Proxy](#use-tilescale-as-vpn-and-ssh-proxy)
  - [Commands](#commands)
    - [boot](#boot)
    - [find](#find)
    - [layer1](#layer1)
    - [layer2](#layer2)
    - [authorized-keys](#authorized-keys)
    - [network](#network)

## Install

Download the latest version from the [releases page](https://github.com/sralloza/rpi-provisioner/releases) for your OS and architecture. Then, move it to a directory in your PATH.

## Quick start

Instructions to quickly have your raspberry pi up and running:

1. Flash ISO in memory card. Use a tool like [balenaEtcher](https://www.balena.io/etcher/) to do it.
2. Create pi user, enable ssh and setup wifi connection in the SD card -> use the [boot](#boot) command
3. Insert the SD card in the raspberry and turn it on. Wait a couple minutes, as the first boot takes a while.
4. Find the raspberry's IP address -> use the [find](#find) command
5. If you don't have a ssh key, create one with `ssh-keygen -t rsa`
6. Create the authorized_keys file -> more info about its format in the [authorized-keys](#authorized-keys) docs
7. Create deployer user, configure SSH, disable pi login and setup static IP -> use the [layer1](#layer1) command
8. Update and upgrade packages, install some libraries, zsh, tailscale and docker -> use the [layer2](#layer2) command

## SSH Access & Tailscale

Even if you are unable to open the port 22 in your router, you can still access your raspberry via ssh. You just need to setup tilescale. There are two ways to do it:

1. Use tilescale only as VPN
2. Use tilescale as VPN and SSH Proxy

Both ways need you to have a tilescale account and have it installed in your PC. To install it in your raspberry, just use the `rpi-provisioner layer2` command.

### Use tilescale only as VPN

Tailscale will create a virtual network interface in your PC. You can ssh into your raspberry using this interface. You can also use it to access your raspberry's services (like a web server) using the tailscale's IP address (or the alias) assigned to the raspberry.

In this case, the raspberry will manage the SSH access. You will need to add your public ssh key to the raspberry's authorized_keys file. You can do it manually or using the [authorized-keys](#authorized-keys) command.

### Use tilescale as VPN and SSH Proxy

The only difference with the previous option is that when starting a SSH connection using the tailscale's IP address, tailscale will manage the access. This means you can ssh from a PC that doesn't have your public ssh key but needs to have tailscale installed. You can even ssh from a web browser using the [tailscale's web interface](https://login.tailscale.com/admin/machines).

Keep in mind that when using the raspberry's real IP address, tailscale will not manage the access, so you will need to add your public ssh key to the raspberry's authorized_keys file.

If you want to use this option, you will to start the tailscale daemon in your raspberry enabling ssh access:

```shell
sudo tailscale up --ssh --accept-risk=lose-ssh
```

Keep in mind that running this command will close your current SSH connection.

## Commands

The commands are sorted by the order you will probably use them. Some functionality was seen to be useful outside its command, so it was extracted to a separate command (like [network](#network) or [authorized-keys](#authorized-keys)), which are at the end.

Each command has its own examples to show how to use it. For more information, use the `--help` flag in any command.

### boot

After flashing the raspbian ISO into the SD card, you must do some stuff before you can insert it into the raspberry.

The boot command will:

- Enable ssh connections, as Raspbian doesn't enable ssh connection.
- Make the raspberry create the user `pi` with password `raspberry` during the first boot, as Raspbian doesn't add a default user.
- Setup the WiFi connection (optional), so you can still use the raspberry in headless mode even if you don't have an ethernet connection.
- Setup the raspberry hostname.

Example:

```shell
# In MacOS
$ rpi-provisioner boot --wifi-ssid MOVISTAR_34XC --wifi-pass '7074Lly/R4nD0M' --hostname 'rpi-provisioner-example' /Volumes/bootfs

# In Windows
$ rpi-provisioner boot --wifi-ssid MOVISTAR_34XC --wifi-pass '7074Lly/R4nD0M' --hostname 'rpi-provisioner-example' E:/
```

**Note: this command can only be executed one time - before the first boot. If you want to connect your raspberry to another interface, use the raspi-config command.**

### find

This command will find your raspberry pi in your local network. It will try to connect to each host in your local network using SSH. If it is able to connect, it will print the host's IP address.

Examples:

```shell
# After executing the boot command, the user 'pi' will be created with password 'raspberry' (default values for the this command)
$ rpi-provisioner find

# You can use this command after changing the user and limiting the access to use ssh keys
$ rpi-provisioner find --user $USER --ssh-key

# If for some reason you want to login with a different user and password (not the ssh key):
$ rpi-provisioner find --user $USER --password $PASSWORD
```

More useful info:

- `--subnet`: this is the most important flag. You won't probably use it, but with this flag you can specify your local network's IP. If you left this blank, the program will try to generate it from your local IP address. If it is wrong, use this flag to really find your raspberry pi in your local network (and open an issue so it can be fixed).
- `--live`: By default when you start the analysis, the valid raspberry's IP will only be shown at the end. You can use this flag to see as soon as it is discovered.
- `--port`: just in case the default SSH port is not 22, use this flag to set it right.
- `--timeout`: Timeout in nanoseconds to wait in SSH connections. It is directly passed to the SSH Dial method. To be fair I don't really know if this works, so don't use it. By default is 1, but I don't know if it affects performance. If you know more about this flag, feel free to open an issue or a PR correcting the documentation.

### layer1

The layer1 command will set up the _infrastructure_ or your raspberry pi (meaning only configuration, no libraries or programs).

It will:

- Create the deployer user (the user you will use to ssh into the raspberry)
- Disable login with the pi user
- Setup the ssh connection (add the ssh keys and disable any password login). For more information about the --keys-uri option, refer to the [authorized-keys](#authorized-keys) command.
- Set up the static IP address (optional). For more information about the --primary-ip and the --secondary-ip options, refer to the [network](#network) command.

Examples:

```shell
# Create the deployer user 'deployer' with password 'p422w0rD', update the authorized_keys and set the primary interface's IP address to 192.172.0.71 (the router assigned the raspberry initially the IP address 192.168.0.144 using DCHP)
$ rpi-provisioner layer1 --deployer-user deployer --deployer-password p422w0rD --host 192.168.0.144 --keys-uri=/path/to/public-ssh-keys.json --ip 192.168.0.71
```

**Important: make sure that the authorized-keys file includes your public ssh key, otherwise you will lose SSH access to the raspberry.**

**Note: this command is designed to be executed only once. It uses the login with user:password but it disables the password login, so the second time it's executed it will return an error during the connection. If you wish to setup the static IP address again please refer to the [network](#network) command.**

### layer2

The layer2 command will install some useful libraries and programs. It will:

- Update and upgrade packages
- Install some useful libraries
- Install zsh
- Install and configure oh-my-zsh
- Install some useful oh-my-zsh plugins
- Install tailscale (and optionally configure it with a pregenerated auth key)
- Install docker and docker compose v2

By default (without the option --ts-auth-key) the layer2 command will just install tailscale, showing a message at the end with more instructions about how to configure it. If you want to configure tailscale to manage SSH access you must not use the --ts-auth-key option, but follow the instructions after the command finishes.

```shell
# Run the layer2 command in the host 192.168.0.71 using the user 'deployer' and the ssh key
$ rpi-provisioner layer2 --host 192.168.0.71 --user deployer

# Run the layer2 command configuring tailscale with a pregenerated auth key
# You can generate the ssh-key from https://login.tailscale.com/admin/settings/keys
$ rpi-provisioner layer2 --host 192.168.0.71 --user deployer --ts-auth-key s0m3-rand0m-7a1lscal3-k3y
```

You can run this command as many times as you want. It will always update the packages and install the libraries and programs. Tailscale will only be setup once.

### authorized-keys

This command is used to update the authorized_keys file in the raspberry. It will join the current authorized_keys file with the keys in the file specified in the `--keys-uri` flag.

The format of the file must be like this:

```json
[
  {
    "alias": "name-of-the-key",
    "type": "ssh-rsa",
    "key": "the-public-key"
  }
]
```

In this example, the key will be added in the authorized_keys as expected:

```text
ssh-rsa the-public-key name-of-the-key
```

The `--keys-uri` flag supports three formats:

- As a local file: `--keys-uri=/path/to/public-ssh-keys.json`
- As a S3 file: `--keys-uri=s3://<REGION>/<BUCKET_NAME>/<FILE_NAME>`
- As a URL: `--keys-uri=https://example.com/public-ssh-keys.json`

The recommended way is uploading the file to [Google Drive](https://drive.google.com) and getting the shareable link. Example: `--keys-uri https://drive.google.com/file/d/sfw3jirjoisdvx89werjkf/view?usp=sharing`. In reality this URL does not return the file exactly, but this command is smart enough to detect the Google Drive URL and process it correctly.

**Make sure that the file includes your public ssh key, otherwise you will lose SSH access to the raspberry.**

```shell
# After launching the layer1 command, update the authorized_keys file (using the ssh key to login)
$ rpi-provisioner authorized-keys --host 192.168.0.33 --keys-uri https://drive.google.com/... --ssh-key --user deployer

# You can also update the authorized_keys for the pi user before the layer1 command
$ rpi-provisioner authorized-keys --host 192.168.0.33 --keys-uri https://drive.google.com/... --user pi --password raspberry
```

### network

This command will add the new static IP addresses and then delete the old ones. You may be asked to restart the server to fully remove the old IP addresses. In that case, please run the network command again after restart to be sure that the old IP addresses are deleted.

This commands configures the same IP Adress for all the network interfaces of the Raspberry Pi (eth0 and wlan0), but it gives priority to eth0. This means that if the ethernet cable is connected it will use it, otherwise it will use the wifi connection (both cases with the same IP address).

**Note: you can only set the static IP for eth0 if the ethernet cable is connected.**

**Note: if you want to move the raspberry to another network, is recommended to remove the static IP addresses and let DHCP assign the IP address, because the new network might have a different IP address or your static IP address might be assigned to another device. Future releases of the `rpi-provisioner` command will support this.**
