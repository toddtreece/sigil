//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const (
	defaultE2EBaseURL      = "http://localhost:8080"
	defaultE2EMySQLDSN     = "sigil:sigil@tcp(localhost:3306)/sigil?parseTime=true"
	defaultE2EHTTPTimeout  = 10 * time.Second
	defaultE2EPollInterval = 500 * time.Millisecond
	defaultE2EStartupWait  = 45 * time.Second
)

// TestStorageHotColdRoundTripLocal exercises local hot+cold query behavior
// against a running docker-compose stack:
//
// 1. export multiple generations in one request
// 2. verify hot-path query behavior
// 3. wait for compaction to mark rows durable in cold storage
// 4. remove one hot row and verify mixed hot+cold query behavior
// 5. remove remaining hot rows and verify cold-path query fallback
//
// Required runtime: SIGIL service, MySQL, and object storage up locally.
//
// Optional environment overrides:
//   - SIGIL_E2E_BASE_URL (default http://localhost:8080)
//   - SIGIL_E2E_MYSQL_DSN (default sigil:sigil@tcp(localhost:3306)/sigil?parseTime=true)
//   - SIGIL_E2E_TENANT_ID (default fake)
//   - SIGIL_E2E_STARTUP_WAIT (default 45s)
//   - SIGIL_E2E_COMPACTION_WAIT (default 3m)
func TestStorageHotColdRoundTripLocal(t *testing.T) {
	baseURL := strings.TrimSpace(getenvDefault("SIGIL_E2E_BASE_URL", defaultE2EBaseURL))
	mysqlDSN := strings.TrimSpace(getenvDefault("SIGIL_E2E_MYSQL_DSN", defaultE2EMySQLDSN))
	tenantID := strings.TrimSpace(getenvDefault("SIGIL_E2E_TENANT_ID", "fake"))
	startupWait := getenvDurationDefault("SIGIL_E2E_STARTUP_WAIT", defaultE2EStartupWait)
	compactionWait := getenvDurationDefault("SIGIL_E2E_COMPACTION_WAIT", 3*time.Minute)

	client := &http.Client{Timeout: defaultE2EHTTPTimeout}
	ctx := context.Background()

	waitForCondition(t, startupWait, defaultE2EPollInterval, "sigil health endpoint", func() (bool, error) {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/healthz", nil)
		if err != nil {
			return false, err
		}
		response, err := client.Do(request)
		if err != nil {
			return false, nil
		}
		defer func() { _ = response.Body.Close() }()
		return response.StatusCode == http.StatusOK, nil
	})

	db, err := sql.Open("mysql", mysqlDSN)
	if err != nil {
		t.Fatalf("open mysql connection: %v", err)
	}
	defer func() { _ = db.Close() }()

	pingCtx, cancelPing := context.WithTimeout(ctx, 15*time.Second)
	defer cancelPing()
	if err := db.PingContext(pingCtx); err != nil {
		t.Fatalf("ping mysql: %v", err)
	}

	runID := strconv.FormatInt(time.Now().UnixNano(), 10)
	conversationID := "conv-e2e-" + runID
	generationIDs := []string{"gen-e2e-" + runID + "-1", "gen-e2e-" + runID + "-2"}

	startedAt := time.Now().UTC().Add(-2 * time.Second)
	completedAt := time.Now().UTC().Add(-1 * time.Second)
	exportBody := map[string]any{
		"generations": []map[string]any{
			{
				"id":              generationIDs[0],
				"conversation_id": conversationID,
				"mode":            "GENERATION_MODE_SYNC",
				"model": map[string]any{
					"provider": "openai",
					"name":     "gpt-4o",
				},
				"started_at":   startedAt.Format(time.RFC3339Nano),
				"completed_at": completedAt.Format(time.RFC3339Nano),
				"agent_name":   "e2e-agent",
			},
			{
				"id":              generationIDs[1],
				"conversation_id": conversationID,
				"mode":            "GENERATION_MODE_SYNC",
				"model": map[string]any{
					"provider": "openai",
					"name":     "gpt-4o-mini",
				},
				"started_at":   startedAt.Add(10 * time.Millisecond).Format(time.RFC3339Nano),
				"completed_at": completedAt.Add(10 * time.Millisecond).Format(time.RFC3339Nano),
				"agent_name":   "e2e-agent",
			},
		},
	}

	exportResponse := postJSON(t, client, baseURL+"/api/v1/generations:export", exportBody, http.StatusAccepted)
	resultsRaw, ok := exportResponse["results"].([]any)
	if !ok || len(resultsRaw) != len(generationIDs) {
		t.Fatalf("unexpected export results payload: %#v", exportResponse["results"])
	}
	for i, resultRaw := range resultsRaw {
		result, ok := resultRaw.(map[string]any)
		if !ok {
			t.Fatalf("unexpected export result item: %#v", resultRaw)
		}
		accepted, _ := result["accepted"].(bool)
		if !accepted {
			t.Fatalf("expected accepted export for generation %s, got %#v", generationIDs[i], result)
		}
	}

	hotRows, err := countGenerationRows(ctx, db, tenantID, generationIDs)
	if err != nil {
		t.Fatalf("count hot generation rows: %v", err)
	}
	if hotRows != len(generationIDs) {
		t.Fatalf("expected %d hot rows immediately after export, got %d", len(generationIDs), hotRows)
	}

	assertConversationContainsGenerations(t, client, baseURL, conversationID, generationIDs)
	for _, generationID := range generationIDs {
		assertGenerationExists(t, client, baseURL, generationID)
	}

	waitForCondition(t, compactionWait, defaultE2EPollInterval, "generation rows compacted", func() (bool, error) {
		compactedCount, err := countCompactedGenerationRows(ctx, db, tenantID, generationIDs)
		if err != nil {
			return false, err
		}
		return compactedCount == len(generationIDs), nil
	})

	blockCount, err := countCompactionBlocks(ctx, db, tenantID)
	if err != nil {
		t.Fatalf("count compaction blocks: %v", err)
	}
	if blockCount == 0 {
		t.Fatalf("expected at least one compaction block for tenant %q", tenantID)
	}

	if err := deleteGenerationRows(ctx, db, tenantID, []string{generationIDs[0]}); err != nil {
		t.Fatalf("delete first hot row: %v", err)
	}
	remainingRows, err := countGenerationRows(ctx, db, tenantID, generationIDs)
	if err != nil {
		t.Fatalf("count remaining hot rows after first delete: %v", err)
	}
	if remainingRows != 1 {
		t.Fatalf("expected 1 remaining hot row after first delete, got %d", remainingRows)
	}

	// One row is still hot and one row must resolve from cold blocks.
	assertConversationContainsGenerations(t, client, baseURL, conversationID, generationIDs)
	for _, generationID := range generationIDs {
		assertGenerationExists(t, client, baseURL, generationID)
	}

	if err := deleteGenerationRows(ctx, db, tenantID, []string{generationIDs[1]}); err != nil {
		t.Fatalf("delete hot rows: %v", err)
	}
	remainingRows, err = countGenerationRows(ctx, db, tenantID, generationIDs)
	if err != nil {
		t.Fatalf("count remaining hot rows: %v", err)
	}
	if remainingRows != 0 {
		t.Fatalf("expected 0 remaining hot rows after delete, got %d", remainingRows)
	}

	// Hot rows are gone; these reads now require cold block fallback.
	assertConversationContainsGenerations(t, client, baseURL, conversationID, generationIDs)
	for _, generationID := range generationIDs {
		assertGenerationExists(t, client, baseURL, generationID)
	}
}

func postJSON(t *testing.T, client *http.Client, endpoint string, payload any, wantStatus int) map[string]any {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal request payload: %v", err)
	}
	request, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("post %s: %v", endpoint, err)
	}
	defer func() { _ = response.Body.Close() }()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if response.StatusCode != wantStatus {
		t.Fatalf("unexpected status for %s: got=%d want=%d body=%s", endpoint, response.StatusCode, wantStatus, string(responseBody))
	}

	var decoded map[string]any
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		t.Fatalf("decode response body: %v body=%s", err, string(responseBody))
	}
	return decoded
}

func getJSON(t *testing.T, client *http.Client, endpoint string, wantStatus int) map[string]any {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("get %s: %v", endpoint, err)
	}
	defer func() { _ = response.Body.Close() }()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if response.StatusCode != wantStatus {
		t.Fatalf("unexpected status for %s: got=%d want=%d body=%s", endpoint, response.StatusCode, wantStatus, string(responseBody))
	}

	var decoded map[string]any
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		t.Fatalf("decode response body: %v body=%s", err, string(responseBody))
	}
	return decoded
}

func assertGenerationExists(t *testing.T, client *http.Client, baseURL, generationID string) {
	t.Helper()
	payload := getJSON(t, client, baseURL+"/api/v1/generations/"+generationID, http.StatusOK)
	if payload["generation_id"] != generationID {
		t.Fatalf("expected generation_id=%q, got payload=%#v", generationID, payload)
	}
}

func assertConversationContainsGenerations(t *testing.T, client *http.Client, baseURL, conversationID string, generationIDs []string) {
	t.Helper()
	payload := getJSON(t, client, baseURL+"/api/v1/conversations/"+conversationID, http.StatusOK)
	generationsRaw, ok := payload["generations"].([]any)
	if !ok {
		t.Fatalf("missing generations array in payload: %#v", payload)
	}
	if len(generationsRaw) != len(generationIDs) {
		t.Fatalf("unexpected generations length: got=%d want=%d payload=%#v", len(generationsRaw), len(generationIDs), payload)
	}

	expected := make(map[string]struct{}, len(generationIDs))
	for _, generationID := range generationIDs {
		expected[generationID] = struct{}{}
	}
	for _, generationRaw := range generationsRaw {
		generation, ok := generationRaw.(map[string]any)
		if !ok {
			t.Fatalf("unexpected generation payload item: %#v", generationRaw)
		}
		generationID, _ := generation["generation_id"].(string)
		if _, found := expected[generationID]; !found {
			t.Fatalf("unexpected generation id in conversation payload: %q payload=%#v", generationID, generation)
		}
		delete(expected, generationID)
	}
	if len(expected) != 0 {
		t.Fatalf("missing expected generation ids from conversation payload: %#v", expected)
	}
}

func countGenerationRows(ctx context.Context, db *sql.DB, tenantID string, generationIDs []string) (int, error) {
	query, args := buildGenerationIDQuery(
		"SELECT COUNT(*) FROM generations WHERE tenant_id = ? AND generation_id IN (%s)",
		tenantID,
		generationIDs,
	)
	var count int
	err := db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

func countCompactedGenerationRows(ctx context.Context, db *sql.DB, tenantID string, generationIDs []string) (int, error) {
	query, args := buildGenerationIDQuery(
		"SELECT COUNT(*) FROM generations WHERE tenant_id = ? AND compacted = TRUE AND generation_id IN (%s)",
		tenantID,
		generationIDs,
	)
	var count int
	err := db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

func countCompactionBlocks(ctx context.Context, db *sql.DB, tenantID string) (int, error) {
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM compaction_blocks WHERE tenant_id = ? AND deleted = FALSE", tenantID).Scan(&count)
	return count, err
}

func deleteGenerationRows(ctx context.Context, db *sql.DB, tenantID string, generationIDs []string) error {
	query, args := buildGenerationIDQuery(
		"DELETE FROM generations WHERE tenant_id = ? AND generation_id IN (%s)",
		tenantID,
		generationIDs,
	)
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

func buildGenerationIDQuery(format string, tenantID string, generationIDs []string) (string, []any) {
	placeholders := make([]string, 0, len(generationIDs))
	args := make([]any, 0, len(generationIDs)+1)
	args = append(args, tenantID)
	for _, generationID := range generationIDs {
		placeholders = append(placeholders, "?")
		args = append(args, generationID)
	}
	return fmt.Sprintf(format, strings.Join(placeholders, ",")), args
}

func waitForCondition(t *testing.T, timeout, interval time.Duration, description string, check func() (bool, error)) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		ok, err := check()
		if err != nil {
			t.Fatalf("wait for %s: %v", description, err)
		}
		if ok {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out after %s waiting for %s", timeout, description)
		}
		time.Sleep(interval)
	}
}

func getenvDefault(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func getenvDurationDefault(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
