package main

import (
	"fmt"
	"github.com/docopt/docopt-go"
	"github.com/mitchellh/colorstring"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Args struct {
	Auth         bool
	Server       bool
	Listen       string
	TokenFile    string
	ClientId     string
	ClientSecret string
	Scope        string
	RadioName    string
	ApiUrl       string
	Interval     time.Duration
}

func makeArgs(args map[string]interface{}) Args {
	auth := args["auth"].(bool)
	server := args["server"].(bool)
	listen := args["--listen"].(string)
	tokenFile := args["--token-file"].(string)
	intervalOpt := args["--interval"].(string)
	clientId := ""
	clientSecret := ""
	radioName := ""
	apiUrl := ""
	if auth {
		clientId = args["<client-id>"].(string)
		clientSecret = args["<client-secret>"].(string)
	}
	if server {
		radioName = args["<radio-name>"].(string)
		apiUrl = args["<api-url>"].(string)
	}
	scope := "playlist-modify playlist-modify-private " +
		"playlist-read-private"
	interval, err := strconv.Atoi(intervalOpt)
	if err != nil || interval < 1 {
		fmt.Println("--interval must be an positive integer")
		os.Exit(1)
	}
	return Args{
		Auth:         auth,
		Listen:       listen,
		Server:       server,
		TokenFile:    tokenFile,
		ClientId:     clientId,
		ClientSecret: clientSecret,
		Scope:        scope,
		RadioName:    radioName,
		ApiUrl:       apiUrl,
		Interval:     time.Duration(interval),
	}
}

func (args *Args) Url() string {
	if strings.HasPrefix(args.Listen, ":") {
		return "http://localhost" + args.Listen
	}
	return "http://" + args.Listen
}

func (args *Args) CallbackUrl() string {
	return args.Url() + "/callback"
}

func main() {
	usage := `Listen to NRK radio channels in Spotify.

Usage:
  nrk-spotify auth [-l <address>] [-f <token-file>] <client-id> <client-secret>
  nrk-spotify server [-f <token-file>] [-i <interval>] <radio-name> <api-url>
  nrk-spotify -h | --help

Options:
  -h --help                     Show help
  -f --token-file=<token-file>  Token file to use [default: .token.json]
  -l --listen=<address>         Auth server listening address [default: :8080]
  -i --interval=<minutes>       Polling interval [default: 5]`

	arguments, _ := docopt.Parse(usage, nil, true, "", false)
	args := makeArgs(arguments)

	if args.Auth {
		spotifyAuth := SpotifyAuth{
			ClientId:     args.ClientId,
			ClientSecret: args.ClientSecret,
			CallbackUrl:  args.CallbackUrl(),
			TokenFile:    args.TokenFile,
			Scope:        args.Scope,
		}
		http.HandleFunc("/login", spotifyAuth.login)
		http.HandleFunc("/callback", spotifyAuth.callback)

		fmt.Printf(colorstring.Color(
			"Visit [green]%s/login[reset] to authenticate "+
				"with Spotify.\n"), args.Url())
		fmt.Printf(colorstring.Color(
			"Listening at [green]%s[reset]\n"), args.Listen)
		http.ListenAndServe(args.Listen, nil)
	}
	if args.Server {
		token, err := ReadToken(args.TokenFile)
		if err != nil {
			log.Fatalf("Failed to read file: %s", err)
		}
		// Set and save current user if profile is empty
		if token.Profile.Id == "" {
			profile, err := token.currentUser()
			if err != nil {
				log.Fatalf("Failed to get current user: %s\n",
					err)

			}
			token.Profile = *profile
			if err := token.save(token.Auth.TokenFile); err != nil {
				log.Fatalf("Failed to save token: %s\n", err)
			}
		}

		radio := NrkRadio{
			Name: args.RadioName,
			Url:  args.ApiUrl,
		}
		spotifyPlaylist, err := token.getOrCreatePlaylist(radio.Name)
		if err != nil {
			log.Fatalf("Failed to get or create playlist: %s", err)
		}
		server := SyncServer{
			Spotify:  token,
			Playlist: spotifyPlaylist,
			Radio:    &radio,
			Interval: args.Interval,
		}
		server.Serve()
	}
}
