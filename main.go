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
	RadioName    string
	RadioId      string
	Interval     time.Duration
}

func makeArgs(args map[string]interface{}) (*Args, error) {
	auth := args["auth"].(bool)
	server := args["server"].(bool)
	listen := args["--listen"].(string)
	tokenFile := args["--token-file"].(string)
	intervalOpt := args["--interval"].(string)
	clientId := ""
	clientSecret := ""
	radioName := ""
	radioId := ""
	if auth {
		clientId = args["<client-id>"].(string)
		clientSecret = args["<client-secret>"].(string)
	}
	if server {
		radioName = args["<radio-name>"].(string)
		radioId = args["<radio-id>"].(string)
	}
	interval, err := strconv.Atoi(intervalOpt)
	if err != nil || interval < 1 {
		return nil, fmt.Errorf("--interval must be an positive integer")
	}
	return &Args{
		Auth:         auth,
		Listen:       listen,
		Server:       server,
		TokenFile:    tokenFile,
		ClientId:     clientId,
		ClientSecret: clientSecret,
		RadioName:    radioName,
		RadioId:      radioId,
		Interval:     time.Duration(interval),
	}, nil
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
  nrk-spotify server [-f <token-file>] [-i <interval>] <radio-name> <radio-id>
  nrk-spotify -h | --help

Options:
  -h --help                     Show help
  -f --token-file=<token-file>  Token file to use [default: .token.json]
  -l --listen=<address>         Auth server listening address [default: :8080]
  -i --interval=<minutes>       Polling interval [default: 5]`

	arguments, _ := docopt.Parse(usage, nil, true, "", false)
	args, err := makeArgs(arguments)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if args.Auth {
		spotifyAuth := SpotifyAuth{
			ClientId:     args.ClientId,
			ClientSecret: args.ClientSecret,
			CallbackUrl:  args.CallbackUrl(),
			TokenFile:    args.TokenFile,
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
			Id:   args.RadioId,
		}
		server := SyncServer{
			Spotify:  token,
			Radio:    &radio,
			Interval: args.Interval,
		}
		server.Serve()
	}
}
