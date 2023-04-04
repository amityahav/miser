package notifier

import "fmt"

type ErrUnsupportedNotifier struct {
	name string
}

func (e ErrUnsupportedNotifier) Error() string {
	return fmt.Sprintf("unsupported notifier of type: %s", e.name)
}
