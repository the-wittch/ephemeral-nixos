package main

import (
	"fmt"
	"regexp"
	"strings"
)

// The same rules as scripts/common.sh, so nothing passes here that install.sh would refuse.
var (
	hostnameRe = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)
	usernameRe = regexp.MustCompile(`^[a-z_][a-z0-9_-]*$`)
	swapRe     = regexp.MustCompile(`^[0-9]+[KMGT]?$`)
	timezoneRe = regexp.MustCompile(`^[A-Za-z0-9_/+-]+$`)
	layoutRe   = regexp.MustCompile(`^[a-z]{2,3}(,[a-z]{2,3})*$`)
	variantRe  = regexp.MustCompile(`^[a-z0-9_-]*(,[a-z0-9_-]*)*$`)
	localeRe   = regexp.MustCompile(`^[A-Za-z_]+\.UTF-8$`)
	controldRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
)

func checkHostname(s string) error {
	if !hostnameRe.MatchString(s) {
		return fmt.Errorf("letters, digits and hyphens only, such as my-laptop")
	}
	return nil
}

func checkUsername(s string) error {
	if !usernameRe.MatchString(s) || len(s) > 32 {
		return fmt.Errorf("lowercase letters, digits, _ and -, starting with a letter, up to 32")
	}
	return nil
}

func checkSwap(s string) error {
	if !swapRe.MatchString(s) {
		return fmt.Errorf("a size such as 4G or 8192M")
	}
	return nil
}

func checkTimezone(s string) error {
	if !timezoneRe.MatchString(s) {
		return fmt.Errorf("a zone such as Europe/Berlin")
	}
	return nil
}

func checkLayout(s string) error {
	if !layoutRe.MatchString(s) {
		return fmt.Errorf("a layout code such as us, de or us,de")
	}
	return nil
}

func checkVariant(s string) error {
	if !variantRe.MatchString(s) {
		return fmt.Errorf("a variant code such as nodeadkeys, or nothing")
	}
	return nil
}

func checkLocale(s string) error {
	if !localeRe.MatchString(s) {
		return fmt.Errorf("a language such as en_US.UTF-8")
	}
	return nil
}

func checkVault(s string) error {
	if s == "" || strings.ContainsAny(s, "\"$`\\") || strings.IndexFunc(s, isControl) >= 0 {
		return fmt.Errorf("a vault name without quotes, $, backticks or backslashes")
	}
	return nil
}

func checkControlD(s string) error {
	if !controldRe.MatchString(s) {
		return fmt.Errorf("the resolver id from the Control D dashboard, such as abcd1234")
	}
	return nil
}

func checkPassphrase(s, again string) error {
	switch {
	case len(s) < 8:
		return fmt.Errorf("at least 8 characters")
	case again != "" && s != again:
		return fmt.Errorf("the two don't match")
	}
	return nil
}

func checkPassword(s, again string) error {
	switch {
	case s == "":
		return fmt.Errorf("can't be empty")
	case again != "" && s != again:
		return fmt.Errorf("the two don't match")
	}
	return nil
}

func isControl(r rune) bool { return r < 0x20 || r == 0x7f }
