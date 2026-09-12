package checkout

type Queue interface {
	Enqueue(Event) error
}

type Worker interface {
	Process(Event) error
}
