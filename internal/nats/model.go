package nats

type JszResponse struct {
	AccountDetails []AccountDetails `json:"account_details"`
}

type AccountDetails struct {
	StreamDetail []StreamDetail `json:"stream_detail"`
}

type StreamDetail struct {
	Name           string           `json:"name"`
	ConsumerDetail []ConsumerDetail `json:"consumer_detail"`
}

type ConsumerDetail struct {
	Name       string `json:"name"`
	NumPending int    `json:"num_pending"`
}
