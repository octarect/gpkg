package gpkg

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

var defaultTestEventBuilder = newEventBuilder(&NopSpec{})

// NopSpec implements PackageSpec interface and its methods do nothing.
// You can use this struct in test code.
type NopSpec struct {
	*CommonSpec
}

var _ PackageSpec = &NopSpec{}

func (s *NopSpec) Validate() error {
	return nil
}

func (s *NopSpec) DirName() string {
	return "nop"
}

func (s *NopSpec) DisplayName() string {
	return "nop"
}

// DummyDownloader implements Downloader interface.
// When you call read(), it returns `buf` instead of actually downloading from remote.
type DummyDownloader struct {
	io.ReadCloser
	total int64
}

var _ Downloader = &DummyDownloader{}

func newDummyDownloader() *DummyDownloader {
	buf := "dummy contents"
	r := io.NopCloser(bytes.NewBufferString(buf))
	return &DummyDownloader{ReadCloser: r, total: int64(len(buf))}
}

func (dl *DummyDownloader) GetAssetName() string {
	return "dummy"
}

func (dl *DummyDownloader) GetContentLength() int64 {
	return dl.total
}

func TestNewEventBuilder(t *testing.T) {
	eb := newEventBuilder(&NopSpec{})
	assert.Equal(t, eb.spec, &NopSpec{})
}

func TestEventBuilder_started(t *testing.T) {
	got := defaultTestEventBuilder.started()
	expected := &Event{
		Type: EventStarted,
	}
	checkDiff(t, Event{}, expected, got, "Spec")
}

func TestEventBuilder_completed(t *testing.T) {
	got := defaultTestEventBuilder.completed()
	expected := &Event{
		Type: EventCompleted,
	}
	checkDiff(t, Event{}, expected, got, "Spec")
}

func TestEventBuilder_downloadStarted(t *testing.T) {
	dl := newDummyDownloader()
	got := defaultTestEventBuilder.downloadStarted(dl)
	expected := &Event{
		Type: EventDownloadStarted,
		Data: EventDataDownload{
			ContentLength: dl.GetContentLength(),
		},
	}
	checkDiff(t, Event{}, expected, got, "Spec")
}

func TestEventBuilder_downloadCompleted(t *testing.T) {
	got := defaultTestEventBuilder.downloadCompleted()
	expected := &Event{
		Type: EventDownloadCompleted,
	}
	checkDiff(t, Event{}, expected, got, "Spec")
}

func TestEventBuilder_pickStarted(t *testing.T) {
	got := defaultTestEventBuilder.pickStarted()
	expected := &Event{
		Type: EventPickStarted,
	}
	checkDiff(t, Event{}, expected, got, "Spec")
}
