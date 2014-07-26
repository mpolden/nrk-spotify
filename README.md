# nrk-spotify

[![Build Status](https://travis-ci.org/martinp/nrk-spotify.png)](https://travis-ci.org/martinp/nrk-spotify)

A server that combines the NRK radio API and Spotify API to create Spotify
playlists based on current radio playlist.

This essentially makes it possible to stream the current radio playlist at a
much higher bitrate.

## Usage

```
$ nrk-spotify -h
Listen to NRK radio channels in Spotify.

Usage:
  nrk-spotify auth [-l <address>] [-f <file>] <client-id> <client-secret>
  nrk-spotify server [-f <file>] [-i <interval>] [-a] [-c <max>] <name> <radio-id>
  nrk-spotify -h | --help

Options:
  -h --help                Show help
  -f --token-file=<file>   Token file to use [default: .token.json]
  -l --listen=<address>    Auth server listening address [default: :8080]
  -i --interval=<minutes>  Polling interval [default: 5]
  -c --cache-size=<max>    Max entries to keep in cache [default: 100]
  -a --adaptive            Automatically determine sync interval
```

## Compiling and installing

To compile you need the [Golang compiler](http://golang.org/doc/install).

Compile:

`$ make deps build`

Install:

`$ PREFIX=/usr/local make install`

## In action

![Screenshot](docs/output.png)

## Tutorial

### Create a Spotify application (required for API access)

1. Create a Spotify application at https://developer.spotify.com/my-applications/
2. Add `http://localhost:8080/callback` as a redirect URL for your app.
3. Make a note of the `Client ID` and `Client Secret` keys.

### Run local auth server to create a access token

```
$ nrk-spotify auth CLIENT_ID CLIENT_SECRET
Visit http://localhost:8080/login to authenticate with Spotify.
```

Use the link to authenticate with Spotify. If the authentication succeeds, a
success message should be shown. After authentication succeeds, the auth server
can be stopped with `Ctrl+C`.

### Run the sync server

Choose a playlist name and a radio channel to sync. The following radio IDs can
be used:

`p1pluss p2 p3 p13 mp3 radio_super klassisk jazz folkemusikk urort
radioresepsjonen national_rap_show pyro`

Start sync server:

```
$ nrk-spotify server 'NRK P3 Pyro' pyro
```

`NRK P3 Pyro` will become the playlist name in Spotify, which will be
created if it doesn't exist.

The playlist will be updated with new songs every 5 minutes.

## License
Licensed under the MIT license.
