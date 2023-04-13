# MISER
Miser provides extended support for rule-type alert connectors in elasticsearch such as: Custom webhook, Slack, PagerDuty, etc.. .
It was created for people/organizations who own the free version of elastic and only have the basic connectors such as: Index and Server log.

# How it Works

Rules will be created using the Index connector and Miser will manage the state of these rule's alerts via the index specified in the connector.

Miser will support all types of rules as long as the Action part of the alert is defined as follows:

![](action_documents/how-to.gif)

in the above demo I've created a log-threshold rule via the Kibana UI, but it can be any other rule as well.

you can find the action's payloads [here](./action_documents)

NOTE: action payloads contain multiple fields from several rule types all combined. 
 `matching_docs` and `grouping_key` will be populated when using Log Threshold rules while `value` will be populated when using Elastic Query rules.
you can differentiate between alert types by the `rule_type` field in the payload.
## Configuration

```yaml
# Elasticsearch
es_host: http://localhost:9200/
es_username : ""
es_password : ""
alerts_index : alerts* # Index/ Data-view configured in the Index-connector

# Miser
sync_interval: 1m # the interval which miser will process the alerts in <alerts_index>
notifiers :
  - type: webhook
    name: my-webhook
    endpoint: http://127.0.0.1:5000/
    retries: 1 # num of retries when notify fails
    headers: # custom headers for the webhook request
      Content-Type: application/json
```
Note: currently only Webhook type notifier is supported.
## How to run
1. ``make bin``
2. ``cd bin ``
3. `./Miser --config=path/to/config`

### Some caveats:
1. I was testing Miser with Elasticsearch v8.6.2.
2. There are some strange behaviors I faced when enabling/disabling rules in Kibana, you can see [this](https://discuss.elastic.co/t/elastic-log-threshold-rule-problem/329213) post.

   1. If a user disables a rule, no 'resolved' event will be written to the index. hence Miser will keep notify about it, so there's to be done outside of Miser in terms of manually/Programmatically delete those events when disabling a rule.
   2. Because of the issue where events are not written when re-enabling a rule, I would suggest to store those rules somewhere else, and when disabling a rule -> delete the rule in Elastic and when enabling it -> recreate it, since re-creation of the rule fixes this issue.
3. regarding (2) Miser relies on the state of the alerts managed by elastic and written to the alerts index, so if those events are not written properly, Miser won't function properly as well.
 
## Contribution

Feel free to open a pull request, I would be happy to review it.
