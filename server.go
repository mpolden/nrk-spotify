package main

import (
	"log"
	"time"
)

type SyncServer struct {
	Spotify  *Spotify
	Playlist *Playlist
	Radio    *NrkRadio
}

func (sync *SyncServer) Serve() {
	ticker := time.NewTicker(3 * time.Minute)
	quit := make(chan struct{})

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
		case <-quit:
			log.Print("Shutting down")
			ticker.Stop()
			return
		}
	}
}

func (sync *SyncServer) run() error {
	radioPlaylist, err := sync.Radio.Playlist()
	if err != nil {
		return err
	}

	radioTracks := radioPlaylist.Tracks[1:] // Skip previous track
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
		log.Printf("Added: %+v\n", track)
	}
	return nil
}
