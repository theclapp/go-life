package gesture

// SPDX-License-Identifier: Unlicense OR MIT

import (
	"math"

	"gioui.org/ui"
	"gioui.org/ui/input"
	"gioui.org/ui/pointer"
)

type ScrollXY struct {
	dragging     bool
	pid          pointer.ID
	grab         bool
	lastX, lastY int
	// Leftover scroll.
	scrollX, scrollY float32
}

var (
	touchSlop = ui.Dp(3)
)

const (
	thresholdVelocity = 1
)

func (s *ScrollXY) Add(ops *ui.Ops) {
	oph := pointer.HandlerOp{Key: s, Grab: s.grab}
	oph.Add(ops)
}

func (s *ScrollXY) Stop() {}

func (s *ScrollXY) Dragging() bool {
	return s.dragging
}

func (s *ScrollXY) ScrollXY(cfg ui.Config, q input.Queue) (x, y int) {
	totalX, totalY := 0, 0
	for evt, ok := q.Next(s); ok; evt, ok = q.Next(s) {
		e, ok := evt.(pointer.Event)
		if !ok {
			continue
		}
		switch e.Type {
		case pointer.Press:
			// fmt.Printf("scrollxy: press: %+v\n", e)
			if s.dragging /*|| e.Source != pointer.Touch*/ {
				break
			}
			s.Stop()
			s.lastX = int(math.Round(float64(e.Position.X)))
			s.lastY = int(math.Round(float64(e.Position.Y)))
			s.dragging = true
			s.pid = e.PointerID
			// fmt.Printf("scrollxy: dragging: true\n")
		case pointer.Release:
			// fmt.Printf("scrollxy: release: %+v\n", e)
			if s.pid != e.PointerID {
				break
			}
			fallthrough
		case pointer.Cancel:
			// fmt.Printf("scrollxy: cancel: %+v\n", e)
			s.dragging = false
			s.grab = false
		case pointer.Move:
			// fmt.Printf("scrollxy: move: %+v\n", e)
			// Scroll
			s.scrollX += e.Scroll.X
			s.scrollY += e.Scroll.Y
			iscrollX := int(math.Round(float64(s.scrollX)))
			s.scrollX -= float32(iscrollX)
			totalX += iscrollX
			iscrollY := int(math.Round(float64(s.scrollY)))
			s.scrollY -= float32(iscrollY)
			totalY += iscrollY
			// fmt.Printf("totalX: %d, totalY: %d\n", totalX, totalY)
			if !s.dragging || s.pid != e.PointerID {
				continue
			}
			// Drag
			vX := int(math.Round(float64(e.Position.X)))
			distX := s.lastX - vX
			vY := int(math.Round(float64(e.Position.Y)))
			distY := s.lastY - vY
			if e.Priority < pointer.Grabbed {
				slop := cfg.Px(touchSlop)
				if (distX >= slop || -slop >= distX) ||
					(distY >= slop || -slop >= distY) {
					s.grab = true
				}
			} else {
				s.lastX = vX
				s.lastY = vY
				totalX += distX
				totalY += distY
			}
		}
	}
	return totalX, totalY
}

func (s *ScrollXY) Active() bool {
	return false
}
