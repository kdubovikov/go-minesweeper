package main

import "testing"

func TestNewMinesweeper(t *testing.T) {
	expectedNumBombs := int8(10)
	err, minesweeper := NewMinesweeper(8, 8, expectedNumBombs)

	if err != nil {
		t.Errorf("Error while creating minesweeper: %s", err.Error())
	}

	actualNumBombs := int8(0)
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			if _, cell := minesweeper.Get(i, j); cell.IsBomb() {
				actualNumBombs++
			}
		}
	}

	if actualNumBombs != expectedNumBombs {
		t.Errorf("%d bombs actual != %d bombs expected", actualNumBombs, expectedNumBombs)
	}
}
