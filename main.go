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
)

const (
	TPURL = "https://api.tinify.com/shrink"
)

var (
	auth      []string
	authIndex = 0
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
	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return err
	}
	fmt.Printf(printInfo(filepath.Base(name)))
	return nil
}

func printInfo(name string) string {
	if len(name) < 36 {
		return name + " " + strings.Repeat("-", 36-len(name)) + " Done\n"
	} else {
		return name[:30] + "... --- Done\n"
	}
}

func PrintE(err error) {
	if err != nil {
		fmt.Fprint(os.Stderr, "error:"+err.Error()+"\n")
	}
}

func init() {
	config, e := getConfig()
	PrintE(e)
	auth = config.ApiKey
}

func main() {
	if len(os.Args) == 3 {
		src := os.Args[1]
		dst := os.Args[2]
		srcInfo, e := os.Stat(src)
		var dstInfo os.FileInfo
		dstInfo, e = os.Stat(dst)
		PrintE(e)
		if dstInfo.IsDir() {
			fmt.Println("---Start---\n")
			if srcInfo.IsDir() {
				s := scheduler.NewScheduler()
				s.Start()
				filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
					if ".png" == filepath.Ext(info.Name()) || ".jpg" == filepath.Ext(info.Name()) {
						s.Add(func() {
							compress(path, dst)
						})
					}
					return nil
				})
				s.Wait()
			} else {
				compress(src, dst)
			}
			fmt.Print("\n----End----")
			os.Exit(0)
		}
		os.Exit(1)
	}
	os.Exit(2)
}
