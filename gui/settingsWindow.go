package gui

import (
	"bufio"
	"bytes"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/roffe/cimtool/avr"
)

type settingsWindow struct {
	e                *Gui
	hwVerSelect      *widget.Select
	ignoreError      *widget.Check
	readSliderLabel  *widget.Label
	readSlider       *widget.Slider
	writeSliderLabel *widget.Label
	writeSlider      *widget.Slider
	updateButton     *widget.Button

	fyne.Window
}

func (sw *settingsWindow) layout() fyne.CanvasObject {
	return container.NewVBox(
		//&widget.Label{
		//	Text:      "CIM Tool Version: " + VERSION,
		//	TextStyle: fyne.TextStyle{Bold: true},
		//},
		container.NewHBox(widget.NewLabel("Arduino"), sw.hwVerSelect),
		sw.ignoreError,
		sw.readSliderLabel,
		sw.readSlider,
		sw.writeSliderLabel,
		sw.writeSlider,
		layout.NewSpacer(),
		sw.updateButton,
		//&widget.Button{
		//	Icon: theme.DocumentSaveIcon(),
		//	Text: "Save settings",
		//	OnTapped: func() {
		//		sw.Close()
		//	},
		//},
	)
}

func delayLabel(t string, f float64) string {
	return fmt.Sprintf("%s Pin Delay: %.0f", t, f)
}

func newSettingsView(e *Gui) fyne.CanvasObject {
	sw := &settingsWindow{
		e: e,
		hwVerSelect: widget.NewSelect([]string{"Uno", "Nano", "Nano (old bootloader)"}, func(s string) {
			e.hwVersion.Set(s)
			e.Preferences().SetString("hardware_version", s)
		}),
		ignoreError:      widget.NewCheckWithData("Ignore read validation errors", e.ignoreError),
		readSliderLabel:  widget.NewLabel(""),
		readSlider:       widget.NewSliderWithData(0, 255, e.readDelayValue),
		writeSliderLabel: widget.NewLabel(""),
		writeSlider:      widget.NewSliderWithData(0, 255, e.writeDelayValue),
	}

	if f, err := sw.e.readDelayValue.Get(); err == nil {
		sw.readSliderLabel.SetText(delayLabel("Read", f))
	}

	if f, err := sw.e.writeDelayValue.Get(); err == nil {
		sw.writeSliderLabel.SetText(delayLabel("Write", f))
	}

	sw.hwVerSelect.Alignment = fyne.TextAlignCenter
	sw.hwVerSelect.PlaceHolder = "Select Arduino version"
	if hwVer, err := e.hwVersion.Get(); err == nil {
		sw.hwVerSelect.SetSelected(hwVer)
	}

	sw.ignoreError.OnChanged = func(b bool) {
		sw.e.Preferences().SetBool("ignore_read_errors", b)
		sw.e.ignoreError.Set(b)
	}

	sw.readSlider.OnChanged = func(f float64) {
		sw.readSliderLabel.SetText(delayLabel("Read", f))
		sw.e.Preferences().SetFloat("read_pin_delay", f)
		sw.e.readDelayValue.Set(f)
	}

	sw.writeSlider.OnChanged = func(f float64) {
		sw.writeSliderLabel.SetText(delayLabel("Write", f))
		sw.e.Preferences().SetFloat("write_pin_delay", f)
		sw.e.writeDelayValue.Set(f)
	}

	sw.updateButton = widget.NewButtonWithIcon("Update firmware", theme.WarningIcon(), func() {
		sw.updateButton.Disable()
		sw.e.mw.disableButtons()
		go func() {
			fyne.Do(func() { sw.e.mw.docTab.Select(sw.e.mw.logTab) })
			defer fyne.Do(sw.updateButton.Enable)
			defer sw.e.mw.enableButtons()

			hwVer, err := sw.e.hwVersion.Get()
			if err != nil {
				hwVer = "Uno"
			}

			out, err := avr.Update(sw.e.port, hwVer, sw.e.mw.output)
			if err != nil {
				sw.e.mw.output("Error updating: %v", err)
				return
			}

			r := bytes.NewReader(out)
			scanner := bufio.NewScanner(r)
			for scanner.Scan() {
				sw.e.mw.output("%s", scanner.Text())
			}
			fyne.Do(func() {
				sw.e.mw.docTab.Select(sw.e.mw.settingsTab)
				dialog.ShowInformation("Update", "Firmware update complete", sw.e.mw)
			})
		}()
	})

	return sw.layout()
}
