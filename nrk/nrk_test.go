package nrk

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func testTrack() RadioTrack {
	return RadioTrack{
		Track:      "Like a Rolling Stone",
		Artist:     "Bob Dylan",
		Type:       "Music",
		StartTime_: "/Date(1405971945000+0200)/",
		Duration_:  "PT6M10S",
	}
}

func TestStartTime(t *testing.T) {
	track := testTrack()
	startTime, err := track.StartTime()
	if err != nil {
		t.Fatalf("Failed to parse start time: %s", err)
	}
	expected := time.Unix(1405971945, 0)
	if startTime != expected {
		t.Fatalf("Expected %s, got %s", expected, startTime)
	}
}

func TestDuration(t *testing.T) {
	track := testTrack()
	duration, err := track.Duration()
	if err != nil {
		t.Fatalf("Failed to parse duration: %s", err)
	}
	expected := time.Duration(10*time.Second) +
		time.Duration(6*time.Minute)
	if duration != expected {
		t.Fatalf("Expected %s, got %s", expected, duration)
	}
}

func TestDurationWithoutSeconds(t *testing.T) {
	track := testTrack()
	track.Duration_ = "PT6M"
	duration, err := track.Duration()
	if err != nil {
		t.Fatalf("Failed to parse duration: %s", err)
	}
	expected := time.Duration(6 * time.Minute)
	if duration != expected {
		t.Fatalf("Expected %s, got %s", expected, duration)
	}
}

func TestDurationWithoutMinutes(t *testing.T) {
	track := testTrack()
	track.Duration_ = "PT10S"
	duration, err := track.Duration()
	if err != nil {
		t.Fatalf("Failed to parse duration: %s", err)
	}
	expected := time.Duration(10 * time.Second)
	if duration != expected {
		t.Fatalf("Expected %s, got %s", expected, duration)
	}
}

func TestPosition(t *testing.T) {
	track := testTrack()
	thirtySecsAgo := time.Now().Unix() - 30
	track.StartTime_ = fmt.Sprintf("/Date(%d000+0200)/", thirtySecsAgo)

	position, err := track.Position()
	if err != nil {
		t.Fatalf("Failed to get position: %s", err)
	}
	expected := time.Duration(30 * time.Second)
	if position.Position != expected {
		t.Fatalf("Expected %s, got %s", expected, position)
	}
}

func TestPositionString(t *testing.T) {
	track := testTrack()
	thirtySecsAgo := time.Now().Unix() - 30
	track.StartTime_ = fmt.Sprintf("/Date(%d000+0200)/", thirtySecsAgo)

	position, err := track.Position()
	if err != nil {
		t.Fatalf("Failed to parse position: %s", err)
	}

	pos := position.String()
	expected := "00:30/06:10"
	if pos != expected {
		t.Fatalf("Expected %s, got %s", expected, pos)
	}
}

func TestPositionSymbol(t *testing.T) {
	track := testTrack()
	thirtySecsAgo := time.Now().Unix() - 30
	track.StartTime_ = fmt.Sprintf("/Date(%d000+0200)/", thirtySecsAgo)

	position, err := track.Position()
	if err != nil {
		t.Fatalf("Failed to parse position: %s", err)
	}

	f := func(scale int, expected string) {
		symbol := position.Symbol(scale, false)
		if len(symbol) != scale {
			t.Fatalf("Expected string of length 12, got: %d",
				len(symbol))
		}
		if symbol != expected {
			t.Fatalf("Expected %s, got %s", expected, symbol)
		}
	}
	f(10, "=---------")
	f(20, "==------------------")
}

func TestPositionSymbolRatioGreaterThanOne(t *testing.T) {
	track := testTrack()
	// Start time after track duration
	track.StartTime_ = fmt.Sprintf("/Date(%d000+0200)/",
		time.Now().Unix()-371)

	position, err := track.Position()
	if err != nil {
		t.Fatalf("Failed to parse position: %s", err)
	}

	symbol := position.Symbol(10, false)
	if len(symbol) != 10 {
		t.Fatalf("Expected string of length 10, got: %d",
			len(symbol))
	}
	expected := "=========="
	if symbol != expected {
		t.Fatalf("Expected %s, got %s", expected, symbol)
	}
}

func TestPositionLongerThanDuration(t *testing.T) {
	track := testTrack()
	track.StartTime_ = fmt.Sprintf("/Date(%d000+0200)/",
		time.Now().Unix()-371)

	position, err := track.Position()
	if err != nil {
		t.Fatalf("Failed to parse position: %s", err)
	}
	if position.Position > position.Duration {
		t.Fatalf("Position should NOT be larger than duration: %s > %s",
			position.Position, position.Duration)
	}
}

func TestPositionNegative(t *testing.T) {
	track := testTrack()
	track.StartTime_ = fmt.Sprintf("/Date(%d000+0200)/",
		time.Now().Unix()+370)
	position, err := track.Position()
	if err != nil {
		t.Fatalf("Failed to parse position: %s", err)
	}
	if position.Position != 0 {
		t.Fatalf("Expected 0, got %s", position.Position)
	}
}

func TestNextSync(t *testing.T) {
	currentTrack := testTrack()
	// 42 seconds into current track
	currentTrack.StartTime_ = fmt.Sprintf("/Date(%d000+0200)/",
		time.Now().Unix()-42)

	// Next track is 3m 37s long
	nextTrack := testTrack()
	nextTrack.Duration_ = "PT3M37S"

	playlist := RadioPlaylist{
		Tracks: []RadioTrack{RadioTrack{}, currentTrack, nextTrack},
	}

	nextSync, err := playlist.NextSync()
	if err != nil {
		t.Fatalf("Failed to find next sync time: %s", err)
	}
	expected := time.Duration(9*time.Minute) +
		time.Duration(5*time.Second)
	if nextSync != expected {
		t.Fatalf("Expected %s, got %s", expected, nextSync)
	}
}

func TestCurrentAndNext(t *testing.T) {
	tracks := []RadioTrack{
		RadioTrack{Track: "A"},
		RadioTrack{Track: "B"},
		RadioTrack{Track: "C"},
	}
	playlist := RadioPlaylist{Tracks: tracks}
	currentAndNext, err := playlist.CurrentAndNext()
	if err != nil {
		t.Fatalf("Failed to get current and next track")
	}
	expected := "B"
	if currentAndNext[0].Track != expected {
		t.Fatalf("Expected %s, got %s", expected, currentAndNext[0])
	}
	expected = "C"
	if currentAndNext[1].Track != expected {
		t.Fatalf("Expected %s, got %s", expected, currentAndNext[1])
	}
}

func TestCurrentAndNextInvalidLength(t *testing.T) {
	tracks := []RadioTrack{
		RadioTrack{Track: "A"},
	}
	playlist := RadioPlaylist{Tracks: tracks}
	_, err := playlist.CurrentAndNext()
	if err == nil {
		t.Fatalf("Expected error for playlist length < 3")
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

func TestPlaylist(t *testing.T) {
	server := newTestServer("/", testResponse)
	defer server.Close()
	r := Radio{Name: "P3 Pyro", ID: "pyro", url: server.URL}
	playlist, err := r.Playlist()
	if err != nil {
		t.Fatal(err)
	}
	if len(playlist.Tracks) != 3 {
		t.Fatalf("Expected playlist to contain 3 tracks, got %d",
			len(playlist.Tracks))
	}
	if playlist.Tracks[0].Track != "21st Century Schizoid Man" {
		t.Fatalf("Expected '21st Century Schizoid Man', got %s",
			playlist.Tracks[0].Track)
	}
	if playlist.Tracks[1].Track != "Room 24" {
		t.Fatalf("Expected 'Room 24', got %s",
			playlist.Tracks[1].Track)
	}
	if playlist.Tracks[2].Track != "The King is Dead" {
		t.Fatalf("Expected 'The King is Dead', got %s",
			playlist.Tracks[2].Track)
	}
}

func TestPlaylistInvalidResponse(t *testing.T) {
	server := newTestServer("/", "gopher says: no JSON for you!")
	defer server.Close()
	r := Radio{Name: "P3 Pyro", ID: "pyro", url: server.URL}
	_, err := r.Playlist()
	if err == nil {
		t.Fatal("Expected error for invalid JSON response")
	}
}

const testResponse string = `
[
  {
    "title": "21st Century Schizoid Man",
    "description": "King Crimson",
    "programId": "unknown",
    "channelId": "pyro",
    "startTime": "/Date(1406402993000+0200)/",
    "duration": "PT7M21S",
    "type": "Music",
    "imageUrl": "http://gfx.nrk.no/-VFyp6wjWilG-wNIF3mOiwPAg7Bign7rPjUT_F26vpOQ",
    "programTitle": null,
    "relativeTimeType": "Present",
    "category": null,
    "contributors": "",
    "creators": null
  },
  {
    "title": "Room 24",
    "description": "Volbeat + King Diamond",
    "programId": "unknown",
    "channelId": "pyro",
    "startTime": "/Date(1406403434000+0200)/",
    "duration": "PT5M6S",
    "type": "Music",
    "imageUrl": null,
    "programTitle": null,
    "relativeTimeType": "Present",
    "category": null,
    "contributors": "",
    "creators": null
  },
  {
    "title": "The King is Dead",
    "description": "Audrey Horne",
    "programId": "unknown",
    "channelId": "pyro",
    "startTime": "/Date(1406403741000+0200)/",
    "duration": "PT5M12S",
    "type": "Music",
    "imageUrl": "http://gfx.nrk.no/Uuw5ZZW7q0OzI6iQ_JMS1wlT5xzkzsB-Pf9t_XWvfniQ",
    "programTitle": null,
    "relativeTimeType": "Future",
    "category": null,
    "contributors": "",
    "creators": null
  }
]`
