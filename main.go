package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"srcsync/common"
)

const (
	confFilename = ".remote.srcsync"
)

var (
	EnabledMD5 bool
)

func main() {
	flag.BoolVar(&EnabledMD5,"md5",false,"Enabled MD5 checking.")
	flag.Parse()

	//切换目录到当前git库的根目录
	err := common.ChdirRopoRoot()
		if err != nil { panic(err) }

	//加载配置
	confFile,err := os.Open(confFilename)
		if err != nil { panic(err) }
	defer confFile.Close()
	var conf struct {
		Url string
		Path string
		MD5 bool
	}
	err = json.NewDecoder(confFile).Decode(&conf)
		if err != nil { panic(err) }
	if conf.Url == "" { log.Fatal("'Url' missed in ",confFilename) }
	if conf.Path == "" { log.Fatal("'Path' missed in ",confFilename) }
	log.Print("Sync to: ", conf.Url, ":", conf.Path)

	//获取文件列表
	index,err := common.GetRopoIndex(EnabledMD5 || conf.MD5)
		if err != nil { panic(err) }

	//发起比较请求
	reqBytes,err := json.Marshal(&common.DiffRequest{
		ServerPath : conf.Path,
		MD5 : EnabledMD5 || conf.MD5,
		Index : index,
	})
		if err != nil { panic(err) }
	reqDiff,err := http.NewRequest("POST", conf.Url+"/diff",
		bytes.NewReader(reqBytes))
		if err != nil { panic(err) }
	respDiff,err := http.DefaultClient.Do(reqDiff)
		if err != nil { panic(err) }
	if respDiff.StatusCode != http.StatusOK { log.Fatal(respDiff.Status) }

	//获取更新文件列表
	var diffResp common.DiffResponse
	err = json.NewDecoder(respDiff.Body).Decode(&diffResp)
		if err != nil { panic(err) }
	respDiff.Body.Close()

	//准备上传数据
	reqReader,reqWriter := io.Pipe()
	mw := common.NewMultipartWriter(reqWriter)
	if len(diffResp.Upd) > 0 {
		log.Print("Updating ", len(diffResp.Upd), " files")
	}
	go func() {
		for _,filename := range diffResp.Upd {
			log.Print("Uploading ", filename, "...")
			file,err := os.Open(filename)
				if err != nil { panic(err) }
			filebytes,err := ioutil.ReadAll(file)
				if err != nil { panic(err) }
			mw.WritePart(filebytes)
			file.Close()
		}
		reqWriter.Close()
	}()

	//发起上传请求
	reqUpdate,err := http.NewRequest("POST",
		conf.Url+"/update?sessid="+diffResp.SessionID, reqReader)
		if err != nil { panic(err) }
	respUpdate,err := http.DefaultClient.Do(reqUpdate)
		if err != nil { panic(err) }
	if respUpdate.StatusCode != http.StatusOK {
		log.Fatal(respUpdate.Status) }

	//获取成功确认
	var updateResp common.UpdateResponse
	err = json.NewDecoder(respUpdate.Body).Decode(&updateResp)
		if err != nil { panic(err) }
	log.Print("done")
}
