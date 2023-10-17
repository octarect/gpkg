package main

import (
	"fmt"
	"time"

	"github.com/cheggaaa/pb/v3"
)

type ProgressBar struct {
	bar *pb.ProgressBar

	started bool
}

func newProgressBar(name string) *ProgressBar {
	bar := pb.Full.New(0)
	bar.SetRefreshRate(time.Millisecond * 500)
	bar.Set(pb.Bytes, true)
	bar.Set("prefix", fmt.Sprintf("%s: ", name))

	return &ProgressBar{
		bar: bar,
	}
}

func (b *ProgressBar) SetTotal(n int64) {
	b.bar.SetTotal(n)
}

func (b *ProgressBar) Write(data []byte) (int, error) {
	n := len(data)
	b.bar.Add(n)
	return n, nil
}

func (b *ProgressBar) Start() {
	b.started = true
	b.bar.Start()
}

func (b *ProgressBar) Finish() {
	if b.started {
		b.bar.Finish()
	}
}
