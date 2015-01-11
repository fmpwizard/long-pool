package main

import (
	"encoding/json"
	_ "expvar"
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
	log.Printf("\n\nNumGoroutine %v\n", runtime.NumGoroutine())
	rw.Header().Set("Content-Type", "application/json")
	currentComet := req.FormValue("cometid")
	currentIndex, _ := strconv.ParseUint(req.FormValue("index"), 10, 64)
	cookie, _ := req.Cookie("gsessionid")
	cometStore.Lock()
	cometStore.m[session(cookie.Value)] = comet{currentComet, time.Now()} //update timestamp on comet
	cometStore.Unlock()
	var chanMessages = make(chan Responses)
	var done = make(chan bool)
	tick := time.NewTicker(500 * 2 * time.Millisecond)
	key := sessionCometKey(cookie.Value + currentComet)
	go func() {
		for {
			select {
			case <-tick.C:
				getMessages(key, currentIndex, chanMessages, done)
			case <-done:
				return
			}
		}
	}()

	select {
	case messages := <-chanMessages:
		done <- true
		rw.Write(messages.Encode())
	case <-time.After(time.Second * 5):
		done <- true
		rw.Write(Responses{[]Response{Response{Value: jsCmd{""}, Error: ""}}, currentIndex}.Encode())
	}
}

func getMessages(key sessionCometKey, currentIndex uint64, result chan Responses, done chan bool) {
	messageStore.RLock()
	messages, found := messageStore.m[key]
	lastId := messageStore.LastIndex
	messageStore.RUnlock()
	if found {
		var payload Responses
		for _, msg := range messages {
			if currentIndex < msg.index {
				payload.Res = append(payload.Res, Response{jsCmd{msg.Value.Js}, ""})
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
	data = "console.log('" + data + "');"
	currentComet := req.FormValue("cometid")
	cookie, _ := req.Cookie("gsessionid")
	messageStore.Lock()
	messageStore.LastIndex++
	messageStore.m[sessionCometKey(cookie.Value+currentComet)] = append(messageStore.m[sessionCometKey(cookie.Value+currentComet)], message{messageStore.LastIndex, jsCmd{data}, time.Now()})
	messageStore.Unlock()
	rw.Write([]byte("Added a message"))
}

func gc() {
	for _ = range time.Tick(5 * time.Second) {
		log.Println("Started gc")
		start := time.Now()
		messageStore.Lock()
		for storeKey, messages := range messageStore.m {
			var temp []message
			for key, message := range messages {
				//if message.Stamp.Sub(time.Now()) > 20*time.Second {
				if time.Now().Sub(message.Stamp) > 20*time.Second {
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
	Value jsCmd `json:"value"`
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
	Value jsCmd
	Error string
}

type Responses struct {
	Res       []Response
	LastIndex uint64
}

func (r Responses) Encode() []byte {
	b, err := json.Marshal(r)
	if err != nil {
		return []byte("")
	}
	return b
}

type sessionCometKey string

type session string

type jsCmd struct {
	Js string `json:"js"`
}
