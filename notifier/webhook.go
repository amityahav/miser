package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"miser/rules"
	"net/http"
	"time"
)

type Webhook struct {
	Name        string
	Client      http.Client
	Endpoint    string
	Headers     map[string]string
	Retries     int
	failedCache []rules.Alert
	logger      *logrus.Logger
}

func NewWebhookNotifier(name, endpoint string, headers map[string]string, retries int) (*Webhook, error) {
	l := logrus.New()
	l.SetFormatter(&logrus.JSONFormatter{})
	w := Webhook{
		Name: name,
		Client: http.Client{
			Transport: http.DefaultTransport,
			Timeout:   0,
		},
		Endpoint: endpoint,
		Headers:  headers,
		Retries:  retries,
		logger:   l,
	}

	return &w, nil
}

func (w *Webhook) Notify(alerts []rules.Alert) error {
	type as struct {
		Arr []rules.Alert `json:"alerts"`
	}

	a := as{Arr: alerts}

	for i := 0; i < w.Retries; i++ {
		j, err := json.Marshal(a)
		if err != nil {
			w.logger.WithError(err).Errorf("Webhook %s", w.Name)
			time.Sleep(time.Second)
			continue
		}

		body := bytes.NewReader(j)

		req, err := http.NewRequest(http.MethodPost, w.Endpoint, body)
		if err != nil {
			w.logger.WithError(err).Errorf("Webhook %s", w.Name)
			time.Sleep(time.Second)
			continue
		}

		addHeaders(req, w.Headers)

		res, err := w.Client.Do(req)
		if err != nil {
			w.logger.WithError(err).Errorf("Webhook %s", w.Name)
			time.Sleep(time.Second)
			continue
		}

		if res.StatusCode != http.StatusOK {
			w.logger.Errorf("webhook %s returned %d code", w.Name, res.StatusCode)
			time.Sleep(time.Second)
			continue
		} else {
			return nil
		}
	}

	return fmt.Errorf("%s webhook - url: %s", w.Name, w.Endpoint)
}

func (w *Webhook) GetType() string { return "webhook" }

func (w *Webhook) GetName() string { return w.Name }

func addHeaders(req *http.Request, headers map[string]string) {
	for k, v := range headers {
		req.Header.Add(k, v)
	}
}
