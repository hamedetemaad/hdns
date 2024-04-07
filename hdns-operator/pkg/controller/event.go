package controller

type eventType string

const (
	addDNSBlock             eventType = "addDNSBlock"
)

type event struct {
	eventType      eventType
	oldObj, newObj interface{}
}