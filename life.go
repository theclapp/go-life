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

var delay float32 = 800.0
var stop = 0
var sessionNum = 0

var u *universe
var uCh chan *universe
var numCPU = runtime.NumCPU()
var eventCh = make(chan string)
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

func curFuncName() string {
	pc, _, _, _ := runtime.Caller(1)
	return runtime.FuncForPC(pc).Name()
}

func LifeServer(w http.ResponseWriter, req *http.Request) {
	_, err := req.Cookie("session")
	if err != nil {
		fmt.Printf("%s: No session cookie; setting %d\n", curFuncName(), sessionNum)
		http.SetCookie(w, &http.Cookie{
			Name:   "session",
			Value:  fmt.Sprintf("%d", sessionNum),
			MaxAge: 10,
		})
		sessionNum++
	}
	http.ServeFile(w, req, "life.html")
}

func LifeJS(w http.ResponseWriter, req *http.Request) {
	http.ServeFile(w, req, "life.js")
}

func LifeImage(w http.ResponseWriter, req *http.Request) {
	fmt.Printf("gen: %d\r", gen)
	gen++
	png.Encode(w, display(<-uCh))
}

func Button(w http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		return
	}
	fmt.Printf("title is %s\n", req.FormValue("title"))
	switch req.FormValue("title") {
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
	eventCh <- event
}

// Long-polled URL.  What happens if the connection times out, or is closed?
// Indeed, that's exactly what happens (the socket is closed) if you press
// Reload on the browser.
func Updates(w http.ResponseWriter, req *http.Request) {
	fmt.Printf("starting Updates, req is %p\n", req)
	event := <-eventCh
	fmt.Printf("event recieved for req %p; event is %s\n", req, event)
	_, err := w.Write([]byte(event))
	if err != nil {
		fmt.Printf("write error in Updates for %p\n", req)
	}
	fmt.Printf("Updates finished for req %p\n", req)
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
