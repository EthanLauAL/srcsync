package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path"
	"runtime"
	"sync"
	"time"
	"srcsync/common"
)

var (
	authUsername, authPassword string
)

var (
	sessionMtx sync.Mutex
	sessionID string
	updates *common.Updates
)

func main() {
	flag.StringVar(&authUsername, "u", "", "username")
	flag.StringVar(&authPassword, "p", "", "password")
	flag.Parse()
	if flag.Arg(0) == "" {
		fmt.Println("Usage:", os.Args[0], "[options...] <listen>")
		return
	}
	rand.Seed(time.Now().Unix())
	http.HandleFunc("/diff", handleDiff)
	http.HandleFunc("/update", handleUpdate)
	http.HandleFunc("/stacks", handleStacks)
	log.Print("Listening: ", flag.Arg(0))
	log.Fatal(http.ListenAndServe(flag.Arg(0), nil))
}

func handleDiff(w http.ResponseWriter, r *http.Request) {
	sessionMtx.Lock()
	defer sessionMtx.Unlock()

	log.Print("Diff: ", r.RemoteAddr)
	if !commonCheck(w,r) { return }

	req := new(common.DiffRequest)
	err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	log.Print("Comparing local path: ", req.ServerPath)
	err = common.MkdirAndChdir(req.ServerPath)
		if err != nil {
			log.Print(err)
			http.Error(w, err.Error(),
				http.StatusInternalServerError)
			return
		}
	updates,err = common.GetDiff(req.MD5, req.Index)
		if err != nil {
			log.Print(err)
			http.Error(w, err.Error(),
				http.StatusInternalServerError)
			return
		}
	sessionID = fmt.Sprintf("%016x", rand.Int63())
	err = json.NewEncoder(w).Encode(&common.DiffResponse{
		SessionID : sessionID,
		Upd : updates.Upd,
	})
	if err != nil { log.Print(err) }
}

func handleUpdate(w http.ResponseWriter, r *http.Request) {
	sessionMtx.Lock()
	defer sessionMtx.Unlock()

	log.Print("Updating: ", r.RemoteAddr)
	if !commonCheck(w,r) { return }

	values := r.URL.Query()
	if len(values["sessid"]) == 0 {
		http.Error(w, "'sessid' argument required",
			http.StatusBadRequest)
		return
	}
	if values["sessid"][0] != sessionID {
		http.Error(w, "Invalid 'sessid'",
			http.StatusBadRequest)
		return
	}
	sessionID = ""

	log.Print("Updating: ", len(updates.Upd),
		" Deletion:", len(updates.Del))

	mr := common.NewMultipartReader(r.Body)
	for _,filename := range updates.Upd {
		log.Print("Fetching ", filename)
		filereader,err := mr.NextPart()
			if err != nil { panic(err) }
		bytes,err := ioutil.ReadAll(filereader)
			if err != nil { panic(err) }
		err = os.MkdirAll(path.Dir(filename), os.ModeDir | os.ModePerm)
			if err != nil { panic(err) }
		file,err :=  os.Create(filename)
			if err != nil { panic(err) }
		_,err = file.Write(bytes)
			if err != nil { panic(err) }
		file.Close()
	}

	for _,filename := range updates.Del {
		err := os.Remove(filename)
			if err != nil { panic(err) }
		log.Print(filename, " deleted.")
	}

	log.Print("done")
	err := json.NewEncoder(w).Encode(&common.UpdateResponse{true})
	if err != nil { log.Print(err) }
}

func handleStacks(w http.ResponseWriter, r *http.Request) {
	if !commonCheck(w, r) {
		return
	}

	buf := make([]byte, 1<<20)
	n := runtime.Stack(buf, true)
	w.Write(buf[0:n])
}

func commonCheck(w http.ResponseWriter, r *http.Request) bool {
	user,pass,_ := r.BasicAuth()
		if user != authUsername || pass != authPassword {
			httpError(w, http.StatusUnauthorized)
			return false
		}
	return true
}

func httpError(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}

