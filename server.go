package main

import (
	"github.com/golang/groupcache/lru"
	"github.com/mitchellh/colorstring"
	"log"
	"time"
)

type SyncServer struct {
	Spotify   *Spotify
	Radio     *Radio
	Interval  time.Duration
	Adaptive  bool
	CacheSize int
	playlist  *Playlist
	cache     *lru.Cache
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
	sync.cache = lru.New(sync.CacheSize)
	for _, t := range sync.playlist.Tracks.Items {
		sync.addTrack(&t.Track)
	}
}

func (sync *SyncServer) Serve() {
	log.Printf("Server started")
	if err := sync.initPlaylist(); err != nil {
		log.Fatalf("Failed to get or create playlist: %s", err)
	}
	log.Printf("Spotify playlist: %s", sync.playlist.String())

	sync.initCache()
	log.Printf("LRU cache initialized: %d/%d", sync.cache.Len(),
		sync.cache.MaxEntries)

	if sync.Adaptive {
		log.Print("Automatically determining interval")
	} else {
		log.Printf("Syncing every %s", sync.Interval)
	}
	for {
		select {
		case <-sync.scheduleRun():
		}
	}
}

func (sync *SyncServer) scheduleRun() <-chan time.Time {
	duration, err := sync.run()
	if err != nil {
		log.Printf("Sync failed: %s\n", err)
		log.Print("Retrying in 1 minute")
		return time.After(1 * time.Minute)
	}
	log.Printf("Next sync in %s", duration)
	return time.After(duration)
}

func (sync *SyncServer) run() (time.Duration, error) {
	logColorf("[light_magenta]Running sync[reset]")

	radioPlaylist, err := sync.Radio.Playlist()
	if err != nil {
		return time.Duration(0), err
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

	added := 0
	for _, t := range radioTracks {
		logColorf("Searching for: %s", t.String())
		if !t.IsMusic() {
			logColorf("[green]Not music, skipping: %s[reset]",
				t.String())
			continue
		}
		tracks, err := sync.Spotify.SearchArtistTrack(t.Artist, t.Track)
		if err != nil {
			logColorf("[red]Search failed: %s (%s)[reset]",
				t.String(), err)
			continue
		}
		if len(tracks) == 0 {
			logColorf("[yellow]Track not found: %s (%s)[reset]",
				t.String(), err)
			continue
		}
		track := &tracks[0]
		if sync.isCached(track) {
			logColorf("[yellow]Already added: %s[reset]",
				track.String())
			added++
			continue
		}
		if err = sync.Spotify.AddTrack(sync.playlist,
			track); err != nil {
			logColorf("[red]Failed to add: %s (%s)[reset]",
				track.String(), err)
			continue
		}
		sync.addTrack(track)
		added++
		logColorf("[green]Added track: %s[reset]", track.String())
	}
	log.Printf("Cache size: %d/%d", sync.cache.Len(),
		sync.cache.MaxEntries)
	if !sync.Adaptive {
		return sync.Interval, nil
	}
	if added != len(radioTracks) {
		log.Printf("%d/%d tracks were added. Falling back to "+
			" regular interval", added, len(radioTracks))
		return sync.Interval, nil
	}
	return radioPlaylist.NextSync()
}
