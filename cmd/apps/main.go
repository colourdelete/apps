package main

import (
	"fyne.io/apps/pkg/apps"
	"fyne.io/fyne"
	app2 "fyne.io/fyne/app"
	"log"
	"time"
)

func main() {
	app := app2.NewWithID("io.fyne.apps")
	win := app.NewWindow("Fyne Applications")
	timeout := app.Preferences().String("http-timeout")
	d, err := time.ParseDuration(timeout)
	if err != nil {
		log.Printf("failed to parse settings http-timeout, falling back to 1 second: %s", err)
		d = 1 * time.Second
	}
	rawAppList, err := apps.LoadAppListFromWeb(d)
	if err != nil {
		log.Printf("failed to get app list from web, getting from cache: %s", err)
		rawAppList, err = apps.LoadAppListFromCache()

		if err != nil {
			fyne.LogError("Cache load error", err)
			return
		}
	}

	defer rawAppList.Close()

	appList, err := apps.ParseAppList(rawAppList)
	if err != nil {
		fyne.LogError("Parse error", err)
		return
	}

	win.SetContent(apps.NewApps(appList, win))
	x, y := app.Preferences().Int("window-x"), app.Preferences().Int("window-y")
	win.Resize(fyne.NewSize(x, y))

	win.ShowAndRun()
}
