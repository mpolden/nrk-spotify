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
	radioName := args["<name>"].(string)
	radioId := args["<radio-id>"].(string)
	tokenFile := args["--token-file"].(string)
	adaptive := args["--adaptive"].(bool)
	intervalOpt := args["--interval"].(string)
	interval, err := strconv.Atoi(intervalOpt)
	if err != nil || interval < 1 {
		return nil, fmt.Errorf("--interval must be an positive integer")
	}
	cacheSizeOpt := args["--cache-size"].(string)
	cacheSize, err := strconv.Atoi(cacheSizeOpt)
	if err != nil || interval < 1 {
		return nil, fmt.Errorf(
			"--cache-size must be an positive integer")
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
	if !radio.isValidId() {
		return nil, fmt.Errorf("'%s' is not a valid radio ID", radio.Id)
	}
	return &SyncServer{
		Spotify:   spotify,
		Radio:     &radio,
		Interval:  time.Duration(interval) * time.Minute,
		Adaptive:  adaptive,
		CacheSize: cacheSize,
	}, nil
}

func main() {
	usage := `Listen to NRK radio channels in Spotify.

Usage:
  nrk-spotify auth [-l <address>] [-f <file>] <client-id> <client-secret>
  nrk-spotify server [-f <file>] [-i <minutes>] [-a] [-c <max>] <name> <radio-id>
  nrk-spotify list
  nrk-spotify -h | --help

Options:
  -h --help                Show help
  -f --token-file=<file>   Token file to use [default: .token.json]
  -l --listen=<address>    Auth server listening address [default: :8080]
  -i --interval=<minutes>  Polling interval [default: 5]
  -c --cache-size=<max>    Max entries to keep in cache [default: 100]
  -a --adaptive            Automatically determine sync interval`

	arguments, _ := docopt.Parse(usage, nil, true, "", false)
	auth := arguments["auth"].(bool)
	server := arguments["server"].(bool)

	if auth {
		spotifyAuth := makeSpotifyAuth(arguments)
		spotifyAuth.Serve()
	} else if server {
		server, err := makeServer(arguments)
		if err != nil {
			log.Fatalf("Failed to initialize server: %s", err)
		}
		server.Serve()
	} else {
		fmt.Println("Available radio IDs:")
		for _, id := range radioIds {
			fmt.Println(id)
		}
	}
}
