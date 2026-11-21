package main

import (
	"fmt"
	"regexp"
	"strings"
)

// The rules are the config's, from the contract; only what to say when an answer breaks one is
// the installer's.

func checkHostname(s string) error {
	if !rules.allows("hostname", s) {
		return fmt.Errorf("letters, digits, _ and hyphens, starting and ending with a letter or digit, such as my-laptop")
	}
	return nil
}

func checkUsername(s string) error {
	if !rules.allows("username", s) {
		return fmt.Errorf("lowercase letters, digits, _ and -, starting with a letter or _, up to 32")
	}
	return nil
}

func checkSwap(s string) error {
	if !rules.allows("swapSize", s) {
		return fmt.Errorf("a size such as 4G or 8192M")
	}
	return nil
}

func checkTimezone(s string) error {
	if !rules.allows("timeZone", s) {
		return fmt.Errorf("a zone such as Europe/Berlin")
	}
	return nil
}

func checkLayout(s string) error {
	if !rules.allows("keyboardLayout", s) {
		return fmt.Errorf("a layout code such as us, de or us,de")
	}
	return nil
}

func checkVariant(s string) error {
	if !rules.allows("keyboardVariant", s) {
		return fmt.Errorf("a variant code such as nodeadkeys, or nothing")
	}
	return nil
}

func checkLocale(s string) error {
	if !rules.allows("locale", s) {
		return fmt.Errorf("a language such as en_US.UTF-8")
	}
	return nil
}

func checkVault(s string) error {
	if !rules.allows("passSshVault", s) {
		return fmt.Errorf("a vault name without quotes, $, backticks or backslashes")
	}
	return nil
}

func checkControlD(s string) error {
	if !rules.allows("controldResolver", s) {
		return fmt.Errorf("the resolver id from the Control D dashboard, such as abcd1234")
	}
	return nil
}

func checkSSHKey(s string) error {
	if !rules.allows("sshKeys", s) {
		return fmt.Errorf("a public key, such as ssh-ed25519 AAAA… you@laptop, and not the private one")
	}
	return nil
}

// GitHub's own rule for user names, checked before the name goes into a URL
var githubNameRe = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?$`)

func checkGitHubName(s string) error {
	if !githubNameRe.MatchString(s) {
		return fmt.Errorf("a GitHub user name, such as github:octocat")
	}
	return nil
}

func checkDesktop(s string) error {
	if !rules.allows("desktop", s) {
		return fmt.Errorf("one of %s", strings.Join(rules.values["desktop"], ", "))
	}
	return nil
}

func checkNvidia(s string) error {
	if !rules.allows("nvidia", s) {
		return fmt.Errorf("one of %s", strings.Join(rules.values["nvidia"], ", "))
	}
	return nil
}

func checkProfile(s string) error {
	if !rules.allows("profile", s) {
		return fmt.Errorf("one of %s", strings.Join(rules.values["profile"], ", "))
	}
	return nil
}

// disko reads them a line at a time, so a newline would split one in two
func checkPassphrase(s, again string) error {
	switch {
	case len(s) < 8:
		return fmt.Errorf("at least 8 characters")
	case strings.IndexFunc(s, isControl) >= 0:
		return fmt.Errorf("no newlines, tabs or other control characters")
	case again != "" && s != again:
		return fmt.Errorf("the two don't match")
	}
	return nil
}

func checkPassword(s, again string) error {
	switch {
	case s == "":
		return fmt.Errorf("can't be empty")
	case strings.IndexFunc(s, isControl) >= 0:
		return fmt.Errorf("no newlines, tabs or other control characters")
	case again != "" && s != again:
		return fmt.Errorf("the two don't match")
	}
	return nil
}

func isControl(r rune) bool { return r < 0x20 || r == 0x7f }

// The DNS menu, in the order the options screen shows it. Control D goes in useControlD, not in
// encryptedDns.
func dnsChoices() []choice { return rules.menus["encryptedDns"] }

func checkDNS(s string) error {
	if !rules.allows("encryptedDns", s) {
		return fmt.Errorf("one of %s", strings.Join(rules.values["encryptedDns"], ", "))
	}
	return nil
}

func checkCustomDNS(s string) error {
	servers := strings.Fields(s)
	if len(servers) == 0 {
		return fmt.Errorf("at least one server, as address#name")
	}
	for _, sv := range servers {
		if !rules.allows("customDns", sv) {
			return fmt.Errorf("%s isn't address#name, such as 194.242.2.2#dns.mullvad.net", sv)
		}
	}
	return nil
}
