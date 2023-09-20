package gpkg

type EventType uint8
const (
	EventStarted   EventType = iota
	EventCompleted EventType = iota
	EventDownloadStarted
)

type Event struct {
	Type EventType
	Spec PackageSpec
	Data interface{}
}

type EventDataDownload struct {
	ContentLength int64
}

type EventBuilder struct {
	spec PackageSpec
}

func newEventBuilder(spec PackageSpec) *EventBuilder {
	return &EventBuilder{spec}
}

func (b *EventBuilder) started() *Event {
	return &Event{
		Type: EventStarted,
		Spec: b.spec,
	}
}

func (b *EventBuilder) completed() *Event {
	return &Event{
		Type: EventCompleted,
		Spec: b.spec,
	}
}

func (b *EventBuilder) downloadStarted(dl Downloader) *Event {
	return &Event{
		Type: EventDownloadStarted,
		Spec: b.spec,
		Data: EventDataDownload{
			ContentLength: dl.GetContentLength(),
		},
	}
}
