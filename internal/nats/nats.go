package nats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Av1shay/nats-scaler/internal/errs"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	GlobalAccountName = "$G" // we'll use only the global account for the sake of the demo
)

var NoAccountFoundErr = errors.New("no accounts found")

type Service struct {
	httpClient *http.Client
}

func NewService(httpClient *http.Client) *Service {
	return &Service{httpClient: httpClient}
}

func (c *Service) GetPendingMessages(ctx context.Context, baseURL, streamName, consumerName string) (int, error) {

	// NOTE: This uses an HTTP call because the assignment explicitly requires querying
	// the monitoring endpoint. In production, I would use the official NATS Go client:
	// https://pkg.go.dev/github.com/nats-io/nats.go
	// which communicates over NATS using the JetStream wire protocol:
	// https://docs.nats.io/reference/reference-protocols/nats_api_reference

	v := url.Values{}
	v.Add("acc", GlobalAccountName)
	v.Add("consumers", "1")
	v.Add("leader_only", "1")
	u := fmt.Sprintf("%s/jsz?%s", strings.TrimRight(baseURL, "/"), v.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create new http request to %s: %w", u, err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to query nats subscriptions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return 0, &errs.HTTPStatusCodeErr{
			Code: resp.StatusCode,
			Body: b,
		}
	}

	var data JszResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, fmt.Errorf("failed to decode JSON: %w", err)
	}

	if len(data.AccountDetails) == 0 {
		return 0, NoAccountFoundErr
	}

	// assuming we only have one account, we can just look at the first one (that supposes to be $G)
	account := data.AccountDetails[0]
	for _, stream := range account.StreamDetail {
		if stream.Name == streamName {
			for _, consumer := range stream.ConsumerDetail {
				if consumer.Name == consumerName {
					return consumer.NumPending, nil
				}
			}
		}
	}

	return 0, fmt.Errorf("couldn't find NATS account <%s>, stram <%s>, consumer <%s>", GlobalAccountName, streamName, consumerName)
}
