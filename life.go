package main

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"
	"os"
	"time"

	"gioui.org/ui"
	"gioui.org/ui/app"
	"gioui.org/ui/f32"
	ggesture "gioui.org/ui/gesture"
	"gioui.org/ui/key"
	"gioui.org/ui/layout"
	"gioui.org/ui/measure"
	"gioui.org/ui/paint"
	"gioui.org/ui/pointer"
	"gioui.org/ui/text"
	"github.com/theclapp/go-life/gesture"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"
)

const (
	fontHeight = 14
)

type (
	Pos struct {
		x, y int
	}

	Universe struct {
		cells map[Pos]bool
		gen   int
	}

	Window struct {
		w       *app.Window
		u       *Universe
		scale   int
		regular *sfnt.Font
		faces   measure.Faces

		scrollXY         gesture.ScrollXY
		scrollX, scrollY int
		paused           bool

		interval time.Duration
		genTimer *time.Ticker

		buttons map[string]*ggesture.Click
	}
)

func main() {
	go func() {
		w := &Window{
			scale:   10,
			buttons: map[string]*ggesture.Click{},
		}
		w.u = NewUniverse()
		// w.u.RPentomino()
		w.u.Random(100, 100, 333)

		w.w = app.NewWindow(&app.WindowOptions{
			Width:  ui.Dp(800),
			Height: ui.Dp(600),
			Title:  "Gio Life",
		})

		w.regular, _ = sfnt.Parse(goregular.TTF)

		ops := &ui.Ops{}

		w.interval = 100 * time.Millisecond
		w.genTimer = time.NewTicker(w.interval)
		for {
			select {
			case <-w.genTimer.C:
				w.u = w.u.NextGen()
				w.w.Invalidate()
			case e := <-w.w.Events():
				switch e := e.(type) {
				case key.EditEvent:
					switch e.Text {
					case "q":
						os.Exit(0)
					case "p":
						w.pause()
					case "c":
						w.cont()
					// Restart with another random universe
					case "r":
						w.u.Random(100, 100, 333)
						w.w.Invalidate()
					case "-":
						w.zoomOut()
					case "+", "=":
						w.zoomIn()
					case ">", ".":
						w.goFaster()
					case "<", ",":
						w.goSlower()
					default:
						// fmt.Printf("key name: %s\n", e.Text)
					}
				case app.UpdateEvent:
					ops.Reset()
					w.Layout(e, ops)
					w.w.Update(ops)
				default:
					// fmt.Printf("event %T %+v\n", e, e)
				}
			}
		}
	}()
	app.Main()
}

func NewUniverse() *Universe {
	return &Universe{
		cells: make(map[Pos]bool),
	}
}

func (u *Universe) RPentomino() {
	//  xx
	// xx
	//  x
	for _, rec := range []Pos{
		{1, 2}, {2, 2},
		{0, 1}, {1, 1},
		{1, 0},
	} {
		u.Set(rec)
	}
}

func (u *Universe) Set(p Pos) {
	u.cells[p] = true
}

func (u *Universe) IsSet(p Pos) int {
	if u.cells[p] {
		return 1
	}
	return 0
}

func (u *Universe) NextGen() *Universe {
	next := *u
	next.cells = make(map[Pos]bool)
	next.gen = u.gen + 1
	checked := make(map[Pos]bool)
	for pos := range u.cells {
		// Will this cell be alive in the next generation?
		if n := u.NumNeighbors(pos); n == 2 || n == 3 {
			next.Set(pos)
		}

		// Will any of this cell's neighbors be alive in the next generation?
		xm1, xp1 := pos.x-1, pos.x+1
		ym1, yp1 := pos.y-1, pos.y+1
		for _, neighbor := range []Pos{
			{xm1, ym1}, {pos.x, ym1}, {xp1, ym1},
			{xm1, pos.y}, {xp1, pos.y},
			{xm1, yp1}, {pos.x, yp1}, {xp1, yp1}} {
			if next.cells[neighbor] || checked[neighbor] {
				continue
			}
			if u.NumNeighbors(neighbor) == 3 {
				next.Set(neighbor)
			}
			checked[neighbor] = true
		}
	}
	return &next
}

// Aborts at 4
func (u *Universe) NumNeighbors(pos Pos) int {
	xm1, xp1 := pos.x-1, pos.x+1
	ym1, yp1 := pos.y-1, pos.y+1
	n := u.IsSet(Pos{xm1, ym1}) + u.IsSet(Pos{pos.x, ym1}) + u.IsSet(Pos{xp1, ym1}) +
		u.IsSet(Pos{xm1, pos.y})
	if n == 4 {
		return n
	}
	for _, neighbor := range []Pos{
		{xp1, pos.y},
		{xm1, yp1}, {pos.x, yp1}, {xp1, yp1}} {
		if u.cells[neighbor] {
			n++
			if n == 4 {
				return n
			}
		}
	}
	return n
}

func min(n1, n2 int) int {
	if n1 < n2 {
		return n1
	}
	return n2
}

func max(n1, n2 int) int {
	if n1 > n2 {
		return n1
	}
	return n2
}

func (w *Window) Layout(e app.UpdateEvent, ops *ui.Ops) {
	cfg := &e.Config
	w.faces.Reset(cfg)
	cs := layout.Constraints{
		Width:  layout.Constraint{Max: e.Size.X},
		Height: layout.Constraint{Max: e.Size.Y},
	}

	// Do scrolling
	scrollX, scrollY := w.scrollXY.ScrollXY(cfg, w.w.Queue())
	if scrollX != 0 || scrollY != 0 {
		// fmt.Printf("window layout: scrollX: %d, scrollY: %d\n", scrollX, scrollY)
		w.scrollX -= scrollX
		w.scrollY -= scrollY
	}
	r := image.Rectangle{Max: e.Size}
	pointer.RectAreaOp{Rect: r}.Add(ops)
	w.scrollXY.Add(ops)

	var stack ui.StackOp
	stack.Push(ops)

	// Draw the top buttons
	paint.ColorOp{
		Color: color.RGBA{A: 0x80, R: 0xff, B: 0xff, G: 0xff},
	}.Add(ops)

	// Layout buttons and process clicks
	{
		partial := func(s string, action func()) { w.button(ops, cs, s, action) }
		if w.paused {
			partial("Continue |", w.cont)
		} else {
			partial("Pause |", w.pause)
		}
		partial(" Zoom in |", w.zoomIn)
		partial(" Zoom out |", w.zoomOut)
		partial(" Go slower |", w.goSlower)
		partial(" Go faster |", w.goFaster)
	}

	lbl := text.Label{
		Face: w.faces.For(w.regular, ui.Sp(fontHeight)),
		Text: fmt.Sprintf(" Gen: %d | Scale: %d | GenTime: %v", w.u.gen, w.scale, w.interval),
	}
	if w.paused {
		lbl.Text += " | PAUSED"
	}
	dims := lbl.Layout(ops, cs)
	ui.TransformOp{}.Offset(f32.Point{X: float32(dims.Size.X)}).Add(ops)

	stack.Pop()

	// Clip cells so they don't obscure the caption
	paint.RectClip(image.Rectangle{
		Min: image.Point{Y: dims.Size.Y},
		Max: e.Size,
	}).Add(ops)
	ui.TransformOp{}.Offset(f32.Point{
		X: float32(w.scrollX),
		Y: float32(w.scrollY),
	}).Add(ops)
	paint.ColorOp{Color: color.RGBA{A: 0xff, G: 0xff}}.Add(ops)
	for pos := range w.u.cells {
		paint.PaintOp{
			Rect: f32.Rectangle{
				Min: f32.Point{float32(w.scale * pos.x), float32(w.scale * pos.y)},
				Max: f32.Point{float32(w.scale * (pos.x + 1)), float32(w.scale * (pos.y + 1))},
			},
		}.Add(ops)
	}

}

func (u *Universe) Random(x, y, density int) {
	*u = *NewUniverse()
	for px := 0; px < x; px++ {
		for py := 0; py < y; py++ {
			if rand.Intn(1000) < density {
				u.Set(Pos{px, py})
			}
		}
	}
}

func (w *Window) zoomIn() {
	w.scale++
	w.w.Invalidate()
}

func (w *Window) zoomOut() {
	if w.scale > 1 {
		w.scale--
	}
	w.w.Invalidate()
}

func (w *Window) goFaster() {
	w.genTimer.Stop()
	w.interval /= 2
	w.genTimer = time.NewTicker(w.interval)
	w.w.Invalidate()
}

func (w *Window) goSlower() {
	w.genTimer.Stop()
	w.interval *= 2
	w.genTimer = time.NewTicker(w.interval)
	w.w.Invalidate()
}

func (w *Window) pause() {
	w.paused = true
	w.genTimer.Stop()
	w.w.Invalidate()
}

func (w *Window) cont() {
	w.paused = false
	w.genTimer = time.NewTicker(w.interval)
	w.w.Invalidate()
}

type label struct {
	w     *Window
	click ggesture.Click
}

func (w *Window) button(ops *ui.Ops, cs layout.Constraints, label string, action func()) {
	click, ok := w.buttons[label]
	if !ok {
		click = &ggesture.Click{}
		w.buttons[label] = click
	}

	lbl := text.Label{
		Face: w.faces.For(w.regular, ui.Sp(fontHeight)),
		Text: label,
	}
	dims := lbl.Layout(ops, cs)

	var stack ui.StackOp
	stack.Push(ops)

	pointer.RectAreaOp{
		Rect: image.Rectangle{Max: image.Point{X: dims.Size.X, Y: dims.Size.Y}},
	}.Add(ops)
	click.Add(ops)

	stack.Pop()

	ui.TransformOp{}.Offset(f32.Point{X: float32(dims.Size.X)}).Add(ops)

	for _, e := range click.Events(w.w.Queue()) {
		if e.Type == ggesture.TypeClick {
			action()
		}
	}
}
