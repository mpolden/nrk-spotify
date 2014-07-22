package main

import (
	"github.com/golang/groupcache/lru"
	"github.com/mitchellh/colorstring"
	"log"
	"time"
)

type SyncServer struct {
	Spotify  *Spotify
	Radio    *Radio
	Interval time.Duration
	playlist *Playlist
	cache    *lru.Cache
}

func logColorf(format string, v ...interface{}) {
	log.Printf(colorstring.Color(format), v...)
}

func (sync *SyncServer) isCached(track *Track) bool {
	_, exists := sync.cache.Get(track.Id)
	return exists
}

func (sync *SyncServer) addTrack(track *Track) {
	if !sync.isCached(track) {
		sync.cache.Add(track.Id, track)
	}
}

func (sync *SyncServer) initPlaylist() error {
	playlist, err := sync.Spotify.GetOrCreatePlaylist(sync.Radio.Name)
	if err != nil {
		return err
	}
	sync.playlist = playlist
	return nil
}

func (sync *SyncServer) initCache() {
	sync.cache = lru.New(100)
	for _, t := range sync.playlist.Tracks.Items {
		sync.addTrack(&t.Track)
	}
}

func (sync *SyncServer) Serve() {
	ticker := time.NewTicker(sync.Interval * time.Minute)

	logColorf("Server started. Syncing every [bold]%d[reset] minute(s)",
		sync.Interval)

	if err := sync.initPlaylist(); err != nil {
		log.Fatalf("Failed to get or create playlist: %s", err)
	}
	log.Printf("Playlist has %d entries", len(sync.playlist.Tracks.Items))
	sync.initCache()

	logColorf("[green]Tracks will be added to Spotify playlist: %s[reset]",
		sync.playlist.String())
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

func (sync *SyncServer) run() error {
	radioPlaylist, err := sync.Radio.Playlist()
	if err != nil {
		return err
	}

	radioTracks := radioPlaylist.Tracks[1:] // Skip previous track
	if current, err := radioPlaylist.Current(); err != nil {
		logColorf("[red]Failed to get current track: %s[reset]", err)
	} else {
		if position, err := current.Position(); err != nil {
			logColorf("[red]Failed to parse metadata: %s[reset]",
				err)
		} else {
			logColorf("[cyan]%s is currently playing: %s - %s"+
				"[reset] (%s) [%s]",
				sync.Radio.Name, current.Artist, current.Track,
				position.String(),
				position.Symbol(10, true))
		}
	}

	for _, t := range radioTracks {
		logColorf("Searching for: %s", t.String())
		if !t.IsMusic() {
			logColorf("[green]Not music, skipping: %s[reset]",
				t.String())
			continue
		}
		track, err := sync.Spotify.SearchArtistTrack(t.Artist, t.Track)
		if err != nil {
			logColorf("[red]Not found: %s (%s)[reset]",
				t.String(), err)
			continue
		}
		if sync.isCached(track) {
			logColorf("[yellow]Already added: %s[reset]",
				track.String())
			continue
		}
		if err = sync.Spotify.AddTrack(sync.playlist,
			track); err != nil {
			logColorf("[red]Failed to add: %s (%s)[reset]",
				track.String(), err)
			continue
		}
		sync.addTrack(track)
		logColorf("[green]Added track: %s (cache size: %d)[reset]",
			track.String(), sync.cache.Len())
	}
	return nil
}
