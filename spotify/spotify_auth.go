package spotify

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/mitchellh/colorstring"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

const defaultAuthURL string = "https://accounts.spotify.com"
const stateKey string = "spotify_auth_state"
const scope string = "playlist-modify-public playlist-modify-private " +
	"playlist-read-private"

type Auth struct {
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	TokenFile    string `json:"token_file"`
	Listen       string
	listenURL    string
	url          string
}

func (auth *Auth) URL() string {
	if auth.url == "" {
		return defaultAuthURL
	}
	return auth.url
}

func (auth *Auth) ListenURL() string {
	if auth.listenURL != "" {
		return auth.listenURL
	}
	if strings.HasPrefix(auth.Listen, ":") {
		return "http://localhost" + auth.Listen
	}
	return "http://" + auth.Listen
}

func (auth *Auth) CallbackURL() string {
	return auth.ListenURL() + "/callback"
}

func (auth *Auth) authHeader() string {
	data := auth.ClientId + ":" + auth.ClientSecret
	return "Basic " + base64.StdEncoding.EncodeToString(
		[]byte(data))
}

func base64Rand(size int) (string, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func (auth *Auth) Login(w http.ResponseWriter, r *http.Request) {
	state, err := base64Rand(32)
	if err != nil {
		http.Error(w, "Failed to generate state value", 400)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:  stateKey,
		Value: state,
	})
	params := url.Values{
		"response_type": {"code"},
		"client_id":     {auth.ClientId},
		"scope":         {scope},
		"redirect_uri":  {auth.CallbackURL()},
		"state":         {state},
	}
	url := auth.URL() + "/authorize?" + params.Encode()
	http.Redirect(w, r, url, http.StatusFound)
}

func (auth *Auth) getToken(code []string) (*Spotify, error) {
	formData := url.Values{
		"code":          code,
		"redirect_uri":  {auth.CallbackURL()},
		"grant_type":    {"authorization_code"},
		"client_id":     {auth.ClientId},
		"client_secret": {auth.ClientSecret},
	}
	url := auth.URL() + "/api/token"
	resp, err := http.PostForm(url, formData)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var token Spotify
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	token.Auth = *auth
	return &token, nil
}

func (auth *Auth) Callback(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	state := params["state"]
	cookie, _ := r.Cookie(stateKey)

	if state == nil || cookie == nil || cookie.Value != state[0] {
		http.Error(w, "Could not validate request", 400)
		return
	}

	code, exists := params["code"]
	if !exists {
		http.Error(w, "Missing required query parameter: code", 400)
		return
	}
	token, err := auth.getToken(code)
	if err != nil {
		http.Error(w, "Failed to retrieve token from Spotify", 400)
		return
	}
	err = token.Save(auth.TokenFile)
	if err != nil {
		fmt.Printf("Failed to save token: %s\n", err)
		http.Error(w, "Failed to save token", 400)
		return
	}
	fmt.Fprintf(w, "Success! Wrote token file to %s", auth.TokenFile)
}

func (auth *Auth) Serve() {
	http.HandleFunc("/login", auth.Login)
	http.HandleFunc("/callback", auth.Callback)
	fmt.Printf(colorstring.Color(
		"Visit [green]%s/login[reset] to authenticate "+
			"with Spotify.\n"), auth.ListenURL())
	http.ListenAndServe(auth.Listen, nil)
}
