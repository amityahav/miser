package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/elastic/go-elasticsearch/v8"
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
	Logger       *logrus.Logger
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
		Logger:       logrus.New(),
	}

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
		alertsToNotify    []*rules.Alert
		alertDocsToDelete []string
	)
	alertsMap := make(map[string][]*rules.Alert)
	statusToIndex := map[string]int{
		"active":   0,
		"resolved": 1,
	}

	for _, alert := range er.Hits.HitsArr {
		a := alert
		key := fmt.Sprintf("%s%s", a.Source.RuleId, a.Source.GroupingKey)
		status := a.Source.Status

		if _, ok := alertsMap[key]; !ok {
			alertsMap[key] = make([]*rules.Alert, 2)
		}

		al := alertsMap[key][statusToIndex[status]]
		if al != nil { // alert was already spotted with <status>, save only the latest
			if al.Source.Triggered.Before(a.Source.Triggered) {
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
			alertsToNotify = append(alertsToNotify, resolved)
			alertDocsToDelete = append(alertDocsToDelete, resolved.DocId)
			if active != nil {
				if active.Source.Triggered.After(resolved.Source.Triggered) {
					alertsToNotify = append(alertsToNotify, active)
				} else {
					alertDocsToDelete = append(alertDocsToDelete, active.DocId)
				}
			}
		} else {
			alertsToNotify = append(alertsToNotify, active)
		}
	}

	if len(alertsToNotify) > 0 {
		for _, n := range m.notifiers {
			go n.Notify(alertsToNotify)
		}
	}

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
