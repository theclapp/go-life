package main

import (
	"math"
	"fmt"
	"time"
	"runtime"
)

// data structures
type cell struct {
	x, y int
}

type universe struct {
	cells map[cell]bool
	minx, miny, maxx, maxy int
}
var u *universe

// processing
// several channels to process cells (# CPUs - 1?)
// one channel read by goroutine to build next generation

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	u = make_universe()
	for {
		display(u)
		u = next_gen()
	}
}

func make_universe() *universe {
	//   x
	// x x
	//  xx
	u := newUniverse(nil)
	n := 0
	                                    u.addXY(n+2, n+0)
	u.addXY(n+0,n+1);                   u.addXY(n+2, n+1)
	                  u.addXY(n+1,n+2); u.addXY(n+2, n+2)

	n = 5
	                                    u.addXY(n+2, n+0)
	u.addXY(n+0,n+1);                   u.addXY(n+2, n+1)
	                  u.addXY(n+1,n+2); u.addXY(n+2, n+2)

	n = 10
	                                    u.addXY(n+2, n+0)
	u.addXY(n+0,n+1);                   u.addXY(n+2, n+1)
	                  u.addXY(n+1,n+2); u.addXY(n+2, n+2)
	return u
}

func (u *universe) addXY(x, y int) {
	u.addCell(cell{x, y })
}

func (u *universe) addCell(c cell) {
	u.cells[c] = true
	u.minx = min(u.minx, c.x)
	u.miny = min(u.miny, c.y)
	u.maxx = max(u.maxx, c.x)
	u.maxy = max(u.maxy, c.y)
}

func next_gen() *universe {
	checkCells := make(chan cell)
	newCells := make(chan cell)
	newUniverseCh := make(chan *universe)
	doneCh := make(chan int)
	for n := 0; n < runtime.NumCPU(); n++ {
		go calcCell(checkCells, newCells, doneCh)
	}
	go recordResults(newCells, newUniverseCh)
	seen := make(map[cell]bool)
	for c := range u.cells {
		checkCells <- c
		checkNeighbor(checkCells, seen, c, -1, -1)
		checkNeighbor(checkCells, seen, c,  0, -1)
		checkNeighbor(checkCells, seen, c, +1, -1)
		checkNeighbor(checkCells, seen, c, -1,  0)
		checkNeighbor(checkCells, seen, c,  1,  0)
		checkNeighbor(checkCells, seen, c, -1,  1)
		checkNeighbor(checkCells, seen, c,  0,  1)
		checkNeighbor(checkCells, seen, c, -1,  1)
	}
	close(checkCells)
	for n := 0; n < runtime.NumCPU(); n++ {
		<- doneCh
	}
	close(newCells)
	return <- newUniverseCh
}

func checkNeighbor(checkCells chan cell, seen map[cell]bool, c cell, x, y int) {
	neighborCell := cell{c.x+x, c.y+y}
	if u.cells[neighborCell] ||
	   seen[neighborCell] {
		return
	}
	seen[neighborCell] = true
	checkCells <- neighborCell
}

func calcCell(checkCells, newCells chan cell, doneCh chan int) {
	for c := range checkCells {
		n := neighbors(c)
		if n == 3 ||
		   (n == 2 &&
			 u.cells[c]) {
			newCells <- c
		}
	}
	doneCh <- 1
}

func recordResults(newCells chan cell, newUniverseCh chan *universe) {
	newU := newUniverse(u)
	for c := range newCells {
		newU.addCell(c)
	}
	newUniverseCh <- newU
}

func newUniverse(parent *universe) *universe {
	var u universe
	u.cells = make(map[cell]bool)
	if (parent == nil) {
		u.minx = math.MaxInt16
		u.miny = math.MaxInt16
	} else {
		u.minx = parent.minx
		u.miny = parent.miny
	}
	u.maxx = -math.MaxInt16
	u.maxy = -math.MaxInt16
	return &u
}

func neighbors(c cell) int {
	n := 0

	if u.cells[cell{c.x-1, c.y-1}] { n++ }
	if u.cells[cell{c.x,   c.y-1}] { n++ }
	if u.cells[cell{c.x+1, c.y-1}] { n++ }

	if u.cells[cell{c.x-1, c.y  }] { n++ }
	if u.cells[cell{c.x+1, c.y  }] { n++ }

	if u.cells[cell{c.x-1, c.y+1}] { n++ }
	if u.cells[cell{c.x,   c.y+1}] { n++ }
	if u.cells[cell{c.x+1, c.y+1}] { n++ }

	return n
}

func display(u *universe) {
	var c cell
	fmt.Printf( "%c[H%c[2J", 27, 27 )
	fmt.Println()
	for y := u.miny; y <= u.maxy; y++ {
		found := false
		s := ""
		for x := u.minx; x <= u.maxx; x++ {
			c.x = x; c.y = y
			if u.cells[c] {
				s = s + "X"
				found = true
			} else {
				s = s + " "
			}
		}
		if found { fmt.Print(s) }
		fmt.Println()
	}
	fmt.Println()
	time.Sleep(1e7)
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

