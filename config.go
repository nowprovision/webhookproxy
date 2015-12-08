package webhookproxy

import "time"
import "net"

type Config struct {
	ShowDebugInfo      bool
	BackQueueSize      int
	MaxWaitSeconds     time.Duration
	WebhookWhiteList   []*net.IPNet
	PollReplyWhiteList []*net.IPNet
	MaxPayloadSize     int64
	TryLaterStatusCode int
	UseLongPoll        bool
	LongPollWait       time.Duration
	Secret             string
}
