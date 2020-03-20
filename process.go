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
	Total int
}

func NewProcess(name string, total int) *process {
	if total <= 0 {
		return nil
	}
	p := &process{name, -1, total}
	p.Print(0)
	return p
}

func (p *process) Print(current int) {
	currentProcess := current * 100 / p.Total
	if currentProcess > 100 {
		currentProcess = 100
	}
	if currentProcess > p.Value {
		p.Value = currentProcess
		fmt.Printf("[%s] Process: %d%%\r", p.Name, p.Value)
		if currentProcess == 100 {
			fmt.Println()
		}
	}
}
