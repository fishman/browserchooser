package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Nord palette: https://www.nordtheme.com
var (
	nordPolar1 = color.NRGBA{0x2E, 0x34, 0x40, 0xFF} // #2E3440
	nordPolar2 = color.NRGBA{0x3B, 0x42, 0x52, 0xFF} // #3B4252
	nordPolar3 = color.NRGBA{0x43, 0x4C, 0x5E, 0xFF} // #434C5E
	nordPolar4 = color.NRGBA{0x4C, 0x56, 0x6A, 0xFF} // #4C566A
	nordSnow1  = color.NRGBA{0xD8, 0xDE, 0xE9, 0xFF} // #D8DEE9
	nordSnow2  = color.NRGBA{0xE5, 0xE9, 0xF0, 0xFF} // #E5E9F0
	nordSnow3  = color.NRGBA{0xEC, 0xEF, 0xF4, 0xFF} // #ECEFF4
	nordFrost1 = color.NRGBA{0x8F, 0xBC, 0xBB, 0xFF} // #8FBCBB
	nordFrost2 = color.NRGBA{0x88, 0xC0, 0xD0, 0xFF} // #88C0D0
	nordFrost3 = color.NRGBA{0x81, 0xA1, 0xC1, 0xFF} // #81A1C1
	nordFrost4 = color.NRGBA{0x5E, 0x81, 0xAC, 0xFF} // #5E81AC
	nordRed    = color.NRGBA{0xBF, 0x61, 0x6A, 0xFF} // #BF616A
	nordGreen  = color.NRGBA{0xA3, 0xBE, 0x8C, 0xFF} // #A3BE8C
	nordYellow = color.NRGBA{0xEB, 0xCB, 0x8B, 0xFF} // #EBCB8B
	nordPurple = color.NRGBA{0xB4, 0x8E, 0xAD, 0xFF} // #B48EAD
	nordGray   = color.NRGBA{0x8F, 0x98, 0xA3, 0xFF} // disabled text on snow
)

// nordColors maps each theme color to its [dark, light] nord values. Colors
// not listed fall back to the built-in theme.
var nordColors = map[fyne.ThemeColorName][2]color.Color{
	theme.ColorNameBackground:               {nordPolar1, nordSnow3},
	theme.ColorNameButton:                   {nordPolar2, nordSnow2},
	theme.ColorNameDisabledButton:           {nordPolar1, nordSnow2},
	theme.ColorNameDisabled:                 {nordPolar4, nordGray},
	theme.ColorNameError:                    {nordRed, nordRed},
	theme.ColorNameFocus:                    {nordFrost4, nordFrost4},
	theme.ColorNameForeground:               {nordSnow1, nordPolar1},
	theme.ColorNameForegroundOnError:        {nordSnow3, nordSnow3},
	theme.ColorNameForegroundOnPrimary:      {nordPolar1, nordSnow3},
	theme.ColorNameForegroundOnSuccess:      {nordPolar1, nordPolar1},
	theme.ColorNameForegroundOnWarning:      {nordPolar1, nordPolar1},
	theme.ColorNameHeaderBackground:         {nordPolar2, nordSnow2},
	theme.ColorNameHover:                    {nordPolar2, nordSnow2},
	theme.ColorNameHyperlink:                {nordFrost3, nordFrost4},
	theme.ColorNameInnerWindowBorder:        {nordPolar2, nordSnow2},
	theme.ColorNameInnerWindowBorderInactive: {nordPolar3, nordSnow1},
	theme.ColorNameInputBackground:          {nordPolar2, nordSnow2},
	theme.ColorNameInputBorder:              {nordPolar4, nordSnow1},
	theme.ColorNameMenuBackground:           {nordPolar2, color.White},
	theme.ColorNameOverlayBackground:        {nordPolar2, color.White},
	theme.ColorNamePlaceHolder:              {nordPolar4, nordGray},
	theme.ColorNamePressed:                  {nordPolar3, nordSnow1},
	theme.ColorNamePrimary:                  {nordFrost2, nordFrost4},
	theme.ColorNameScrollBar:                {nordPolar4, nordSnow1},
	theme.ColorNameScrollBarBackground:      {nordPolar2, nordSnow2},
	theme.ColorNameSelection:                {nordPolar3, nordSnow1},
	theme.ColorNameSeparator:                {nordPolar3, nordSnow1},
	theme.ColorNameShadow:                   {color.NRGBA{0, 0, 0, 0x73}, color.NRGBA{0, 0, 0, 0x33}},
	theme.ColorNameSuccess:                  {nordGreen, nordGreen},
	theme.ColorNameWarning:                  {nordYellow, nordYellow},
}

type nordTheme struct{ dark bool }

func (t *nordTheme) Color(n fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	if c, ok := nordColors[n]; ok {
		if t.dark {
			return c[0]
		}
		return c[1]
	}
	// the map above covers every color fyne defines; this is a neutral fallback
	if t.dark {
		return nordPolar1
	}
	return nordSnow3
}

func (t *nordTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *nordTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *nordTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
