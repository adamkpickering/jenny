package notify

// Notifier allows one goroutine to send a notification to one or more other
// goroutines.
type Notifier struct {
	channels map[string]chan struct{}
}

func New() *Notifier {
	notifier := &Notifier{
		channels: make(map[string]chan struct{}),
	}
	return notifier
}

func (notifier *Notifier) Register(id string) {
	notifier.channels[id] = make(chan struct{})
}

func (notifier *Notifier) Notify() {
	for _, channel := range notifier.channels {
		channel <- struct{}{}
	}
}

func (notifier *Notifier) Get(id string) chan struct{} {
	channel, ok := notifier.channels[id]
	if ok {
		return channel
	}
	return nil
}

func (notifier *Notifier) Close(id string) {
	channel, ok := notifier.channels[id]
	if ok {
		close(channel)
		delete(notifier.channels, id)
	}
}

func (notifier *Notifier) CloseAll() {
	for id := range notifier.channels {
		notifier.Close(id)
	}
}
