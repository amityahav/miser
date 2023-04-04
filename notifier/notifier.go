package notifier

import (
	"miser"
	"miser/rules"
)

type Notifier interface {
	Notify([]*rules.Alert)
}

func NewNotifier(n miser.Notifier) (Notifier, error) {
	switch n.Type {
	case "webhook":
		return NewWebhookNotifier(n.Name, n.Endpoint, n.Headers, n.Retries)
	default:
		return nil, ErrUnsupportedNotifier{name: n.Type}
	}
}
