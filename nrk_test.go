package main

import (
	"fmt"
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
