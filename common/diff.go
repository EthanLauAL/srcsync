package common

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Profile struct {
	Name string
	Size int64
	ModTime time.Time
	MD5 string
}

type Updates struct {
	Upd []string
	Del []string
}

type Index map[string]*Profile

func exists(filename string) bool {
	_,err := getProfile(filename, false)
	return !os.IsNotExist(err)
}

func getProfile(filename string, enabledMD5 bool) (*Profile,error) {
	info,err := os.Stat(filename)
		if err != nil { return nil,err }
	profile := &Profile{
		Name : info.Name(),
		Size : info.Size(),
		ModTime : info.ModTime(),
		MD5 : "",
	}

	if enabledMD5 {
		file, err := os.Open(filename)
			if err != nil { return nil,err }
		hasher := md5.New()
		_,err = io.Copy(hasher, file)
			if err != nil { file.Close() ; return nil,err }
		profile.MD5 = fmt.Sprintf("%x", hasher.Sum(nil))
	}

	return profile, nil
}

func getRopo() (files []string, err error) {
	cmd := exec.Command("git", "ls-files")
	r,err := cmd.StdoutPipe()
		if err != nil { return nil, err }
	err = cmd.Start()
		if err != nil { return nil, err }

	_files := make([]string, 0)
	err = ForEachLine(r,func(filename string){
		if filename == ".gitignore" { return }
		_files = append(_files, filename)
	})
	err = cmd.Wait()
		if err != nil { return nil, err }

	return _files, nil
}

func getFiles() (files []string, err error) {
	cmd := exec.Command("find", ".", "-type", "f")
	r,err := cmd.StdoutPipe()
		if err != nil { return nil, err }
	err = cmd.Start()
		if err != nil { return nil, err }

	_files := make([]string, 0)
	err = ForEachLine(r,func(filename string){
		filename = strings.TrimPrefix(filename, "./")
		_files = append(_files, filename)
	})
	err = cmd.Wait()
		if err != nil { return nil, err }

	return _files, nil
}

func getIndex(files []string, enabledMD5 bool) (Index,error) {
	index := make(Index)
	for _,filename := range files {
		profile,err := getProfile(filename, enabledMD5)
			if err != nil { return nil, err }
		index[filename] = profile
	}

	return index, nil
}

func ChdirRopoRoot() error {
	pathBytes, err := exec.Command(
		"git", "rev-parse", "--show-toplevel").Output()
		if err != nil { return err }
	path := strings.TrimSuffix(string(pathBytes),"\n")

	err = os.Chdir(path)
		if err != nil { return err }
	return nil
}

func GetRopoIndex(enabledMD5 bool) (Index, error) {
	files, err := getRopo()
		if err != nil { return nil, err }
	return getIndex(files, enabledMD5)
}

func MkdirAndChdir(path string) error {
	err := os.MkdirAll(path, os.ModePerm)
		if err != nil { return err }
	err = os.Chdir(path)
		if err != nil { return err }
	return err
}

func GetDiff(enabledMD5 bool, src Index) (*Updates, error) {
	files, err := getFiles()
	dst, err := getIndex(files, enabledMD5)
		if err != nil { return nil, err }

	upd := make([]string, 0)
	del := make([]string, 0)

	for filename,s := range src {
		if d,ok := dst[filename] ; ok {
			if d.Size == s.Size &&
				!d.ModTime.Before(s.ModTime) &&
				d.MD5 == s.MD5 {
				continue
			}
		}
		upd = append(upd, filename)
	}

	for filename,_ := range dst {
		if _,found := src[filename] ; !found {
			del = append(del, filename)
		}
	}

	return &Updates{Upd:upd, Del:del}, nil
}

