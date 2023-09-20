package main

import (
	"fmt"
	"time"

	"github.com/cheggaaa/pb/v3"
)

type ProgressBar struct {
	Bar *pb.ProgressBar
}

func newProgressBar(name string) *ProgressBar {
	bar := pb.Full.New(0)
	bar.SetRefreshRate(time.Millisecond * 500)
	bar.Set(pb.Bytes, true)
	bar.Set("prefix", fmt.Sprintf("%s: ", name))

	return &ProgressBar{
		Bar: bar,
	}
}

func (b *ProgressBar) SetTotal(n int64) {
	b.Bar.SetTotal(n)
}

func (b *ProgressBar) Write(data []byte) (int, error) {
	n := len(data)
	b.Bar.Add(n)
	return n, nil
}
