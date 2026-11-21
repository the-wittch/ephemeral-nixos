package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"slices"
)

// The settings rules and menus from the config release this installer pins: its settings-contract
// package, which package.nix points EPHEMERAL_CONTRACT at. The config owns them; this only
// applies them.
type contract struct {
	patterns map[string]*regexp.Regexp
	values   map[string][]string
	menus    map[string][]choice
	// every key settings.json can have
	settings map[string]bool
	// the app categories, in the order to show them
	categories []string
}

// the settings the screens ask for from a menu
var menuSettings = []string{"desktop", "nvidia", "profile", "encryptedDns"}

var rules contract

func loadContract() (contract, error) {
	path := os.Getenv("EPHEMERAL_CONTRACT")
	if path == "" {
		return contract{}, fmt.Errorf("EPHEMERAL_CONTRACT isn't set, so there are no settings rules to check the answers against")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return contract{}, err
	}
	var raw struct {
		Rules    map[string]json.RawMessage `json:"rules"`
		Defaults map[string]json.RawMessage `json:"defaults"`
		Choices  map[string][]struct {
			Value, Label, Description, Tag string
		} `json:"choices"`
		AppCategories []string `json:"appCategories"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return contract{}, fmt.Errorf("%s: %w", path, err)
	}
	c := contract{patterns: map[string]*regexp.Regexp{}, values: map[string][]string{}, menus: map[string][]choice{}, settings: map[string]bool{}}
	for key := range raw.Defaults {
		c.settings[key] = true
	}
	for key, rule := range raw.Rules {
		c.settings[key] = true
		var pattern string
		var values []string
		switch {
		case json.Unmarshal(rule, &pattern) == nil:
			// the config's are POSIX extended expressions, matched against the whole value
			re, err := regexp.Compile("^(?:" + pattern + ")$")
			if err != nil {
				return contract{}, fmt.Errorf("the rule for %s: %w", key, err)
			}
			c.patterns[key] = re
		case json.Unmarshal(rule, &values) == nil:
			c.values[key] = values
		default:
			return contract{}, fmt.Errorf("the rule for %s is neither a pattern nor a list of values", key)
		}
	}
	if len(raw.AppCategories) == 0 {
		return contract{}, fmt.Errorf("%s has no order for the app categories; the config release is older than this installer", path)
	}
	c.categories = raw.AppCategories
	for _, key := range menuSettings {
		if len(raw.Choices[key]) == 0 {
			return contract{}, fmt.Errorf("%s has no menu for %s; the config release is older than this installer", path, key)
		}
		for _, e := range raw.Choices[key] {
			c.menus[key] = append(c.menus[key], choice{e.Value, e.Label, e.Description, e.Tag})
		}
	}
	return c, nil
}

// false for a key the contract doesn't have, so a renamed setting refuses rather than passes
func (c contract) allows(key, value string) bool {
	if re, ok := c.patterns[key]; ok {
		return re.MatchString(value)
	}
	return slices.Contains(c.values[key], value)
}
