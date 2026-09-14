package gui

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/roffe/cim"
	"github.com/roffe/cimtool/adapter"
	"github.com/roffe/cimtool/assets"
)

type mainWindow struct {
	e *Gui

	hw     fyne.Window
	docTab *container.DocTabs

	logTab      *container.TabItem
	settingsTab *container.TabItem

	logList binding.StringList
	log     *widget.List

	rescanButton *widget.Button
	portList     *widget.Select

	openButton  *widget.Button
	readButton  *widget.Button
	writeButton *widget.Button
	eraseButton *widget.Button
	helpButton  *widget.Button
	copyButton  *widget.Button
	clearButton *widget.Button

	readMIUButton *widget.Button
	//writeMIUButton *widget.Button

	progressBar *widget.ProgressBar

	fyne.Window
}

var mainSize = fyne.NewSize(1280, 700)

func newMainWindow(e *Gui) *mainWindow {
	m := &mainWindow{
		Window:      e.NewWindow("Saab CIM Tool " + fyne.CurrentApp().Metadata().Version + " Build: " + strconv.Itoa(fyne.CurrentApp().Metadata().Build)),
		e:           e,
		logList:     binding.NewStringList(),
		progressBar: widget.NewProgressBar(),
	}

	m.docTab = container.NewDocTabs()
	m.docTab.CloseIntercept = func(i *container.TabItem) {
		switch i.Text {
		case "Home", "Log", "Settings":
			return
		}
		dialog.ShowConfirm("Close", "Are you sure you want to close this tab?", func(b bool) {
			if b {
				m.docTab.Remove(i)
			}
		}, m.Window)
	}

	logo := &canvas.Image{
		FillMode:  canvas.ImageFillContain,
		ScaleMode: canvas.ImageScaleSmooth,
		Resource:  fyne.NewStaticResource("logo.png", assets.LogoBytes),
	}
	logo.SetMinSize(fyne.NewSize(180, 180))

	m.docTab.Append(
		container.NewTabItemWithIcon("Start", theme.HomeIcon(), container.NewCenter(
			container.NewVBox(
				logo,
				&widget.Label{
					Text:      "Welcome to the Saab CIM Tool",
					Alignment: fyne.TextAlignCenter,
					TextStyle: fyne.TextStyle{Bold: true},
					SizeName:  theme.SizeNameHeadingText,
				},
				&widget.Label{
					Text:       "Pick a serial port in the toolbar, then press Read to dump your CIM eeprom,\nor press Open to inspect an existing bin file.",
					Alignment:  fyne.TextAlignCenter,
					Importance: widget.LowImportance,
				},
			),
		)),
	)
	m.log = widget.NewListWithData(
		m.logList,
		func() fyne.CanvasObject {
			return &widget.Label{
				TextStyle:  fyne.TextStyle{Monospace: true},
				Selectable: true,
			}
		},
		func(item binding.DataItem, obj fyne.CanvasObject) {
			i := item.(binding.String)
			txt, err := i.Get()
			if err != nil {
				panic(err)
			}
			if v, ok := obj.(*widget.Label); ok {
				v.SetText(txt)
			}
		},
	)

	message, ports, err := adapter.ListPorts()
	if err != nil {
		m.outputStr(err.Error())
	}
	if message != "" {
		m.outputStr(message)
	}

	m.rescanButton = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		message, ports, err := adapter.ListPorts()
		if err != nil {
			m.outputStr(err.Error())
			return
		}
		m.portList.Options = ports
		m.portList.Refresh()
		m.outputStr(message)
	})

	m.portList = &widget.Select{
		PlaceHolder: "Select port",
		Alignment:   fyne.TextAlignCenter,
		Options:     ports,
		Selected:    m.e.port,
		OnChanged: func(s string) {
			m.e.port = s
			m.e.Preferences().SetString("port", s)
		},
	}

	m.openButton = widget.NewButtonWithIcon("Open", theme.FolderOpenIcon(), m.viewClickHandler)
	m.readButton = widget.NewButtonWithIcon("Read", theme.DownloadIcon(), m.readClickHandler)
	m.readButton.Importance = widget.HighImportance
	m.writeButton = widget.NewButtonWithIcon("Write", theme.UploadIcon(), m.writeClickHandler)
	m.eraseButton = widget.NewButtonWithIcon("Erase", theme.DeleteIcon(), m.eraseClickHandler)
	m.eraseButton.Importance = widget.DangerImportance
	m.helpButton = widget.NewButtonWithIcon("Help", theme.HelpIcon(), func() {
		if m.hw == nil {
			m.hw = newHelpWindow(e)
		} else {
			m.hw.RequestFocus()
		}
	})
	m.copyButton = widget.NewButtonWithIcon("Copy log", theme.ContentCopyIcon(), func() {
		if content, err := m.logList.Get(); err == nil {
			m.Clipboard().SetContent(strings.Join(content, "\n"))
		}
	})
	m.clearButton = widget.NewButtonWithIcon("Clear log", theme.ContentClearIcon(), func() {
		m.logList.Set([]string{})
	})
	m.readMIUButton = widget.NewButtonWithIcon("Read MIU", theme.DownloadIcon(), func() {
		b, err := m.readMIU()
		if err != nil {
			dialog.ShowError(err, m.Window)
			return
		}

		m.output("%X", md5.Sum(b))

		if err := os.WriteFile(time.Now().Format("15_04_05")+".bin", b, 0644); err != nil {
			dialog.ShowError(err, m.Window)
		}
	})

	/*
		m.writeMIUButton = widget.NewButtonWithIcon("Write MIU", theme.UploadIcon(), func() {
			b, err := os.ReadFile("128_test.bin")
			if err != nil {
				dialog.ShowError(err, m.Window)
			}

			if err := m.writeMIU(m.e.port, b); err != nil {
				dialog.ShowError(err, m.Window)
			}
		})
	*/

	m.SetContent(m.layout())
	m.Resize(mainSize)
	m.SetMaster()
	m.Show()
	return m
}

func (m *mainWindow) layout() fyne.CanvasObject {
	logView := container.NewBorder(
		container.NewHBox(layout.NewSpacer(), m.copyButton, m.clearButton),
		nil, nil, nil,
		m.log,
	)

	m.logTab = container.NewTabItemWithIcon("Log", theme.ListIcon(), logView)
	m.settingsTab = container.NewTabItemWithIcon("Settings", theme.SettingsIcon(), newSettingsView(m.e))
	m.docTab.Append(m.logTab)
	m.docTab.Append(m.settingsTab)
	m.docTab.SelectIndex(0)

	toolbar := container.NewHBox(
		m.openButton,
		m.readButton,
		m.writeButton,
		m.eraseButton,
		m.helpButton,
		layout.NewSpacer(),
		m.portList,
		m.rescanButton,
		//m.readMIUButton,
		//m.writeMIUButton,
	)

	return container.NewBorder(
		container.NewVBox(toolbar, widget.NewSeparator()),
		container.NewVBox(widget.NewSeparator(), container.NewPadded(m.progressBar)),
		nil,
		nil,
		m.docTab,
	)
}

func (m *mainWindow) viewClickHandler() {
	m.openButton.Disable()
	go func() {
		defer fyne.Do(m.openButton.Enable)
		filename, err := openFileNative("Select file to view")
		if err != nil {
			m.output("%s", err.Error())
			return
		}
		if filename == "" {
			return
		}

		bin, err := cim.MustLoad(filename)
		if err != nil {
			fyne.Do(func() {
				dialog.ShowConfirm("File verification failed", fmt.Sprintf("File verification failed: %v. View anyway?", err), func(ok bool) {
					if ok {
						rawbin, err := os.ReadFile(filename)
						if err != nil {
							dialog.ShowError(err, m)
							return
						}
						m.docTab.Append(container.NewTabItemWithIcon(filepath.Base(filename), theme.FileIcon(), newViewerView(m.e, filename, rawbin, false)))
						m.docTab.SelectIndex(len(m.docTab.Items) - 1)
					}
				}, m)
			})
			return
		}
		b, err := bin.XORBytes()
		if err != nil {
			fyne.Do(func() { dialog.ShowError(err, m) })
			return
		}
		fyne.Do(func() {
			d := container.NewTabItemWithIcon(filepath.Base(filename), theme.FileIcon(), newViewerView(m.e, filename, b, false))
			m.docTab.Append(d)
			m.docTab.SelectIndex(len(m.docTab.Items) - 1)
		})
	}()
}

func (m *mainWindow) readClickHandler() {
	m.disableButtons()
	go func() {
		defer m.enableButtons()
		if m.e.port == "" {
			m.outputStr("Please select a port first")
			return
		}
		ignoreReadErrors, _ := m.e.ignoreError.Get()
		rawBytes, bin, err := m.readCIM()
		if err != nil {
			m.outputStr(err.Error())
			if err.Error() == "Timeout reading eeprom" {
				return
			}
			if ignoreReadErrors {
				m.saveFile("Save raw bin file", fmt.Sprintf("cim_raw_%s.bin", time.Now().Format("20060102-150405")), rawBytes)
			} else {
				fyne.Do(func() {
					m.docTab.Select(m.logTab)
					dialog.ShowConfirm("Error reading CIM", "There was errors reading, view anyway?", func(ok bool) {
						if ok {
							m.docTab.Append(container.NewTabItemWithIcon(fmt.Sprintf("Raw read at %s", time.Now().Format("15:04:05")), theme.FileIcon(), newViewerView(m.e, fmt.Sprintf("failed read from %s", time.Now().Format(time.RFC1123Z)), rawBytes, true)))
							m.docTab.SelectIndex(len(m.docTab.Items) - 1)
						}
					}, m)
				})
			}
			return
		}
		xorBytes, err := bin.XORBytes()
		if err != nil {
			fyne.Do(func() { dialog.ShowError(err, m) })
			return
		}
		fyne.Do(func() {
			m.docTab.Append(container.NewTabItemWithIcon(fmt.Sprintf("Read at %s", time.Now().Format("15:04:05")), theme.FileIcon(), newViewerView(m.e, fmt.Sprintf("successful read from %s", time.Now().Format(time.RFC1123Z)), xorBytes, true)))
			m.docTab.SelectIndex(len(m.docTab.Items) - 1)
		})
	}()
}

func (m *mainWindow) writeClickHandler() {
	if m.e.port == "" {
		m.output("Please select a port first")
		return
	}

	go func() {
		_, bin, err := loadFile()
		if err != nil {
			m.outputStr(err.Error())
			return
		}
		if bin == nil {
			return
		}
		fyne.Do(func() {
			dialog.ShowConfirm("Write to CIM?", "Continue writing to CIM?", func(ok bool) {
				if ok {
					start := time.Now()
					go func() {
						m.disableButtons()
						defer m.enableButtons()
						if err := m.writeCIM(bin); err != nil {
							fyne.Do(func() {
								dialog.ShowError(err, m)
								m.docTab.Select(m.logTab)
							})
							return
						}
						fyne.Do(func() {
							dialog.ShowInformation("Write done", fmt.Sprintf("Write successfull, took %s", time.Since(start).Round(time.Millisecond).String()), m)
						})
					}()
				}
			}, m)
		})
	}()
}

func (m *mainWindow) eraseClickHandler() {
	if m.e.port == "" {
		m.output("Please select a port first")
		return
	}
	dialog.ShowConfirm("Erase CIM?", "Continue erasing CIM?", func(ok bool) {
		if ok {
			go func() {
				m.disableButtons()
				defer m.enableButtons()

				start := time.Now()

				client := m.newAdapter()
				if err := client.Open(m.e.port, VERSION); err != nil {
					m.output("Failed to init adapter: %v", err)
					return
				}
				defer client.Close()

				m.output("Erasing ... ")
				if err := client.EraseCIM(); err != nil {
					m.outputStr(err.Error())
					return
				}
				m.output("Erase took %s", time.Since(start).String())
				fyne.Do(func() {
					dialog.ShowInformation("Erase done", fmt.Sprintf("Erase successfull, took %s", time.Since(start).Round(time.Millisecond).String()), m)
				})
			}()

		}
	}, m)
}

func (m *mainWindow) saveFile(title, suggestedFilename string, data []byte) bool {
	filename, err := saveFileNative(title, suggestedFilename)
	if err != nil {
		m.outputStr(err.Error())
		return false
	}
	if filename == "" {
		return false
	}
	filename = addSuffix(filename, ".bin")

	if err := os.WriteFile(filename, data, 0644); err == nil {
		m.output("Saved to %s", filename)
	} else {
		m.outputStr(err.Error())
		return false
	}
	return true
}

// loadFile returns nil data without error if the user cancelled.
func loadFile() (string, []byte, error) {
	filename, err := openFileNative("Load bin file")
	if err != nil || filename == "" {
		return "", nil, err
	}

	bin, err := os.ReadFile(filename)
	if err != nil {
		return "", nil, err
	}
	return filename, bin, nil
}

func addSuffix(s, suffix string) string {
	if !strings.HasSuffix(s, suffix) {
		return s + suffix
	}
	return s
}

func (m *mainWindow) outputStr(text string) {
	m.output("%s", text)
}

func (m *mainWindow) output(format string, values ...interface{}) {
	fyne.Do(func() {
		scanner := bufio.NewScanner(strings.NewReader(fmt.Sprintf(format, values...)))
		for scanner.Scan() {
			var text string
			if format != "" {
				text = fmt.Sprintf("%s - %s", time.Now().Format("15:04:05.000"), scanner.Text())
			}
			m.logList.Append(text)
			m.log.ScrollToBottom()
		}
	})
}

func (m *mainWindow) disableButtons() {
	fyne.Do(func() {
		m.rescanButton.Disable()
		m.portList.Disable()
		//m.openButton.Disable()
		m.readButton.Disable()
		m.writeButton.Disable()
		m.eraseButton.Disable()
	})
}

func (m *mainWindow) enableButtons() {
	fyne.Do(func() {
		m.rescanButton.Enable()
		m.readButton.Enable()
		m.portList.Enable()
		//m.openButton.Enable()
		m.readButton.Enable()
		m.writeButton.Enable()
		m.eraseButton.Enable()
	})
}
