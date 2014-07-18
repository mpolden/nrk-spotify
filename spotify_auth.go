package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type SpotifyAuth struct {
	ClientId     string
	ClientSecret string
	CallbackUrl  string
	Scope        string
}

type SpotifyToken struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    uint   `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

func (auth *SpotifyAuth) login(w http.ResponseWriter, r *http.Request) {
	params := url.Values{
		"response_type": {"code"},
		"client_id":     {auth.ClientId},
		"scope":         {auth.Scope},
		"redirect_uri":  {auth.CallbackUrl},
	}
	url := "https://accounts.spotify.com/authorize?" + params.Encode()
	http.Redirect(w, r, url, http.StatusFound)
}

func (auth *SpotifyAuth) callback(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	code, exists := queryParams["code"]
	if !exists {
		http.Error(w, "code param not given", 400)
		return
	}

	formData := url.Values{
		"code":          code,
		"redirect_uri":  {auth.CallbackUrl},
		"grant_type":    {"authorization_code"},
		"client_id":     {auth.ClientId},
		"client_secret": {auth.ClientSecret},
	}
	url := "https://accounts.spotify.com/api/token"
	resp, err := http.PostForm(url, formData)
	if err != nil {
		fmt.Printf("POST failed: %s\n", err)
		http.Error(w, "failed to get token", 400)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read response: %s\n", err)
		http.Error(w, "failed to read response", 400)
		return
	}
	var token SpotifyToken
	if err := json.Unmarshal(body, &token); err != nil {
		fmt.Printf("Failed to unmarshal JSON: %s\n", err)
		http.Error(w, "failed to read json response", 400)
		return
	}
	fmt.Printf("%+v\n", token)
}
