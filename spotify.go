package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Spotify struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    uint   `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

func (spotify *Spotify) authHeader() string {
	return spotify.TokenType + " " + spotify.AccessToken
}

func (spotify *Spotify) get(url string) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", spotify.authHeader())
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

func (spotify *Spotify) save(filepath string) error {
	json, err := json.Marshal(spotify)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath, json, 0600)
	if err != nil {
		return err
	}
	return nil
}

func ReadToken(filepath string) (*Spotify, error) {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	var spotify Spotify
	if err := json.Unmarshal(data, &spotify); err != nil {
		return nil, err
	}
	return &spotify, nil
}
