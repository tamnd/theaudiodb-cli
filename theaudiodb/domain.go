package theaudiodb

import (
	"context"
	"strings"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes theaudiodb as a kit Domain: a driver that a multi-domain
// host (ant) enables with a single blank import,
//
//	import _ "github.com/tamnd/theaudiodb-cli/theaudiodb"
//
// exactly as a database/sql program enables a driver with `import _
// "github.com/lib/pq"`. The init below registers it; the host then dereferences
// theaudiodb:// URIs by routing to the operations Register installs. The same
// Domain also builds the standalone theaudiodb binary (see cli.NewApp), so the
// binary and a host share one source of truth.
func init() { kit.Register(Domain{}) }

// Domain is the theaudiodb driver. It carries no state; the per-run client is
// built by the factory Register hands kit.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against, and
// the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "theaudiodb",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "theaudiodb",
			Short:  "A command line for TheAudioDB music data.",
			Long: `A command line for TheAudioDB music data.

theaudiodb reads public TheAudioDB data over plain HTTPS, shapes it into
clean records, and prints output that pipes into the rest of your tools. No paid
API key needed.`,
			Site: Host,
			Repo: "https://github.com/tamnd/theaudiodb-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	// artist: search for an artist by name
	kit.Handle(app, kit.OpMeta{Name: "artist", Group: "read", Single: true,
		Summary: "Search for an artist by name",
		Args:    []kit.Arg{{Name: "name", Help: "artist name"}}}, searchArtist)

	// album: search for an album by artist + album name
	kit.Handle(app, kit.OpMeta{Name: "album", Group: "read", Single: true,
		Summary: "Search for an album by artist and album name",
		Args: []kit.Arg{
			{Name: "artist", Help: "artist name"},
			{Name: "album", Help: "album name"},
		}}, searchAlbum)

	// track: search for a track by artist + track name
	kit.Handle(app, kit.OpMeta{Name: "track", Group: "read", Single: true,
		Summary: "Search for a track by artist and track name",
		Args: []kit.Arg{
			{Name: "artist", Help: "artist name"},
			{Name: "track", Help: "track name"},
		}}, searchTrack)

	// discography: list all albums by an artist
	kit.Handle(app, kit.OpMeta{Name: "discography", Group: "read", List: true,
		Summary: "List all albums by an artist",
		Args:    []kit.Arg{{Name: "artist", Help: "artist name"}}}, discography)
}

// newClient builds the client from the host-resolved config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	dcfg := DefaultConfig()
	if cfg.UserAgent != "" {
		dcfg.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		dcfg.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		dcfg.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		dcfg.Timeout = cfg.Timeout
	}
	return NewClientWithConfig(dcfg), nil
}

// --- inputs ---

type artistIn struct {
	Name   string  `kit:"arg" help:"artist name"`
	Client *Client `kit:"inject"`
}

type albumIn struct {
	Artist string  `kit:"arg" help:"artist name"`
	Album  string  `kit:"arg" help:"album name"`
	Client *Client `kit:"inject"`
}

type trackIn struct {
	Artist string  `kit:"arg" help:"artist name"`
	Track  string  `kit:"arg" help:"track name"`
	Client *Client `kit:"inject"`
}

type discographyIn struct {
	Artist string  `kit:"arg" help:"artist name"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

// --- handlers ---

func searchArtist(ctx context.Context, in artistIn, emit func(*Artist) error) error {
	artists, err := in.Client.SearchArtist(ctx, in.Name)
	if err != nil {
		return mapErr(err)
	}
	for _, a := range artists {
		if err := emit(a); err != nil {
			return err
		}
	}
	return nil
}

func searchAlbum(ctx context.Context, in albumIn, emit func(*Album) error) error {
	albums, err := in.Client.SearchAlbum(ctx, in.Artist, in.Album)
	if err != nil {
		return mapErr(err)
	}
	for _, a := range albums {
		if err := emit(a); err != nil {
			return err
		}
	}
	return nil
}

func searchTrack(ctx context.Context, in trackIn, emit func(*Track) error) error {
	tracks, err := in.Client.SearchTrack(ctx, in.Artist, in.Track)
	if err != nil {
		return mapErr(err)
	}
	for _, t := range tracks {
		if err := emit(t); err != nil {
			return err
		}
	}
	return nil
}

func discography(ctx context.Context, in discographyIn, emit func(*Album) error) error {
	albums, err := in.Client.Discography(ctx, in.Artist)
	if err != nil {
		return mapErr(err)
	}
	for i, a := range albums {
		if in.Limit > 0 && i >= in.Limit {
			break
		}
		if err := emit(a); err != nil {
			return err
		}
	}
	return nil
}

// --- Resolver: URI driver string functions ---

// Classify turns any accepted input into the canonical (type, id).
func (Domain) Classify(input string) (uriType, id string, err error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", "", errs.Usage("empty theaudiodb reference")
	}
	return "artist", input, nil
}

// Locate is the inverse: the live https URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	switch uriType {
	case "artist":
		return "https://" + Host + "/artist/" + id, nil
	case "album":
		return "https://" + Host + "/album/" + id, nil
	case "track":
		return "https://" + Host + "/track/" + id, nil
	default:
		return "", errs.Usage("theaudiodb has no resource type %q", uriType)
	}
}

// mapErr converts a library error into the kit error kind that carries the right
// exit code.
func mapErr(err error) error {
	return err
}
