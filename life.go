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

type sessionId string

type session struct {
	sid        sessionId
	nextPageId int
	// map pageIds to listener channels
	listeners map[string]chan string
	delay     float32
	stop      int
	uCh		 chan *universe
	u         *universe
	numU      int
	gen       int
	sync.Mutex
}

var sessions = make(map[sessionId]*session)
var sessionsLock sync.Mutex
var templates = make(map[string]*template.Template)

var nextSessionNum = 0

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
	pc, _, _, _ := runtime.Caller(n+1)
	return runtime.FuncForPC(pc).Name()
}

func LifeServer(w http.ResponseWriter, req *http.Request) {
	s := getSession(w, req)
	err := templates["life"].Execute(w, struct{PageId string}{s.getNextPageId()})
	if err != nil {
		fmt.Println("Error rendering life.html template")
	}
	// Force an update on all listeners; clears closed listeners.
	for _, listener := range s.listeners {
		listener <- "true"
	}
}

// FIXME Locks the session
func (s *session) getNextPageId() (result string) {
	result = fmt.Sprintf("%d", s.nextPageId)
	s.Lock(); defer s.Unlock()
	s.nextPageId++
	return
}

func LifeJS(w http.ResponseWriter, req *http.Request) {
	fmt.Println("Serving LifeJS")
	http.ServeFile(w, req, "life.js")
}

// FIXME Locks the session.
func LifeImage(w http.ResponseWriter, req *http.Request) {
	pageId := req.FormValue("pageId")
	s := getSession(w, req)
	s.Lock();
	if s.numU < len(s.listeners) {
		s.numU++
	} else {
		s.u = <-s.uCh
		s.gen++
		s.numU = 1
	}
	curU := s.u
	fmt.Printf("session: %s, pageId: %s, gen: %d, numU: %d, #listeners: %d\n",
		s.sid, pageId, s.gen, s.numU, len(s.listeners))
	s.Unlock()
	// FIXME this renders the image repeatedly and is quite inefficient
	png.Encode(w, display(curU))
}

// FIXME Locks the session.
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
		fmt.Println("No listeners for session", s.sid)
		return
	}

	s.Lock();
	switch whichButton {
	case "delayMore":
		s.delay *= 2
	case "delayLess":
		s.delay /= 2
	case "stopLife":
		s.stop = 1
		s.delay++
	case "startLife":
		s.stop = 0
		s.delay--
	case "clearSession":
		http.SetCookie(w, &http.Cookie{
			Name:   "session",
			MaxAge: -1,
		})
	}
	s.Unlock()

	event := fmt.Sprintf(`refresh({"delay":%f,"stop":%d})`, s.delay, s.stop)
	fmt.Println("event is " + event)
	for _, listener := range s.listeners {
		listener <- event
	}
}

// Long-polled URL.  What happens if the connection times out, or is closed?
// Indeed, that's exactly what happens (the socket is closed) if you press
// reload on the browser.
// FIXME Locks the session.
func Updates(w http.ResponseWriter, req *http.Request) {
	s := getSession(w, req)
	pageId := req.FormValue("pageId")
	fmt.Printf("starting Updates, pageId is %s, req is %p\n", pageId, req)
	listener := make(chan string)
	s.Lock()
	s.listeners[pageId] = listener
	s.Unlock()

	event := <-listener
	fmt.Printf("event recieved for req %p, pageId %s; event is %s\n", req, pageId, event)
	_, err := w.Write([]byte(event))
	if err != nil {
		fmt.Printf("write error in Updates for %p\n", req)
	}
	fmt.Printf("Updates finished for req %p, pageId %s\n", req, pageId)

	s.Lock()
	delete(s.listeners, pageId)
	s.Unlock()
}

// FIXME locks sessions[]
func getSession(w http.ResponseWriter, req *http.Request) *session {
	var sid sessionId
	sessionsLock.Lock(); defer sessionsLock.Unlock()
	sessionCookie, err := req.Cookie("session")
	if err == nil {
		sid = sessionId(sessionCookie.Value)
		if sessions[sid] == nil {
			fmt.Printf("%s: Invalid session cookie: %s\n", curFuncName(1), sid)
		} else {
			return sessions[sid]
		}
	}

	fmt.Printf("%s: No session cookie; setting %d\n", curFuncName(1), nextSessionNum)
	sid = sessionId(fmt.Sprintf("%d", nextSessionNum))
	sessionCookie = &http.Cookie{
		Name:   "session",
		Value:  string(sid),
		MaxAge: 3600,
	}
	http.SetCookie(w, sessionCookie)
	sessions[sid] = &session{
		sid: sid,
		listeners: make(map[string]chan string),
		delay: 800.00,
		uCh: make(chan *universe),
	}
	nextSessionNum++
	go func(s *session) {
		u := makeUniverse()
		for {
			s.uCh <- u
			u = nextGen(u)
		}
	}(sessions[sid])
	return sessions[sid]
}

func display(u *universe) (m *image.NRGBA) {
	red := color.RGBA{255, 0, 0, 255}
	m = image.NewNRGBA(image.Rect(u.minx, u.miny, u.maxx+1, u.maxy+1))
	for c := range u.cells {
		m.Set(c.x, c.y, red)
	}
	return
}

func makeUniverse() *universe {
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

func nextGen(u *universe) *universe {
	checkCellsCh := make(chan cell)
	neighborCountCh := make(chan map[cell]byte)
	newUniverseCh := make(chan *universe)
	var wg sync.WaitGroup
	for n := 0; n < numCPU; n++ {
		wg.Add(1)
		go calcCell(checkCellsCh, neighborCountCh, &wg)
	}
	go recordResults(u, neighborCountCh, newUniverseCh)
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

func recordResults(u *universe,
	neighborCountCh chan map[cell]byte,
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
