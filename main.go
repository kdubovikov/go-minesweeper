package main

import (
	"container/list"
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
	x         int8
	y         int8
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
	// consider all bombs are placed at the start
	// for each bomb we will swap it with random element
	for i := 0; i < int(numBombs+1); i++ {
		// generate a second cell index to swap with
		i2 := i + rand.Intn(int(width)*int(height)-i)

		// convert sequential index to row number
		row1 := i / int(width)

		// if indices are different, then we should place bomb at
		// newly generated index
		if i != i2 {
			// convert sequential index to row and col numbers
			row2 := i2 / int(width)
			col2 := (i2 - row2*int(width)) % int(width)

			field[row2][col2].isBomb = true
		} else {
			// convert sequential index to col number
			col1 := (i - row1*int(width)) % int(width)

			// else, leave the bomb at the current index
			field[row1][col1].isBomb = true
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
			field[i][j].x = j
			field[i][j].y = i
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
// Surrounding empty cells are uncovered automatically
func (ms *Minesweeper) Uncover(x, y int) (error, bool) {
	if x >= ms.height || y >= ms.width {
		return errors.New("x or y is larger than a field size"), false
	}

	cell := &ms.field[y][x]
	if !cell.isBomb {
		// uncover surrounding cells
		queue := list.New()
		queue.PushBack(cell)

		for queue.Len() > 0 {
			elem := queue.Front()
			currentCell := elem.Value.(*Cell)
			currentCell.uncovered = true
			queue.Remove(elem)

			for i := Max(0, currentCell.y-1); i < Min(int8(ms.height), currentCell.y+2); i++ {
				for j := Max(0, currentCell.x-1); j < Min(int8(ms.width), currentCell.x+2); j++ {
					neighbourCell := &ms.field[i][j]
					if !neighbourCell.isBomb && !neighbourCell.uncovered && neighbourCell.label == 0 {
						queue.PushBack(neighbourCell)
					}
				}
			}
		}
	} else {
		cell.uncovered = true
	}

	return nil, cell.isBomb
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

	// q := list.New()
	// cell := Cell{}
	// q.PushBack(&cell)
	// val := q.Front().Value.(*Cell)
	// val.uncovered = true
	// fmt.Println(val)
}
