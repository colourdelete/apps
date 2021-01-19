package apps

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type App struct {
	ID, Name, Icon     string
	Developer, Summary string
	URL, Website       string
	Screenshots        []AppScreenshot

	Date    time.Time
	Version string

	Source   AppSource
	Requires string
}

type AppScreenshot struct {
	Image, Type string
}

type AppSource struct {
	Git, Package string
}

type AppList []App

func ParseAppList(reader io.Reader) (AppList, error) {
	decode := json.NewDecoder(reader)

	appList := AppList{}
	err := decode.Decode(&appList)

	if err != nil {
		return nil, err
	}

	appList = appList.filterCompatible()
	sort.Slice(appList, func(a, b int) bool {
		return appList[a].Name < appList[b].Name
	})

	return appList, nil
}

func LoadAppListFromWeb(Timeout time.Duration) (io.ReadCloser, error) {
	client := http.Client{
		Timeout: Timeout,
	}

	req, err := client.Get("https://apps.fyne.io/api/v1/list.json")

	if err != nil || (req != nil && req.StatusCode != 200) {
		return nil, err
	}
	return req.Body, nil
}

func LoadAppListFromCache() (io.ReadCloser, error) {
	res, err := os.Open(filepath.Join("testdata", "apps-list.json"))
	if err != nil {
		return nil, err
	}

	return res, nil
}
