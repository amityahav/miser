package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"miser"
	"miser/notifier"
	"miser/rules"
	"strings"
	"time"
)

type Miser struct {
	esClient     *elasticsearch.Client
	syncInterval time.Duration
	index        string
	notifiers    []notifier.Notifier
	metrics      Metrics
	Logger       *logrus.Logger
}

type Metrics struct {
	promReg    *prometheus.Registry
	notifyFail *prometheus.GaugeVec
}

func NewMiser(cfg *miser.Config) (*Miser, error) {
	esCfg := elasticsearch.Config{
		Addresses: []string{cfg.ESHost},
		Username:  cfg.ESUsername,
		Password:  cfg.ESPassword,
	}

	c, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, err
	}

	m := Miser{
		esClient:     c,
		syncInterval: cfg.SyncInterval,
		index:        cfg.AlertsIndex,
		metrics: Metrics{
			promReg: prometheus.NewRegistry(),
			notifyFail: prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Name: "notify_fail",
			}, []string{"type", "name"}),
		},
		Logger: logrus.New(),
	}

	m.metrics.promReg.MustRegister(m.metrics.notifyFail)

	for _, n := range cfg.Notifiers {
		nt, err := notifier.NewNotifier(n)
		if err != nil {
			return nil, err
		}

		m.notifiers = append(m.notifiers, nt)
	}

	m.Logger.SetFormatter(&logrus.JSONFormatter{})

	return &m, nil
}

func (m *Miser) Sync() error {
	ticker := time.NewTicker(m.syncInterval)

	for {
		select {
		case <-ticker.C:
			m.Logger.Info("Started new iteration...")
			err := m.sync()
			if err != nil {
				m.Logger.Errorf("Sync: %s", err.Error())
			}
		}
	}
}

func (m *Miser) sync() error {
	var buf bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	q := rules.NewSearchPayload()
	if err := json.NewEncoder(&buf).Encode(q); err != nil {
		return err
	}
	body := strings.NewReader(buf.String())

	res, err := m.esClient.Search(
		m.esClient.Search.WithContext(ctx),
		m.esClient.Search.WithBody(body),
		m.esClient.Search.WithIndex(m.index),
		m.esClient.Search.WithErrorTrace(),
		m.esClient.Search.WithPretty())

	if err != nil {
		return err
	}

	defer res.Body.Close()

	var er rules.ElasticResult

	if res.IsError() {
		return errors.New(res.Status())
	}

	err = json.NewDecoder(res.Body).Decode(&er)
	if err != nil {
		return err
	}

	var (
		alertsToNotify    []rules.Alert
		alertDocsToDelete []string
	)
	alertsMap := make(map[string][]*rules.AlertDoc)
	statusToIndex := map[string]int{
		"active":   0,
		"resolved": 1,
	}

	for _, alert := range er.Hits.HitsArr {
		a := alert
		key := a.AlertFields.Alert.UniqueKey()
		status := a.AlertFields.Alert.GetStatus()

		if _, ok := alertsMap[key]; !ok {
			alertsMap[key] = make([]*rules.AlertDoc, 2)
		}

		al := alertsMap[key][statusToIndex[status]]
		if al != nil { // alert was already spotted with <status>, save only the latest
			if al.AlertFields.Alert.TriggeredTime().Before(a.AlertFields.Alert.TriggeredTime()) {
				alertDocsToDelete = append(alertDocsToDelete, al.DocId)
				alertsMap[key][statusToIndex[status]] = &a
			} else {
				alertDocsToDelete = append(alertDocsToDelete, a.DocId)
			}
		} else {
			alertsMap[key][statusToIndex[status]] = &a
		}
	}

	for _, alerts := range alertsMap {
		active, resolved := alerts[0], alerts[1]
		if resolved != nil {
			alertsToNotify = append(alertsToNotify, resolved.AlertFields.Alert)
			alertDocsToDelete = append(alertDocsToDelete, resolved.DocId)
			if active != nil {
				if active.AlertFields.Alert.TriggeredTime().After(resolved.AlertFields.Alert.TriggeredTime()) {
					alertsToNotify = append(alertsToNotify, active.AlertFields.Alert)
				} else {
					alertDocsToDelete = append(alertDocsToDelete, active.DocId)
				}
			}
		} else {
			alertsToNotify = append(alertsToNotify, active.AlertFields.Alert)
		}
	}

	if len(alertsToNotify) > 0 {
		for _, n := range m.notifiers {
			go func(notifier notifier.Notifier) {
				labels := prometheus.Labels{
					"type": notifier.GetType(),
					"name": notifier.GetName()}

				err = notifier.Notify(alertsToNotify)
				if err != nil {
					m.Logger.WithError(err).Error("failed to notify")
					m.metrics.notifyFail.With(labels).Set(1)
				} else {
					m.metrics.notifyFail.With(labels).Set(0)
				}
			}(n)
		}
	}

	// TODO: handle the case where some resolved alerts failed to be notified but their docs were deleted successfully.
	if len(alertDocsToDelete) > 0 {
		err = m.DeleteDocs(alertDocsToDelete)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Miser) DeleteDocs(docsIds []string) error {
	var buf bytes.Buffer

	q := rules.NewDeletePayload(docsIds)
	if err := json.NewEncoder(&buf).Encode(q); err != nil {
		return err
	}
	body := strings.NewReader(buf.String())
	res, err := m.esClient.DeleteByQuery([]string{m.index}, body)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.IsError() {
		return errors.New(res.Status())
	}

	return nil
}

func (m *Miser) GetPromRegistry() *prometheus.Registry { return m.metrics.promReg }
