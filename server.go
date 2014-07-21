package main

import (
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
		log.Print("Running sync")
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

func (sync *SyncServer) run() error {
	radioPlaylist, err := sync.Radio.Playlist()
	if err != nil {
		return err
	}

	radioTracks := radioPlaylist.Tracks[1:] // Skip previous track
	if current, err := radioPlaylist.Current(); err != nil {
		log.Printf("Failed to get currently playing song: %s", err)
	} else {
		if meta, err := current.PositionMeta(); err != nil {
			log.Printf("Failed to get metadata: %s", err)
		} else {
			log.Printf("%s is currently playing: %s - %s [%s] [%s]",
				sync.Radio.Name, current.Artist, current.Track,
				meta.PositionString(),
				meta.PositionSymbol(10, true))
		}
	}
	log.Printf("Looking for %+v", radioTracks)

	for _, t := range radioTracks {
		if !t.IsMusic() {
			log.Printf("Not music, skipping: %+v", t)
			continue
		}
		track, err := sync.Spotify.searchArtistTrack(t.Artist, t.Track)
		if err != nil {
			log.Printf("No result for %+v: %s", t, err)
			continue
		}
		if sync.Playlist.contains(*track) {
			log.Printf("Already added: %+v", track)
			continue
		}
		if err = sync.Spotify.addTrack(sync.Playlist,
			track); err != nil {
			log.Printf("Failed to add %+v: %s", track, err)
			continue
		}
		// Refresh playlist
		playlist, err := sync.Spotify.playlistById(sync.Playlist.Id)
		if err != nil {
			log.Printf("Failed to refresh playlist: %s", err)
			continue
		}
		sync.Playlist = playlist
		log.Printf("Added: %+v\n", track)
	}
	return nil
}
