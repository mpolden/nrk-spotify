package main

import (
	"testing"
)

func TestURL(t *testing.T) {
	auth := SpotifyAuth{listen: ":8080"}
	if auth.ListenURL() != "http://localhost:8080" {
		t.Fatalf("Expected http://localhost:8080, got %s",
			auth.ListenURL())
	}
	auth = SpotifyAuth{listen: "1.2.3.4:8080"}
	if auth.ListenURL() != "http://1.2.3.4:8080" {
		t.Fatalf("Expected http://1.2.3.4:8080, got %s",
			auth.ListenURL())
	}
}

func TestAuthHeader(t *testing.T) {
	auth := SpotifyAuth{
		ClientId:     "foo",
		ClientSecret: "bar",
	}
	expected := "Basic Zm9vOmJhcg==" // base64("foo:bar")
	if auth.authHeader() != expected {
		t.Fatalf("Expected '%s', got '%s'", expected, auth.authHeader())
	}
}
