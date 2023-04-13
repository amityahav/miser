package rules

import "time"

type ElasticResult struct {
	Hits struct {
		HitsArr []Alert `json:"hits"`
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

type Alert struct {
	DocId  string      `json:"_id"`
	Source AlertFields `json:"_source"`
}

type AlertFields struct {
	RuleId         string    `json:"rule_id"`
	RuleName       string    `json:"rule_name"`
	RuleType       string    `json:"rule_type"`
	AlertId        string    `json:"alert_id"`
	Triggered      time.Time `json:"triggered"`
	Status         string    `json:"status"`
	ContextMessage string    `json:"context_message"`

	// Fields for Log-Threshold rules
	MatchingDocs string `json:"matching_docs"`
	GroupingKey  string `json:"grouping_key"`

	// Fields for Elastic-Query rules
	Value string `json:"value"`

	CustomData map[string]interface{} `json:"custom_data"`
}
