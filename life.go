package main

// vim:sw=3:ts=3:fdm=indent

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"net/http"
	"runtime"
	"sync"
	"html/template"
	// "time"
)

// data structures
type cell struct {
	x, y int
}

type universe struct {
	cells                  map[cell]byte
	minx, miny, maxx, maxy int
}

type Page struct {
	PageId string
}

type session struct {
	id string
	nextPageId int
	// map pageIds to listener channels
	listeners map[string]chan string
}

var sessions = make(map[string]*session)
var templates = make(map[string]*template.Template)

var delay float32 = 800.0
var stop = 0
var nextSessionNum = 0

var u *universe
var uCh chan *universe
var numCPU = runtime.NumCPU()
var gen = 0

func main() {
	runtime.GOMAXPROCS(numCPU)
	u = make_universe()
	uCh = make(chan *universe, 10)
	go func() {
		for {
			uCh <- u
			u = next_gen()
		}
	}()

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
	pc, _, _, _ := runtime.Caller(n+1)
	return runtime.FuncForPC(pc).Name()
}

func LifeServer(w http.ResponseWriter, req *http.Request) {
	s := getSession(w, req)
	p := Page{PageId: s.getNextPageId()}
	err := templates["life"].Execute(w, p)
	if err != nil {
		fmt.Println("Error rendering life.html template")
	}
	// Force an update on all listeners; clears closed listeners.
	for _, listener := range s.listeners {
		listener <- "true"
	}
}

func (s *session) getNextPageId() (result string) {
	result = fmt.Sprintf("%d", s.nextPageId)
	s.nextPageId++
	return
}

func getSession(w http.ResponseWriter, req *http.Request) *session {
	var sessionId string
	sessionIdCookie, err := req.Cookie("session")
	if err == nil {
		sessionId = sessionIdCookie.Value
		if sessions[sessionId] == nil {
			fmt.Printf("%s: Invalid session cookie: %s\n", curFuncName(1), sessionId)
		} else {
			return sessions[sessionId]
		}
	}

	fmt.Printf("%s: No session cookie; setting %d\n", curFuncName(1), nextSessionNum)
	sessionId = fmt.Sprintf("%d", nextSessionNum)
	sessionIdCookie = &http.Cookie{
		Name:   "session",
		Value:  sessionId,
		MaxAge: 3600,
	}
	http.SetCookie(w, sessionIdCookie)
	sessions[sessionId] = &session{
		id: sessionId,
		nextPageId: 0,
		listeners: make(map[string]chan string),
	}
	nextSessionNum++
	return sessions[sessionId]
}

func LifeJS(w http.ResponseWriter, req *http.Request) {
	fmt.Println("Serving LifeJS")
	http.ServeFile(w, req, "life.js")
}

// FIXME ticks once per *window*.  So 3 windows, 3 generations per tick.
func LifeImage(w http.ResponseWriter, req *http.Request) {
	fmt.Printf("gen: %d\r", gen)
	gen++
	png.Encode(w, display(<-uCh))
}

func Button(w http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		fmt.Println("Could not parse form in Button()")
		return
	}
	whichButton := req.FormValue("title")
	fmt.Printf("title is %s\n", whichButton)

	s := getSession(w, req)

	if len(s.listeners) == 0 {
		fmt.Println("No listeners for session", s.id)
		return
	}

	switch whichButton {
	case "delayMore":
		delay *= 2
	case "delayLess":
		delay /= 2
	case "stopLife":
		stop = 1
		delay++
	case "startLife":
		stop = 0
		delay--
	case "clearSession":
		http.SetCookie(w, &http.Cookie{
			Name:   "session",
			MaxAge: -1,
		})
	}

	event := fmt.Sprintf(`refresh({"delay":%f,"stop":%d})`, delay, stop)
	fmt.Println("event is " + event)
	for _, listener := range s.listeners {
		listener <- event
	}
}

// Long-polled URL.  What happens if the connection times out, or is closed?
// Indeed, that's exactly what happens (the socket is closed) if you press
// Reload on the browser.
// Current implementation is not threadsafe.
func Updates(w http.ResponseWriter, req *http.Request) {
	s := getSession(w, req)
	pageId := req.FormValue("pageId")
	fmt.Printf("starting Updates, pageId is %s, req is %p\n", pageId, req)
	listener := make(chan string)
	s.listeners[pageId] = listener

	event := <-listener
	fmt.Printf("event recieved for req %p, pageId %s; event is %s\n", req, pageId, event)
	_, err := w.Write([]byte(event))
	if err != nil {
		fmt.Printf("write error in Updates for %p\n", req)
	}
	fmt.Printf("Updates finished for req %p, pageId %s\n", req, pageId)

	// FIXME this is not threadsafe
	delete(s.listeners, pageId)
}

func display(u *universe) (m *image.NRGBA) {
	red := color.RGBA{255, 0, 0, 255}
	m = image.NewNRGBA(image.Rect(u.minx, u.miny, u.maxx+1, u.maxy+1))
	for c := range u.cells {
		m.Set(c.x, c.y, red)
	}
	return
}

func make_universe() *universe {
	u := newUniverse()

	// R-pentomino
	//  XX
	// XX
	//  X
	u.addXY(1, 0)
	u.addXY(2, 0)
	u.addXY(0, 1)
	u.addXY(1, 1)
	u.addXY(1, 2)
	return u
}

func (u *universe) addXY(x, y int) {
	u.addCell(cell{x, y})
}

func (u *universe) addCell(c cell) {
	u.cells[c] = 1
	u.minx = min(u.minx, c.x)
	u.miny = min(u.miny, c.y)
	u.maxx = max(u.maxx, c.x)
	u.maxy = max(u.maxy, c.y)
}

func next_gen() *universe {
	checkCellsCh := make(chan cell)
	neighborCountCh := make(chan map[cell]byte)
	newUniverseCh := make(chan *universe)
	var wg sync.WaitGroup
	for n := 0; n < numCPU; n++ {
		wg.Add(1)
		go calcCell(checkCellsCh, neighborCountCh, &wg)
	}
	go recordResults(neighborCountCh, newUniverseCh)
	for c := range u.cells {
		checkCellsCh <- c
	}
	close(checkCellsCh)
	wg.Wait()
	close(neighborCountCh)
	return <-newUniverseCh
}

func calcCell(checkCellsCh chan cell,
	neighborCountCh chan map[cell]byte,
	wg *sync.WaitGroup) {
	neighborCount := make(map[cell]byte)
	for c := range checkCellsCh {
		neighborCount[cell{c.x - 1, c.y - 1}]++
		neighborCount[cell{c.x, c.y - 1}]++
		neighborCount[cell{c.x + 1, c.y - 1}]++

		neighborCount[cell{c.x - 1, c.y}]++
		neighborCount[cell{c.x + 1, c.y}]++

		neighborCount[cell{c.x - 1, c.y + 1}]++
		neighborCount[cell{c.x, c.y + 1}]++
		neighborCount[cell{c.x + 1, c.y + 1}]++
	}
	neighborCountCh <- neighborCount
	wg.Done()
}

func recordResults(neighborCountCh chan map[cell]byte,
	newUniverseCh chan *universe) {
	newU := newUniverse()
	allNeighbors := <-neighborCountCh
	for moreNeighbors := range neighborCountCh {
		for c, neighbors := range moreNeighbors {
			allNeighbors[c] += neighbors
		}
	}
	for c, neighbors := range allNeighbors {
		if neighbors == 3 ||
			u.cells[c] == 1 && (neighbors == 2) {
			newU.addCell(c)
		}
	}
	newUniverseCh <- newU
}

func newUniverse() *universe {
	var u universe
	u.cells = make(map[cell]byte)
	u.minx = math.MaxInt16
	u.miny = math.MaxInt16
	u.maxx = -math.MaxInt16
	u.maxy = -math.MaxInt16
	return &u
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}
