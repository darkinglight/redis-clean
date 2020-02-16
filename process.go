package main

import (
	"fmt"
)

/**
 * 进度
 */
type process struct {
	Name  string
	Value int
}

func (p *process) Print(currentProcess int) {
	if currentProcess > 100 {
		currentProcess = 100
	}
	if currentProcess > p.Value {
		p.Value = currentProcess
		fmt.Printf("%s Process: %d%%\r", p.Name, p.Value)
		if currentProcess == 100 {
			fmt.Println()
		}
	}
}
