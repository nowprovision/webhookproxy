package webhookproxy

import "time"
import "net"

type Config struct {
	ShowDebugInfo      bool
	BackQueueSize      int
	MaxWaitSeconds     time.Duration
	WebhookFilters     []*net.IPNet
	PollReplyFilters   []*net.IPNet
	MaxPayloadSize     int64
	TryLaterStatusCode int
	UseLongPoll        bool
	LongPollWait       time.Duration
	Secret             string
	FilteringEnabled   bool
	Hostname           string
	Id                 string
	Autoreply          bool
}
