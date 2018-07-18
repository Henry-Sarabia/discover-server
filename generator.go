package main

import (
	"math"

	"github.com/pkg/errors"

	"github.com/zmb3/spotify"
)

const (
	playlistSize          = 20
	maxRequestTracks      = 50
	maxRequestArtists     = 50
	maxRecommendations    = 100
	requiredRecentTracks  = 30
	requiredRecentArtists = 30
	requiredTopArtists    = 30
	requiredTopTracks     = 30
	playlistName          = "The Discover Now Project" //add date?
)

var (
	errRecentTracksUnavailable   = errors.New("cannot retrieve recent tracks")
	errRecentTracksInsufficient  = errors.New("insufficient recent track data")
	errRecentArtistsInsufficient = errors.New("insufficient recent artists data")
	errTopArtistsInsufficient    = errors.New("insufficient top artist data")
	errCannotCreatePlaylist      = errors.New("cannot create any relevant playlist")
	errTopTracksInsufficient     = errors.New("insufficient top track data")
)

type generator struct {
	c              *clientBuffer
	invalidArtists map[string]bool
}

func newGenerator(nc clienter) *generator {
	return &generator{
		c:              &clientBuffer{c: nc},
		invalidArtists: make(map[string]bool),
	}
}

// BestPlaylist returns the most relevant playlist that can be generated for
// the current user. Recent, varied, and plentiful playback data will result in
// a more relevant playlist.
func (g *generator) BestPlaylist() (*spotify.FullPlaylist, error) {
	if pl, err := g.TrackDiscover(); err == nil {
		return pl, nil
	}

	if pl, err := g.ArtistDiscover(); err == nil {
		return pl, nil
	}

	if pl, err := g.TrackSummary("short"); err == nil {
		return pl, nil
	}

	if pl, err := g.ArtistSummary("short"); err == nil {
		return pl, nil
	}

	if pl, err := g.ArtistSummary("medium"); err == nil {
		return pl, nil
	}

	if pl, err := g.ArtistSummary("long"); err == nil {
		return pl, nil
	}

	return nil, errCannotCreatePlaylist
}

// TrackDiscover returns a playlist based on the analysis of a user's recently
// played tracks.
func (g *generator) TrackDiscover() (*spotify.FullPlaylist, error) {
	rt, err := g.c.RecentTracks(maxRequestTracks)
	if err != nil {
		return nil, errRecentTracksUnavailable
	}
	if len(rt) < requiredRecentTracks {
		return nil, errRecentTracksInsufficient
	}

	recs, err := g.recsByTrack(rt)
	if err != nil {
		return nil, errors.WithMessage(err, "discoverTracks: cannot retrieve recommendations")
	}

	name := playlistName
	IDs := extractTrackIDs(recs)
	pl, err := g.c.Playlist(name, IDs)
	if err != nil {
		return nil, errors.WithMessage(err, "trackDiscover: cannot create playlist")
	}

	return pl, nil
}

// ArtistDiscover creates a playlist based on the analysis of the user's
// recently played artists.
func (g *generator) ArtistDiscover() (*spotify.FullPlaylist, error) {
	artists, err := g.c.RecentArtists(maxRequestArtists)
	if err != nil {
		return nil, err
	}

	if len(artists) < requiredRecentArtists {
		return nil, errRecentArtistsInsufficient
	}

	genres, err := extractGenres(artists)
	if err != nil {
		return nil, err
	}

	recs, err := g.recsByGenre(genres)
	if err != nil {
		return nil, err
	}

	IDs := extractTrackIDs(shuffleTracks(recs))
	pl, err := g.c.Playlist(playlistName, IDs)
	if err != nil {
		return nil, errors.WithMessage(err, "ArtistDiscover: cannot create playlist")
	}

	return pl, nil
}

// TrackSummary returns a playlist based on a summary analysis of the user's
// track playback data for the provided time period. Valid time periods range
// from short, medium, and long.
func (g *generator) TrackSummary(time string) (*spotify.FullPlaylist, error) {
	top, err := g.c.TopTracks(maxRequestTracks, time)
	if err != nil {
		return nil, err
	}

	if len(top) < requiredTopTracks {
		return nil, errTopTracksInsufficient
	}

	recs, err := g.recsByTrack(simpleToFull(top...))
	if err != nil {
		return nil, err
	}

	IDs := extractTrackIDs(shuffleTracks(recs))
	pl, err := g.c.Playlist(playlistName, IDs)
	if err != nil {
		return nil, err
	}

	return pl, nil
}

// ArtistSummary returns a playlist based on a summary analysis of the user's
// artist playback data for the provided time period. Valid time periods range
// from short, medium, and long.
func (g *generator) ArtistSummary(time string) (*spotify.FullPlaylist, error) {
	ta, err := g.c.TopArtists(requiredTopArtists, time)
	if err != nil {
		return nil, err
	}

	if len(ta) < requiredTopArtists {
		return nil, errTopArtistsInsufficient
	}

	genres, err := extractGenres(ta)
	if err != nil {
		return nil, err
	}

	recs, err := g.recsByGenre(genres)
	if err != nil {
		return nil, err
	}

	IDs := extractTrackIDs(shuffleTracks(recs))
	pl, err := g.c.Playlist(playlistName, IDs)
	if err != nil {
		return nil, err
	}

	return pl, nil
}

// discoverTracks returns a list of tracks based on the analysis of the logged
// in user's recently played tracks.
func (g *generator) discoverTracks() ([]spotify.SimpleTrack, error) {
	rt, err := g.c.RecentTracks(maxRequestTracks)
	if err != nil {
		return nil, errRecentTracksUnavailable
	}
	if len(rt) < requiredRecentTracks {
		return nil, errRecentTracksInsufficient
	}

	recs, err := g.recsByTrack(rt)
	if err != nil {
		return nil, errors.WithMessage(err, "discoverTracks: cannot retrieve recommendations")
	}

	return recs, nil
}

// removeDuplicateArtists returns a list with only unique artists.
// func removeDuplicateArtists(old []spotify.FullArtist) []spotify.FullArtist {
// 	visited := make(map[string]bool)
// 	new := make([]spotify.FullArtist, 0)
// 	for _, a := range old {
// 		if ok := visited[a.Name]; !ok {
// 			visited[a.Name] = true
// 			new = append(new, a)
// 		}
// 	}
// 	return new
// }

// recommendations returns a list of num unique recommended tracks based on the
// provided seeds.
func (g *generator) recommendations(num int, sd spotify.Seeds) ([]spotify.SimpleTrack, error) {
	uniq := make([]spotify.SimpleTrack, 0)

	buf := boundedInt(num*3, maxRecommendations)

	recs, err := g.c.Recommendation(sd, buf)
	if err != nil {
		return nil, err
	}

	i := 0
	for _, r := range recs {
		if i >= num {
			break
		}
		if g.isValidArtist(r.Artists[0].Name) {
			uniq = append(uniq, r)
			g.invalidateArtist(r.Artists[0].Name)
			i++
		}
	}

	return uniq, nil
}

// recsByGenre returns a list of playlistSize tracks recommended based on the
// provided genres.
func (g *generator) recsByGenre(gens []*genre) ([]spotify.SimpleTrack, error) {
	sum := sumScore(gens)

	ta, err := g.c.TopArtistsVar(maxRequestArtists, "short", "medium", "long")
	if err != nil {
		return nil, err
	}

	for _, a := range ta {
		g.invalidateArtist(a.Name)
	}

	for _, gen := range gens {
		for n := range gen.artists {
			g.invalidateArtist(n)
		}
	}

	recs := make([]spotify.SimpleTrack, 0)
	for _, gen := range gens {
		ratio := float64(gen.score()) / float64(sum)
		tracks := ratio * playlistSize
		num := math.Ceil(tracks)

		rec, err := g.recommendations(int(num), gen.seed())
		if err != nil {
			return nil, errors.WithMessage(err, "recsByGenre: cannot retrieve a recommendation")
		}
		recs = append(recs, rec...)
	}

	return recs, nil
}

// recsByTrack returns a list of playlistSize tracks recommended based on the
// provided tracks.
func (g *generator) recsByTrack(tracks []spotify.SimpleTrack) ([]spotify.SimpleTrack, error) {
	sds := trackSeeds(shuffleTracks(tracks))

	ta, err := g.c.TopArtistsVar(maxRequestArtists, "short", "medium", "long")
	if err != nil {
		return nil, errors.WithMessage(err, "recsByTrack: cannot retrieve all top artists")
	}

	for _, a := range ta {
		g.invalidateArtist(a.Name)
	}

	recs := make([]spotify.SimpleTrack, 0)
	num := playlistSize / len(sds)
	for _, sd := range sds {
		rec, err := g.recommendations(num, sd)
		if err != nil {
			return nil, errors.WithMessage(err, "recsByTrack: cannot get recommendation with a track seed")
		}
		recs = append(recs, rec...)
	}

	return recs, nil
}

// invalidateArtist adds the provided artist name to the list of invalid
// artists.
func (g *generator) invalidateArtist(a string) {
	g.invalidArtists[a] = true
	return
}

// isValidArtist returns true if the artist is absent from the invalid artist
//  list. Returns false if the artist is present in the invalid artist list.
func (g *generator) isValidArtist(a string) bool {
	ok := g.invalidArtists[a]
	return !ok
}
