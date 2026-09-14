package main

import (
	_ "embed"
	"log"

	"fyne.io/fyne/v2/app"
	"github.com/roffe/cimtool/gui"
)

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
}

func main() {
	application := app.New()

	application.Settings().SetTheme(&gui.MyTheme{})
	ui, err := gui.New(application)
	if err != nil {
		log.Fatal(err)
	}
	//application.Lifecycle().SetOnStarted(ui.CheckUpdate)
	//application.Run()
	ui.Run()
}
