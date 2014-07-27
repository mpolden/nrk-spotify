package spotify

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
)

func TestURL(t *testing.T) {
	auth := Auth{Listen: ":8080"}
	if auth.ListenURL() != "http://localhost:8080" {
		t.Fatalf("Expected http://localhost:8080, got %s",
			auth.ListenURL())
	}
	auth = Auth{Listen: "1.2.3.4:8080"}
	if auth.ListenURL() != "http://1.2.3.4:8080" {
		t.Fatalf("Expected http://1.2.3.4:8080, got %s",
			auth.ListenURL())
	}
}

func TestAuthHeader(t *testing.T) {
	auth := Auth{
		ClientId:     "foo",
		ClientSecret: "bar",
	}
	expected := "Basic Zm9vOmJhcg==" // base64("foo:bar")
	if auth.authHeader() != expected {
		t.Fatalf("Expected '%s', got '%s'", expected, auth.authHeader())
	}
}

func newTestServer(path string, body string) *httptest.Server {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}
	mux := http.NewServeMux()
	mux.HandleFunc(path, handler)
	return httptest.NewServer(mux)
}

func TestLogin(t *testing.T) {
	auth := Auth{
		ClientId:     "foo",
		ClientSecret: "bar",
	}
	server := httptest.NewServer(http.HandlerFunc(auth.Login))
	defer server.Close()

	auth.listenURL = server.URL

	req, err := http.NewRequest("GET", auth.listenURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	transport := http.Transport{}
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 302 {
		t.Fatalf("Expected 302, got %d", resp.StatusCode)
	}
	if len(resp.Cookies()) == 0 {
		t.Fatal("Expected cookies to be set")
	}
	cookie := resp.Cookies()[0]
	if cookie.Name != "spotify_auth_state" || len(cookie.Value) == 0 {
		t.Fatal("Cookie 'spotify_auth_state' not set")
	}
	location := resp.Header.Get("Location")
	u, err := url.Parse(location)
	if err != nil {
		t.Fatal(err)
	}
	values, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		t.Fatal(err)
	}
	expected := "foo"
	if values["client_id"][0] != "foo" {
		t.Fatalf("Expected %s, got %s", expected,
			values["client_id"][0])
	}
	expected = auth.listenURL + "/callback"
	if values["redirect_uri"][0] != expected {
		t.Fatalf("Expected %s, got %s", expected,
			values["redirect_uri"][0])
	}
	expected = "code"
	if values["response_type"][0] != expected {
		t.Fatalf("Expected %s, got %s", expected,
			values["response_type"][0])
	}
	expected = "playlist-modify-public playlist-modify-private " +
		"playlist-read-private"
	if values["scope"][0] != expected {
		t.Fatalf("Expected %s, got %s", expected,
			values["scope"][0])
	}
	if values["state"] == nil {
		t.Fatalf("Expected 'state' to be set")
	}
}

func TestGetToken(t *testing.T) {
	server := newTestServer("/api/token", tokenResponse)
	defer server.Close()
	auth := Auth{
		ClientId:     "foo",
		ClientSecret: "bar",
		url:          server.URL,
	}
	spotify, err := auth.getToken([]string{"foobar"})
	if err != nil {
		t.Fatal(err)
	}
	expected := "NgCXRK...MzYjw"
	if spotify.AccessToken != expected {
		t.Fatalf("Expected '%s', got '%s'",
			expected, spotify.AccessToken)
	}
	expected = "Bearer"
	if spotify.TokenType != expected {
		t.Fatalf("Expected '%s', got '%s'",
			expected, spotify.TokenType)
	}
	if spotify.ExpiresIn != 3600 {
		t.Fatalf("Expected 3600, got %d", spotify.ExpiresIn)
	}
	expected = "NgAagA...Um_SHo"
	if spotify.RefreshToken != expected {
		t.Fatalf("Expected '%s', got '%s'",
			expected, spotify.RefreshToken)
	}
}

func TestCallback(t *testing.T) {
	tempFile, err := ioutil.TempFile("", "spotify_auth")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Remove(tempFile.Name()); err != nil {
			t.Fatal(err)
		}
	}()
	auth := Auth{
		ClientId:     "foo",
		ClientSecret: "bar",
		TokenFile:    tempFile.Name(),
	}
	tokenHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, tokenResponse)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/token", tokenHandler)
	mux.HandleFunc("/callback", auth.Callback)
	server := httptest.NewServer(mux)
	defer server.Close()

	auth.listenURL = server.URL
	auth.url = server.URL

	req, err := http.NewRequest("GET",
		auth.CallbackURL()+"?code=foobar&state=secret", nil)
	req.AddCookie(&http.Cookie{
		Name:  "spotify_auth_state",
		Value: "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, body)
	}
}

const tokenResponse string = `
{
   "access_token": "NgCXRK...MzYjw",
   "token_type": "Bearer",
   "expires_in": 3600,
   "refresh_token": "NgAagA...Um_SHo"
}`
