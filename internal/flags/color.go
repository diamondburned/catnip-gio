package flags

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/spf13/pflag"
)

// ColorNRGBA is a color.NRGBA that implements pflag.Value.
type ColorNRGBA color.NRGBA

var _ pflag.Value = (*ColorNRGBA)(nil)

// NewColorNRGBA creates a new ColorNRGBA using the given RGBA values.
func NewColorNRGBA(r, g, b, a uint8) *ColorNRGBA {
	return &ColorNRGBA{r, g, b, a}
}

// MustParseColorNRGBA parses a hexadecimal color string and panics on error.
func MustParseColorNRGBA(s string) *ColorNRGBA {
	c, err := ParseColorNRGBA(s)
	if err != nil {
		panic(err)
	}
	return c
}

// ParseColorNRGBA parses a hexadecimal color string.
func ParseColorNRGBA(s string) (*ColorNRGBA, error) {
	if !strings.HasPrefix(s, "#") {
		return nil, fmt.Errorf("invalid color: %q", s)
	}

	s = s[1:]
	var c ColorNRGBA
	var err error

	switch len(s) {
	case 3:
		_, err = fmt.Sscanf(s, "%1x%1x%1x", &c.R, &c.G, &c.B)
		c.R *= 17
		c.G *= 17
		c.B *= 17
		c.A = 255
	case 4:
		_, err = fmt.Sscanf(s, "%1x%1x%1x%1x", &c.R, &c.G, &c.B, &c.A)
		c.R *= 17
		c.G *= 17
		c.B *= 17
		c.A *= 17
	case 6:
		_, err = fmt.Sscanf(s, "%2x%2x%2x", &c.R, &c.G, &c.B)
		c.A = 255
	case 8:
		_, err = fmt.Sscanf(s, "%2x%2x%2x%2x", &c.R, &c.G, &c.B, &c.A)
	default:
		return nil, fmt.Errorf("invalid hexadecimal color %q", "#"+s)
	}

	if err != nil {
		return nil, fmt.Errorf("invalid hexadecimal color %q: %w", "#"+s, err)
	}

	return &c, nil
}

func (c *ColorNRGBA) Set(s string) error {
	cc, err := ParseColorNRGBA(s)
	if err != nil {
		return err
	}
	*c = ColorNRGBA(*cc)
	return nil
}

func (c *ColorNRGBA) String() string {
	return fmt.Sprintf("#%02x%02x%02x%02x", c.R, c.G, c.B, c.A)
}

func (c *ColorNRGBA) NRGBA() color.NRGBA {
	return color.NRGBA(*c)
}

func (c *ColorNRGBA) Type() string {
	return "color"
}
