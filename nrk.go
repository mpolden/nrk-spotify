package main

import (
	"encoding/json"
	"fmt"
	"github.com/mitchellh/colorstring"
	"io/ioutil"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type NrkRadio struct {
	Name string
	Id   string
}

type RadioPlaylist struct {
	Tracks []RadioTrack
}

type RadioTrack struct {
	Track      string `json:"title"`
	Artist     string `json:"description"`
	Type       string `json:"type"`
	StartTime_ string `json:"startTime"`
	Duration_  string `json:"duration"`
}

type PositionMeta struct {
	Position time.Duration
	Duration time.Duration
}

func (radio *NrkRadio) Url() string {
	return fmt.Sprintf(
		"http://v7.psapi.nrk.no/channels/%s/liveelements/now", radio.Id)
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

func (track *RadioTrack) StartTime() (time.Time, error) {
	re := regexp.MustCompile("^/Date\\((\\d+)\\+\\d+\\)/$")
	matches := re.FindAllStringSubmatch(track.StartTime_, 2)
	if len(matches) > 0 && len(matches[0]) == 2 {
		timestamp, err := strconv.ParseInt(matches[0][1], 10, 0)
		if err != nil {
			return time.Time{}, err
		}
		return time.Unix(timestamp/1000, 0), nil
	}
	return time.Time{}, fmt.Errorf("Could not parse start time")
}

func (track *RadioTrack) Duration() (time.Duration, error) {
	duration := strings.ToLower(track.Duration_)
	if strings.HasPrefix(duration, "pt") {
		duration = duration[2:]
	}
	return time.ParseDuration(duration)
}

func (track *RadioTrack) Position() (time.Duration, error) {
	startTime, err := track.StartTime()
	if err != nil {
		return time.Duration(0), err
	}
	return time.Now().Truncate(1 * time.Second).Sub(startTime), nil
}

func (track *RadioTrack) PositionMeta() (PositionMeta, error) {
	position, err := track.Position()
	if err != nil {
		return PositionMeta{}, err
	}
	duration, err := track.Duration()
	if err != nil {
		return PositionMeta{}, err
	}
	return PositionMeta{
		Position: position,
		Duration: duration,
	}, nil
}

func (track *RadioTrack) String() string {
	return fmt.Sprintf("%s - %s", track.Artist, track.Track)
}

func (meta *PositionMeta) PositionString() string {
	floor := func(n float64) int {
		return int(n) % 60
	}
	return fmt.Sprintf("%02d:%02d/%02d:%02d",
		floor(meta.Position.Minutes()),
		floor(meta.Position.Seconds()),
		floor(meta.Duration.Minutes()),
		floor(meta.Duration.Seconds()))
}

func (meta *PositionMeta) PositionSymbol(scale int, colorize bool) string {
	ratio := meta.Position.Seconds() / meta.Duration.Seconds()

	elapsed := scale
	remaining := 0
	if ratio < 1 {
		elapsed = int(math.Ceil(ratio * float64(scale)))
		remaining = (1 * scale) - elapsed
	}

	elapsedSymbol := strings.Repeat("=", elapsed)
	remainingSymbol := strings.Repeat("-", remaining)

	if colorize {
		elapsedSymbol = fmt.Sprintf(
			colorstring.Color("[green]%s[reset]"), elapsedSymbol)
	}

	return elapsedSymbol + remainingSymbol
}

func (radio *NrkRadio) Playlist() (*RadioPlaylist, error) {
	resp, err := http.Get(radio.Url())
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
