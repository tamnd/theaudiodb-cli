// Package theaudiodb is the library behind the theaudiodb command line:
// the HTTP client, request shaping, and the typed data models for TheAudioDB API.
//
// The Client here is the spine every command shares. It sets a real
// User-Agent, paces requests so a busy session stays polite, and retries the
// transient failures (429 and 5xx) that any public API throws under load.
package theaudiodb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Host is the site this client talks to.
const Host = "www.theaudiodb.com"

// Config holds all tuneable client settings.
type Config struct {
	BaseURL   string
	APIKey    string
	UserAgent string
	Rate      time.Duration
	Timeout   time.Duration
	Retries   int
}

// DefaultConfig returns sensible defaults for the TheAudioDB free API.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://www.theaudiodb.com",
		APIKey:    "2",
		UserAgent: "theaudiodb-cli/0.1 (tamnd87@gmail.com)",
		Rate:      500 * time.Millisecond,
		Timeout:   15 * time.Second,
		Retries:   3,
	}
}

// Artist is a music artist record from TheAudioDB.
type Artist struct {
	ID         string `json:"id"          kit:"id"`
	Name       string `json:"name"`
	Genre      string `json:"genre"`
	Country    string `json:"country"`
	FormedYear string `json:"formed_year"`
	Biography  string `json:"biography"`
}

// Album is a music album record from TheAudioDB.
type Album struct {
	ID     string `json:"id"     kit:"id"`
	Name   string `json:"name"`
	Artist string `json:"artist"`
	Year   string `json:"year"`
	Genre  string `json:"genre"`
	Style  string `json:"style"`
	Score  string `json:"score"`
}

// Track is a music track record from TheAudioDB.
type Track struct {
	ID          string `json:"id"           kit:"id"`
	Name        string `json:"name"`
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	Duration    string `json:"duration"`
	TrackNumber string `json:"track_number"`
	Genre       string `json:"genre"`
}

// Client talks to TheAudioDB API over HTTP.
type Client struct {
	HTTP    *http.Client
	cfg     Config
	mu      sync.Mutex
	lastReq time.Time
}

// NewClient returns a Client with DefaultConfig settings.
func NewClient() *Client {
	return NewClientWithConfig(DefaultConfig())
}

// NewClientWithConfig returns a Client configured with the given Config.
func NewClientWithConfig(cfg Config) *Client {
	return &Client{
		HTTP: &http.Client{Timeout: cfg.Timeout},
		cfg:  cfg,
	}
}

// apiURL builds the full URL for a given endpoint and query params.
func (c *Client) apiURL(endpoint string, params url.Values) string {
	return fmt.Sprintf("%s/api/v1/json/%s/%s?%s", c.cfg.BaseURL, c.cfg.APIKey, endpoint, params.Encode())
}

// SearchArtist searches for artists by name.
func (c *Client) SearchArtist(ctx context.Context, name string) ([]*Artist, error) {
	params := url.Values{"s": {name}}
	body, err := c.Get(ctx, c.apiURL("search.php", params))
	if err != nil {
		return nil, err
	}
	var resp struct {
		Artists []struct {
			IDArtist       string `json:"idArtist"`
			StrArtist      string `json:"strArtist"`
			StrGenre       string `json:"strGenre"`
			StrCountry     string `json:"strCountry"`
			IntFormedYear  string `json:"intFormedYear"`
			StrBiographyEN string `json:"strBiographyEN"`
		} `json:"artists"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode artists: %w", err)
	}
	if resp.Artists == nil {
		return nil, nil
	}
	out := make([]*Artist, len(resp.Artists))
	for i, a := range resp.Artists {
		bio := a.StrBiographyEN
		if len(bio) > 200 {
			bio = bio[:200]
		}
		out[i] = &Artist{
			ID:         a.IDArtist,
			Name:       a.StrArtist,
			Genre:      a.StrGenre,
			Country:    a.StrCountry,
			FormedYear: a.IntFormedYear,
			Biography:  bio,
		}
	}
	return out, nil
}

// Discography returns all albums for a given artist name.
func (c *Client) Discography(ctx context.Context, artistName string) ([]*Album, error) {
	params := url.Values{"s": {artistName}}
	body, err := c.Get(ctx, c.apiURL("discography.php", params))
	if err != nil {
		return nil, err
	}
	var resp struct {
		Album []struct {
			IDAlbum         string `json:"idAlbum"`
			StrAlbum        string `json:"strAlbum"`
			StrArtist       string `json:"strArtist"`
			IntYearReleased string `json:"intYearReleased"`
			StrGenre        string `json:"strGenre"`
			StrStyle        string `json:"strStyle"`
			IntScore        string `json:"intScore"`
		} `json:"album"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode discography: %w", err)
	}
	if resp.Album == nil {
		return nil, nil
	}
	out := make([]*Album, len(resp.Album))
	for i, a := range resp.Album {
		out[i] = &Album{
			ID:     a.IDAlbum,
			Name:   a.StrAlbum,
			Artist: a.StrArtist,
			Year:   a.IntYearReleased,
			Genre:  a.StrGenre,
			Style:  a.StrStyle,
			Score:  a.IntScore,
		}
	}
	return out, nil
}

// SearchAlbum searches for albums by artist name and album name.
func (c *Client) SearchAlbum(ctx context.Context, artistName, albumName string) ([]*Album, error) {
	params := url.Values{"s": {artistName}, "a": {albumName}}
	body, err := c.Get(ctx, c.apiURL("searchalbum.php", params))
	if err != nil {
		return nil, err
	}
	var resp struct {
		Album []struct {
			IDAlbum         string `json:"idAlbum"`
			StrAlbum        string `json:"strAlbum"`
			StrArtist       string `json:"strArtist"`
			IntYearReleased string `json:"intYearReleased"`
			StrGenre        string `json:"strGenre"`
			StrStyle        string `json:"strStyle"`
			IntScore        string `json:"intScore"`
		} `json:"album"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode album: %w", err)
	}
	if resp.Album == nil {
		return nil, nil
	}
	out := make([]*Album, len(resp.Album))
	for i, a := range resp.Album {
		out[i] = &Album{
			ID:     a.IDAlbum,
			Name:   a.StrAlbum,
			Artist: a.StrArtist,
			Year:   a.IntYearReleased,
			Genre:  a.StrGenre,
			Style:  a.StrStyle,
			Score:  a.IntScore,
		}
	}
	return out, nil
}

// SearchTrack searches for tracks by artist name and track name.
func (c *Client) SearchTrack(ctx context.Context, artistName, trackName string) ([]*Track, error) {
	params := url.Values{"s": {artistName}, "t": {trackName}}
	body, err := c.Get(ctx, c.apiURL("searchtrack.php", params))
	if err != nil {
		return nil, err
	}
	var resp struct {
		Track []struct {
			IDTrack        string `json:"idTrack"`
			StrTrack       string `json:"strTrack"`
			StrArtist      string `json:"strArtist"`
			StrAlbum       string `json:"strAlbum"`
			IntDuration    string `json:"intDuration"`
			IntTrackNumber string `json:"intTrackNumber"`
			StrGenre       string `json:"strGenre"`
		} `json:"track"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode track: %w", err)
	}
	if resp.Track == nil {
		return nil, nil
	}
	out := make([]*Track, len(resp.Track))
	for i, t := range resp.Track {
		out[i] = &Track{
			ID:          t.IDTrack,
			Name:        t.StrTrack,
			Artist:      t.StrArtist,
			Album:       t.StrAlbum,
			Duration:    t.IntDuration,
			TrackNumber: t.IntTrackNumber,
			Genre:       t.StrGenre,
		}
	}
	return out, nil
}

// Get fetches rawURL and returns the response body. It paces and retries
// according to the client's settings.
func (c *Client) Get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) (body []byte, retry bool, err error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

// pace blocks until at least Rate has passed since the previous request.
func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.lastReq); wait > 0 {
		time.Sleep(wait)
	}
	c.lastReq = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}
