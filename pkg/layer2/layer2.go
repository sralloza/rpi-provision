package layer2

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/rs/zerolog"
	"github.com/sralloza/rpi-provisioner/pkg/info"
	"github.com/sralloza/rpi-provisioner/pkg/logging"
	"github.com/sralloza/rpi-provisioner/pkg/ssh"
)

type Layer2Args struct {
	User             string
	Host             string
	Port             int
	TailscaleAuthKey string
}

func NewManager() *layer2Manager {
	return &layer2Manager{
		log: logging.Get(),
	}
}

type layer2Manager struct {
	conn ssh.SSHConnection
	log  *zerolog.Logger
}

type Layer2Result struct {
	NeedManualTailscaleLogin bool
	DockerInstallErr         error
}

// Returns (needManualTailscaleLogin, dockerInstallErr, error)
func (m *layer2Manager) Provision(args Layer2Args) (Layer2Result, error) {
	result := Layer2Result{
		NeedManualTailscaleLogin: false,
		DockerInstallErr:         nil,
	}
	address := fmt.Sprintf("%s:%d", args.Host, args.Port)

	info.Title("Connecting to server")
	m.conn = ssh.SSHConnection{UseSSHKey: true}
	err := m.conn.Connect(args.User, address)
	if err != nil {
		info.Fail()
		return result, err
	}
	info.Ok()
	defer m.conn.Close()

	return m.provisionLayer2(args)
}

// Returns (needManualTailscaleLogin, dockerInstallErr, error)
func (m *layer2Manager) provisionLayer2(args Layer2Args) (Layer2Result, error) {
	result := Layer2Result{
		NeedManualTailscaleLogin: false,
		DockerInstallErr:         nil,
	}

	info.Title("Updating and upgrading packages")
	if err := m.installLibraries(); err != nil {
		info.Fail()
		return result, err
	}
	info.Ok()

	info.Title("Installing zsh")
	if installed, err := m.installZsh(args); err != nil {
		info.Fail()
		return result, err
	} else if installed {
		info.Ok()
	} else {
		info.Skipped()
	}

	info.Title("Installing oh-my-zsh")
	if installed, err := m.installOhMyZsh(args); err != nil {
		info.Fail()
		return result, err
	} else if installed {
		info.Ok()
	} else {
		info.Skipped()
	}

	info.Title("Configuring zsh plugins")
	if installed, err := m.configureZshPlugins(); err != nil {
		info.Fail()
		return result, err
	} else if installed {
		info.Ok()
	} else {
		info.Skipped()
	}

	info.Title("Installing powerlevel10k")
	if installed, err := m.installPowerlevel10k(); err != nil {
		info.Fail()
		return result, err
	} else if installed {
		info.Ok()
	} else {
		info.Skipped()
	}

	info.Title("Installing tailscale")
	if installed, err := m.installTailscale(); err != nil {
		info.Fail()
		return result, err
	} else if installed {
		info.Ok()
	} else {
		info.Skipped()
	}

	info.Title("Starting and setting up tailscale")
	tailscaleStarted, needManualLogin, err := m.startAndSetupTailscale(args.TailscaleAuthKey, args.User)
	result.NeedManualTailscaleLogin = needManualLogin
	if err != nil {
		info.Fail()
		return result, err
	} else if tailscaleStarted {
		info.Ok()
	} else {
		info.Skipped()
	}

	info.Title("Installing docker")
	installed, dockerInstallErr, err := m.installDocker(args)
	if err != nil {
		info.Fail()
		result.DockerInstallErr = dockerInstallErr
		return result, err
	} else if installed {
		info.Ok()
	} else {
		info.Skipped()
	}
	return result, nil
}

func (m *layer2Manager) installLibraries() error {
	_, _, err := m.conn.RunSudo("apt-get update")
	if err != nil {
		return fmt.Errorf("error updating apt registry: %w", err)
	}

	_, _, err = m.conn.RunSudo("apt-get upgrade -y")
	if err != nil {
		return fmt.Errorf("error upgrading libraries: %w", err)
	}

	libraries := []string{
		"bat",
		"build-essential",
		"cmake",
		"cron",
		"curl",
		"git",
		"jq",
		"libffi-dev",
		"mailutils",
		"mdadm",
		"nano",
		"progress",
		"ripgrep",
		"sqlite3",
		"tcpdump",
		"tree",
		"wget",
	}
	installCmd := fmt.Sprintf("apt-get install %s -y", strings.Join(libraries, " "))
	_, _, err = m.conn.RunSudo(installCmd)
	if err != nil {
		return fmt.Errorf("error installing needed libraries: %w", err)
	}

	// Bat in debian is called 'batcat'
	// https://github.com/sharkdp/bat/issues/982
	_, _, err = m.conn.Run("which bat")
	if err != nil {
		_, _, err = m.conn.RunSudo("ln -s /usr/bin/batcat /usr/bin/bat")
		if err != nil {
			return fmt.Errorf("error creating bat symlink: %w", err)
		}
	}

	return nil
}

func (m *layer2Manager) installZsh(args Layer2Args) (bool, error) {
	_, _, err := m.conn.Run("which zsh")
	if err == nil {
		return false, nil
	}

	_, _, err = m.conn.RunSudo("apt install zsh -y")
	if err != nil {
		return false, fmt.Errorf("error installing zsh: %w", err)
	}

	chshCmd := fmt.Sprintf("chsh -s /usr/bin/zsh %s", args.User)
	_, _, err = m.conn.RunSudo(chshCmd)
	if err != nil {
		return false, fmt.Errorf("error setting deployer's shell to zsh: %w", err)
	}

	return true, nil
}

func (m *layer2Manager) installOhMyZsh(args Layer2Args) (bool, error) {
	_, _, err := m.conn.Run("file ~/.oh-my-zsh -E")
	if err == nil {
		return false, nil
	}

	_, _, err = m.conn.Run("curl -L https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh > /tmp/omz.sh")
	if err != nil {
		return false, fmt.Errorf("error downloading oh-my-zsh installer: %w", err)
	}

	_, _, err = m.conn.Run("sh /tmp/omz.sh")
	if err != nil {
		return false, fmt.Errorf("error running oh-my-zsh installer: %w", err)
	}

	_, _, err = m.conn.Run("rm /tmp/omz.sh")
	if err != nil {
		return false, fmt.Errorf("error removing oh-my-zsh installer: %w", err)
	}
	return true, nil
}

func (m *layer2Manager) configureZshPlugins() (bool, error) {
	zshrc, _, err := m.conn.Run("cat ~/.zshrc")
	if err != nil {
		return false, fmt.Errorf("error getting zshrc: %w", err)
	}

	r := regexp.MustCompile(`(?m)^plugins=\([a-z\s-]+\)`)

	if !r.MatchString(zshrc) {
		return false, fmt.Errorf("error finding plugins in zshrc")
	}

	zshSuggClone, err := m.cloneGitRepo(
		"https://github.com/zsh-users/zsh-autosuggestions.git",
		"${ZSH_CUSTOM:-~/.oh-my-zsh/custom}/plugins/zsh-autosuggestions")
	if err != nil {
		return false, err
	}

	syntaxHighlClone, err := m.cloneGitRepo("https://github.com/zsh-users/zsh-syntax-highlighting.git", "${ZSH_CUSTOM:-~/.oh-my-zsh/custom}/plugins/zsh-syntax-highlighting")
	if err != nil {
		return false, err
	}

	fzfPluginClone, err := m.cloneGitRepo("https://github.com/unixorn/fzf-zsh-plugin.git", "${ZSH_CUSTOM:-~/.oh-my-zsh/custom}/plugins/fzf-zsh-plugin")
	if err != nil {
		return false, err
	}

	plugins := []string{
		"fzf-zsh-plugin",
		"git",
		"zsh-autosuggestions",
		"zsh-syntax-highlighting",
	}
	pluginsWithSpace := []string{}
	for _, plugin := range plugins {
		pluginsWithSpace = append(pluginsWithSpace, fmt.Sprintf(" %s", plugin))
	}

	newZshrc := r.ReplaceAllString(zshrc, fmt.Sprintf("plugins=(\n%s\n)", strings.Join(pluginsWithSpace, "\n")))

	zshChanged := newZshrc != zshrc
	if zshChanged {
		m.log.Info().Msg("zshrc plugins changed, updating")
		err = m.conn.WriteToFile("/home/deployer/.zshrc", []byte(newZshrc))
		if err != nil {
			return false, fmt.Errorf("error setting plugins in zshrc: %w", err)
		}
	}

	return zshSuggClone || syntaxHighlClone || fzfPluginClone || zshChanged, nil
}

func (m *layer2Manager) installPowerlevel10k() (bool, error) {
	repoCloned, err := m.cloneGitRepo(
		"https://github.com/romkatv/powerlevel10k.git",
		"${ZSH_CUSTOM:-$HOME/.oh-my-zsh/custom}/themes/powerlevel10k")

	if err != nil {
		return false, err
	}

	_, _, err = m.conn.Run("grep 'ZSH_THEME=\"powerlevel10k/powerlevel10k\"' .zshrc")
	missingTheme := err != nil
	if missingTheme {
		_, _, err = m.conn.Run("sed -i 's/ZSH_THEME=\".*\"/ZSH_THEME=\"powerlevel10k\\/powerlevel10k\"/' ~/.zshrc")
		if err != nil {
			return false, fmt.Errorf("error setting ZSH_THEME: %w", err)
		}
	}

	// Disable configuration wizard
	_, _, err = m.conn.Run("grep \"POWERLEVEL9K_DISABLE_CONFIGURATION_WIZARD=true\" ~/.zshrc")
	missingWizardDisable := err != nil
	if missingWizardDisable {
		_, _, err = m.conn.Run("echo \"POWERLEVEL9K_DISABLE_CONFIGURATION_WIZARD=true\" >> ~/.zshrc")
		if err != nil {
			return false, fmt.Errorf("error disabling powerlevel10k configuration wizard: %w", err)
		}
	}

	m.log.Info().
		Bool("repoCloned", repoCloned).
		Bool("missingTheme", missingTheme).
		Bool("missingWizardDisable", missingWizardDisable).
		Msg("powerlevel10k configured")
	return repoCloned || missingTheme || missingWizardDisable, nil
}

func (m *layer2Manager) installDocker(args Layer2Args) (bool, error, error) {
	_, _, err := m.conn.Run("which docker")
	if err == nil {
		_, _, err = m.conn.Run("docker compose")
		if err != nil {
			return false, nil, fmt.Errorf("docker is installed but docker compose v2 is not")
		}
		_, _, err := m.conn.Run("docker ps")
		if err != nil {
			return false, nil, fmt.Errorf("docker is installed but not running: %w", err)
		}
		return false, nil, nil
	}

	_, _, err = m.conn.Run("curl -fsSL https://get.docker.com -o /tmp/get-docker.sh")
	if err != nil {
		return false, nil, fmt.Errorf("error downloading docker installer: %w", err)
	}

	// Docker instalation may fail, but a reboot should fix it
	// https://stackoverflow.com/questions/59752840/docker-socket-failed-with-result-service-start-limit-hit-after-protecting-doc
	var dockerInstallErr error
	_, _, err = m.conn.Run("sudo sh /tmp/get-docker.sh")
	if err != nil {
		m.log.Warn().Msgf("error executing docker installer: %v", err)
		dockerInstallErr = fmt.Errorf("error executing docker installer: %w", err)
	}

	_, _, err = m.conn.Run("rm /tmp/get-docker.sh")
	if err != nil {
		return false, dockerInstallErr, fmt.Errorf("error removing docker installer: %w", err)
	}

	_, _, err = m.conn.Run(fmt.Sprintf("sudo usermod -aG docker %s", args.User))
	if err != nil {
		return false, dockerInstallErr, fmt.Errorf("error adding deployer to docker group: %w", err)
	}

	return true, dockerInstallErr, nil
}

func (m *layer2Manager) installTailscale() (bool, error) {
	_, _, err := m.conn.Run("which tailscale")
	if err == nil {
		return false, nil
	}

	_, _, err = m.conn.Run("curl -fsSL https://tailscale.com/install.sh -o /tmp/install-tailscale.sh")
	if err != nil {
		return false, fmt.Errorf("error downloading tailscale installer: %w", err)
	}

	_, _, err = m.conn.Run("sh /tmp/install-tailscale.sh")
	if err != nil {
		return false, fmt.Errorf("error executing tailscale installer: %w", err)
	}

	_, _, err = m.conn.Run("rm /tmp/install-tailscale.sh")
	if err != nil {
		return false, fmt.Errorf("error removing tailscale installer: %w", err)
	}

	return true, nil
}

// Starts tailscale and logs in if needed
// Returns (tailscaleStarted, needManualLogin, error)
func (m *layer2Manager) startAndSetupTailscale(authKey, user string) (bool, bool, error) {
	status, err := m.getTailScaleStatus()
	if err != nil {
		return false, false, err
	}
	m.log.Debug().Str("status", string(status)).Msg("tailscale status")

	if status == tailscaleUp {
		m.log.Debug().Msg("tailscale is already running")
		return false, false, nil
	}

	if status == tailscaleLoggedOut {
		m.log.Debug().Msg("tailscale is logged out, logging in")
		if authKey == "" {
			m.log.Debug().Msg("tailscale auth key not provided, will not login")
			return false, true, nil
		}

		if err := m.tailscaleLogin(authKey); err != nil {
			return false, false, fmt.Errorf("error logging in to tailscale: %w", err)
		}
	}

	if err := m.tailscaleUp(user); err != nil {
		return false, false, fmt.Errorf("error starting tailscale: %w", err)
	}

	return true, false, nil
}

func (m *layer2Manager) cloneGitRepo(repo, path string) (bool, error) {
	_, _, err := m.conn.Run(fmt.Sprintf("file %s -E", path))
	if err != nil {
		_, _, err = m.conn.Run(fmt.Sprintf("git clone --depth 1 %s %s", repo, path))
		if err != nil {
			return false, fmt.Errorf("error cloning repo %s: %w", repo, err)
		}
		return true, nil
	}
	return false, nil
}
