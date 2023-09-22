package gpkg

import (
	"fmt"
	"io"
	"time"

	"github.com/cheggaaa/pb/v3"
)

type ProgressWriter interface {
	io.Writer
	Print(string)
	SetTotal(int64)
}

type ProgressUI struct {
	pool *pb.Pool
	started bool
}

func NewProgressUI() *ProgressUI {
	return &ProgressUI{
		pool: pb.NewPool(),
		started: false,
	}
}

func (u *ProgressUI) AddProgressBar(msg string) *ProgressBar {
	b := NewProgressBar(msg)
	u.pool.Add(b.Bar)
	return b
}

func (u *ProgressUI) Start() {
	u.pool.Start()
}

func (u *ProgressUI) Stop() {
	u.pool.Stop()
}

type ProgressBar struct {
	Bar   *pb.ProgressBar
	total int64
	fixedTotal bool
}

func NewProgressBar(name string) *ProgressBar {
	bar := pb.Full.New(0)
	bar.SetRefreshRate(time.Millisecond * 500)
	bar.Set(pb.Bytes, true)
	bar.Set("prefix", fmt.Sprintf("%s: ", name))

	return &ProgressBar{ Bar: bar }
}

func (b *ProgressBar) Write(data []byte) (int, error) {
	if !b.fixedTotal {
		b.SetTotal(b.Bar.Current() + 2048)
	}

	n := len(data)
	b.Bar.Add(n)
	return n, nil
}

func (b *ProgressBar) Print(msg string) {
	b.Bar.SetTemplateString("aborted")
	return
}

func (b *ProgressBar) SetTotal(total int64) {
	b.fixedTotal = true
	b.Bar.SetTotal(total)
}
