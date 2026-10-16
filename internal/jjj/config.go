// Package jjj holds what makes jjj different from jjui at configuration
// level: compiled-in defaults and the jjj-only config tables.
package jjj

import (
	_ "embed"

	"github.com/BurntSushi/toml"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/journal"
)

//go:embed defaults.toml
var defaultsTOML string

type JournalConfig struct {
	Enabled    bool   `toml:"enabled"`
	Template   string `toml:"template"` // empty → journal.RowTemplateJournal
	Limit      int    `toml:"limit"`
	DailyNote  string `toml:"daily_note"`
	Callout    string `toml:"callout"`
	CreateNote bool   `toml:"create_note"`
}

type ShellConfig struct {
	Enabled       bool     `toml:"enabled"`
	NarrowColumns int      `toml:"narrow_columns"`
	LeftWidth     float64  `toml:"left_width"`
	NotesRows     int      `toml:"notes_rows"`
	Hidden        []string `toml:"hidden"`
}

type TasksConfig struct {
	Enabled  bool `toml:"enabled"`
	ShowDone bool `toml:"show_done"`
}

type StatsConfig struct {
	Weeks int `toml:"weeks"`
}

// Config is the jjj-only part of config.toml. Upstream tables in the same
// file are decoded by config.Config; unknown tables are ignored on both sides.
type Config struct {
	Journal JournalConfig `toml:"journal"`
	Shell   ShellConfig   `toml:"shell"`
	Tasks   TasksConfig   `toml:"tasks"`
	Stats   StatsConfig   `toml:"stats"`
}

// Current is the live jjj config, filled by ApplyDefaults and then by each
// Load of the user's global and repo config text.
var Current = &Config{}

// Load merges the jjj tables from a config.toml text into c.
func (c *Config) Load(data string) error {
	_, err := toml.Decode(data, c)
	return err
}

// ApplyDefaults loads defaults.toml into the upstream config (revset, preview
// commands) and into Current, and installs the bundled graph-mode row template.
func ApplyDefaults(upstream *config.Config) error {
	if err := upstream.Load(defaultsTOML, ""); err != nil {
		return err
	}
	upstream.Revisions.Template = journal.RowTemplateGraph
	return Current.Load(defaultsTOML)
}

// lightSelection replaces upstream's light-theme selection background, ANSI
// white. Many light terminal palettes render white as a mid grey, so the
// dimmed (bright black) text on a selected row became grey on grey. 253 is
// a fixed light grey in every 256-colour palette.
var lightSelection = map[string]config.Color{
	":selected":                  {Bg: "253"},
	"revisions details:selected": {Bg: "253", Bold: ptr(true)},
	"revset completion:selected": {Bg: "253", Bold: ptr(true)},
	"picker:selected":            {Fg: "cyan", Bg: "253", Bold: ptr(true), Underline: ptr(false)},
}

func ptr[T any](v T) *T { return &v }

// ApplyTheme layers jjj's theme colours onto a resolved theme. A selector
// the user set in [ui.colors] is left alone.
func ApplyTheme(colors map[string]config.Color, isDark bool) {
	if isDark {
		return
	}
	user := config.NormalizeColorSelectors(config.Current.UI.Colors)
	for key, c := range config.NormalizeColorSelectors(lightSelection) {
		if _, set := user[key]; !set {
			colors[key] = c
		}
	}
}
