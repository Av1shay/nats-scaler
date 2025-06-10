package nats

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/Av1shay/nats-scaler/internal/errs"
	"github.com/stretchr/testify/require"
)

type mockNatsServer struct {
	t              *testing.T
	resp           JszResponse
	statusCode     int
	forceError     bool
	gotQueryParams url.Values
}

func (m *mockNatsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m.forceError {
		http.Error(w, "simulated error", m.statusCode)
		return
	}

	require.Equal(m.t, http.MethodGet, r.Method)
	require.Equal(m.t, "/jsz", r.URL.Path)

	m.gotQueryParams = r.URL.Query()

	w.WriteHeader(m.statusCode)
	err := json.NewEncoder(w).Encode(m.resp)
	require.NoError(m.t, err)
}

func TestService_GetPendingMessages(t *testing.T) {
	ctx := context.Background()
	httpClient := &http.Client{Timeout: 5 * time.Second}
	s := NewService(httpClient)

	natsServer := &mockNatsServer{
		t: t,
		resp: JszResponse{
			AccountDetails: []AccountDetails{{
				StreamDetail: []StreamDetail{
					{Name: "EVENTS", ConsumerDetail: []ConsumerDetail{{
						Name:       "xxx",
						NumPending: 250,
					}}},
				},
			}},
		},
		statusCode: 200,
	}
	ts := httptest.NewServer(natsServer)
	t.Cleanup(ts.Close)

	res, err := s.GetPendingMessages(ctx, ts.URL, "EVENTS", "xxx")
	require.NoError(t, err)
	require.Equal(t, 250, res)
	require.Equal(t, url.Values{"acc": {"$G"}, "consumers": {"1"}, "leader_only": {"1"}}, natsServer.gotQueryParams)

	_, err = s.GetPendingMessages(ctx, ts.URL, "NOT-EXIST", "xxx")
	require.ErrorContains(t, err, "couldn't find NATS account")

	natsServer.resp = struct {
		AccountDetails []AccountDetails `json:"account_details"`
	}{AccountDetails: nil}
	_, err = s.GetPendingMessages(ctx, ts.URL, "EVENTS", "xxx")
	require.ErrorIs(t, err, ErrNoAccountFound)

	natsServer.statusCode = 500
	res, err = s.GetPendingMessages(ctx, ts.URL, "EVENTS", "xxx")
	var scerr *errs.HTTPStatusCodeErr
	require.ErrorAs(t, err, &scerr)
	require.Equal(t, 0, res)
	require.Equal(t, 500, scerr.Code)
	require.Equal(t, "{\"account_details\":null}\n", string(scerr.Body))
}
