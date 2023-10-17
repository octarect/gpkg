package gpkg

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

var defaultTestEventBuilder = newEventBuilder(&NopSpec{})

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
	got := defaultTestEventBuilder.downloadStarted(dl, "v0.1", "v0.2")
	expected := &Event{
		Type: EventDownloadStarted,
		Data: EventDataDownload{
			ContentLength: dl.GetContentLength(),
			CurrentRef:    "v0.1",
			NextRef:       "v0.2",
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

func TestEventBuilder_skipped(t *testing.T) {
	got := defaultTestEventBuilder.skipped("v1.0.0")
	expected := &Event{
		Type: EventSkipped,
		Data: EventDataSkipped{
			CurrentRef: "v1.0.0",
		},
	}
	checkDiff(t, Event{}, expected, got, "Spec")
}
