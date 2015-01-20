package server

import (
	"github.com/cenkalti/backoff"
	"github.com/golang/groupcache/lru"
	"github.com/martinp/nrk-spotify/nrk"
	"github.com/martinp/nrk-spotify/spotify"
	"github.com/mitchellh/colorstring"
	"log"
	"os"
	"runtime/pprof"
	"time"
)

var Colorize colorstring.Colorize

func init() {
	Colorize = colorstring.Colorize{
		Colors: colorstring.DefaultColors,
		Reset:  true,
	}
}

type Sync struct {
	Spotify       *spotify.Spotify
	Radio         *nrk.Radio
	Interval      time.Duration
	Adaptive      bool
	CacheSize     int
	DeleteEvicted bool
	playlist      *spotify.Playlist
	cache         *lru.Cache
	MemProfile    string
}

func logColorf(format string, v ...interface{}) {
	log.Printf(Colorize.Color(format), v...)
}

func (sync *Sync) isCached(track *spotify.Track) bool {
	_, exists := sync.cache.Get(track.Id)
	return exists
}

func (sync *Sync) addTrack(track *spotify.Track) {
	if !sync.isCached(track) {
		sync.cache.Add(track.Id, *track)
	}
}

func (sync *Sync) initPlaylist() error {
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
		ticker.Stop()
		break
	}
	if err != nil {
		return err
	}
	sync.playlist = playlist
	return nil
}

func (sync *Sync) deleteEvicted(key lru.Key, value interface{}) {
	track, ok := value.(spotify.Track)
	if !ok {
		log.Print("Could not cast to spotify.Track: %+v", value)
		return
	}
	if err := sync.retryDeleteTrack(&track); err != nil {
		log.Printf("Failed to delete track: %s", err)
		return
	}
	logColorf("[dark_gray]Deleted evicted track: %s[reset]", track.String())
}

func (sync *Sync) initCache() error {
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = time.Duration(5 * time.Minute)
	ticker := backoff.NewTicker(b)
	var tracks []spotify.PlaylistTrack
	var err error
	for _ = range ticker.C {
		tracks, err = sync.Spotify.RecentTracks(sync.playlist,
			sync.playlist.Tracks.Total)
		if err != nil {
			log.Printf("Failed to get recent tracks: %s", err)
			log.Println("Retrying...")
			continue
		}
		ticker.Stop()
		break
	}
	if err != nil {
		return err
	}
	sync.cache = lru.New(sync.CacheSize)
	if sync.DeleteEvicted {
		log.Print("Deleting evicted tracks from playlist")
		sync.cache.OnEvicted = sync.deleteEvicted
	}
	for _, t := range tracks {
		sync.addTrack(&t.Track)
	}
	return nil
}

func (sync *Sync) Serve() {
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

func (sync *Sync) memProfile() error {
	f, err := os.Create(sync.MemProfile)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := pprof.WriteHeapProfile(f); err != nil {
		return err
	}
	return nil
}

func (sync *Sync) runForever() <-chan time.Time {
	duration, err := sync.run()
	if err != nil {
		log.Printf("Sync failed: %s", err)
		duration = sync.Interval
	}
	log.Printf("Next sync in %s", duration)
	if sync.MemProfile != "" {
		log.Printf("Writing memory profile to: %s", sync.MemProfile)
		if err := sync.memProfile(); err != nil {
			log.Print(err)
		}
	}
	return time.After(duration)
}

func (sync *Sync) retryPlaylist() (*nrk.Playlist, error) {
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = time.Duration(1 * time.Minute)
	ticker := backoff.NewTicker(b)
	var playlist *nrk.Playlist
	var err error
	for _ = range ticker.C {
		playlist, err = sync.Radio.Playlist()
		if err != nil {
			log.Printf("Retrieving radio playlist failed: %s", err)
			log.Println("Retrying...")
			continue
		}
		ticker.Stop()
		break
	}
	return playlist, err
}

func (sync *Sync) retrySearch(track *nrk.Track) ([]spotify.Track, error) {
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = time.Duration(1 * time.Minute)
	ticker := backoff.NewTicker(b)
	var tracks []spotify.Track
	var err error
	for _ = range ticker.C {
		tracks, err = sync.Spotify.SearchArtistTrack(
			track.ArtistName(), track.Track)
		if err != nil {
			log.Printf("Search failed: %s", err)
			log.Println("Retrying...")
			continue
		}
		ticker.Stop()
		break
	}
	return tracks, err
}

func (sync *Sync) retryAddTrack(track *spotify.Track) error {
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
		ticker.Stop()
		break
	}
	return err
}

func (sync *Sync) retryDeleteTrack(track *spotify.Track) error {
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = time.Duration(1 * time.Minute)
	ticker := backoff.NewTicker(b)
	var err error
	for _ = range ticker.C {
		err = sync.Spotify.DeleteTrack(sync.playlist, track)
		if err != nil {
			log.Printf("Delete track failed: %s", err)
			log.Println("Retrying...")
			continue
		}
		ticker.Stop()
		break
	}
	return err
}

func (sync *Sync) logCurrentTrack(playlist *nrk.Playlist) {
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
		position.String(), position.Symbol(10, !Colorize.Disable))
}

func (sync *Sync) run() (time.Duration, error) {
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
	added := make([]nrk.Track, 0, len(radioTracks))
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
			added = append(added, t)
			continue
		}
		if err = sync.retryAddTrack(track); err != nil {
			logColorf("[red]Failed to add: %s (%s)[reset]",
				track.String(), err)
			continue
		}
		sync.addTrack(track)
		added = append(added, t)

		logColorf("[green]Added track: %s[reset]", track.String())
	}
	log.Printf("Cache size: %d/%d", sync.cache.Len(),
		sync.cache.MaxEntries)
	if !sync.Adaptive {
		return sync.Interval, nil
	}
	return radioPlaylist.NextSync(added)
}
