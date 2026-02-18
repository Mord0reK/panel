package tests

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"backend/internal/api"
	"backend/internal/buffer"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSEAPI(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bm := buffer.NewBufferManager()
	handler := api.NewSSEHandler(db, bm, "*")
	r := mux.NewRouter()
	r.HandleFunc("/api/metrics/live/all", handler.HandleLiveAll).Methods("GET")

	// Prepare data
	_, err := db.Exec("INSERT INTO servers (uuid, hostname, approved) VALUES (?, ?, ?)", "s1", "host1", true)
	require.NoError(t, err)
	bm.AddHostMetric("s1", buffer.HostMetricPoint{Timestamp: 1000, CPU: 15.5, MemUsed: 1234})

	// Start server
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Client request
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/api/metrics/live/all", nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))

	// Read first message
	scanner := bufio.NewScanner(resp.Body)
	found := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			assert.Contains(t, line, "15.5")
			found = true
			break
		}
	}
	assert.True(t, found)
}
