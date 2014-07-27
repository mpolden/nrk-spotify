package main

import (
	"github.com/cenkalti/backoff"
	"github.com/golang/groupcache/lru"
	"github.com/mitchellh/colorstring"
	"github.com/martinp/nrk-spotify/spotify"
	"github.com/martinp/nrk-spotify/nrk"
	"log"
	"time"
)

type SyncServer struct {
	Spotify   *spotify.Spotify
	Radio     *nrk.Radio
	Interval  time.Duration
	Adaptive  bool
	CacheSize int
	playlist  *spotify.Playlist
	cache     *lru.Cache
}

func logColorf(format string, v ...interface{}) {
	log.Printf(colorstring.Color(format), v...)
}

func (sync *SyncServer) isCached(track *spotify.Track) bool {
	_, exists := sync.cache.Get(track.Id)
	return exists
}

func (sync *SyncServer) addTrack(track *spotify.Track) {
	if !sync.isCached(track) {
		sync.cache.Add(track.Id, track)
	}
}

func (sync *SyncServer) initPlaylist() error {
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = time.Duration(5 * time.Minute)
	ticker := backoff.NewTicker(b)
	var playlist *spotify.Playlist
	var err error
	for _ = range ticker.C {
		playlist, err = sync.Spotify.GetOrCreatePlaylist(
			sync.Radio.Name)
		if err != nil {
			log.Printf("Failed to get playlist: %s", err)
			log.Println("Retrying...")
			continue
		}
		break
	}
	if err != nil {
		return err
	}
	sync.playlist = playlist
	return nil
}

func (sync *SyncServer) initCache() error {
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = time.Duration(5 * time.Minute)
	ticker := backoff.NewTicker(b)
	var tracks []spotify.PlaylistTrack
	var err error
	for _ = range ticker.C {
		tracks, err = sync.Spotify.RecentTracks(sync.playlist)
		if err != nil {
			log.Printf("Failed to get recent tracks: %s", err)
			log.Println("Retrying...")
			continue
		}
		break
	}
	if err != nil {
		return err
	}
	sync.cache = lru.New(sync.CacheSize)
	for _, t := range tracks {
		sync.addTrack(&t.Track)
	}
	return nil
}

func (sync *SyncServer) Serve() {
	log.Printf("Server started")

	log.Print("Initializing Spotify playlist")
	if err := sync.initPlaylist(); err != nil {
		log.Fatalf("Failed to initialize playlist: %s", err)
	}
	log.Printf("Playlist: %s", sync.playlist.String())

	log.Print("Initializing cache")
	if err := sync.initCache(); err != nil {
		log.Fatalf("Failed to init cache: %s", err)
	}
	log.Printf("Size: %d/%d", sync.cache.Len(), sync.cache.MaxEntries)

	if sync.Adaptive {
		log.Print("Using adaptive interval")
	} else {
		log.Printf("Syncing every %s", sync.Interval)
	}
	for {
		select {
		case <-sync.runForever():
		}
	}
}

func (sync *SyncServer) runForever() <-chan time.Time {
	duration, err := sync.run()
	if err != nil {
		log.Printf("Sync failed: %s", err)
		duration = sync.Interval
	}
	log.Printf("Next sync in %s", duration)
	return time.After(duration)
}

func (sync *SyncServer) retryPlaylist() (*nrk.RadioPlaylist, error) {
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = time.Duration(1 * time.Minute)
	ticker := backoff.NewTicker(b)
	var playlist *nrk.RadioPlaylist
	var err error
	for _ = range ticker.C {
		playlist, err = sync.Radio.Playlist()
		if err != nil {
			log.Printf("Retrieving radio playlist failed: %s", err)
			log.Println("Retrying...")
			continue
		}
		break
	}
	return playlist, err
}

func (sync *SyncServer) retrySearch(track *nrk.RadioTrack) ([]spotify.Track, error) {
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = time.Duration(1 * time.Minute)
	ticker := backoff.NewTicker(b)
	var tracks []spotify.Track
	var err error
	for _ = range ticker.C {
		tracks, err = sync.Spotify.SearchArtistTrack(track.Artist,
			track.Track)
		if err != nil {
			log.Printf("Search failed: %s", err)
			log.Println("Retrying...")
			continue
		}
		break
	}
	return tracks, err
}

func (sync *SyncServer) retryAddTrack(track *spotify.Track) error {
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = time.Duration(1 * time.Minute)
	ticker := backoff.NewTicker(b)
	var err error
	for _ = range ticker.C {
		err = sync.Spotify.AddTrack(sync.playlist, track)
		if err != nil {
			log.Printf("Add track failed: %s", err)
			log.Println("Retrying...")
			continue
		}
		break
	}
	return err
}

func (sync *SyncServer) logCurrentTrack(playlist *nrk.RadioPlaylist) {
	current, err := playlist.Current()
	if err != nil {
		logColorf("[red]Failed to get current track: %s[reset]", err)
		return
	}
	position, err := current.Position()
	if err != nil {
		logColorf("[red]Failed to parse metadata: %s[reset]", err)
		return
	}
	logColorf("[cyan]%s is currently playing: %s - %s[reset] (%s) [%s]",
		sync.Radio.Name, current.Artist, current.Track,
		position.String(), position.Symbol(10, true))
}

func (sync *SyncServer) run() (time.Duration, error) {
	logColorf("[light_magenta]Running sync[reset]")

	radioPlaylist, err := sync.retryPlaylist()
	if err != nil {
		return time.Duration(0), err
	}
	sync.logCurrentTrack(radioPlaylist)

	radioTracks, err := radioPlaylist.CurrentAndNext()
	if err != nil {
		return time.Duration(0), err
	}
	added := 0
	for _, t := range radioTracks {
		logColorf("Searching for: %s", t.String())
		if !t.IsMusic() {
			logColorf("[yellow]Not music, skipping: %s[reset]",
				t.String())
			continue
		}
		tracks, err := sync.retrySearch(&t)
		if err != nil {
			logColorf("[red]Search failed: %s (%s)[reset]",
				t.String(), err)
			continue
		}
		if len(tracks) == 0 {
			logColorf("[yellow]Track not found: %s[reset]",
				t.String())
			continue
		}
		track := &tracks[0]
		if sync.isCached(track) {
			logColorf("[yellow]Already added: %s[reset]",
				track.String())
			added++
			continue
		}
		if err = sync.retryAddTrack(track); err != nil {
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
