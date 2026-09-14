//go:build !linux

package gui

import (
	sdialog "github.com/sqweek/dialog"
)

// openFileNative returns the selected path, or "" if the user cancelled.
func openFileNative(title string) (string, error) {
	filename, err := sdialog.File().Filter("Bin file", "bin").Title(title).Load()
	if err == sdialog.ErrCancelled {
		return "", nil
	}
	return filename, err
}

// saveFileNative returns the selected path, or "" if the user cancelled.
func saveFileNative(title, suggestedFilename string) (string, error) {
	filename, err := sdialog.File().Filter("Bin file", "bin").SetStartFile(suggestedFilename).Title(title).Save()
	if err == sdialog.ErrCancelled {
		return "", nil
	}
	return filename, err
}
