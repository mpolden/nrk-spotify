package main

import (
	"fmt"
	"github.com/docopt/docopt-go"
	"github.com/martinp/nrk-spotify/nrk"
	"github.com/martinp/nrk-spotify/server"
	"github.com/martinp/nrk-spotify/spotify"
	"log"
	"strconv"
	"time"
)

func makeSpotifyAuth(args map[string]interface{}) (string, *spotify.Auth) {
	clientId := args["<client-id>"].(string)
	clientSecret := args["<client-secret>"].(string)
	listen := args["--listen"].(string)
	tokenFile := args["--token-file"].(string)
	return listen, &spotify.Auth{
		ClientId:     clientId,
		ClientSecret: clientSecret,
		TokenFile:    tokenFile,
	}
}

func makeServer(args map[string]interface{}) (*server.Sync, error) {
	radioName := args["<name>"].(string)
	radioID := args["<radio-id>"].(string)
	tokenFile := args["--token-file"].(string)
	adaptive := args["--adaptive"].(bool)
	deleteEvicted := args["--delete-evicted"].(bool)
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
	s, err := spotify.New(tokenFile)
	if err != nil {
		return nil, err
	}
	radio, err := nrk.New(radioName, radioID)
	if err != nil {
		return nil, err
	}
	return &server.Sync{
		Spotify:       s,
		Radio:         radio,
		Interval:      time.Duration(interval) * time.Minute,
		Adaptive:      adaptive,
		CacheSize:     cacheSize,
		DeleteEvicted: deleteEvicted,
	}, nil
}

func main() {
	usage := `Listen to NRK radio channels in Spotify.

Usage:
  nrk-spotify auth [-l <address>] [-f <file>] <client-id> <client-secret>
  nrk-spotify server [-f <file>] [-i <minutes>] [-a] [-d] [-c <max>] <name> <radio-id>
  nrk-spotify list
  nrk-spotify -h | --help

Options:
  -h --help                Show help
  -f --token-file=<file>   Token file to use [default: .token.json]
  -l --listen=<address>    Auth server listening address [default: :8080]
  -i --interval=<minutes>  Polling interval [default: 5]
  -c --cache-size=<max>    Max entries to keep in cache [default: 100]
  -a --adaptive            Automatically determine sync interval
  -d --delete-evicted      Delete evicted (uncached) tracks from playlist`

	arguments, _ := docopt.Parse(usage, nil, true, "", false)
	auth := arguments["auth"].(bool)
	server := arguments["server"].(bool)

	if auth {
		listen, spotifyAuth := makeSpotifyAuth(arguments)
		spotifyAuth.Serve(listen)
	} else if server {
		server, err := makeServer(arguments)
		if err != nil {
			log.Fatalf("Failed to initialize server: %s", err)
		}
		server.Serve()
	} else {
		fmt.Println("Available radio IDs:")
		for _, id := range nrk.IDs() {
			fmt.Println(id)
		}
	}
}
