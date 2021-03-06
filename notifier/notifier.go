package notifier

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/dxas90/cole/configuration"
	"github.com/dxas90/cole/dmtimer"
	"github.com/dxas90/cole/slack"
	"github.com/dxas90/cole/teams"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/prometheus/alertmanager/template"
	log "github.com/sirupsen/logrus"
)

// NotificationSet - the body of the alert message from cole
type NotificationSet struct {
	Message template.Data
	Config  configuration.Conf
	Timers  dmtimer.DmTimers
}

func (n *NotificationSet) constructBody() ([]byte, error) {
	jsonBody, err := json.Marshal(n)
	if err != nil {
		return jsonBody, err
	}
	return jsonBody, nil

}

func (n *NotificationSet) Alert() {
	log.Println("Sending Alert. Missed deadman switch notification.")
	// set up for future specific notification types
	// switch on n.Config.SenderType
	switch n.Config.SenderType {
	case "slack":
		n.slack()
	case "pagerduty":
		n.pagerDuty()
	case "teams":
		n.teams()
	case "telegram":
		n.telegram()
	default:
		// thinking I should just pass the whole alert message here
		n.slack()
	}
	//

}

// genericWebHook - takes url as and expects json to be the payload
func (n *NotificationSet) genericWebHook(jsonBody []byte) {

	log.Println("genericWebHook method")
	req, err := http.NewRequest(
		n.Config.HTTPMethod,
		n.Config.HTTPEndpoint,
		bytes.NewBuffer(jsonBody),
	)
	log.Info("Notification received to web hook!")
	if err != nil {
		log.Error("Error:", err)
		// If we couldn't create a request object, we have nothing to send
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error("Error:", err)
		// Sending the webhook failed, so don't attempt to process the response
		return
	}
	defer resp.Body.Close()

	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("Error reading response body", err)
	}

	log.Info(string(respData))

}

func (n *NotificationSet) teams() {
	// DEBUG
	log.Println("teams method")

	fact1 := teams.Fact{Name: "Alertname:", Value: "Watchdog Failure"}
	fact2 := teams.Fact{Name: "Severity:", Value: "Critical"}
	fact3 := teams.Fact{Name: "Environment:", Value: n.Config.ClusterLabel}
	fact4 := teams.Fact{Name: "Message:", Value: "Your entire Prometheus alerting pipeline is failing. Deadman switch noticed a Watchdog failure"}
	fact5 := teams.Fact{Name: "Action Required:", Value: "Please make sure Alert manager service is functional"}
	section := teams.Section{ActivityTitle: "*** IMPORTANT !!! ***", Facts: []teams.Fact{fact1, fact2, fact3, fact4, fact5}}

	payload := teams.TeamPayload{
		Summary:  "New Alert by @Cole",
		Title:    "Prometheus Alertmanager Service DOWN",
		Text:     "Generated by: Cole Deadman Switch",
		Color:    "F74721",
		Sections: []teams.Section{section},
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		log.Error("Error marshalling new data", err)
	}
	n.genericWebHook(jsonBody)
}

func (n *NotificationSet) slack() {
	// DEBUG
	log.Println("slack method")
	payload := slack.Payload{
		Text:      "Missed DeadManSwitch Alert  - " + n.Message.CommonAnnotations["message"],
		Username:  n.Config.SlackUsername,
		Channel:   n.Config.SlackChannel,
		IconEmoji: n.Config.SlackIcon,
	}
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		log.Error("Error marshalling new data", err)
	}
	n.genericWebHook(jsonBody)
}

func (n *NotificationSet) pagerDuty() {

	pdPayload := pagerduty.V2Payload{
		Summary:  "Missed DeadManSwitch Check in.",
		Source:   "cole",
		Severity: n.Message.CommonLabels["severity"],

		Timestamp: time.Now().Format(time.RFC3339),
		Group:     n.Message.CommonLabels["job"],
		Class:     n.Message.CommonLabels["alertname"],
		Details:   n.Message.CommonAnnotations["message"],
	}
	event := pagerduty.V2Event{
		RoutingKey: n.Config.PDIntegrationKey,
		Action:     "trigger",
		DedupKey:   n.Message.CommonLabels["alertname"],
		Client:     "Cole - Dead Man Switch Monitor",
		Payload:    &pdPayload,
	}

	resp, err := pagerduty.ManageEvent(event)
	if err != nil {
		log.Errorf("Error Created Event in Pager Duty: %s", err)
		return
	}
	log.Printf("%+v", resp)

}

func (n *NotificationSet) telegram() {
	bot, err := tgbotapi.NewBotAPI(n.Config.TelegramToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized Telegram on account %s", bot.Self.UserName)

	msg := tgbotapi.NewMessage(n.Config.TelegramChatID, "Missed DeadManSwitch Alert  - "+n.Message.CommonAnnotations["message"])

	resp, err := bot.Send(msg)
	if err != nil {
		log.Errorf("Error Created Event in Pager Duty: %s", err)
		return
	}
	log.Printf("%+v", resp)
}
