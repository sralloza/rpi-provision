package ssh

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/sralloza/rpi-provisioner/pkg/authorizedkeys"
	"github.com/sralloza/rpi-provisioner/pkg/logging"
	"golang.org/x/crypto/ssh"
)

type SSHConnection struct {
	config    *ssh.ClientConfig
	conn      *ssh.Client
	Password  string
	UseSSHKey bool
	Timeout   int64
}

func (c *SSHConnection) Connect(user string, address string) error {
	var auth []ssh.AuthMethod

	if c.UseSSHKey {
		auth = append(auth, publicKey("~/.ssh/id_rsa"))
	} else {
		auth = append(auth, ssh.Password(c.Password))
	}

	c.config = &ssh.ClientConfig{
		User:            user,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	conn, err := ssh.Dial("tcp", address, c.config)
	if err != nil {
		return fmt.Errorf("could not stablish ssh connection: %w", err)
	}
	c.conn = conn
	return nil
}

func (c SSHConnection) Run(cmd string) (string, string, error) {
	return runCommand(cmd, c.conn)
}

func (c SSHConnection) RunSudo(cmd string) (string, string, error) {
	return c.Run(basicSudoStdin(cmd, ""))
}

func (c SSHConnection) RunSudoPassword(cmd string, password string) (string, string, error) {
	return c.Run(basicSudoStdin(cmd, password))
}

func (c SSHConnection) Close() {
	c.conn.Close()
}

func basicSudoStdin(cmd string, password string) string {
	return fmt.Sprintf("echo %s | sudo -S bash -c '%s'", password, cmd)
}

func runCommand(cmd string, conn *ssh.Client) (string, string, error) {
	sess, err := conn.NewSession()
	if err != nil {
		panic(err)
	}
	defer sess.Close()
	sessStdOut, err := sess.StdoutPipe()
	if err != nil {
		panic(err)
	}
	sessStderr, err := sess.StderrPipe()
	if err != nil {
		panic(err)
	}
	err = sess.Run(cmd)

	bufOut := new(strings.Builder)
	io.Copy(bufOut, sessStdOut)
	bufErr := new(strings.Builder)
	io.Copy(bufErr, sessStderr)

	logger := logging.Get()
	logger.Debug().
		Str("cmd", cmd).
		Str("stdout", bufOut.String()).
		Str("stderr", bufErr.String()).
		Msg("Running command via ssh")

	return bufOut.String(), bufErr.String(), err
}

type UploadsshKeysArgs struct {
	User     string
	Password string
	Group    string
	KeysUri  string
}

func UploadsshKeys(conn SSHConnection, args UploadsshKeysArgs) (bool, error) {
	mkdirCmd := fmt.Sprintf("mkdir -p /home/%s/.ssh", args.User)
	_, _, err := conn.Run(mkdirCmd)
	if err != nil {
		return false, fmt.Errorf("error creating user's ssh directory: %w", err)
	}

	catCmd := fmt.Sprintf("cat /home/%s/.ssh/authorized_keys", args.User)
	fileContent, _, err := conn.Run(catCmd)

	var authorizedKeys []string
	if err != nil {
		authorizedKeys = []string{}
	} else {
		authorizedKeys = strings.Split(strings.Trim(fileContent, "\n"), "\n")
	}

	newKeysInfo, err := authorizedkeys.Get(args.KeysUri)
	if err != nil {
		return false, fmt.Errorf("error getting authorized keys: %w", err)
	}

	newKeys := []string{}
	for _, key := range newKeysInfo {
		newKeys = append(newKeys, key.String())
	}

	finalKeys := removeDuplicateStr(newKeys)
	sort.Strings(finalKeys)

	newFileContent := strings.Trim(strings.Join(finalKeys, "\n"), "\n")

	if len(authorizedKeys) == len(finalKeys) {
		equal := true
		for i := 0; i < len(authorizedKeys); i++ {
			if authorizedKeys[i] != finalKeys[i] {
				equal = false
				continue
			}
		}
		if equal {
			return false, nil
		}
	}

	updateKeysCmd := fmt.Sprintf("echo \"%s\" > /home/%s/.ssh/authorized_keys", newFileContent, args.User)
	_, _, err = conn.RunSudoPassword(updateKeysCmd, args.Password)
	if err != nil {
		return false, fmt.Errorf("error updating authorized_keys: %w", err)
	}

	sshFolder := fmt.Sprintf("/home/%s/.ssh", args.User)
	authorizedKeysPath := fmt.Sprintf("%s/authorized_keys", sshFolder)

	chmodsshCmd := fmt.Sprintf("chmod 700 %s", sshFolder)
	_, _, err = conn.RunSudoPassword(chmodsshCmd, args.Password)
	if err != nil {
		return false, fmt.Errorf("error setting permissions to ssh folder: %w", err)
	}

	chmodAkpath := fmt.Sprintf("chmod 600 %s", authorizedKeysPath)
	_, _, err = conn.RunSudoPassword(chmodAkpath, args.Password)
	if err != nil {
		return false, fmt.Errorf("error setting permissions to authorized_keys: %w", err)
	}

	ownership := fmt.Sprintf("%s:%s", args.User, args.Group)
	chownsshCmd := fmt.Sprintf("chown %s %s", ownership, sshFolder)
	_, _, err = conn.RunSudoPassword(chownsshCmd, args.Password)
	if err != nil {
		return false, fmt.Errorf("error setting ownership of ssh folder: %w", err)
	}

	chownAkpCmd := fmt.Sprintf("chown %s %s", ownership, authorizedKeysPath)
	_, _, err = conn.RunSudoPassword(chownAkpCmd, args.Password)
	if err != nil {
		return false, fmt.Errorf("error setting ownership of authorized_keys: %w", err)
	}

	return true, nil
}

func expandPath(path string) string {
	res, _ := homedir.Expand(path)
	return res
}

func publicKey(path string) ssh.AuthMethod {
	key, err := os.ReadFile(expandPath(path))
	if err != nil {
		panic(err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		panic(err)
	}
	return ssh.PublicKeys(signer)
}

func removeDuplicateStr(strSlice []string) []string {
	allKeys := make(map[string]bool)
	list := []string{}
	for _, item := range strSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}
