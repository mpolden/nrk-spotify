package main

import (
	"fmt"
	"github.com/docopt/docopt-go"
	"log"
	"strconv"
	"time"
)

func makeSpotifyAuth(args map[string]interface{}) *SpotifyAuth {
	clientId := args["<client-id>"].(string)
	clientSecret := args["<client-secret>"].(string)
	listen := args["--listen"].(string)
	tokenFile := args["--token-file"].(string)
	return &SpotifyAuth{
		ClientId:     clientId,
		ClientSecret: clientSecret,
		TokenFile:    tokenFile,
		listen:       listen,
	}
}

func makeServer(args map[string]interface{}) (*SyncServer, error) {
	radioName := args["<radio-name>"].(string)
	radioId := args["<radio-id>"].(string)
	tokenFile := args["--token-file"].(string)
	intervalOpt := args["--interval"].(string)
	interval, err := strconv.Atoi(intervalOpt)
	if err != nil || interval < 1 {
		return nil, fmt.Errorf("--interval must be an positive integer")
	}
	spotify, err := ReadToken(tokenFile)
	if err != nil {
		return nil, err
	}
	// Set and save current user if profile is empty
	if spotify.Profile.Id == "" {
		if err := spotify.SetCurrentUser(); err != nil {
			return nil, err
		}
	}
	radio := Radio{
		Name: radioName,
		Id:   radioId,
	}
	return &SyncServer{
		Spotify:  spotify,
		Radio:    &radio,
		Interval: time.Duration(interval),
	}, nil
}

func main() {
	usage := `Listen to NRK radio channels in Spotify.

Usage:
  nrk-spotify auth [-l <address>] [-f <token-file>] <client-id> <client-secret>
  nrk-spotify server [-f <token-file>] [-i <interval>] <radio-name> <radio-id>
  nrk-spotify -h | --help

Options:
  -h --help                     Show help
  -f --token-file=<token-file>  Token file to use [default: .token.json]
  -l --listen=<address>         Auth server listening address [default: :8080]
  -i --interval=<minutes>       Polling interval [default: 5]`

	arguments, _ := docopt.Parse(usage, nil, true, "", false)
	auth := arguments["auth"].(bool)
	server := arguments["server"].(bool)

	if auth {
		spotifyAuth := makeSpotifyAuth(arguments)
		spotifyAuth.Serve()
	}
	if server {
		server, err := makeServer(arguments)
		if err != nil {
			log.Fatalf("Failed to initialize server: %s", err)
		}
		server.Serve()
	}
}
