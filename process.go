package main

import (
	"fmt"
)

/**
 * 进度
 */
type process struct {
	Name  string
	TotalNum int
    SearchNum int
    MatchNum int
    SaveKeyNum int
    SaveDataNum int
    DeleteNum int
    Percent float64
}

func NewProcess(name string) *process {
	p := &process{name, 0, 0, 0, 0, 0, 0, 0}
	return p
}

func (p *process) SetTotal(total int) {
    p.TotalNum = total
}

func (p *process) IncrSearchNum(num int) {
    p.SearchNum += num
}

func (p *process) IncrMatchNum(num int) {
    p.MatchNum += num
}

func (p *process) IncrSaveKeyNum(num int) {
    p.SaveKeyNum += num
}

func (p *process) IncrSaveDataNum(num int) {
    p.SaveDataNum += num
}

func (p *process) IncrDeleteNum(num int) {
    p.DeleteNum += num
}

func (p *process) Print() {
	percent := float64(p.SearchNum) * 100 / float64(p.TotalNum)
	if percent > 100 {
		percent = 100
	}
	if percent > p.Percent {
		p.Percent = percent
        fmt.Printf("[%s] Process: %.2f%% Total Keys: %d  Search Keys: %d  Match Key: %d  Save Keys: %d  Save Data: %d  Delete: %d\r", p.Name, p.Percent, p.TotalNum, p.SearchNum, p.MatchNum, p.SaveKeyNum, p.SaveDataNum, p.DeleteNum)
		if percent == 100 {
			fmt.Println()
		}
	}
}
