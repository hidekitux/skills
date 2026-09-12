package checkout

type Event struct {
	ID      string
	Payload []byte
}

func Handle(event Event) error {
	return process(event)
}

func process(Event) error {
	return nil
}
