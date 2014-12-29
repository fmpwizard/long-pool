package main

import (
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"time"
)

//messageStore 's key is sessionId + cometId'
var messageStore = struct {
	sync.RWMutex
	m map[string][]message
}{m: make(map[string][]message)}

//cometStore 's key is sessionId'
var cometStore = struct {
	sync.RWMutex
	m map[string]comet
}{m: make(map[string]comet)}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	http.HandleFunc("/index", home)
	http.HandleFunc("/add", addMessage)
	http.HandleFunc("/comet", handleComet)
	log.Println("starting on :7070")
	log.Fatalf("failed to start %s", http.ListenAndServe(":7070", nil))
}

func home(rw http.ResponseWriter, req *http.Request) {
	t := template.New("index.html")
	t, err := t.ParseFiles("index.html")
	cookie, err := req.Cookie("gsessionid")
	if err == http.ErrNoCookie {
		rand.Seed(time.Now().UnixNano())
		sess := strconv.FormatFloat(rand.Float64(), 'f', 20, 64)
		cookie = &http.Cookie{
			Name:    "gsessionid",
			Value:   sess,
			Path:    "/",
			Expires: time.Now().Add(60 * time.Hour),
		}
		http.SetCookie(rw, cookie)
	}
	var cometId string
	var index uint64
	rw.Header().Add("Content-Type", "text/html; charset=UTF-8")
	cometStore.RLock()
	cometVal, found := cometStore[cookie.Value]
	cometStore.RUnlock()
	if found {
		cometId = cometVal.Value
	} else {
		//create comet for the first time
		rand.Seed(time.Now().UnixNano())
		cometId = strconv.FormatFloat(rand.Float64(), 'f', 20, 64)
		cometStore[cookie.Value] = comet{cometId, 0, time.Now()}
	}

	err = t.ExecuteTemplate(rw, "index.html", CometInfo{cometId, index})
	if err != nil {
		log.Fatalf("got error: %s", err)
	}

}

func handleComet(rw http.ResponseWriter, req *http.Request) {
	currentComet := req.FormValue("cometid")
	currentIndex, _ := strconv.ParseUint(req.FormValue("index"), 10, 64)
	cookie, _ := req.Cookie("gsessionid")
	log.Printf("comet id %s\n", currentComet)
	log.Printf("session id %s\n", cookie.Value)
	messageStore.RLock()
	messages, found := messageStore[cookie.Value+currentComet]
	messageStore.RUnlock()
	if found {
		cometStore[cookie.Value] = comet{currentComet, time.Now()} //update timestamp on comet
		for _, msg := range messages {
			if currentIndex < msg.Index && currentIndex != 0 {
				log.Println("sending message")
				fmt.Fprintf(rw, "here we are "+msg.Value+" on "+msg.stamp.Format("Jan 2, 2006 at 3:04pm (EST)"))
			} else {
				log.Printf("not sending message %+v\n", msg)
			}

		}
	} else {
		log.Printf("Didn't find comet message\n")
	}

}

func addMessage(rw http.ResponseWriter, req *http.Request) {
	currentComet := req.FormValue("cometid")
	cookie, _ := req.Cookie("gsessionid")
	messageStore.Lock()
	messageStore[cookie.Value+currentComet] = append(messageStore[cookie.Value+currentComet], message{1, "Hi", time.Now()})
	sort.Sort(ByIndex(messageStore))
	messageStore.Unlock()
	fmt.Fprintf(rw, "Added a message")
}

type message struct {
	Index uint64
	Value string
	stamp time.Time
}

type ByIndex []message

func (a ByIndex) Len() int           { return len(a) }
func (a ByIndex) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByIndex) Less(i, j int) bool { return a[i].Index < a[j].Index }

type comet struct {
	Value     string
	LastIndex uint64
	LastSeen  time.Time
}

type CometInfo struct {
	CometId string
	Index   uint64
}
