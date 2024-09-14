package layer2

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
)

type tailscaleStatus string

const (
	// Logged in and running
	tailscaleUp tailscaleStatus = "up"
	// Logged in but not running
	tailscaleStopped tailscaleStatus = "stopped"
	// Not logged in and not running
	tailscaleLoggedOut tailscaleStatus = "logged-out"
)

func (m *layer2Manager) getTailScaleStatus() (tailscaleStatus, error) {
	// We ignore the error because if tailscale is in status 'Stopped' it will return 1
	stdout, _, _ := m.conn.Run("tailscale status")
	stdout = strings.Trim(stdout, "\n")

	m.log.Debug().Str("stdout", stdout).Msg("Parsing tailscale status")

	switch stdout {
	case "Logged out.":
		return tailscaleLoggedOut, nil
	case "Tailscale is stopped.":
		return tailscaleStopped, nil
	}

	lines := strings.Split(stdout, "\n")
	if len(lines) == 0 {
		log.Error().Str("stdout", stdout).Msg("Cannot parse tailscale status (no lines)")
		return "", fmt.Errorf("cannot parse tailscale status (no lines): %s", stdout)
	}
	fields := strings.Fields(lines[0])
	if len(fields) == 0 {
		log.Error().Str("stdout", stdout).Msg("Cannot parse tailscale status (no fields)")
		return "", fmt.Errorf("cannot parse tailscale status (no fields): %s", stdout)
	}

	// Apply IPv4 regex to the first field
	ipRegex := regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	if !ipRegex.MatchString(fields[0]) {
		m.log.Error().Strs("fields", fields).Msg("Cannot parse tailscale status (first field not a IPv4)")
		return "", fmt.Errorf("cannot parse tailscale status (first field not a IPv4): %s", fields[0])
	}

	return tailscaleUp, nil
}

func (m *layer2Manager) tailscaleLogin(authKey string) error {
	_, _, err := m.conn.RunSudo(fmt.Sprintf("tailscale login --auth-key %s", authKey))
	if err != nil {
		return fmt.Errorf("error logging in to tailscale: %w", err)
	}

	return nil
}

func (m *layer2Manager) tailscaleUp(user string) error {
	_, _, err := m.conn.Run(fmt.Sprintf("tailscale up --operator=%s", user))
	if err != nil {
		return fmt.Errorf("error starting tailscale: %w", err)
	}

	return nil
}
