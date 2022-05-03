package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
	"golang.org/x/exp/constraints"
)

type Cell struct {
	isBomb    bool
	label     int8
	flagged   bool
	uncovered bool
}

type Minesweeper struct {
	field  [][]Cell
	width  int
	height int
}

type Renderer struct {
	minesweeper *Minesweeper
	screen      tcell.Screen
	defStyle    tcell.Style
}

func (c Cell) IsBomb() bool {
	return c.isBomb
}

// NewMinesweeper creates a new minesweeper field.
func NewMinesweeper(width, height, numBombs int8) (error, *Minesweeper) {
	rand.Seed(time.Now().UnixNano())
	if width > 32 || height > 32 {
		return errors.New("Width or height can't be > 32"), nil
	}

	if numBombs > (width * height) {
		return errors.New("Too many bombs"), nil
	}

	field := make([][]Cell, height)

	for i := range field {
		field[i] = make([]Cell, width)
	}

	// generate bombs at random positions
	bombCount := int8(0)
	for {
		i := rand.Intn(int(height))
		j := rand.Intn(int(width))

		if !field[i][j].isBomb {
			field[i][j].isBomb = true
			bombCount++
		} else {
			continue
		}

		if bombCount == numBombs {
			break
		}
	}

	// let's calculate all labels using matrix convolution
	countBombsAround := func(x, y int8) int8 {
		bombCount := int8(0)
		for i := Max(0, y-1); i < Min(int8(height), y+2); i++ {
			for j := Max(0, x-1); j < Min(int8(width), x+2); j++ {
				if field[i][j].isBomb {
					bombCount++
				}
			}
		}

		return bombCount
	}

	for i := int8(0); i < int8(height); i++ {
		for j := int8(0); j < int8(width); j++ {
			field[i][j].label = countBombsAround(j, i)
		}
	}

	return nil, &Minesweeper{field, int(width), int(height)}
}

// Get returns cell at position x, y
func (ms Minesweeper) Get(x, y int) (error, *Cell) {
	if x > ms.height || y > ms.width {
		return errors.New("x or y is larger than a field size"), nil
	}
	return nil, &ms.field[x][y]
}

// Uncover acts on a Cell at position x, y and returns if it's a bomb.
// If cell is not a bomb, it's label is also updated to comtain the number of surronding bombs
func (ms *Minesweeper) Uncover(x, y int) (error, bool) {
	if x >= ms.height || y >= ms.width {
		return errors.New("x or y is larger than a field size"), false
	}

	ms.field[y][x].uncovered = true
	return nil, ms.field[y][x].isBomb
}

// NewRenderer creates new rederer for given Minesweeper reference
func NewRenderer(ms *Minesweeper) (error, *Renderer) {
	s, err := tcell.NewScreen()

	if err != nil {
		return err, nil
	}

	if err := s.Init(); err != nil {
		return err, nil
	}

	defStyle := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite)
	s.SetStyle(defStyle)
	s.EnableMouse()
	s.EnablePaste()

	s.Clear()
	return nil, &Renderer{ms, s, defStyle}
}

// render draws minesweeper field on screen
func (r Renderer) render() {
	for i := 0; i < r.minesweeper.height; i++ {
		for j := 0; j < r.minesweeper.width; j++ {
			_, cell := r.minesweeper.Get(i, j)
			if cell.isBomb && cell.uncovered {
				r.screen.SetContent(j, i, 'x', nil, r.defStyle.Foreground(tcell.ColorRed))
			} else if cell.uncovered {
				r.screen.SetContent(j, i, rune(48+cell.label), nil, r.defStyle)
			} else {
				r.screen.SetContent(j, i, 'o', nil, r.defStyle)
			}
		}
	}
}

// StartLoop launches main rendering loop
func (r Renderer) StartLoop() {
	// render everything the first time
	r.render()

	for {
		// Update screen
		r.screen.Show()

		// Poll event
		ev := r.screen.PollEvent()

		// Process event
		switch ev := ev.(type) {
		case *tcell.EventResize:
			r.screen.Sync()
		case *tcell.EventKey:
			r.handleKeyPressed(ev.Key())
		case *tcell.EventMouse:
			buttons := ev.Buttons()
			x, y := ev.Position()
			drawText(r.screen, 20, 5, 30, 5, r.defStyle, fmt.Sprintf("%d, %d", x, y))
			r.handleMousePressed(x, y, buttons)
		}
	}
}

func (r Renderer) handleMousePressed(x, y int, buttons tcell.ButtonMask) {
	switch buttons {
	case tcell.Button1:
		_, hasBlownUp := r.minesweeper.Uncover(x, y)

		if hasBlownUp {
			// TODO do something more interesting
			// quit()
			drawText(r.screen, 20, 21, 30, 21, r.defStyle.Foreground(tcell.ColorRed), "BLOWN UP")
		}

	}
	r.render()
}

func (r Renderer) handleKeyPressed(key tcell.Key) {
	if key == tcell.KeyEscape || key == tcell.KeyCtrlC {
		r.quit()
	}
}

func (r Renderer) quit() {
	r.screen.Fini()
	os.Exit(0)
}

// drawText draws text on screen from (x1, y1) to (x2, y2)
func drawText(s tcell.Screen, x1, y1, x2, y2 int, style tcell.Style, text string) {
	row := y1
	col := x1
	for _, r := range []rune(text) {
		s.SetContent(col, row, r, nil, style)
		col++
		if col >= x2 {
			row++
			col = x1
		}
		if row > y2 {
			break
		}
	}
}

func Max[T constraints.Ordered](a, b T) T {
	if a > b {
		return a
	} else {
		return b
	}
}

func Min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	} else {
		return b
	}
}

func main() {
	err, minesweeper := NewMinesweeper(8, 8, 10)

	if err != nil {
		log.Panicf("Error while creating minesweeper: %s", err)
	}

	err, renderer := NewRenderer(minesweeper)

	if err != nil {
		log.Panicf("Error while creating renderer: %s", err)
	}

	renderer.StartLoop()
}
