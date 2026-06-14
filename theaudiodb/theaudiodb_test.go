package theaudiodb_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tamnd/theaudiodb-cli/theaudiodb"
)

func newTestClient(baseURL string) *theaudiodb.Client {
	cfg := theaudiodb.DefaultConfig()
	cfg.BaseURL = baseURL
	cfg.Rate = 0
	return theaudiodb.NewClientWithConfig(cfg)
}

func TestSearchArtist(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/json/2/search.php" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("s") != "nirvana" {
			t.Errorf("unexpected s param: %s", r.URL.Query().Get("s"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"artists":[{"idArtist":"111378","strArtist":"Nirvana","strGenre":"Rock","strCountry":"United States","intFormedYear":"1987","strBiographyEN":"Nirvana was an American rock band."}]}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	artists, err := c.SearchArtist(context.Background(), "nirvana")
	if err != nil {
		t.Fatal(err)
	}
	if len(artists) != 1 {
		t.Fatalf("want 1 artist, got %d", len(artists))
	}
	a := artists[0]
	if a.ID != "111378" {
		t.Errorf("ID = %q, want 111378", a.ID)
	}
	if a.Name != "Nirvana" {
		t.Errorf("Name = %q, want Nirvana", a.Name)
	}
	if a.Genre != "Rock" {
		t.Errorf("Genre = %q, want Rock", a.Genre)
	}
	if a.Country != "United States" {
		t.Errorf("Country = %q, want United States", a.Country)
	}
	if a.FormedYear != "1987" {
		t.Errorf("FormedYear = %q, want 1987", a.FormedYear)
	}
}

func TestSearchArtistBioTruncated(t *testing.T) {
	longBio := string(make([]byte, 300))
	for i := range longBio {
		longBio = longBio[:i] + "x" + longBio[i+1:]
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"artists":[{"idArtist":"1","strArtist":"Test","strBiographyEN":"` + longBio + `"}]}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	artists, err := c.SearchArtist(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(artists) != 1 {
		t.Fatalf("want 1 artist, got %d", len(artists))
	}
	if len(artists[0].Biography) > 200 {
		t.Errorf("Biography length = %d, want <= 200", len(artists[0].Biography))
	}
}

func TestSearchArtistNullResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"artists":null}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	artists, err := c.SearchArtist(context.Background(), "unknownxyzartist")
	if err != nil {
		t.Fatal(err)
	}
	if artists != nil {
		t.Errorf("expected nil for no results, got %v", artists)
	}
}

func TestDiscography(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/json/2/discography.php" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"album":[{"idAlbum":"2110","strAlbum":"Nevermind","strArtist":"Nirvana","intYearReleased":"1991","strGenre":"Grunge","strStyle":"Alternative Rock","intScore":"8.2"},{"idAlbum":"2111","strAlbum":"Bleach","strArtist":"Nirvana","intYearReleased":"1989","strGenre":"Grunge","strStyle":"Alternative Rock","intScore":"7.5"}]}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	albums, err := c.Discography(context.Background(), "nirvana")
	if err != nil {
		t.Fatal(err)
	}
	if len(albums) != 2 {
		t.Fatalf("want 2 albums, got %d", len(albums))
	}
	if albums[0].Name != "Nevermind" {
		t.Errorf("albums[0].Name = %q, want Nevermind", albums[0].Name)
	}
	if albums[0].Year != "1991" {
		t.Errorf("albums[0].Year = %q, want 1991", albums[0].Year)
	}
	if albums[0].Artist != "Nirvana" {
		t.Errorf("albums[0].Artist = %q, want Nirvana", albums[0].Artist)
	}
	if albums[0].Style != "Alternative Rock" {
		t.Errorf("albums[0].Style = %q, want Alternative Rock", albums[0].Style)
	}
}

func TestSearchAlbum(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/json/2/searchalbum.php" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("s") != "nirvana" || q.Get("a") != "nevermind" {
			t.Errorf("unexpected params s=%q a=%q", q.Get("s"), q.Get("a"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"album":[{"idAlbum":"2110","strAlbum":"Nevermind","strArtist":"Nirvana","intYearReleased":"1991","strGenre":"Grunge","strStyle":"Alternative Rock","intScore":"8.2"}]}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	albums, err := c.SearchAlbum(context.Background(), "nirvana", "nevermind")
	if err != nil {
		t.Fatal(err)
	}
	if len(albums) != 1 {
		t.Fatalf("want 1 album, got %d", len(albums))
	}
	if albums[0].Score != "8.2" {
		t.Errorf("Score = %q, want 8.2", albums[0].Score)
	}
	if albums[0].Genre != "Grunge" {
		t.Errorf("Genre = %q, want Grunge", albums[0].Genre)
	}
}

func TestSearchTrack(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/json/2/searchtrack.php" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"track":[{"idTrack":"32656061","strTrack":"Smells Like Teen Spirit","strArtist":"Nirvana","strAlbum":"Nevermind","intDuration":"278413","intTrackNumber":"1","strGenre":"Rock"}]}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	tracks, err := c.SearchTrack(context.Background(), "nirvana", "smells like teen spirit")
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 1 {
		t.Fatalf("want 1 track, got %d", len(tracks))
	}
	tr := tracks[0]
	if tr.Name != "Smells Like Teen Spirit" {
		t.Errorf("Name = %q, want Smells Like Teen Spirit", tr.Name)
	}
	if tr.Album != "Nevermind" {
		t.Errorf("Album = %q, want Nevermind", tr.Album)
	}
	if tr.Duration != "278413" {
		t.Errorf("Duration = %q, want 278413", tr.Duration)
	}
	if tr.TrackNumber != "1" {
		t.Errorf("TrackNumber = %q, want 1", tr.TrackNumber)
	}
	if tr.Artist != "Nirvana" {
		t.Errorf("Artist = %q, want Nirvana", tr.Artist)
	}
}

func TestGetRetriesOn503(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"artists":[{"idArtist":"1","strArtist":"Test"}]}`))
	}))
	defer srv.Close()

	cfg := theaudiodb.DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	cfg.Retries = 5
	c := theaudiodb.NewClientWithConfig(cfg)

	artists, err := c.SearchArtist(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(artists) != 1 {
		t.Fatalf("want 1 artist after retry, got %d", len(artists))
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
}
