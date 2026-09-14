package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

var (
	errorColor   = color.NRGBA{R: 0xf8, G: 0x51, B: 0x49, A: 0xff}
	successColor = color.NRGBA{R: 0x3f, G: 0xb9, B: 0x50, A: 0xff}
	warningColor = color.NRGBA{R: 0xd2, G: 0x99, B: 0x22, A: 0xff}
	accentColor  = color.NRGBA{R: 0x4c, G: 0x8d, B: 0xff, A: 0xff}
)

type MyTheme struct{}

var _ fyne.Theme = (*MyTheme)(nil)

func (m MyTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 0x11, G: 0x13, B: 0x18, A: 0xff}
	case theme.ColorNameButton:
		return color.NRGBA{R: 0x21, G: 0x26, B: 0x2e, A: 0xff}
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 0x54, G: 0x5b, B: 0x66, A: 0xff}
	case theme.ColorNameDisabledButton:
		return color.NRGBA{R: 0x1a, G: 0x1e, B: 0x24, A: 0xff}
	case theme.ColorNameError:
		return errorColor
	case theme.ColorNameFocus:
		return color.NRGBA{R: 0x4c, G: 0x8d, B: 0xff, A: 0x55}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 0xe6, G: 0xe8, B: 0xeb, A: 0xff}
	case theme.ColorNameHeaderBackground:
		return color.NRGBA{R: 0x16, G: 0x1a, B: 0x21, A: 0xff}
	case theme.ColorNameHover:
		return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x14}
	case theme.ColorNameHyperlink:
		return accentColor
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 0x1a, G: 0x1e, B: 0x24, A: 0xff}
	case theme.ColorNameInputBorder:
		return color.NRGBA{R: 0x30, G: 0x36, B: 0x40, A: 0xff}
	case theme.ColorNameMenuBackground:
		return color.NRGBA{R: 0x1c, G: 0x21, B: 0x29, A: 0xff}
	case theme.ColorNameOverlayBackground:
		return color.NRGBA{R: 0x16, G: 0x1a, B: 0x21, A: 0xff}
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 0x8b, G: 0x94, B: 0x9e, A: 0xff}
	case theme.ColorNamePressed:
		return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x33}
	case theme.ColorNamePrimary:
		return accentColor
	case theme.ColorNameScrollBar:
		return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x50}
	case theme.ColorNameSelection:
		return color.NRGBA{R: 0x4c, G: 0x8d, B: 0xff, A: 0x44}
	case theme.ColorNameSeparator:
		return color.NRGBA{R: 0x2a, G: 0x2f, B: 0x3a, A: 0xff}
	case theme.ColorNameShadow:
		return color.NRGBA{A: 0x66}
	case theme.ColorNameSuccess:
		return successColor
	case theme.ColorNameWarning:
		return warningColor
	}

	return theme.DefaultTheme().Color(name, variant)
}

func (m MyTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (m MyTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (m MyTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 6
	case theme.SizeNameInnerPadding:
		return 8
	case theme.SizeNameLineSpacing:
		return 4
	case theme.SizeNameText:
		return 13
	case theme.SizeNameHeadingText:
		return 24
	case theme.SizeNameSubHeadingText:
		return 18
	case theme.SizeNameCaptionText:
		return 11
	case theme.SizeNameInlineIcon:
		return 20
	case theme.SizeNameInputBorder:
		return 1
	case theme.SizeNameInputRadius:
		return 8
	case theme.SizeNameButtonRadius:
		return 8
	case theme.SizeNameSelectionRadius:
		return 6
	case theme.SizeNameMenuRadius:
		return 10
	case theme.SizeNamePopupRadius:
		return 10
	case theme.SizeNameDialogRadius:
		return 12
	case theme.SizeNameCardRadius:
		return 12
	case theme.SizeNameScrollBar:
		return 12
	case theme.SizeNameScrollBarSmall:
		return 4
	case theme.SizeNameScrollBarRadius:
		return 6
	case theme.SizeNameSeparatorThickness:
		return 1
	}
	return theme.DefaultTheme().Size(name)
}
