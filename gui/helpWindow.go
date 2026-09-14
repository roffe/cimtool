package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/roffe/cimtool/assets"
)

const helpIntroConnect = `
## Connecting to the CIM

To read or write the CIM's memory, connect either a SOP8 test clip or a pogo pin probe to the EEPROM chip on the CIM's circuit board, shown below. A clip stays attached on its own, while a pogo pin probe needs to be pressed against the EEPROM's legs and held steady for the whole read or write.
`

const helpIntroPrepare = `
## Preparing the EEPROM

The legs of the EEPROM are covered by a protective conformal coating that prevents the clip from making electrical contact. Gently scrape the coating off the legs with a sharp knife or razor blade, then clean them with IPA or acetone on a cotton swab.

**Take care** — excessive force can break the legs, which would require soldering in a new EEPROM.

## Clip / probe orientation

Before attaching the SOP8 clip or pressing the pogo pin probe on, check the cable orientation: the red wire goes to the corner of the EEPROM with the indentation (pin 1), as shown below.
`

const helpTroubleshooting = `
## Failed reads

A failed read is usually caused by one of the following, in order of likelihood:

- **Conformal coating still on the EEPROM** — the clip or probe cannot make good contact until the legs are scraped clean.
- **SOP8 clip or pogo pin probe not seated properly** — re-seat the clip and make sure it sits straight and grips all eight legs, or keep the probe pressed firmly and evenly against the EEPROM for the whole operation.
- **Pin delays too low** — worn or aged EEPROMs may need slower timing. Increase the read/write pin delays in Settings.
- **Corrupted or defective EEPROM** — see below.

## DTC B1000-36 / corrupted EEPROM

If you consistently get the same CRC error on repeated reads, the EEPROM content has probably become corrupted. DTC B1000-36 usually indicates this. Common causes are power loss during a write operation, a stray bit flip or a failing chip.

What to do:

1. When the read validation error appears, choose to view the file anyway, then save it.
2. Contact Roffe via TrionicTuning.com — the content can often be repaired, and you can then flash it back with no further action required.
3. If the data cannot be recovered, it may still be possible to virginize the CIM and marry it to the car with Tech2, avoiding the cost of a new CIM module.

If the EEPROM keeps failing read validation even after a known-good binary has been flashed, the chip itself is most likely defective and needs to be replaced.
`

const helpSettings = `
## Arduino

Select which board your programmer is built on: Uno, Nano or Nano with the old bootloader. This is used when updating the programmer firmware.

## Ignore read validation errors

When enabled, reads that fail CRC validation can still be saved to disk. Useful for salvaging data from a corrupted EEPROM.

## Read / Write pin delay

The timing used when talking to the EEPROM. Higher values are slower but more reliable. If reads or flashes fail, try increasing the delays — worn EEPROMs often need it.

## Update firmware

Flashes the latest programmer firmware to your Arduino over the selected serial port.
`

func helpText(md string) *widget.RichText {
	rt := widget.NewRichTextFromMarkdown(md)
	rt.Wrapping = fyne.TextWrapWord
	return rt
}

func helpImage(name string, content []byte) fyne.CanvasObject {
	return container.NewCenter(&canvas.Image{
		ScaleMode: canvas.ImageScaleFastest,
		FillMode:  canvas.ImageFillOriginal,
		Resource:  fyne.NewStaticResource(name, content),
	})
}

func newHelpWindow(e *Gui) fyne.Window {
	introTab := container.NewTabItemWithIcon("Getting started", theme.QuestionIcon(),
		container.NewVScroll(container.NewVBox(
			helpText(helpIntroConnect),
			helpImage("pcb.jpg", assets.PcbBytes),
			helpText(helpIntroPrepare),
			helpImage("eeprom.jpg", assets.EepromBytes),
		)),
	)

	failedTab := container.NewTabItemWithIcon("Troubleshooting", theme.ErrorIcon(),
		container.NewVScroll(helpText(helpTroubleshooting)),
	)

	settingsTab := container.NewTabItemWithIcon("Settings", theme.SettingsIcon(),
		container.NewVScroll(helpText(helpSettings)),
	)

	w := e.NewWindow("Help")
	w.SetOnClosed(func() {
		e.mw.hw = nil
	})

	w.SetContent(container.NewAppTabs(
		introTab,
		failedTab,
		settingsTab,
	))
	w.Resize(fyne.NewSize(920, 800))
	w.Show()
	return w
}
