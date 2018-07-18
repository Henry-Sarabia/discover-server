package main

import (
	"errors"

	"github.com/zmb3/spotify"
)

const (
	targetPopularity = 40
	maxPopularity    = 50
	publicPlaylist   = true
)

var (
	errRecentTracksMissing   = errors.New("recentTracks: there are no recent tracks to return")
	errNumRange              = errors.New("input is out of range")
	errTimeMissing           = errors.New("TopArtistsVar: missing time range")
	errTimeInvalid           = errors.New("TopArtists: invalid time range")
	errPlaylistTracksMissing = errors.New("Playlist: missing playlist tracks")
)

// clienter is implemented by any type that has certain Spotify functions.
// Helpful for testing purposes.
type clienter interface {
	CurrentUser() (*spotify.PrivateUser, error)
	CurrentUsersTopArtistsOpt(*spotify.Options) (*spotify.FullArtistPage, error)
	CurrentUsersTopTracksOpt(*spotify.Options) (*spotify.FullTrackPage, error)
	PlayerRecentlyPlayedOpt(*spotify.RecentlyPlayedOptions) ([]spotify.RecentlyPlayedItem, error)
	GetRecommendations(spotify.Seeds, *spotify.TrackAttributes, *spotify.Options) (*spotify.Recommendations, error)
	CreatePlaylistForUser(string, string, bool) (*spotify.FullPlaylist, error)
	AddTracksToPlaylist(string, spotify.ID, ...spotify.ID) (string, error)
	GetArtists(...spotify.ID) ([]*spotify.FullArtist, error)
}

// clientBuffer contains a client used to make Spotify API calls and stores the
// salient details of the response in case of duplicate API calls in subsequent
// function calls.
type clientBuffer struct {
	c               clienter
	user            *spotify.PrivateUser
	topArtistsShort []spotify.FullArtist
	topArtistsMed   []spotify.FullArtist
	topArtistsLong  []spotify.FullArtist
	topTracksShort  []spotify.FullTrack
	topTracksMed    []spotify.FullTrack
	topTracksLong   []spotify.FullTrack
	recentTracks    []spotify.SimpleTrack
	recentArtists   []spotify.FullArtist
	requests        int
}

// User returns the currently logged in Spotify user.
func (cb *clientBuffer) User() (*spotify.PrivateUser, error) {
	if cb.user != nil {
		return cb.user, nil
	}

	u, err := cb.c.CurrentUser()
	cb.requests++
	if err != nil {
		return nil, err
	}

	cb.user = u

	return u, nil
}

// TopArtistsVar returns a concatenated list of the num top artists from each
// provided time range for the current user.
func (cb *clientBuffer) TopArtistsVar(num int, time ...string) ([]spotify.FullArtist, error) {
	if len(time) <= 0 {
		return nil, errTimeMissing
	}

	top := make([]spotify.FullArtist, 0)
	for _, t := range time {
		curr, err := cb.TopArtists(num, t)
		if err != nil {
			return nil, err
		}
		top = append(top, curr...)
	}
	return top, nil
}

// TopArtists returns a list of the num top artists from the provided time
// range for the current user.
func (cb *clientBuffer) TopArtists(num int, time string) ([]spotify.FullArtist, error) {
	if num <= 0 || num > maxRequestArtists {
		return nil, errNumRange
	}

	opt := &spotify.Options{Limit: &num, Timerange: &time}
	switch time {
	case "short":
		if cb.topArtistsShort != nil {
			return cb.topArtistsShort, nil
		}

		top, err := cb.c.CurrentUsersTopArtistsOpt(opt)
		cb.requests++
		if err != nil {
			return nil, err
		}

		cb.topArtistsShort = top.Artists

		return top.Artists, nil

	case "medium":
		if cb.topArtistsMed != nil {
			return cb.topArtistsMed, nil
		}

		top, err := cb.c.CurrentUsersTopArtistsOpt(opt)
		cb.requests++
		if err != nil {
			return nil, err
		}

		cb.topArtistsMed = top.Artists

		return top.Artists, nil

	case "long":
		if cb.topArtistsLong != nil {
			return cb.topArtistsLong, nil
		}

		top, err := cb.c.CurrentUsersTopArtistsOpt(opt)
		cb.requests++
		if err != nil {
			return nil, err
		}

		cb.topArtistsLong = top.Artists

		return top.Artists, nil

	default:
		return nil, errTimeInvalid
	}
}

// TopTracksVar returns a concatenated list of the num top tracks from each
// provided time range for the current user.
func (cb *clientBuffer) TopTracksVar(num int, time ...string) ([]spotify.FullTrack, error) {
	if len(time) <= 0 {
		return nil, errTimeMissing
	}

	top := make([]spotify.FullTrack, 0)
	for _, t := range time {
		curr, err := cb.TopTracks(num, t)
		if err != nil {
			return nil, err
		}
		top = append(top, curr...)
	}

	return top, nil
}

// TopTracks returns a list of the num top tracks from the provided time range
// for the current user.
func (cb *clientBuffer) TopTracks(num int, time string) ([]spotify.FullTrack, error) {
	if num <= 0 || num > maxRequestTracks {
		return nil, errNumRange
	}

	opt := &spotify.Options{Limit: &num, Timerange: &time}
	switch time {
	case "short":
		if cb.topTracksShort != nil {
			return cb.topTracksShort, nil
		}

		top, err := cb.c.CurrentUsersTopTracksOpt(opt)
		cb.requests++
		if err != nil {
			return nil, err
		}

		cb.topTracksShort = top.Tracks
		return top.Tracks, nil
	case "medium":
		if cb.topTracksMed != nil {
			return cb.topTracksMed, nil
		}

		top, err := cb.c.CurrentUsersTopTracksOpt(opt)
		cb.requests++
		if err != nil {
			return nil, err
		}

		cb.topTracksMed = top.Tracks
		return top.Tracks, nil
	case "long":
		if cb.topTracksLong != nil {
			return cb.topTracksLong, nil
		}

		top, err := cb.c.CurrentUsersTopTracksOpt(opt)
		cb.requests++
		if err != nil {
			return nil, err
		}

		cb.topTracksLong = top.Tracks
		return top.Tracks, nil
	default:
		return nil, errTimeInvalid
	}
}

// RecentTracks returns a list of the num most recently played tracks for the
// current user.
func (cb *clientBuffer) RecentTracks(num int) ([]spotify.SimpleTrack, error) {
	if num <= 0 || num > maxRequestTracks {
		return nil, errNumRange
	}

	if cb.recentTracks != nil {
		return cb.recentTracks, nil
	}

	recent, err := cb.c.PlayerRecentlyPlayedOpt(&spotify.RecentlyPlayedOptions{Limit: num})
	cb.requests++
	if err != nil {
		return nil, err
	}
	if len(recent) == 0 {
		return nil, errRecentTracksMissing
	}

	tracks := make([]spotify.SimpleTrack, 0)
	for _, t := range recent {
		tracks = append(tracks, t.Track)
	}

	cb.recentTracks = tracks

	return tracks, nil
}

// RecentArtists returns a list of the most recently played artists for the
// current user.
func (cb *clientBuffer) RecentArtists(num int) ([]spotify.FullArtist, error) {
	if num <= 0 || num > maxRequestArtists {
		return nil, errNumRange
	}

	if cb.recentArtists != nil {
		return cb.recentArtists, nil
	}

	tracks, err := cb.RecentTracks(num)
	cb.requests++
	if err != nil {
		return nil, err
	}

	IDs := extractArtistIDs(tracks)
	artists, err := cb.c.GetArtists(IDs...)
	cb.requests++
	if err != nil {
		return nil, err
	}

	for _, a := range artists {
		cb.recentArtists = append(cb.recentArtists, *a)
	}

	return cb.recentArtists, nil
}

// Recommendation returns a list of num recommended tracks generated using the
// provided seeds for the current user.
func (cb *clientBuffer) Recommendation(sd spotify.Seeds, num int) ([]spotify.SimpleTrack, error) {
	if num <= 0 || num > maxRecommendations {
		return nil, errNumRange
	}

	attr := spotify.NewTrackAttributes().TargetPopularity(targetPopularity).MaxPopularity(maxPopularity)
	opt := &spotify.Options{Limit: &num}
	recs, err := cb.c.GetRecommendations(sd, attr, opt)
	cb.requests++
	if err != nil {
		return nil, err
	}

	return recs.Tracks, nil
}

// Playlist creates and returns a playlist with the provided name and track IDs
// for the current user.
func (cb *clientBuffer) Playlist(name string, IDs []spotify.ID) (*spotify.FullPlaylist, error) {
	if len(IDs) <= 0 {
		return nil, errPlaylistTracksMissing
	}

	u, err := cb.User()
	if err != nil {
		return nil, err
	}

	pl, err := cb.c.CreatePlaylistForUser(u.ID, name, publicPlaylist)
	cb.requests++
	if err != nil {
		return nil, err
	}

	_, err = cb.c.AddTracksToPlaylist(u.ID, pl.ID, IDs...)
	cb.requests++
	if err != nil {
		return nil, err
	}

	return pl, nil
}
