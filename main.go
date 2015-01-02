package main

import (
	"encoding/json"
	_ "expvar"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"
)

//messageStore 's key is sessionId + cometId'
var messageStore = struct {
	sync.RWMutex
	LastIndex uint64
	m         map[sessionCometKey][]message
}{m: make(map[sessionCometKey][]message)}

//cometStore 's key is sessionId'
var cometStore = struct {
	sync.RWMutex
	m map[session]comet
}{m: make(map[session]comet)}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	http.HandleFunc("/index", home)
	http.HandleFunc("/add", addMessage)
	http.HandleFunc("/comet", handleComet)
	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("js"))))
	log.Println("starting on :7070")
	go gc()
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
	cometVal, found := cometStore.m[session(cookie.Value)]
	cometStore.RUnlock()
	if found {
		cometId = cometVal.Value
	} else {
		//create comet for the first time
		rand.Seed(time.Now().UnixNano())
		cometId = strconv.FormatFloat(rand.Float64(), 'f', 20, 64)
		cometStore.Lock()
		cometStore.m[session(cookie.Value)] = comet{cometId, time.Now()}
		cometStore.Unlock()
	}

	messageStore.RLock()
	_, found = messageStore.m[sessionCometKey(cookie.Value+cometId)]
	lastId := messageStore.LastIndex
	messageStore.RUnlock()
	if found {
		index = lastId
	}

	err = t.ExecuteTemplate(rw, "index.html", CometInfo{cometId, index})
	if err != nil {
		log.Fatalf("got error: %s", err)
	}

}

func handleComet(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("Content-Type", "application/json")
	currentComet := req.FormValue("cometid")
	currentIndex, _ := strconv.ParseUint(req.FormValue("index"), 10, 64)
	cookie, _ := req.Cookie("gsessionid")
	cometStore.Lock()
	cometStore.m[session(cookie.Value)] = comet{currentComet, time.Now()} //update timestamp on comet
	cometStore.Unlock()
	var chanMessages = make(chan Responses)
	tick := time.NewTicker(500 * time.Millisecond)
	key := sessionCometKey(cookie.Value + currentComet)
	go func() {
		for _ = range tick.C {
			getMessages(key, currentIndex, chanMessages)
			log.Printf("started ticking %v\n", currentIndex)
		}
		return
	}()

	select {
	case messages := <-chanMessages:
		tick.Stop()
		fmt.Fprint(rw, messages)
		return
	case <-time.After(time.Second * 5):
		tick.Stop()
		fmt.Fprint(rw, Responses{[]Response{Response{Value: "", Error: ""}}, currentIndex})
		return
	}

}

func getMessages(key sessionCometKey, currentIndex uint64, result chan Responses) {
	var lastId uint64
	messageStore.RLock()
	messages, found := messageStore.m[key]
	lastId = messageStore.LastIndex
	messageStore.RUnlock()
	if found {
		var payload Responses
		for _, msg := range messages {
			if currentIndex < msg.index {
				payload.Res = append(payload.Res, Response{"here we are " + msg.Value + " on " + msg.Stamp.Format("Jan 2, 2006 at 3:04pm (EST)"), ""})
			} else {
				log.Printf("not sending message %+v\n", msg)
			}
		}
		if len(payload.Res) > 0 {
			payload.LastIndex = lastId
			result <- payload
		}
	}
}

func addMessage(rw http.ResponseWriter, req *http.Request) {
	data := req.FormValue("data")
	currentComet := req.FormValue("cometid")
	cookie, _ := req.Cookie("gsessionid")
	messageStore.Lock()
	messageStore.LastIndex++
	messageStore.m[sessionCometKey(cookie.Value+currentComet)] = append(messageStore.m[sessionCometKey(cookie.Value+currentComet)], message{messageStore.LastIndex, data, time.Now()})
	messageStore.Unlock()
	log.Printf("NumGoroutine %v\n", runtime.NumGoroutine())
	fmt.Fprintf(rw, "Added a message")
}

func gc() {
	for _ = range time.Tick(5 * time.Second) {
		log.Println("Started gc")
		start := time.Now()
		messageStore.Lock()
		for storeKey, messages := range messageStore.m {
			var temp []message
			for key, message := range messages {
				if message.Stamp.Sub(time.Now()) > 10*time.Second {
					temp = append(messages[:key], messages[key+1:]...)
				}
			}
			messageStore.m[storeKey] = temp
		}
		messageStore.Unlock()
		log.Printf("Ended gc. It took %v ms\n", time.Now().Sub(start).Nanoseconds()/1000)
	}
}

type message struct {
	index uint64
	Value string
	Stamp time.Time
}

type comet struct {
	Value    string
	LastSeen time.Time
}

type CometInfo struct {
	CometId string
	Index   uint64
}

type Response struct {
	Value string
	Error string
}

type Responses struct {
	Res       []Response
	LastIndex uint64
}

func (r Responses) String() string {
	b, err := json.Marshal(r)
	if err != nil {
		return ""
	}
	return string(b)
}

type sessionCometKey string

type session string

//http://127.0.0.1:7070/index
//http://127.0.0.1:7070/add?data=Diego+was+here+1&cometid=0.89506036657873655482
//http://127.0.0.1:7070/comet?index=3&cometid=0.89506036657873655482
