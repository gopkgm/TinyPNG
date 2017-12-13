package main

import (
	"io/ioutil"
	"encoding/json"
	"errors"
	"net/http"
	"bytes"
	"fmt"
	"encoding/base64"
	"os"
	"io"
	"TinyPNG/scheduler"
	"strings"
	"path/filepath"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/labstack/gommon/log"
)

const (
	TPURL     = "https://api.tinify.com/shrink"
	NO_CONFIG = `error:no config file
create config.json
{
    "api_key": [
        "Get Your API key from https://tinypng.com/developers"
    ]
}
`
)

var (
	auth      []string
	authIndex = 0
	outTE     *walk.TextEdit
)

type Config struct {
	ApiKey []string `json:"api_key"`
}

type ImgResp struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Input struct {
		Size int    `json:"size"`
		Type string `json:"type"`
	} `json:"input"`
	Output struct {
		Size   int     `json:"size"`
		Type   string  `json:"type"`
		Width  int     `json:"width"`
		Height int     `json:"height"`
		Ratio  float32 `json:"ratio"`
		Url    string  `json:"url"`
	} `json:"output"`
}

func getConfig() (*Config, error) {
	bytes, e := ioutil.ReadFile("config.json")
	config := &Config{}
	if e != nil {
		return config, errors.New("file error")
	}
	json.Unmarshal(bytes, config)
	return config, nil
}

func compress(file, path string) error {
	if len(auth) <= 0 {
		return errors.New("no api_key")
	}
	client := &http.Client{}
	f, err := os.Open(file)
	fName := f.Name()
	if err != nil {
		return err
	}
	defer f.Close()
	var imgByte []byte
	imgByte, err = ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	var req *http.Request
	req, err = http.NewRequest("POST", TPURL, bytes.NewReader(imgByte))
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("api:"+auth[authIndex])))
	resp, err := client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
		var bytes []byte
		bytes, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		imgResp := &ImgResp{}
		json.Unmarshal(bytes, imgResp)
		//fmt.Printf("%+v\n", imgResp)
		if len(imgResp.Error) > 0 {
			if strings.Contains(strings.ToLower(imgResp.Error), "unauthorized") {
				authIndex++
				if len(auth) > authIndex {
					compress(file, path)
				} else {
					return errors.New("api_key invalid")
				}
			}
			return errors.New(imgResp.Error)
		}
		err = download(imgResp.Output.Url, fName, path)
		if err != nil {
			return err
		}
	}
	return nil
}

func download(url, name, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	var f *os.File
	suffix := strings.HasSuffix(path, string(filepath.Separator))
	if !suffix {
		path = path + string(filepath.Separator)
	}
	f, err = os.Create(path + filepath.Base(name))
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return err
	}
	printProgress(printInfo(filepath.Base(name), "done"))
	return nil
}

func printProgress(s string) {
	if len(os.Args) == 3 {
		fmt.Println(s)
	} else if outTE != nil {
		outTE.SetText(outTE.Text() + s + "\r\n")
	}
}

func printInfo(name, status string) string {
	if len(name) < 36 {
		return name + " " + strings.Repeat("-", 36-len(name)) + " " + status
	} else {
		return name[:30] + "... --- " + status
	}
}

func PrintE(err error) {
	if err != nil {
		fmt.Fprint(os.Stderr, "error:"+err.Error()+"\n")
	}
}

func initConfig() error {
	config, e := getConfig()
	if e != nil {
		fmt.Fprint(os.Stderr, NO_CONFIG)
		return e
	}
	auth = config.ApiKey
	return nil
}

func doCompress(src, dst string) {
	var e error
	var srcInfo os.FileInfo
	srcInfo, e = os.Stat(src)
	var dstInfo os.FileInfo
	dstInfo, e = os.Stat(dst)
	PrintE(e)
	if dstInfo.IsDir() {
		printProgress("---Start---\r\n")
		if srcInfo.IsDir() {
			s := scheduler.NewScheduler()
			s.Start()
			filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
				if ".png" == filepath.Ext(info.Name()) || ".jpg" == filepath.Ext(info.Name()) {
					s.Add(func() {
						e := compress(path, dst)
						if e != nil {
							printProgress(printInfo(filepath.Base(path), "failed"))
						}
					})
				}
				return nil
			})
			s.Wait()
		} else {
			compress(src, dst)
		}
		printProgress("\r\n----End----")
		if len(os.Args) == 3 {
			os.Exit(0)
		}
	}
	if len(os.Args) == 3 {
		os.Exit(1)
	}
}

func main() {
	e := initConfig()
	if e != nil {
		return
	}
	if len(os.Args) == 3 {
		src := os.Args[1]
		dst := os.Args[2]
		doCompress(src, dst)
	} else {
		gui()
	}
}

type MyMainWindow struct {
	*walk.MainWindow
}

func (mw *MyMainWindow) openFolder(prevFilePath string) (string, error) {
	diaolg := new(walk.FileDialog)
	diaolg.FilePath = prevFilePath
	//diaolg.Filter = "Image Files (*.emf;*.bmp;*.exif;*.gif;*.jpeg;*.jpg;*.png;*.tiff)|*.emf;*.bmp;*.exif;*.gif;*.jpeg;*.jpg;*.png;*.tiff"
	diaolg.Title = "Select"
	if ok, err := diaolg.ShowBrowseFolder(mw); err != nil {
		return "", err
	} else if !ok {
		return "", nil
	}
	return diaolg.FilePath, nil
}

func (mw *MyMainWindow) aboutAction() {
	walk.MsgBox(mw, "About", "TinyPNGHelper", walk.MsgBoxIconInformation)
}

func gui() {
	var srcLE, dstLE *walk.LineEdit
	var mw = new(MyMainWindow)
	icon, _ := walk.NewIconFromFile("./img/icon.ico")
	MainWindow{
		AssignTo: &mw.MainWindow,
		Icon:     icon,
		Title:    "TinyPNGHelper",
		MinSize:  Size{600, 400},
		Layout:   VBox{},
		OnDropFiles: func(strings []string) {
			if len(strings) > 0 {
				f := strings[0]
				info, _ := os.Stat(f)
				if !info.IsDir() {
					if ".png" != filepath.Ext(f) && ".jpg" != filepath.Ext(f) {
						return
					}
				}
				if srcLE.Focused() {
					srcLE.SetText(f)
				} else if dstLE.Focused() {
					dstLE.SetText(f)
				}
			}
		},
		Children: []Widget{
			Composite{
				Layout: HBox{},
				Children: []Widget{
					Label{Text: "    源文件: "},
					LineEdit{AssignTo: &srcLE},
					PushButton{
						Text: "浏览",
						OnClicked: func() {
							s, e := mw.openFolder(srcLE.Text())
							if e != nil {
								log.Error(e.Error())
								return
							}
							srcLE.SetText(s)
						},
					},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					Label{Text: "目标文件: "},
					LineEdit{AssignTo: &dstLE},
					PushButton{
						Text: "浏览",
						OnClicked: func() {
							s, e := mw.openFolder(dstLE.Text())
							if e != nil {
								log.Error(e.Error())
								return
							}
							dstLE.SetText(s)
						},
					},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					TextEdit{
						AssignTo: &outTE,
						ReadOnly: true,
						MinSize:  Size{400, 200},
					},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					PushButton{
						Text:    "操作",
						MaxSize: Size{100, 50},
						OnClicked: func() {
							go doCompress(srcLE.Text(), dstLE.Text())
						},
					},
					PushButton{
						Text:    "清空日志",
						MaxSize: Size{100, 50},
						OnClicked: func() {
							outTE.SetText("")
						},
					},
					PushButton{
						Text:    "关于",
						MaxSize: Size{100, 50},
						OnClicked: func() {
							mw.aboutAction()
						},
					},
				},
			},
		},
	}.Run()
	srcLE.SetFocus()
}
