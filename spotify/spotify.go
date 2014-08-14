package spotify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

type Spotify struct {
	Token   `json:"token"`
	Auth    `json:"auth"`
	Profile `json:"profile"`
}

type Token struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

type Profile struct {
	ExternalUrls map[string]string `json:"external_urls"`
	Href         string            `json:"href"`
	Id           string            `json:"id"`
	Type         string            `json:"type"`
	Uri          string            `json:"uri"`
}

type Playlist struct {
	Id     string         `json:"id"`
	Name   string         `json:"name"`
	Tracks PlaylistTracks `json:"tracks"`
}

type PlaylistTracks struct {
	Limit    int             `json:"limit"`
	Next     string          `json:"next"`
	Offset   int             `json:"offset"`
	Previous string          `json:"previous"`
	Total    int             `json:"total"`
	Items    []PlaylistTrack `json:"items"`
}

type PlaylistTrack struct {
	Track Track `json:"track"`
}

type Playlists struct {
	Items []Playlist `json:"items"`
}

type NewPlaylist struct {
	Name   string `json:"name"`
	Public bool   `json:"public"`
}

type SearchResult struct {
	Tracks SearchTracks `json:"tracks"`
}

type SearchTracks struct {
	Items []Track `json:"items"`
}

type Track struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Uri  string `json:"uri"`
}

func (playlist *Playlist) Contains(track Track) bool {
	for _, item := range playlist.Tracks.Items {
		if item.Track.Id == track.Id {
			return true
		}
	}
	return false
}

func (playlist *Playlist) String() string {
	return fmt.Sprintf("%s (%s) [%d songs]", playlist.Name, playlist.Id,
		playlist.Tracks.Total)
}

func (spotify *Spotify) update(newToken *Spotify) {
	spotify.AccessToken = newToken.AccessToken
	spotify.TokenType = newToken.TokenType
	spotify.ExpiresIn = newToken.ExpiresIn
}

func (spotify *Spotify) updateToken() error {
	formData := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {spotify.RefreshToken},
	}
	url := "https://accounts.spotify.com/api/token"
	client := &http.Client{}
	req, err := http.NewRequest("POST", url,
		bytes.NewBufferString(formData.Encode()))
	req.Header.Set("Authorization", spotify.Auth.authHeader())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var newToken Spotify
	if err := json.Unmarshal(body, &newToken); err != nil {
		return err
	}
	spotify.update(&newToken)
	return nil
}

func (spotify *Spotify) authHeader() string {
	return spotify.TokenType + " " + spotify.AccessToken
}

type requestFn func() (*http.Response, error)

func (spotify *Spotify) request(reqFn requestFn) ([]byte, error) {
	resp, err := reqFn()
	if resp.StatusCode == 401 {
		if err := spotify.updateToken(); err != nil {
			return nil, err
		}
		if err := spotify.Save(spotify.Auth.TokenFile); err != nil {
			return nil, err
		}
		resp, err = reqFn()
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("request failed (%d): %s",
			resp.StatusCode, body)
	}
	return body, err
}

func (spotify *Spotify) get(url string) ([]byte, error) {
	getFn := func() (*http.Response, error) {
		client := &http.Client{}
		req, err := http.NewRequest("GET", url, nil)
		req.Header.Set("Authorization", spotify.authHeader())
		if err != nil {
			return nil, err
		}
		return client.Do(req)
	}
	return spotify.request(getFn)
}

func (spotify *Spotify) post(url string, body []byte) ([]byte, error) {
	postFn := func() (*http.Response, error) {
		client := &http.Client{}
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
		req.Header.Set("Authorization", spotify.authHeader())
		req.Header.Set("Content-Type", "application/json")
		if err != nil {
			return nil, err
		}
		return client.Do(req)
	}
	return spotify.request(postFn)
}

func (spotify *Spotify) delete(url string, body []byte) ([]byte, error) {
	deleteFn := func() (*http.Response, error) {
		client := &http.Client{}
		req, err := http.NewRequest("DELETE", url,
			bytes.NewBuffer(body))
		req.Header.Set("Authorization", spotify.authHeader())
		req.Header.Set("Content-Type", "application/json")
		if err != nil {
			return nil, err
		}
		return client.Do(req)
	}
	return spotify.request(deleteFn)
}

func (spotify *Spotify) Save(filepath string) error {
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

func New(filepath string) (*Spotify, error) {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	var spotify Spotify
	if err := json.Unmarshal(data, &spotify); err != nil {
		return nil, err
	}
	if spotify.Profile.Id == "" {
		if err := spotify.SetCurrentUser(); err != nil {
			return nil, err
		}
	}
	return &spotify, nil
}

func (spotify *Spotify) CurrentUser() (*Profile, error) {
	url := "https://api.spotify.com/v1/me"
	body, err := spotify.get(url)
	if err != nil {
		return nil, err
	}
	var profile Profile
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

func (spotify *Spotify) Playlists() ([]Playlist, error) {
	url := fmt.Sprintf("https://api.spotify.com/v1/users/%s/playlists",
		spotify.Profile.Id)
	body, err := spotify.get(url)
	if err != nil {
		return nil, err
	}
	var playlists Playlists
	if err := json.Unmarshal(body, &playlists); err != nil {
		return nil, err
	}
	return playlists.Items, nil
}

func (spotify *Spotify) PlaylistById(playlistId string) (*Playlist, error) {
	url := fmt.Sprintf("https://api.spotify.com/v1/users/%s/playlists/%s",
		spotify.Profile.Id, playlistId)
	body, err := spotify.get(url)
	if err != nil {
		return nil, err
	}
	var playlist Playlist
	if err := json.Unmarshal(body, &playlist); err != nil {
		return nil, err
	}
	return &playlist, nil
}

func (spotify *Spotify) Playlist(name string) (*Playlist, error) {
	playlists, err := spotify.Playlists()
	if err != nil {
		return nil, err
	}
	playlistId := ""
	for _, playlist := range playlists {
		if playlist.Name == name {
			playlistId = playlist.Id
			break
		}
	}
	if playlistId == "" {
		return nil, nil
	}
	return spotify.PlaylistById(playlistId)
}

func (spotify *Spotify) GetOrCreatePlaylist(name string) (*Playlist, error) {
	existing, err := spotify.Playlist(name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	url := fmt.Sprintf("https://api.spotify.com/v1/users/%s/playlists",
		spotify.Profile.Id)
	newPlaylist, err := json.Marshal(NewPlaylist{
		Name:   name,
		Public: false,
	})
	if err != nil {
		return nil, err
	}
	body, err := spotify.post(url, newPlaylist)
	if err != nil {
		return nil, err
	}
	var playlist Playlist
	if err := json.Unmarshal(body, &playlist); err != nil {
		return nil, err
	}
	return &playlist, err
}

func (spotify *Spotify) RecentTracks(playlist *Playlist,
	n int) ([]PlaylistTrack, error) {
	// If playlist has <= 100 tracks, return the last n tracks without doing
	// another request
	if playlist.Tracks.Total <= 100 {
		offset := len(playlist.Tracks.Items) - n
		if offset > 0 {
			return playlist.Tracks.Items[offset:], nil
		}
		return playlist.Tracks.Items, nil
	}

	offset := playlist.Tracks.Total - n
	if offset < 0 {
		offset = 0
	}
	params := url.Values{"offset": {strconv.Itoa(offset)}}
	url := fmt.Sprintf(
		"https://api.spotify.com/v1/users/%s/playlists/%s/tracks?",
		spotify.Profile.Id, playlist.Id)

	tracks := make([]PlaylistTrack, 0, n)
	nextUrl := url + params.Encode()
	for nextUrl != "" {
		body, err := spotify.get(nextUrl)
		if err != nil {
			return nil, err
		}
		var playlistTracks PlaylistTracks
		if err := json.Unmarshal(body, &playlistTracks); err != nil {
			return nil, err
		}
		tracks = append(tracks, playlistTracks.Items...)
		nextUrl = playlistTracks.Next
	}
	return tracks, nil
}

func (spotify *Spotify) Search(query string, types string, limit int) ([]Track,
	error) {
	params := url.Values{
		"q":     {query},
		"type":  {types},
		"limit": {strconv.Itoa(limit)},
	}
	url := "https://api.spotify.com/v1/search?" + params.Encode()
	body, err := spotify.get(url)
	if err != nil {
		return nil, err
	}
	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result.Tracks.Items, nil
}

func (spotify *Spotify) SearchArtistTrack(artist string, track string) ([]Track,
	error) {
	query := fmt.Sprintf("artist:%s track:%s", artist, track)
	tracks, err := spotify.Search(query, "track", 1)
	if err != nil {
		return nil, err
	}
	return tracks, nil
}

func (spotify *Spotify) AddTracks(playlist *Playlist, tracks []Track) error {
	url := fmt.Sprintf(
		"https://api.spotify.com/v1/users/%s/playlists/%s/tracks",
		spotify.Profile.Id, playlist.Id)

	uris := make([]string, len(tracks))
	for i, track := range tracks {
		uris[i] = track.Uri
	}
	jsonUris, err := json.Marshal(uris)
	if err != nil {
		return err
	}
	if _, err := spotify.post(url, jsonUris); err != nil {
		return err
	}
	return nil
}

func (spotify *Spotify) AddTrack(playlist *Playlist, track *Track) error {
	return spotify.AddTracks(playlist, []Track{*track})
}

func (track *Track) String() string {
	return fmt.Sprintf("%s (%s)", track.Name, track.Id)
}

func (spotify *Spotify) SetCurrentUser() error {
	profile, err := spotify.CurrentUser()
	if err != nil {
		return err
	}
	spotify.Profile = *profile
	if err := spotify.Save(spotify.Auth.TokenFile); err != nil {
		return err
	}
	return nil
}

func (spotify *Spotify) DeleteTracks(playlist *Playlist, tracks []Track) error {
	url := fmt.Sprintf(
		"https://api.spotify.com/v1/users/%s/playlists/%s/tracks",
		spotify.Profile.Id, playlist.Id)

	uris := make([]map[string]string, len(tracks))
	for i, track := range tracks {
		uris[i] = map[string]string{
			"uri": track.Uri,
		}
	}
	trackUris := map[string][]map[string]string{
		"tracks": uris,
	}
	jsonUris, err := json.Marshal(trackUris)
	if err != nil {
		return err
	}
	if _, err := spotify.delete(url, jsonUris); err != nil {
		return err
	}
	return nil
}

func (spotify *Spotify) DeleteTrack(playlist *Playlist, track *Track) error {
	return spotify.DeleteTracks(playlist, []Track{*track})
}
