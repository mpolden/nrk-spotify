package nrk

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

var ids = [...]string{
	"p1pluss",
	"p2",
	"p3",
	"p13",
	"mp3",
	"radio_super",
	"klassisk",
	"jazz",
	"folkemusikk",
	"urort",
	"radioresepsjonen",
	"national_rap_show",
	"pyro",
}

const defaultURL string = "http://v7.psapi.nrk.no"

type Radio struct {
	Name string
	ID   string
	url  string
}

type Playlist struct {
	Tracks []Track
}

type Track struct {
	Track      string `json:"title"`
	Artist     string `json:"description"`
	Type       string `json:"type"`
	StartTime_ string `json:"startTime"`
	Duration_  string `json:"duration"`
}

type Position struct {
	Position time.Duration
	Duration time.Duration
}

func IDs() [13]string {
	return ids
}

func New(name string, id string) (*Radio, error) {
	radio := Radio{
		Name: name,
		ID:   id,
	}
	if !radio.isValidID() {
		return nil, fmt.Errorf("%s is not a valid radio ID", radio.ID)
	}
	return &radio, nil
}

func (radio *Radio) URL() string {
	url := radio.url
	if url == "" {
		url = defaultURL
	}
	return url + fmt.Sprintf("/channels/%s/liveelements/now", radio.ID)
}

func (playlist *Playlist) Previous() (*Track, error) {
	if len(playlist.Tracks) > 0 {
		return &playlist.Tracks[0], nil
	}
	return nil, fmt.Errorf("previous track not found")
}

func (playlist *Playlist) Current() (*Track, error) {
	if len(playlist.Tracks) > 1 {
		return &playlist.Tracks[1], nil
	}
	return nil, fmt.Errorf("current track not found")
}

func (playlist *Playlist) Next() (*Track, error) {
	if len(playlist.Tracks) > 2 {
		return &playlist.Tracks[2], nil
	}
	return nil, fmt.Errorf("next track not found")
}

func (playlist *Playlist) CurrentAndNext() ([]Track, error) {
	if len(playlist.Tracks) < 3 {
		return nil, fmt.Errorf("playlist only contains %d tracks",
			len(playlist.Tracks))
	}
	return playlist.Tracks[1:], nil
}

func (track *Track) IsMusic() bool {
	return track.Type == "Music"
}

func (track *Track) StartTime() (time.Time, error) {
	re := regexp.MustCompile("^/Date\\((\\d+)\\+\\d+\\)/$")
	matches := re.FindAllStringSubmatch(track.StartTime_, 2)
	if len(matches) > 0 && len(matches[0]) == 2 {
		timestamp, err := strconv.ParseInt(matches[0][1], 10, 0)
		if err != nil {
			return time.Time{}, err
		}
		return time.Unix(timestamp/1000, 0), nil
	}
	return time.Time{}, fmt.Errorf("failed to parse start time")
}

func (track *Track) Duration() (time.Duration, error) {
	duration := strings.ToLower(track.Duration_)
	if strings.HasPrefix(duration, "pt") {
		duration = duration[2:]
	}
	return time.ParseDuration(duration)
}

func (track *Track) Position() (Position, error) {
	startTime, err := track.StartTime()
	if err != nil {
		return Position{}, err
	}
	duration, err := track.Duration()
	if err != nil {
		return Position{}, err
	}
	position := time.Now().Truncate(1 * time.Second).Sub(startTime)
	// If position is longer than duration, just assume we're at the end
	// This can happen if the API returns a incorrect startTime, or if the
	// system time is incorrect
	if position > duration {
		position = duration
	}
	// If position is negative, assume we're at the beginning
	if position < 0 {
		position = 0
	}
	return Position{
		Position: position,
		Duration: duration,
	}, nil
}

func (track *Track) String() string {
	return fmt.Sprintf("%s - %s", track.Artist, track.Track)
}

func (position *Position) String() string {
	floor := func(n float64) int {
		return int(n) % 60
	}
	return fmt.Sprintf("%02d:%02d/%02d:%02d",
		floor(position.Position.Minutes()),
		floor(position.Position.Seconds()),
		floor(position.Duration.Minutes()),
		floor(position.Duration.Seconds()))
}

func (position *Position) Symbol(scale int, colorize bool) string {
	ratio := position.Position.Seconds() / position.Duration.Seconds()

	elapsed := scale
	remaining := 0
	if ratio > 0 && ratio < 1 {
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

func (playlist *Playlist) NextSync(tracks []*Track) (time.Duration, error) {
	total := time.Duration(0)
	for _, track := range tracks {
		duration, err := playlist.Remaining(track)
		if err != nil {
			return time.Duration(0), err
		}
		total += duration
	}
	return total, nil
}

func (playlist *Playlist) Remaining(track *Track) (time.Duration, error) {
	current, err := playlist.Current()
	if err != nil {
		return time.Duration(0), err
	}
	if *current == *track {
		position, err := current.Position()
		if err != nil {
			return time.Duration(0), err
		}
		return position.Duration - position.Position, nil
	}
	duration, err := track.Duration()
	if err != nil {
		return time.Duration(0), err
	}
	return duration, nil
}

func (radio *Radio) Playlist() (*Playlist, error) {
	resp, err := http.Get(radio.URL())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var tracks []Track
	if err := json.Unmarshal(body, &tracks); err != nil {
		return nil, err
	}
	return &Playlist{Tracks: tracks}, nil
}

func (radio *Radio) isValidID() bool {
	for _, id := range ids {
		if id == radio.ID {
			return true
		}
	}
	return false
}
