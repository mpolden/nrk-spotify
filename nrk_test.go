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

func TestPosition(t *testing.T) {
	track := testTrack()
	thirtySecsAgo := time.Now().Unix() - 30
	track.StartTime_ = fmt.Sprintf("/Date(%d000+0200)/", thirtySecsAgo)

	position, err := track.Position()
	if err != nil {
		t.Fatalf("Failed to get position: %s", err)
	}
	expected := time.Duration(30 * time.Second)
	if position != expected {
		t.Fatalf("Expected %s, got %s", expected, position)
	}
}

func TestPositionString(t *testing.T) {
	track := testTrack()
	thirtySecsAgo := time.Now().Unix() - 30
	track.StartTime_ = fmt.Sprintf("/Date(%d000+0200)/", thirtySecsAgo)

	position, err := track.PositionString()
	if err != nil {
		t.Fatalf("Failed to get position: %s", err)
	}
	expected := "00:30/06:10"
	if position != expected {
		t.Fatalf("Expected %s, got %s", expected, position)		
	}
}
