/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"strings"

	funk "github.com/thoas/go-funk"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

type Layer2Args struct {
	user string
	host string
	port int
}

func NewLayer2Cmd() *cobra.Command {
	args := Layer2Args{}
	var layer2Cmd = &cobra.Command{
		Use:   "layer2",
		Short: "Provision layer 2",
		Long: `Layer 2 uses the deployer user and bash. It consists of:
		- Update and upgrade packages
		- Install libraries: build-essential, cmake, cron, curl, git, libffi-dev, nano, python3-pip, python3, wget
		- Install fish
		- Install docker
		`,
		RunE: func(cmd *cobra.Command, posArgs []string) error {
			fmt.Println("Provisioning layer 2")
			if err := ProvisionLayer2(args); err != nil {
				return err
			}
			fmt.Println("Layer 2 provisioned")
			return nil
		},
	}

	layer2Cmd.Flags().StringVar(&args.user, "user", "", "Login user")
	layer2Cmd.Flags().StringVar(&args.host, "host", "", "Server host")
	layer2Cmd.Flags().IntVar(&args.port, "port", 22, "Server SSH port")

	layer2Cmd.MarkFlagRequired("user")
	layer2Cmd.MarkFlagRequired("host")

	return layer2Cmd

}

func ProvisionLayer2(args Layer2Args) error {
	address := fmt.Sprintf("%s:%d", args.host, args.port)
	config := &ssh.ClientConfig{
		User:            args.user,
		Auth:            []ssh.AuthMethod{publicKey("~/.ssh/id_rsa")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	conn, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return err
	}
	defer conn.Close()

	fmt.Println("Updating and installing system libraries...")
	if err := InstallLibraries(conn); err != nil {
		return err
	}
	fmt.Println("Libraries updated successfully")

	fmt.Println("Provisioning fish...")
	if installed, err := InstallFish(conn, args); err != nil {
		return err
	} else if installed {
		fmt.Println("fish provisioned successfully")
	} else {
		fmt.Println("fish already provisioned")
	}

	fmt.Println("Provisioning docker...")
	if installed, err := InstallDocker(conn, args); err != nil {
		return err
	} else if installed {
		fmt.Println("docker provisioned successfully")
	} else {
		fmt.Println("docker already provisioned")
	}

	fmt.Println("Provisioning docker-compose...")
	if installed, err := InstallDockerCompose(conn, args); err != nil {
		return err
	} else if installed {
		fmt.Println("docker-compose provisioned successfully")
	} else {
		fmt.Println("docker-compose already provisioned")
	}

	return nil
}

func InstallLibraries(conn *ssh.Client) error {
	_, _, err := runCommand(basicSudoStdin("apt-get update", ""), conn)
	if err != nil {
		return err
	}

	_, _, err = runCommand(basicSudoStdin("apt-get upgrade -y", ""), conn)
	if err != nil {
		return err
	}

	libraries := []string{
		"build-essential",
		"cmake",
		"cron",
		"curl",
		"git",
		"libffi-dev",
		"nano",
		"python3-pip",
		"python3",
		"wget",
	}
	installCmd := fmt.Sprintf("apt-get install %s -y", strings.Join(libraries, " "))
	_, _, err = runCommand(basicSudoStdin(installCmd, ""), conn)
	if err != nil {
		return err
	}

	return nil
}

func InstallFish(conn *ssh.Client, args Layer2Args) (bool, error) {
	_, _, err := runCommand("which fish", conn)
	if err == nil {
		fmt.Println("fish already installed")
		return false, nil
	}
	_, _, err = runCommand("echo 'deb http://download.opensuse.org/repositories/shells:/fish:/release:/3/Debian_10/ /' | sudo tee /etc/apt/sources.list.d/shells:fish:release:3.list", conn)
	if err != nil {
		return false, err
	}

	_, _, err = runCommand("curl -fsSL https://download.opensuse.org/repositories/shells:fish:release:3/Debian_10/Release.key | gpg --dearmor | sudo tee /etc/apt/trusted.gpg.d/shells_fish_release_3.gpg", conn)
	if err != nil {
		return false, err
	}

	_, _, err = runCommand("sudo wget -nv https://download.opensuse.org/repositories/shells:fish:release:3/Debian_10/Release.key -O '/etc/apt/trusted.gpg.d/shells_fish_release_3.asc'", conn)
	if err != nil {
		return false, err
	}

	_, _, err = runCommand(basicSudoStdin("apt update", ""), conn)
	if err != nil {
		return false, err
	}

	_, _, err = runCommand(basicSudoStdin("apt install fish -y", ""), conn)
	if err != nil {
		return false, err
	}

	chshCmd := fmt.Sprintf("chsh -s /usr/bin/fish %s", args.user)
	_, _, err = runCommand(basicSudoStdin(chshCmd, ""), conn)
	if err != nil {
		return false, err
	}

	// # Oh My Fish
	_, _, err = runCommand("curl -L https://get.oh-my.fish > /tmp/omf.sh", conn)
	if err != nil {
		return false, err
	}

	_, _, err = runCommand("fish /tmp/omf.sh --noninteractive", conn)
	if err != nil {
		return false, err
	}

	_, _, err = runCommand("rm /tmp/omf.sh", conn)
	if err != nil {
		return false, err
	}

	_, _, err = runCommand("echo omf install agnoster | fish", conn)
	if err != nil {
		return false, err
	}

	_, _, err = runCommand("echo omf theme agnoster | fish", conn)
	if err != nil {
		return false, err
	}

	_, _, err = runCommand("echo omf install bang-bang | fish", conn)
	if err != nil {
		return false, err
	}

	return true, nil
}

func InstallDocker(conn *ssh.Client, args Layer2Args) (bool, error) {
	_, _, err := runCommand("which docker", conn)
	if err == nil {
		fmt.Println("Docker already installed")
		return false, nil
	}
	_, _, err = runCommand("curl -fsSL https://get.docker.com -o /tmp/get-docker.sh", conn)
	if err != nil {
		return false, err
	}

	_, _, err = runCommand("sudo sh /tmp/get-docker.sh", conn)
	if err != nil {
		return false, err
	}

	_, _, err = runCommand("rm /tmp/get-docker.sh", conn)
	if err != nil {
		return false, err
	}

	_, _, err = runCommand(fmt.Sprintf("sudo usermod -aG docker %s", args.user), conn)
	if err != nil {
		return false, err
	}

	return true, nil
}

func InstallDockerCompose(conn *ssh.Client, args Layer2Args) (bool, error) {
	_, _, err := runCommand("which docker-compose", conn)
	if err == nil {
		return false, nil
	}

	_, _, err = runCommand("mkdir -p ~/.local/bin", conn)
	if err != nil {
		return false, err
	}

	localBinPath := fmt.Sprintf("/home/%s/.local/bin", args.user)

	paths, _, err := runCommand("bash -c \"echo $PATH\"", conn)
	if err != nil {
		return false, err
	}
	pathList := strings.Split(strings.Trim(paths, "\n"), ":")

	if !funk.Contains(pathList, localBinPath) {
		_, _, err = runCommand(fmt.Sprintf("echo fish_add_path %s | fish", localBinPath), conn)
		if err != nil {
			return false, err
		}
	}

	_, _, err = runCommand("python3 -m pip install docker-compose", conn)
	if err != nil {
		return false, err
	}

	return true, nil
}
