package gpkg

type EventType uint8

const (
	EventStarted EventType = iota
	EventCompleted
	EventDownloadStarted
	EventDownloadCompleted
	EventPickStarted
	EventSkipped
)

type Event struct {
	Type EventType
	Spec PackageSpec
	Data interface{}
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

type EventDataDownload struct {
	ContentLength int64
	CurrentRef    string
	NextRef       string
}

func (b *EventBuilder) downloadStarted(dl Downloader, currentRef, nextRef string) *Event {
	return &Event{
		Type: EventDownloadStarted,
		Spec: b.spec,
		Data: EventDataDownload{
			ContentLength: dl.GetContentLength(),
			CurrentRef:    currentRef,
			NextRef:       nextRef,
		},
	}
}

func (b *EventBuilder) downloadCompleted() *Event {
	return &Event{
		Type: EventDownloadCompleted,
		Spec: b.spec,
	}
}

func (b *EventBuilder) pickStarted() *Event {
	return &Event{
		Type: EventPickStarted,
		Spec: b.spec,
	}
}

type EventDataSkipped struct {
	CurrentRef string
}

func (b *EventBuilder) skipped(currentRef string) *Event {
	return &Event{
		Type: EventSkipped,
		Spec: b.spec,
		Data: EventDataSkipped{
			CurrentRef: currentRef,
		},
	}
}
