package apps

import (
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"fyne.io/fyne"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/cmd/fyne/commands"
	"fyne.io/fyne/container"
	"fyne.io/fyne/dialog"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/theme"
	"fyne.io/fyne/widget"
)

type Apps struct {
	shownID, shownPkg, shownIcon string
	name, summary, date          *widget.Label
	developer, version           *widget.Label
	link                         *widget.Hyperlink
	icon, screenshot             *canvas.Image
	TasksLock                    sync.Mutex
	TasksUpdated                 bool
	Tasks                        []*Task
}

type Task struct {
	status bool
	err    error
	msg    string
	Lock   sync.Mutex
}

func (w *Apps) loadAppDetail(app App) {
	w.shownID = app.ID
	w.shownPkg = app.Source.Package
	w.shownIcon = app.Icon

	w.name.SetText(app.Name)
	w.developer.SetText(app.Developer)
	w.version.SetText(app.Version)
	w.date.SetText(app.Date.Format("02 Jan 2006"))
	w.summary.SetText(app.Summary)

	w.icon.Resource = nil
	go setImageFromURL(w.icon, app.Icon)

	w.screenshot.Resource = nil
	if len(app.Screenshots) > 0 {
		go setImageFromURL(w.screenshot, app.Screenshots[0].Image)
	}
	w.screenshot.Refresh()

	parsed, err := url.Parse(app.Website)
	if err != nil {
		w.link.SetText("")
		return
	}
	w.link.SetText(parsed.Host)
	w.link.SetURL(parsed)
}

func setImageFromURL(img *canvas.Image, location string) {
	if location == "" {
		return
	}

	res, err := loadResourceFromURL(location)
	if err != nil {
		img.Resource = theme.WarningIcon()
	} else {
		img.Resource = res
	}

	canvas.Refresh(img)
}

func loadResourceFromURL(URL string) (fyne.Resource, error) {
	client := http.Client{
		Timeout: 1 * time.Second,
	}

	req, err := client.Get(URL)

	if err != nil {
		return nil, err
	}

	defer req.Body.Close()

	bytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	parsed, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}

	name := filepath.Base(parsed.Path)

	return fyne.NewStaticResource(name, bytes), nil
}

// iconHoverLayout specifies a layout that floats an icon image top right over other content.
type iconHoverLayout struct {
	content, icon fyne.CanvasObject
}

func (i *iconHoverLayout) Layout(_ []fyne.CanvasObject, size fyne.Size) {
	i.content.Resize(size)

	i.icon.Resize(fyne.NewSize(64, 64))
	i.icon.Move(fyne.NewPos(size.Width-i.icon.Size().Width, 0))
}

func (i *iconHoverLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return i.content.MinSize()
}

func (w *Apps) installer(win fyne.Window) func() {
	return func() {
		if w.shownPkg == "fyne.io/apps" {
			dialog.ShowInformation("System app", "Cannot overwrite the installer app", win)
			return
		}

		prog := dialog.NewProgressInfinite("Installing...", "Please wait while the app is installed", win)
		prog.Show()

		tmpIconChan := make(chan string)
		go func() {
			tmpIconChan <- downloadIcon(w.shownIcon)
		}()

		tmpIcon := <-tmpIconChan
		get := commands.NewGetter()
		get.SetIcon(tmpIcon)
		err := get.Get(w.shownPkg)

		prog.Hide()

		if err != nil {
			dialog.ShowError(err, win)
		} else {
			dialog.ShowInformation("Installed", "App was installed successfully :)", win)
		}

		err = os.Remove(tmpIcon)
		if err != nil {
			dialog.ShowError(err, win)
		}
	}
}

type taskShow struct {
	fyne.CanvasObject
	icon  *widget.Icon
	prog  *widget.ProgressBarInfinite
	label *widget.Label
}

func (a *Apps) tasks(win fyne.Window) func() {
	return func() {
		tasksList := widget.NewList(
			func() int {
				a.TasksLock.Lock()
				defer a.TasksLock.Unlock()
				return len(a.Tasks)
			},
			func() fyne.CanvasObject {
				icon := widget.NewIcon(theme.DownloadIcon())
				prog := widget.NewProgressBarInfinite()
				label := widget.NewLabel("Loading...")
				return taskShow{
					CanvasObject: fyne.NewContainerWithLayout(
						layout.NewHBoxLayout(),
						icon,
						prog,
						label,
					),
					icon:  icon,
					prog:  prog,
					label: label,
				}
			},
			func(id widget.ListItemID, object fyne.CanvasObject) {
				a.TasksLock.Lock()
				task := a.Tasks[id]
				a.TasksUpdated = false
				a.TasksLock.Unlock()
				show := object.(taskShow)
				task.Lock.Lock()

				// make sure nil pointer doesn't happen
				if show.prog == nil {
					show.prog = widget.NewProgressBarInfinite()
				}
				if task.status {
					show.prog.Start()
				} else {
					show.prog.Stop()
				}

				if show.icon == nil {
					show.icon = widget.NewIcon(theme.DownloadIcon())
				}
				if task.err == nil {
					show.icon.SetResource(theme.DownloadIcon())
				} else {
					show.icon.SetResource(theme.ErrorIcon())
				}

				if show.label == nil {
					show.label = widget.NewLabel("Loading...")
				}
				show.label.SetText(task.msg)
				task.Lock.Unlock()
			},
		)
		var tasks fyne.CanvasObject
		tasks = container.NewPadded(
			container.NewBorder(
				container.NewHBox(
					widget.NewLabel("Tasks"),
					layout.NewSpacer(),
					widget.NewButtonWithIcon("Close", theme.CancelIcon(), func() {
						tasks.Hide()
					}),
				),
				nil,
				nil,
				nil,
				tasksList,
			),
		)

		widget.ShowModalPopUp(tasks, win.Canvas())
	}
}

func NewApps(apps AppList, win fyne.Window) fyne.CanvasObject {
	a := new(Apps)
	a.name = widget.NewLabel("")
	a.developer = widget.NewLabel("")
	a.link = widget.NewHyperlink("", nil)
	a.summary = widget.NewLabel("")
	a.summary.Wrapping = fyne.TextWrapWord
	a.version = widget.NewLabel("")
	a.date = widget.NewLabel("")
	a.icon = &canvas.Image{}
	a.icon.FillMode = canvas.ImageFillContain
	a.screenshot = &canvas.Image{}
	a.screenshot.SetMinSize(fyne.NewSize(320, 240))
	a.screenshot.FillMode = canvas.ImageFillContain
	a.Tasks = []*Task{
		{
			status: false,
			err:    nil,
			msg:    "task 1",
		},
		{
			status: true,
			err:    nil,
			msg:    "task 2",
		},
	}

	dateAndVersion := fyne.NewContainerWithLayout(layout.NewGridLayout(2), a.date,
		widget.NewForm(&widget.FormItem{Text: "Version", Widget: a.version}))

	form := widget.NewForm(
		&widget.FormItem{Text: "Name", Widget: a.name},
		&widget.FormItem{Text: "Developer", Widget: a.developer},
		&widget.FormItem{Text: "Website", Widget: a.link},
		&widget.FormItem{Text: "Summary", Widget: a.summary},
		&widget.FormItem{Text: "Date", Widget: dateAndVersion},
	)

	details := fyne.NewContainerWithLayout(&iconHoverLayout{content: form, icon: a.icon}, form, a.icon)

	list := widget.NewList(
		func() int {
			return len(apps)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("A longish app name")
		},
		func(id int, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(apps[id].Name)
		},
	)
	list.OnSelected = func(id int) {
		a.loadAppDetail(apps[id])
	}

	buttons := container.NewHBox(
		widget.NewButton("Tasks", a.tasks(win)),
		layout.NewSpacer(),
		widget.NewButton("Install", a.installer(win)),
	)

	if len(apps) > 0 {
		a.loadAppDetail(apps[0])
	}
	content := container.NewBorder(details, nil, nil, nil, a.screenshot)
	return container.NewBorder(nil, nil, list, nil,
		container.NewBorder(nil, buttons, nil, nil, content))
}

func downloadIcon(url string) string {
	req, err := http.Get(url)
	if err != nil {
		fyne.LogError("Failed to access icon url: "+url, err)
		return ""
	}

	tmp, err := ioutil.TempFile(os.TempDir(), "fyne-icon-*.png")
	if err != nil {
		fyne.LogError("Failed to create temporary file", err)
		return ""
	}
	defer tmp.Close()

	data, err := ioutil.ReadAll(req.Body)

	if err != nil {
		fyne.LogError("Failed tread icon data", err)
		return ""
	}

	_, err = tmp.Write(data)
	if err != nil {
		fyne.LogError("Failed to get write icon to: "+tmp.Name(), err)
		return ""
	}

	return tmp.Name()
}
