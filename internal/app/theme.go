package app

type Theme int

const (
	ThemeSystem Theme = iota
	ThemeDark
	ThemeLight
)

// Dark reports the appearance the theme asks for. A theme that follows the
// system is dark when the system's appearance is unknown.
func (t Theme) Dark(systemDark bool, systemKnown bool) bool {
	switch t {
	case ThemeDark:
		return true
	case ThemeLight:
		return false
	}
	if !systemKnown {
		return true
	}
	return systemDark
}
