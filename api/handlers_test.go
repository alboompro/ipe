package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gorilla/mux"

	"ipe/app"
	channel2 "ipe/channel"
	"ipe/connection"
	"ipe/mocks"
	"ipe/storage"
	"ipe/utils"
)

var (
	testApp  *app.Application
	database storage.Storage
	id       = 0
)

func newTestApp() *app.Application {
	a := app.NewApplication("Test", strconv.Itoa(id), "123", "123", false, false, true, false, "", nil)
	id++

	return a
}

func init() {
	testApp = newTestApp()

	channel := channel2.New("presence-c1")
	testApp.AddChannel(channel)
	testApp.AddChannel(channel2.New("c2"))
	testApp.AddChannel(channel2.New("private-c3"))

	conn := connection.New("123.456", mocks.NewMockSocket())
	_ = testApp.Subscribe(channel, conn, "{}")

	conn = connection.New("321.654", mocks.NewMockSocket())
	_ = testApp.Subscribe(channel, conn, "{}")

	_storage := storage.NewInMemory()
	_ = _storage.AddApp(testApp)

	database = _storage
}

// All channels
func Test_getChannels_all(t *testing.T) {
	appID := testApp.AppID

	r, _ := http.NewRequest("GET", fmt.Sprintf("/apps/%s/channels", appID), nil)
	r = mux.SetURLVars(r, map[string]string{
		"app_id": appID,
	})
	w := httptest.NewRecorder()

	handler := &GetChannels{database}
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("w.Code == %d, wants %d", w.Code, http.StatusOK)
	}

	data := make(map[string]interface{})
	_ = json.Unmarshal(w.Body.Bytes(), &data)

	channels := data["channels"].(map[string]interface{})

	if len(channels) != 3 {
		t.Errorf("len(%q) == %d, want %d", channels, len(channels), 3)
	}
}

func Test_getChannels_filter_by_presence_prefix(t *testing.T) {
	appID := testApp.AppID

	r, _ := http.NewRequest("GET", fmt.Sprintf("/apps/%s/channels?filter_by_prefix=presence-", appID), nil)
	r = mux.SetURLVars(r, map[string]string{
		"app_id": appID,
	})
	w := httptest.NewRecorder()

	handler := &GetChannels{database}
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("w.Code == %d, wants %d", w.Code, http.StatusOK)
	}

	data := make(map[string]interface{})
	_ = json.Unmarshal(w.Body.Bytes(), &data)

	channels := data["channels"].(map[string]interface{})

	if len(channels) != 1 {
		t.Errorf("len(%q) == %d, want %d", channels, len(channels), 1)
	}
}

// Only presence channels and user_count
func Test_getChannels_filter_by_presence_prefix_and_user_count(t *testing.T) {
	appID := testApp.AppID

	r, _ := http.NewRequest("GET", fmt.Sprintf("/apps/%s/channels?filter_by_prefix=presence-&info=user_count", appID), nil)
	r = mux.SetURLVars(r, map[string]string{
		"app_id": appID,
	})
	w := httptest.NewRecorder()

	handler := &GetChannels{database}
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("w.Code == %d, wants %d", w.Code, http.StatusOK)
	}

	data := make(map[string]interface{})
	_ = json.Unmarshal(w.Body.Bytes(), &data)

	channels := data["channels"].(map[string]interface{})

	if len(channels) != 1 {
		t.Errorf("len(%q) == %d, want %d", channels, len(channels), 1)
	}

	c, exists := channels["presence-c1"]

	if !exists {
		t.Errorf("!exists == %t, want %t", !exists, false)
	}

	_channel := c.(map[string]interface{})

	if _channel["user_count"] != float64(1) {
		t.Errorf("_channel['user_count'] == %f, want %d", _channel["user_count"], 1)
	}
}

// User count only allowed in Presence channels
func Test_getChannels_filter_by_private_prefix_and_info_user_count(t *testing.T) {
	appID := testApp.AppID

	r, _ := http.NewRequest("GET", fmt.Sprintf("/apps/%s/channels?filter_by_prefix=private-&info=user_count", appID), nil)
	r = mux.SetURLVars(r, map[string]string{
		"app_id": appID,
	})
	w := httptest.NewRecorder()

	handler := &GetChannels{database}
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("w.Code == %d, wants %d", w.Code, http.StatusBadRequest)
	}
}

func Test_getChannels_filter_by_public_prefix(t *testing.T) {
	appID := testApp.AppID

	r, _ := http.NewRequest("GET", fmt.Sprintf("/apps/%s/channels?filter_by_prefix=public-", appID), nil)
	r = mux.SetURLVars(r, map[string]string{
		"app_id": appID,
	})
	w := httptest.NewRecorder()

	handler := &GetChannels{database}
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("w.Code == %d, wants %d", w.Code, http.StatusOK)
	}

	data := make(map[string]interface{})

	_ = json.Unmarshal(w.Body.Bytes(), &data)

	channels := data["channels"].(map[string]interface{})

	if len(channels) != 1 {
		t.Errorf("len(%q) == %d, want %d", channels, len(channels), 1)
	}

	_, exists := channels["c2"]

	if !exists {
		t.Errorf("!exists == %t, want %t", !exists, false)
	}
}

func Test_getChannels_filter_by_private_prefix(t *testing.T) {
	appID := testApp.AppID

	r, _ := http.NewRequest("GET", fmt.Sprintf("/apps/%s/channels?filter_by_prefix=private-", appID), nil)
	r = mux.SetURLVars(r, map[string]string{
		"app_id": appID,
	})
	w := httptest.NewRecorder()

	handler := &GetChannels{database}
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("w.Code == %d, wants %d", w.Code, http.StatusOK)
	}

	data := make(map[string]interface{})

	_ = json.Unmarshal(w.Body.Bytes(), &data)

	channels := data["channels"].(map[string]interface{})

	if len(channels) != 1 {
		t.Errorf("len(%q) == %d, want %d", channels, len(channels), 1)
	}

	_, exists := channels["private-c3"]

	if !exists {
		t.Errorf("!exists == %t, want %t", !exists, false)
	}
}

// PostEvents handler tests

func Test_PostEvents_SingleChannel(t *testing.T) {
	appID := testApp.AppID
	payload := `{"name":"test-event","channel":"test-channel","data":"{\"message\":\"hello\"}"}`

	r, _ := http.NewRequest("POST", fmt.Sprintf("/apps/%s/events", appID), strings.NewReader(payload))
	r.Header.Set("Content-Type", "application/json")
	r = mux.SetURLVars(r, map[string]string{
		"app_id": appID,
	})
	r.URL.RawQuery = "auth_signature=valid_signature"
	w := httptest.NewRecorder()

	handler := &PostEvents{database}
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("w.Code == %d, wants %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response) != 0 {
		t.Errorf("Expected empty response, got %v", response)
	}
}

func Test_PostEvents_MultipleChannels(t *testing.T) {
	appID := testApp.AppID
	payload := `{"name":"test-event","channels":["channel1","channel2"],"data":"{\"message\":\"hello\"}"}`

	r, _ := http.NewRequest("POST", fmt.Sprintf("/apps/%s/events", appID), strings.NewReader(payload))
	r.Header.Set("Content-Type", "application/json")
	r = mux.SetURLVars(r, map[string]string{
		"app_id": appID,
	})
	w := httptest.NewRecorder()

	handler := &PostEvents{database}
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("w.Code == %d, wants %d", w.Code, http.StatusOK)
	}
}

func Test_PostEvents_WithSocketID(t *testing.T) {
	appID := testApp.AppID
	payload := `{"name":"test-event","channel":"test-channel","data":"{\"message\":\"hello\"}","socket_id":"123.456"}`

	r, _ := http.NewRequest("POST", fmt.Sprintf("/apps/%s/events", appID), strings.NewReader(payload))
	r.Header.Set("Content-Type", "application/json")
	r = mux.SetURLVars(r, map[string]string{
		"app_id": appID,
	})
	w := httptest.NewRecorder()

	handler := &PostEvents{database}
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("w.Code == %d, wants %d", w.Code, http.StatusOK)
	}
}

func Test_PostEvents_EventSizeLimit(t *testing.T) {
	appID := testApp.AppID
	// Create a payload larger than 10KB
	largeData := strings.Repeat("x", 10*1000+1)
	payload := fmt.Sprintf(`{"name":"test-event","channel":"test-channel","data":"%s"}`, largeData)

	r, _ := http.NewRequest("POST", fmt.Sprintf("/apps/%s/events", appID), strings.NewReader(payload))
	r.Header.Set("Content-Type", "application/json")
	r = mux.SetURLVars(r, map[string]string{
		"app_id": appID,
	})
	w := httptest.NewRecorder()

	handler := &PostEvents{database}
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("w.Code == %d, wants %d", w.Code, http.StatusRequestEntityTooLarge)
	}
}

func Test_PostEvents_InvalidJSON(t *testing.T) {
	appID := testApp.AppID
	payload := `invalid json`

	r, _ := http.NewRequest("POST", fmt.Sprintf("/apps/%s/events", appID), strings.NewReader(payload))
	r.Header.Set("Content-Type", "application/json")
	r = mux.SetURLVars(r, map[string]string{
		"app_id": appID,
	})
	w := httptest.NewRecorder()

	handler := &PostEvents{database}
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("w.Code == %d, wants %d", w.Code, http.StatusBadRequest)
	}
}

func Test_PostEvents_NonExistentApp(t *testing.T) {
	payload := `{"name":"test-event","channel":"test-channel","data":"{\"message\":\"hello\"}"}`

	r, _ := http.NewRequest("POST", "/apps/non-existent/events", strings.NewReader(payload))
	r.Header.Set("Content-Type", "application/json")
	r = mux.SetURLVars(r, map[string]string{
		"app_id": "non-existent",
	})
	w := httptest.NewRecorder()

	handler := &PostEvents{database}
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("w.Code == %d, wants %d", w.Code, http.StatusBadRequest)
	}
}

// GetChannel handler tests

func Test_GetChannel_WithoutAttributes(t *testing.T) {
	appID := testApp.AppID
	channelName := "presence-c1"

	r, _ := http.NewRequest("GET", fmt.Sprintf("/apps/%s/channels/%s", appID, channelName), nil)
	r = mux.SetURLVars(r, map[string]string{
		"app_id":       appID,
		"channel_name": channelName,
	})
	w := httptest.NewRecorder()

	handler := &GetChannel{database}
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("w.Code == %d, wants %d", w.Code, http.StatusOK)
	}

	var channel map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &channel); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if occupied, ok := channel["occupied"].(bool); !ok || !occupied {
		t.Errorf("Expected occupied=true, got %v", channel["occupied"])
	}
}

func Test_GetChannel_WithUserCount(t *testing.T) {
	appID := testApp.AppID
	channelName := "presence-c1"

	r, _ := http.NewRequest("GET", fmt.Sprintf("/apps/%s/channels/%s?info=user_count", appID, channelName), nil)
	r = mux.SetURLVars(r, map[string]string{
		"app_id":       appID,
		"channel_name": channelName,
	})
	w := httptest.NewRecorder()

	handler := &GetChannel{database}
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("w.Code == %d, wants %d", w.Code, http.StatusOK)
	}

	var channel map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &channel); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if userCount, ok := channel["user_count"].(float64); !ok || userCount != 1 {
		t.Errorf("Expected user_count=1, got %v", channel["user_count"])
	}
}

func Test_GetChannel_WithSubscriptionCount(t *testing.T) {
	appID := testApp.AppID
	channelName := "presence-c1"

	r, _ := http.NewRequest("GET", fmt.Sprintf("/apps/%s/channels/%s?info=subscription_count", appID, channelName), nil)
	r = mux.SetURLVars(r, map[string]string{
		"app_id":       appID,
		"channel_name": channelName,
	})
	w := httptest.NewRecorder()

	handler := &GetChannel{database}
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("w.Code == %d, wants %d", w.Code, http.StatusOK)
	}

	var channel map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &channel); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if subCount, ok := channel["subscription_count"].(float64); !ok || subCount != 2 {
		t.Errorf("Expected subscription_count=2, got %v", channel["subscription_count"])
	}
}

func Test_GetChannel_WithBothAttributes(t *testing.T) {
	appID := testApp.AppID
	channelName := "presence-c1"

	r, _ := http.NewRequest("GET", fmt.Sprintf("/apps/%s/channels/%s?info=user_count,subscription_count", appID, channelName), nil)
	r = mux.SetURLVars(r, map[string]string{
		"app_id":       appID,
		"channel_name": channelName,
	})
	w := httptest.NewRecorder()

	handler := &GetChannel{database}
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("w.Code == %d, wants %d", w.Code, http.StatusOK)
	}

	var channel map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &channel); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if _, ok := channel["user_count"]; !ok {
		t.Error("Expected user_count in response")
	}

	if _, ok := channel["subscription_count"]; !ok {
		t.Error("Expected subscription_count in response")
	}
}

func Test_GetChannel_UserCountOnNonPresence(t *testing.T) {
	appID := testApp.AppID
	channelName := "c2" // Public channel

	r, _ := http.NewRequest("GET", fmt.Sprintf("/apps/%s/channels/%s?info=user_count", appID, channelName), nil)
	r = mux.SetURLVars(r, map[string]string{
		"app_id":       appID,
		"channel_name": channelName,
	})
	w := httptest.NewRecorder()

	handler := &GetChannel{database}
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("w.Code == %d, wants %d", w.Code, http.StatusBadRequest)
	}
}

func Test_GetChannel_NonExistentChannel(t *testing.T) {
	appID := testApp.AppID
	channelName := "non-existent-channel"

	r, _ := http.NewRequest("GET", fmt.Sprintf("/apps/%s/channels/%s", appID, channelName), nil)
	r = mux.SetURLVars(r, map[string]string{
		"app_id":       appID,
		"channel_name": channelName,
	})
	w := httptest.NewRecorder()

	handler := &GetChannel{database}
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("w.Code == %d, wants %d", w.Code, http.StatusBadRequest)
	}
}

func Test_GetChannel_EmptyChannelName(t *testing.T) {
	appID := testApp.AppID
	channelName := ""

	r, _ := http.NewRequest("GET", fmt.Sprintf("/apps/%s/channels/%s", appID, channelName), nil)
	r = mux.SetURLVars(r, map[string]string{
		"app_id":       appID,
		"channel_name": channelName,
	})
	w := httptest.NewRecorder()

	handler := &GetChannel{database}
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("w.Code == %d, wants %d", w.Code, http.StatusBadRequest)
	}
}

// GetChannelUsers handler tests

func Test_GetChannelUsers_PresenceChannel(t *testing.T) {
	appID := testApp.AppID
	channelName := "presence-c1"

	r, _ := http.NewRequest("GET", fmt.Sprintf("/apps/%s/channels/%s/users", appID, channelName), nil)
	r = mux.SetURLVars(r, map[string]string{
		"app_id":       appID,
		"channel_name": channelName,
	})
	w := httptest.NewRecorder()

	handler := &GetChannelUsers{database}
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("w.Code == %d, wants %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	users, ok := response["users"].([]interface{})
	if !ok {
		t.Fatal("Expected users array in response")
	}

	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}
}

func Test_GetChannelUsers_NonPresenceChannel(t *testing.T) {
	appID := testApp.AppID
	channelName := "c2" // Public channel

	r, _ := http.NewRequest("GET", fmt.Sprintf("/apps/%s/channels/%s/users", appID, channelName), nil)
	r = mux.SetURLVars(r, map[string]string{
		"app_id":       appID,
		"channel_name": channelName,
	})
	w := httptest.NewRecorder()

	handler := &GetChannelUsers{database}
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("w.Code == %d, wants %d", w.Code, http.StatusBadRequest)
	}
}

func Test_GetChannelUsers_NonExistentChannel(t *testing.T) {
	appID := testApp.AppID
	channelName := "non-existent-channel"

	r, _ := http.NewRequest("GET", fmt.Sprintf("/apps/%s/channels/%s/users", appID, channelName), nil)
	r = mux.SetURLVars(r, map[string]string{
		"app_id":       appID,
		"channel_name": channelName,
	})
	w := httptest.NewRecorder()

	handler := &GetChannelUsers{database}
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("w.Code == %d, wants %d", w.Code, http.StatusBadRequest)
	}
}

func Test_GetChannelUsers_EmptyChannel(t *testing.T) {
	appID := testApp.AppID
	// Create a new presence channel with no users
	newChannel := channel2.New("presence-empty")
	testApp.AddChannel(newChannel)

	channelName := "presence-empty"

	r, _ := http.NewRequest("GET", fmt.Sprintf("/apps/%s/channels/%s/users", appID, channelName), nil)
	r = mux.SetURLVars(r, map[string]string{
		"app_id":       appID,
		"channel_name": channelName,
	})
	w := httptest.NewRecorder()

	handler := &GetChannelUsers{database}
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("w.Code == %d, wants %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v, body: %s", err, w.Body.String())
	}

	usersVal, exists := response["users"]
	if !exists {
		t.Fatalf("Expected 'users' key in response, got: %+v, body: %s", response, w.Body.String())
	}

	users, ok := usersVal.([]interface{})
	if !ok {
		// Try to handle null case
		if usersVal == nil {
			users = []interface{}{}
		} else {
			t.Fatalf("Expected users to be []interface{}, got type %T: %+v, body: %s", usersVal, usersVal, w.Body.String())
		}
	}

	if len(users) != 0 {
		t.Errorf("Expected 0 users, got %d", len(users))
	}
}

// Authentication middleware tests

func Test_Authentication_ValidSignature(t *testing.T) {
	appID := testApp.AppID
	method := "POST"
	path := fmt.Sprintf("/apps/%s/events", appID)

	// Create valid signature
	queryString := "param1=value1&param2=value2"
	toSign := method + "\n" + path + "\n" + queryString
	signature := utils.HashMAC([]byte(toSign), []byte(testApp.Secret))

	url := fmt.Sprintf("%s?%s&auth_signature=%s", path, queryString, signature)
	r, _ := http.NewRequest(method, url, nil)
	r = mux.SetURLVars(r, map[string]string{
		"app_id": appID,
	})
	w := httptest.NewRecorder()

	// Create a simple handler that returns 200 if authenticated
	handler := Authentication(database)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("w.Code == %d, wants %d", w.Code, http.StatusOK)
	}
}

func Test_Authentication_InvalidSignature(t *testing.T) {
	appID := testApp.AppID
	method := "POST"
	path := fmt.Sprintf("/apps/%s/events", appID)

	url := fmt.Sprintf("%s?auth_signature=invalid_signature", path)
	r, _ := http.NewRequest(method, url, nil)
	r = mux.SetURLVars(r, map[string]string{
		"app_id": appID,
	})
	w := httptest.NewRecorder()

	handler := Authentication(database)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("w.Code == %d, wants %d", w.Code, http.StatusUnauthorized)
	}
}

func Test_Authentication_MissingSignature(t *testing.T) {
	appID := testApp.AppID
	method := "POST"
	path := fmt.Sprintf("/apps/%s/events", appID)

	r, _ := http.NewRequest(method, path, nil)
	r = mux.SetURLVars(r, map[string]string{
		"app_id": appID,
	})
	w := httptest.NewRecorder()

	handler := Authentication(database)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("w.Code == %d, wants %d", w.Code, http.StatusUnauthorized)
	}
}

func Test_Authentication_NonExistentApp(t *testing.T) {
	method := "POST"
	path := "/apps/non-existent/events"

	r, _ := http.NewRequest(method, path, nil)
	r = mux.SetURLVars(r, map[string]string{
		"app_id": "non-existent",
	})
	w := httptest.NewRecorder()

	handler := Authentication(database)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("w.Code == %d, wants %d", w.Code, http.StatusUnauthorized)
	}
}

func Test_Authentication_QueryParameterOrdering(t *testing.T) {
	appID := testApp.AppID
	method := "POST"
	path := fmt.Sprintf("/apps/%s/events", appID)

	// Test that query parameters are sorted correctly
	queryString := "a=1&b=2&z=3"
	toSign := method + "\n" + path + "\n" + queryString
	signature := utils.HashMAC([]byte(toSign), []byte(testApp.Secret))

	url := fmt.Sprintf("%s?%s&auth_signature=%s", path, queryString, signature)
	r, _ := http.NewRequest(method, url, nil)
	r = mux.SetURLVars(r, map[string]string{
		"app_id": appID,
	})
	w := httptest.NewRecorder()

	handler := Authentication(database)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("w.Code == %d, wants %d", w.Code, http.StatusOK)
	}
}

// CheckAppDisabled middleware tests

func Test_CheckAppDisabled_EnabledApp(t *testing.T) {
	// Create a fresh enabled app
	enabledApp := newTestApp()
	enabledApp.Enabled = true
	_storage := storage.NewInMemory()
	_ = _storage.AddApp(enabledApp)

	appID := enabledApp.AppID
	path := fmt.Sprintf("/apps/%s/events", appID)

	r, _ := http.NewRequest("GET", path, nil)
	r = mux.SetURLVars(r, map[string]string{
		"app_id": appID,
	})
	w := httptest.NewRecorder()

	handler := CheckAppDisabled(_storage)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("w.Code == %d, wants %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func Test_CheckAppDisabled_DisabledApp(t *testing.T) {
	disabledApp := newTestApp()
	disabledApp.Enabled = false
	_storage := storage.NewInMemory()
	_ = _storage.AddApp(disabledApp)

	appID := disabledApp.AppID
	path := fmt.Sprintf("/apps/%s/events", appID)

	r, _ := http.NewRequest("GET", path, nil)
	r = mux.SetURLVars(r, map[string]string{
		"app_id": appID,
	})
	w := httptest.NewRecorder()

	handler := CheckAppDisabled(_storage)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("w.Code == %d, wants %d", w.Code, http.StatusForbidden)
	}
}

func Test_CheckAppDisabled_NonExistentApp(t *testing.T) {
	path := "/apps/non-existent/events"

	r, _ := http.NewRequest("GET", path, nil)
	r = mux.SetURLVars(r, map[string]string{
		"app_id": "non-existent",
	})
	w := httptest.NewRecorder()

	handler := CheckAppDisabled(database)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("w.Code == %d, wants %d", w.Code, http.StatusForbidden)
	}
}
