//go:build bgroup

// Security authorization black-box tests (Sprint 5).
//
// Covers: authentication (401), role-based access (403), horizontal privilege
// escalation, SQL injection, invalid input, and XSS payload handling.
//
// Prerequisites:
//   1. Server running: cd backend && go run ./cmd/server
//   2. Seed data loaded: cd backend && go run ./scripts/seed
//
// Run:
//   cd backend && go test -v -tags=bgroup -run '^TestSec' -count=1 ./tests/

package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

const secBaseURL = "http://localhost:8080/api/v1"

// ── helpers ─────────────────────────────────────────────────────────────────

func secLoginToken(t *testing.T, phone, password string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"phone": phone, "password": password})
	resp, err := http.Post(secBaseURL+"/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	defer resp.Body.Close()
	var out struct {
		Code int `json:"code"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	if out.Data.Token == "" {
		t.Fatalf("no token for %s (code=%d)", phone, out.Code)
	}
	return out.Data.Token
}

func secRequest(method, url, token string, payload interface{}) (int, map[string]interface{}) {
	var bodyReader io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		bodyReader = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, url, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	var data map[string]interface{}
	json.Unmarshal(b, &data)
	return resp.StatusCode, data
}

func secGet(url, token string) (int, map[string]interface{}) {
	return secRequest("GET", url, token, nil)
}

func secPost(url, token string, payload interface{}) (int, map[string]interface{}) {
	return secRequest("POST", url, token, payload)
}

func secPut(url, token string, payload interface{}) (int, map[string]interface{}) {
	return secRequest("PUT", url, token, payload)
}

func secRawRequest(method, url, authHeader, contentType string, body []byte) (int, []byte) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, _ := http.NewRequest(method, url, reader)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b
}

// ═════════════════════════════════════════════════════════════════════════════
// 1. Authentication — unauthenticated access to protected endpoints → 401
// ═════════════════════════════════════════════════════════════════════════════

func TestSec_Auth_NoToken_AllProtectedEndpoints(t *testing.T) {
	endpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/orders?page=1&page_size=1"},
		{"GET", "/orders/1"},
		{"POST", "/orders/1/pay"},
		{"GET", "/enrollments?page=1&page_size=1"},
		{"GET", "/enrollments/1/status"},
		{"POST", "/enrollments"},
		{"POST", "/enrollments/1/cancel"},
		{"GET", "/notifications"},
		{"GET", "/notifications/unread-count"},
		{"PUT", "/notifications/1/read"},
		{"GET", "/auth/profile"},
		{"GET", "/activities/merchant"},
		{"POST", "/activities"},
		{"PUT", "/activities/1"},
		{"PUT", "/activities/1/publish"},
		{"PUT", "/activities/1/preheat"},
		{"POST", "/behaviors"},
		{"POST", "/behaviors/batch"},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+"_"+ep.path, func(t *testing.T) {
			st, _ := secRequest(ep.method, secBaseURL+ep.path, "", nil)
			if st != 401 {
				t.Errorf("want 401, got %d for %s %s", st, ep.method, ep.path)
			}
		})
	}
}

// ═════════════════════════════════════════════════════════════════════════════
// 2. Role-based access — CUSTOMER (USER) accessing MERCHANT endpoints → 403
// ═════════════════════════════════════════════════════════════════════════════

func TestSec_Role_CustomerCreateActivity_403(t *testing.T) {
	userToken := secLoginToken(t, "13800000001", "test123456")
	st, _ := secPost(secBaseURL+"/activities", userToken, map[string]interface{}{
		"title": "unauthorized", "description": "x", "location": "x", "category": "CONCERT",
		"max_capacity": 10, "price": 1.0,
		"enroll_open_at":  "2099-01-01T00:00:00Z",
		"enroll_close_at": "2099-06-01T00:00:00Z",
		"activity_at":     "2099-12-01T00:00:00Z",
	})
	if st != 403 {
		t.Errorf("want 403, got %d", st)
	}
}

func TestSec_Role_CustomerListMerchantActivities_403(t *testing.T) {
	userToken := secLoginToken(t, "13800000001", "test123456")
	st, _ := secGet(secBaseURL+"/activities/merchant", userToken)
	if st != 403 {
		t.Errorf("want 403, got %d", st)
	}
}

func TestSec_Role_CustomerUpdateActivity_403(t *testing.T) {
	userToken := secLoginToken(t, "13800000001", "test123456")
	st, _ := secPut(secBaseURL+"/activities/1", userToken, map[string]interface{}{
		"title": "hacked",
	})
	if st != 403 {
		t.Errorf("want 403, got %d", st)
	}
}

func TestSec_Role_CustomerPublishActivity_403(t *testing.T) {
	userToken := secLoginToken(t, "13800000001", "test123456")
	st, _ := secPut(secBaseURL+"/activities/1/publish", userToken, nil)
	if st != 403 {
		t.Errorf("want 403, got %d", st)
	}
}

func TestSec_Role_CustomerPreheatActivity_403(t *testing.T) {
	userToken := secLoginToken(t, "13800000001", "test123456")
	st, _ := secPut(secBaseURL+"/activities/1/preheat", userToken, nil)
	if st != 403 {
		t.Errorf("want 403, got %d", st)
	}
}

// ═════════════════════════════════════════════════════════════════════════════
// 3. Horizontal privilege escalation — user A accessing user B's resources
// ═════════════════════════════════════════════════════════════════════════════

func secFindFirstResourceID(t *testing.T, token, resourcePath, listKey string) float64 {
	t.Helper()
	st, body := secGet(secBaseURL+resourcePath, token)
	if st != 200 {
		t.Fatalf("list %s: HTTP %d", resourcePath, st)
	}
	data, _ := body["data"].(map[string]interface{})
	list, _ := data[listKey].([]interface{})
	if len(list) == 0 {
		t.Skipf("no %s found for this user, skipping escalation test", listKey)
	}
	first := list[0].(map[string]interface{})
	return first["id"].(float64)
}

func TestSec_Escalation_OrderDetail(t *testing.T) {
	tokenA := secLoginToken(t, "13800000001", "test123456")
	tokenB := secLoginToken(t, "13800000002", "test123456")

	orderID := secFindFirstResourceID(t, tokenA, "/orders?page=1&page_size=1", "list")
	st, body := secGet(fmt.Sprintf("%s/orders/%d", secBaseURL, int(orderID)), tokenB)
	if st == 200 && body["data"] != nil {
		t.Errorf("user B should not see user A's order %d, got HTTP %d", int(orderID), st)
	}
}

func TestSec_Escalation_OrderPay(t *testing.T) {
	tokenA := secLoginToken(t, "13800000001", "test123456")
	tokenB := secLoginToken(t, "13800000002", "test123456")

	orderID := secFindFirstResourceID(t, tokenA, "/orders?page=1&page_size=1", "list")
	st, _ := secPost(fmt.Sprintf("%s/orders/%d/pay", secBaseURL, int(orderID)), tokenB, nil)
	if st == 200 {
		t.Errorf("user B should not pay user A's order %d", int(orderID))
	}
}

func TestSec_Escalation_EnrollmentStatus(t *testing.T) {
	tokenA := secLoginToken(t, "13800000001", "test123456")
	tokenB := secLoginToken(t, "13800000002", "test123456")

	enrollID := secFindFirstResourceID(t, tokenA, "/enrollments?page=1&page_size=1", "list")
	st, body := secGet(fmt.Sprintf("%s/enrollments/%d/status", secBaseURL, int(enrollID)), tokenB)
	if st == 200 && body["data"] != nil {
		t.Errorf("user B should not see user A's enrollment %d status", int(enrollID))
	}
}

func TestSec_Escalation_EnrollmentCancel(t *testing.T) {
	tokenA := secLoginToken(t, "13800000001", "test123456")
	tokenB := secLoginToken(t, "13800000002", "test123456")

	enrollID := secFindFirstResourceID(t, tokenA, "/enrollments?page=1&page_size=1", "list")
	st, _ := secPost(fmt.Sprintf("%s/enrollments/%d/cancel", secBaseURL, int(enrollID)), tokenB, nil)
	if st == 200 {
		t.Errorf("user B should not cancel user A's enrollment %d", int(enrollID))
	}
}

func TestSec_Escalation_NotificationRead(t *testing.T) {
	tokenA := secLoginToken(t, "13800000001", "test123456")
	tokenB := secLoginToken(t, "13800000002", "test123456")

	st, body := secGet(secBaseURL+"/notifications?page=1&page_size=1", tokenA)
	if st != 200 {
		t.Skipf("cannot list notifications for user A: HTTP %d", st)
	}
	data, _ := body["data"].(map[string]interface{})
	list, _ := data["list"].([]interface{})
	if len(list) == 0 {
		t.Skip("no notifications for user A, skipping")
	}
	first := list[0].(map[string]interface{})
	notifID := int(first["id"].(float64))

	st2, _ := secPut(fmt.Sprintf("%s/notifications/%d/read", secBaseURL, notifID), tokenB, nil)
	if st2 == 200 {
		t.Errorf("user B should not mark user A's notification %d as read", notifID)
	}
}

func TestSec_Escalation_MerchantUpdateOtherActivity(t *testing.T) {
	tokenMerchantA := secLoginToken(t, "13800000004", "test123456")
	tokenMerchantB := secLoginToken(t, "13800000005", "test123456")

	actID := secFindFirstResourceID(t, tokenMerchantA, "/activities/merchant?page=1&page_size=1", "list")
	st, _ := secPut(fmt.Sprintf("%s/activities/%d", secBaseURL, int(actID)), tokenMerchantB, map[string]interface{}{
		"title": "hijacked by merchant B",
	})
	if st == 200 {
		t.Errorf("merchant B should not update merchant A's activity %d", int(actID))
	}
}

// ═════════════════════════════════════════════════════════════════════════════
// 4. Input validation — SQL injection, invalid IDs, extreme pagination
// ═════════════════════════════════════════════════════════════════════════════

func TestSec_Input_SQLInjection_ActivityID(t *testing.T) {
	payloads := []string{
		"1 OR 1=1",
		"1; DROP TABLE activities;--",
		"1' UNION SELECT * FROM users--",
		"1 AND 1=1",
	}
	for _, p := range payloads {
		t.Run(p, func(t *testing.T) {
			st, body := secGet(secBaseURL+"/activities/"+p, "")
			if st == 200 {
				t.Errorf("SQL injection payload returned 200: %v", body)
			}
			if st >= 500 {
				t.Errorf("SQL injection caused server error %d", st)
			}
		})
	}
}

func TestSec_Input_SQLInjection_OrderID(t *testing.T) {
	token := secLoginToken(t, "13800000001", "test123456")
	payloads := []string{
		"1 OR 1=1",
		"1; DROP TABLE orders;--",
		"' OR '1'='1",
	}
	for _, p := range payloads {
		t.Run(p, func(t *testing.T) {
			st, _ := secGet(secBaseURL+"/orders/"+p, token)
			if st >= 500 {
				t.Errorf("SQL injection caused server error %d", st)
			}
		})
	}
}

func TestSec_Input_InvalidIDs(t *testing.T) {
	token := secLoginToken(t, "13800000001", "test123456")
	invalidIDs := []string{"-1", "0", "abc", "1.5", "99999999999999999999"}

	endpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/activities/%s"},
		{"GET", "/orders/%s"},
		{"GET", "/enrollments/%s/status"},
		{"POST", "/enrollments/%s/cancel"},
		{"POST", "/orders/%s/pay"},
		{"PUT", "/notifications/%s/read"},
	}

	for _, ep := range endpoints {
		for _, id := range invalidIDs {
			name := fmt.Sprintf("%s_%s_id=%s", ep.method, ep.path, id)
			t.Run(name, func(t *testing.T) {
				url := secBaseURL + fmt.Sprintf(ep.path, id)
				st, _ := secRequest(ep.method, url, token, nil)
				if st >= 500 {
					t.Errorf("invalid ID '%s' caused server error %d", id, st)
				}
			})
		}
	}
}

func TestSec_Input_PaginationEdgeCases(t *testing.T) {
	token := secLoginToken(t, "13800000001", "test123456")

	cases := []struct {
		name  string
		query string
	}{
		{"negative_page", "page=-1&page_size=10"},
		{"negative_page_size", "page=1&page_size=-1"},
		{"zero_page", "page=0&page_size=10"},
		{"zero_page_size", "page=1&page_size=0"},
		{"huge_page_size", "page=1&page_size=999999"},
		{"huge_page", "page=999999&page_size=10"},
		{"string_page", "page=abc&page_size=10"},
		{"missing_params", ""},
	}

	for _, c := range cases {
		t.Run("orders_"+c.name, func(t *testing.T) {
			url := secBaseURL + "/orders"
			if c.query != "" {
				url += "?" + c.query
			}
			st, _ := secGet(url, token)
			if st >= 500 {
				t.Errorf("pagination edge case '%s' caused server error %d", c.name, st)
			}
		})
	}
}

func TestSec_Input_XSSPayload_InActivityCreate(t *testing.T) {
	merchantToken := secLoginToken(t, "13800000004", "test123456")
	xssPayloads := []string{
		`<script>alert(1)</script>`,
		`<img src=x onerror=alert(1)>`,
		`"><svg/onload=alert(1)>`,
		`javascript:alert(1)`,
	}

	for i, xss := range xssPayloads {
		t.Run(fmt.Sprintf("xss_%d", i), func(t *testing.T) {
			st, body := secPost(secBaseURL+"/activities", merchantToken, map[string]interface{}{
				"title":           xss,
				"description":     xss,
				"location":        "test",
				"category":        "CONCERT",
				"max_capacity":    10,
				"price":           1.0,
				"enroll_open_at":  "2099-01-01T00:00:00Z",
				"enroll_close_at": "2099-06-01T00:00:00Z",
				"activity_at":     "2099-12-01T00:00:00Z",
			})
			if st >= 500 {
				t.Errorf("XSS payload caused server error %d", st)
			}
			if st == 200 || st == 201 {
				data, _ := body["data"].(map[string]interface{})
				if data != nil {
					activityID := data["activity_id"]
					if activityID != nil {
						detailSt, detailBody := secGet(fmt.Sprintf("%s/activities/%v", secBaseURL, activityID), "")
						if detailSt == 200 {
							detailData, _ := detailBody["data"].(map[string]interface{})
							if detailData != nil {
								title, _ := detailData["title"].(string)
								if title != xss {
									t.Logf("XSS payload was sanitized/modified: input=%q stored=%q", xss, title)
								} else {
									t.Logf("XSS payload stored as-is (frontend must escape on render): %q", title)
								}
							}
						}
					}
				}
			}
		})
	}
}

// ═════════════════════════════════════════════════════════════════════════════
// 5. Sensitive info leakage — error responses should not leak internals
// ═════════════════════════════════════════════════════════════════════════════

func TestSec_Leakage_ErrorResponseNoStackTrace(t *testing.T) {
	token := secLoginToken(t, "13800000001", "test123456")
	sensitivePatterns := []string{
		"goroutine", "runtime.", "panic", ".go:", "stack trace",
		"sql:", "SELECT", "INSERT", "UPDATE", "DELETE",
		"password", "secret", "DSN", "mysql://",
	}

	endpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/orders/99999999"},
		{"GET", "/enrollments/99999999/status"},
		{"POST", "/orders/99999999/pay"},
		{"POST", "/enrollments/99999999/cancel"},
		{"PUT", "/notifications/99999999/read"},
		{"GET", "/activities/99999999"},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+"_"+ep.path, func(t *testing.T) {
			_, raw := secRawRequest(ep.method, secBaseURL+ep.path, "Bearer "+token, "application/json", nil)
			body := string(raw)
			for _, pat := range sensitivePatterns {
				if strings.Contains(strings.ToLower(body), strings.ToLower(pat)) {
					t.Errorf("response leaks sensitive info matching %q: %s", pat, body[:min(len(body), 200)])
				}
			}
		})
	}
}

func TestSec_Leakage_MetricsEndpointExposed(t *testing.T) {
	st, _ := secRawRequest("GET", "http://localhost:8080/metrics", "", "", nil)
	if st == 200 {
		t.Log("WARNING: /metrics endpoint is publicly accessible without authentication")
	}
}

// ═════════════════════════════════════════════════════════════════════════════
// 6. CORS — verify configuration behavior
// ═════════════════════════════════════════════════════════════════════════════

func TestSec_CORS_PreflightAllowsExpectedHeaders(t *testing.T) {
	req, _ := http.NewRequest("OPTIONS", secBaseURL+"/activities", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Authorization,Content-Type")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("preflight: %v", err)
	}
	defer resp.Body.Close()

	allow := resp.Header.Get("Access-Control-Allow-Headers")
	if allow == "" {
		t.Log("WARNING: CORS preflight did not return Access-Control-Allow-Headers")
	} else {
		if !strings.Contains(strings.ToLower(allow), "authorization") {
			t.Errorf("CORS should allow Authorization header, got: %s", allow)
		}
	}
}

func TestSec_CORS_UnknownOrigin(t *testing.T) {
	req, _ := http.NewRequest("OPTIONS", secBaseURL+"/activities", nil)
	req.Header.Set("Origin", "https://evil.attacker.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("preflight: %v", err)
	}
	defer resp.Body.Close()

	origin := resp.Header.Get("Access-Control-Allow-Origin")
	if origin == "*" || origin == "https://evil.attacker.com" {
		t.Log("WARNING: CORS allows requests from arbitrary origins (check APP_ENV and CORS_ALLOWED_ORIGINS in production)")
	}
}

// ═════════════════════════════════════════════════════════════════════════════
// 7. Rate limiting — registration endpoint
// ═════════════════════════════════════════════════════════════════════════════

func TestSec_RateLimit_LoginEndpointNoRateLimit(t *testing.T) {
	t.Log("NOTE: POST /auth/login has no rate limiting — vulnerable to brute-force")
	t.Log("This is a known risk documented in the security report")
}

