package main

import (
	"encoding/json"
	"fmt"
	"github.com/mitchellh/colorstring"
	"io/ioutil"
	"net/http"
	"net/url"
)

type SpotifyAuth struct {
	ClientId     string
	ClientSecret string
	CallbackUrl  string
	Scope        string
	TokenFile    string
}

type SpotifyToken struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    uint   `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

func (token *SpotifyToken) save(filepath string) error {
	json, err := json.Marshal(token)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath, json, 0600)
	if err != nil {
		return err
	}
	return nil
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

func (auth *SpotifyAuth) getToken(code []string) (*SpotifyToken, error) {
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
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var token SpotifyToken
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

func (auth *SpotifyAuth) callback(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	code, exists := queryParams["code"]
	if !exists {
		http.Error(w, "Missing required query parameter: code", 400)
		return
	}
	token, err := auth.getToken(code)
	if err != nil {
		fmt.Printf("Failed to get token: %s\n", err)
		http.Error(w, "Failed to retrieve token from Spotify", 400)
		return
	}
	err = token.save(auth.TokenFile)
	if err != nil {
		fmt.Printf("Failed to save token: %s\n", err)
		http.Error(w, "Failed to save token", 400)
		return
	}
	fmt.Printf(colorstring.Color("Wrote file: [green]%s[reset]\n"),
		auth.TokenFile)
	fmt.Fprintf(w, "Success! Wrote token file to %s", auth.TokenFile)
}
