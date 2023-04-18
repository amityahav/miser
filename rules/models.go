package rules

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	ElasticQuery = ".es-query"
	LogThreshold = "logs.alert.document.count"
)

type ElasticResult struct {
	Hits struct {
		HitsArr []AlertDoc `json:"hits"`
	} `json:"hits"`
}

type ElasticDeletePayload struct {
	Query Query `json:"query"`
}

type Query struct {
	Ids Ids `json:"ids"`
}

type Ids struct {
	Values []string `json:"values"`
}

func NewDeletePayload(values []string) ElasticDeletePayload {
	return ElasticDeletePayload{Query: Query{Ids: Ids{Values: values}}}
}

type ElasticSearchPayload struct {
	Source bool `json:"_source"`
	Size   int  `json:"size"`
}

func NewSearchPayload() ElasticSearchPayload {
	return ElasticSearchPayload{
		Source: true,
		Size:   10000,
	}
}

type AlertDoc struct {
	DocId       string      `json:"_id"`
	AlertFields AlertFields `json:"_source"`
}

type Alert interface {
	GetStatus() string
	UniqueKey() string
	TriggeredTime() time.Time
}

type AlertFields struct {
	Alert Alert
}

func (af *AlertFields) UnmarshalJSON(data []byte) error {
	p := struct {
		RuleID   string          `json:"rule_id"`
		RuleName string          `json:"rule_name"`
		RuleType string          `json:"rule_type"`
		Alert    json.RawMessage `json:"alert"`
	}{}

	err := json.Unmarshal(data, &p)
	if err != nil {
		return err
	}
	switch p.RuleType {
	case ElasticQuery:
		var eqa ElasticQueryAlert
		err = json.Unmarshal(p.Alert, &eqa)
		if err != nil {
			return err
		}

		eqa.RuleId, eqa.RuleName, eqa.RuleType = p.RuleID, p.RuleName, p.RuleType
		af.Alert = &eqa
	case LogThreshold:
		var lta LogThresholdAlert
		err = json.Unmarshal(p.Alert, &lta)
		if err != nil {
			return err
		}

		lta.RuleId, lta.RuleName, lta.RuleType = p.RuleID, p.RuleName, p.RuleType
		af.Alert = &lta
	default:
		return fmt.Errorf("unkown rule type: %s", p.RuleType)
	}

	return nil
}

type BaseAlert struct {
	RuleId         string                 `json:"rule_id"`
	RuleName       string                 `json:"rule_name"`
	RuleType       string                 `json:"rule_type"`
	AlertId        string                 `json:"alert_id"`
	Triggered      time.Time              `json:"triggered"`
	Status         string                 `json:"status"`
	ContextMessage string                 `json:"context_message"`
	CustomData     map[string]interface{} `json:"custom_data"`
}

func (ba *BaseAlert) GetStatus() string { return ba.Status }

func (ba *BaseAlert) TriggeredTime() time.Time { return ba.Triggered }

type ElasticQueryAlert struct {
	BaseAlert
	Value string `json:"value"`
}

func (a *ElasticQueryAlert) UniqueKey() string { return fmt.Sprintf("%s%s", a.RuleId, a.AlertId) }

type LogThresholdAlert struct {
	BaseAlert
	MatchingDocs string `json:"matching_docs"`
	GroupingKey  string `json:"grouping_key"`
}

func (a *LogThresholdAlert) UniqueKey() string { return fmt.Sprintf("%s%s", a.RuleId, a.GroupingKey) }
