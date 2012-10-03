package main

// vim:sw=3:ts=3:fdm=indent

import (
	"fmt"
	"html/template"
	"image/png"
	"net/http"
	"runtime"
)

var templates = make(map[string]*template.Template)

var numCPU = runtime.NumCPU()

func main() {
	runtime.GOMAXPROCS(numCPU)
	templates["life"] = template.Must(template.ParseFiles("life.html"))

	http.HandleFunc("/life.html", LifeServer)
	http.HandleFunc("/life.js", LifeJS)
	http.HandleFunc("/life.png", LifeImage)
	http.HandleFunc("/button", Button)
	http.HandleFunc("/updates", Updates)
	println("serving")
	err := http.ListenAndServe(":6080", nil)
	if err != nil {
		panic(err)
	}
}

func curFuncName(n int) string {
	pc, _, _, _ := runtime.Caller(n + 1)
	return runtime.FuncForPC(pc).Name()
}

func LifeServer(w http.ResponseWriter, req *http.Request) {
	s := getSession(w, req)
	err := templates["life"].Execute(w, struct{ PageId string }{s.getNextPageId()})
	if err != nil {
		fmt.Println("Error rendering life.html template")
	}
	// Force an update on all listeners; clears closed listeners.
	for _, listener := range s.listeners {
		listener <- "true"
	}
}

func LifeJS(w http.ResponseWriter, req *http.Request) {
	fmt.Println("Serving LifeJS")
	http.ServeFile(w, req, "life.js")
}

func LifeImage(w http.ResponseWriter, req *http.Request) {
	// FIXME this renders the image repeatedly and is quite inefficient
	png.Encode(w, display(getSession(w, req).nextU(req.FormValue("pageId"))))
}

func Button(w http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		fmt.Println("Could not parse form in Button()")
		return
	}

	s := getSession(w, req)

	if len(s.listeners) == 0 {
		fmt.Println("No listeners for session", s.sid)
		return
	}

	s.pressButton(w, req.FormValue("title"))

	event := fmt.Sprintf(`refresh({"delay":%f,"stop":%d})`, s.delay, s.stop)
	fmt.Println("event is " + event)
	for _, listener := range s.listeners {
		listener <- event
	}
}

// Long-polled URL.  What happens if the connection times out, or is closed?
// Indeed, that's exactly what happens (the socket is closed) if you press
// reload on the browser.
func Updates(w http.ResponseWriter, req *http.Request) {
	s := getSession(w, req)
	pageId := req.FormValue("pageId")
	fmt.Printf("starting Updates, pageId is %s, req is %p\n", pageId, req)

	event := <-s.addListener(pageId)
	fmt.Printf("event recieved for req %p, pageId %s; event is %s\n", req, pageId, event)
	_, err := w.Write([]byte(event))
	if err != nil {
		fmt.Printf("write error in Updates for %p\n", req)
	}
	fmt.Printf("Updates finished for req %p, pageId %s\n", req, pageId)

	s.removeListener(pageId)
}
