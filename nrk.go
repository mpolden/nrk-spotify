package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type NrkRadio struct {
	Url string
}

type RadioPlaylist struct {
	Tracks []RadioTrack
}

type RadioTrack struct {
	Track  string `json:"title"`
	Artist string `json:"description"`
	Type   string `json:"type"`
}

func (playlist *RadioPlaylist) Previous() (*RadioTrack, error) {
	if len(playlist.Tracks) > 0 {
		return &playlist.Tracks[0], nil
	}
	return nil, fmt.Errorf("Previous track not found")
}

func (playlist *RadioPlaylist) Current() (*RadioTrack, error) {
	if len(playlist.Tracks) > 1 {
		return &playlist.Tracks[1], nil
	}
	return nil, fmt.Errorf("Current track not found")
}

func (playlist *RadioPlaylist) Next() (*RadioTrack, error) {
	if len(playlist.Tracks) > 2 {
		return &playlist.Tracks[2], nil
	}
	return nil, fmt.Errorf("Next track not found")
}

func (track *RadioTrack) IsMusic() bool {
	return track.Type == "Music"
}

func (radio *NrkRadio) Playlist() (*RadioPlaylist, error) {
	resp, err := http.Get(radio.Url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var tracks []RadioTrack
	if err := json.Unmarshal(body, &tracks); err != nil {
		return nil, err
	}
	return &RadioPlaylist{Tracks: tracks}, nil
}
