package main

import (
	"encoding/json"
	"fmt"
	"image/jpeg"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

type VersionInfo struct {
	Directory string
	Camera    string
	Latest    string
}

func (vi *VersionInfo) Save() {
	filename := path.Join(vi.Directory, "version.json")
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		httplogger.Warningf("Failed to open version file: %s", err.Error())
		return
	}
	defer f.Close()
	data, err := json.Marshal(vi)
	if err != nil {
		httplogger.Warningf("Failed to marshal version json file: %s", err.Error())
		return
	}
	_, err = f.Write(data)
	if err != nil {
		httplogger.Warningf("Failed to write to version file: %s", err.Error())
		return
	}
}

func (vi *VersionInfo) Load(dir string) error {
	filename := path.Join(dir, "version.json")
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		httplogger.Warningf("Failed to read version file: %s", err.Error())
		return err
	}
	err = json.Unmarshal(data, vi)
	if err != nil {
		httplogger.Warningf("Failed to unmarshal version json file: %s", err.Error())
		return err
	}
	return nil
}

var HttpClient = &http.Client{
	Timeout: time.Second * 60,
}

func ensureDir(path string) error {
	logger.Tracef("Ensuring dir '%s' exists", path)
	if !strings.Contains(path, string(os.PathSeparator)) {
		logger.Tracef("Path doesn't contain a %c, bailing", os.PathSeparator)
		return nil
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		logger.Tracef("Path doesn't exist. Attempting to create")
		err = os.MkdirAll(path, os.ModePerm)
		return err
	}
	logger.Tracef("Path exists already")
	return nil
}

func getImage(dir string, cam *Camera) error {
	var filename string
	var filepath string
	var auth *CameraAuth
	auth = cam.Auth
	filepath = path.Join(dir, cam.Name)
	// "%Y%m%d-%H%M%S"
	format := "20060102-150405MST"
	filename = fmt.Sprintf("%s.jpg", time.Now().Format(format))
	var version *VersionInfo = &VersionInfo{filepath, cam.Name, filename}
	filename = path.Join(filepath, filename)
	if err := ensureDir(filepath); err != nil {
		return err
	}
	if cam.SaveTo != "" {
		tmpstr, _ := path.Split(cam.SaveTo)
		if err := ensureDir(tmpstr); err != nil {
			return err
		}
	}
	if auth != nil {
		httplogger.Warningf("[%s] Found Auth, Not implemented, Bailing!", cam.Name)
		return nil

	} else {
		httplogger.Tracef("[%s] Initiating request to %s", cam.Name, cam.URL)
		response, err := HttpClient.Get(cam.URL)
		httplogger.Tracef("[%s] got image from %s", cam.Name, cam.URL)
		if err != nil {
			return err
		}
		httplogger.Tracef("[%s] Saving image from %s", cam.Name, cam.URL)
		fp, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer fp.Close()
		_, err = io.Copy(fp, response.Body)
		response.Body.Close()
		if err != nil {
			return err
		}
		fp.Seek(0, 0)
		err = verifyImageIntegrity(fp)
		if err != nil {
			return err
		}
		if cam.SaveTo != "" {
			httplogger.Tracef("[%s] Saving image to %s", cam.Name, cam.SaveTo)
			fp2, err := os.OpenFile(cam.SaveTo, os.O_RDWR|os.O_CREATE, 0644)
			if err != nil {
				return err
			}
			defer fp2.Close()
			fp.Seek(0, 0)
			io.Copy(fp2, fp)
			httplogger.Infof("[%s] Saved image to %s", cam.Name, cam.SaveTo)
		}
		version.Save()
		httplogger.Infof("[%s] Saved image to %s", cam.Name, filename)
	}
	return nil
}

func verifyImageIntegrity(file io.Reader) error {
	_, err := jpeg.Decode(file)
	if err != nil {
		return err
	}
	return nil
}
