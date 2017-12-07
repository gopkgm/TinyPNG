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
)

const (
	TPURL = "https://api.tinify.com/shrink"
)

var (
	auto string
)

type Config struct {
	ApiKey string `json:"api_key"`
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
	req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("api:"+auto)))
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
		//fmt.Printf("%+v", imgResp)
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
	f, err = os.Create(path + name)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func PrintE(err error) {
	if err != nil {
		fmt.Println(err.Error())
	}
}

func init() {
	config, e := getConfig()
	PrintE(e)
	auto = config.ApiKey
}

func main() {
	//src := ""
	//dst := ""

	s := scheduler.NewScheduler()
	s.Start()
	for i := 0; i < 5; i++ {
		func(i int) {
			s.Add(func() {
				println("aaaaaaaaaaaaaaaa", i)
			})
		}(i)
	}
	s.Wait()
}
