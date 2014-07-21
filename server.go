package main

import (
	"github.com/mitchellh/colorstring"
	"log"
	"time"
)

type SyncServer struct {
	Spotify  *Spotify
	Playlist *Playlist
	Radio    *NrkRadio
	Interval time.Duration
}

func (sync *SyncServer) Serve() {
	ticker := time.NewTicker(sync.Interval * time.Minute)

	log.Printf("Server started. Syncing every %d minute(s)", sync.Interval)
	f := func() {
		logColorf("[light_magenta]Running sync[reset]")
		if err := sync.run(); err != nil {
			log.Printf("Sync failed: %s\n", err)
		}
	}
	f()
	for {
		select {
		case <-ticker.C:
			f()
		}
	}
}

func logColorf(format string, v ...interface{}) {
	log.Printf(colorstring.Color(format), v...)
}

func (sync *SyncServer) run() error {
	radioPlaylist, err := sync.Radio.Playlist()
	if err != nil {
		return err
	}

	radioTracks := radioPlaylist.Tracks[1:] // Skip previous track
	if current, err := radioPlaylist.Current(); err != nil {
		logColorf("[red]Failed to get current track: %s[reset]", err)
	} else {
		if meta, err := current.PositionMeta(); err != nil {
			logColorf("[red]Failed to parse metadata: %s[reset]",
				err)
		} else {
			logColorf("[cyan]%s is currently playing: %s - %s"+
				"[reset] [%s] [%s]",
				sync.Radio.Name, current.Artist, current.Track,
				meta.PositionString(),
				meta.PositionSymbol(10, true))
		}
	}

	for _, t := range radioTracks {
		logColorf("Searching for: %s", t.String())
		if !t.IsMusic() {
			logColorf("[green]Not music, skipping: %s[reset]",
				t.String())
			continue
		}
		track, err := sync.Spotify.searchArtistTrack(t.Artist, t.Track)
		if err != nil {
			logColorf("[red]Not found: %s (%s)[reset]",
				t.String(), err)
			continue
		}
		if sync.Playlist.contains(*track) {
			logColorf("[yellow]Already added: %s[reset]",
				track.String())
			continue
		}
		if err = sync.Spotify.addTrack(sync.Playlist,
			track); err != nil {
			logColorf("[red]Failed to add: %s (%s)[reset]",
				track.String(), err)
			continue
		}
		// Refresh playlist
		playlist, err := sync.Spotify.playlistById(sync.Playlist.Id)
		if err != nil {
			logColorf("[red]Failed to refresh playlist: %s[reset]",
				err)
			continue
		}
		sync.Playlist = playlist
		logColorf("[green]Added track: %s[reset]", track.String())
	}
	return nil
}
