package main

import (
	"errors"

	"github.com/zmb3/spotify"
)

const (
	minArtistData = 15
	maxSeedInput  = 5
)

var (
	errInsufficientArtistGenres = errors.New("insufficient artist data for a genre map")
)

type genre struct {
	name    string
	artists map[string]*spotify.FullArtist
}

// score returns the score of the genre based on the amount of artists in the
// provided genre.
func (g *genre) score() int {
	return len(g.artists)
}

// isTrend returns true if the genre is considered a trend in a user's taste
// profile. A genre is a trend if its score is above 1.
func (g *genre) isTrend() bool {
	if g.score() > 2 {
		return true
	}
	return false
}

// seed returns a seed using the artists of the genre.
func (g *genre) seed() spotify.Seeds {
	sd := spotify.Seeds{}

	for _, a := range g.artists {
		if len(sd.Artists) >= maxSeedInput {
			break
		}
		sd.Artists = append(sd.Artists, a.ID)
	}

	return sd
}

// genreMap contains a list of genre names mapped to their respective genre
// objects.
type genreMap map[string]*genre

// deleteGenre removes the provided genre from the map if it exists.
func (gm genreMap) deleteGenre(g *genre) error {
	if _, ok := gm[g.name]; !ok {
		return errors.New("cannot delete genre as it does not exist in the genreMap")
	}
	delete(gm, g.name)

	return nil
}

// purgeArtist removes all instances of the provided artist from each genre in
// the map.
func (gm genreMap) purgeArtist(fa *spotify.FullArtist) {
	for _, gs := range fa.Genres {
		if _, ok := gm[gs]; ok {
			delete(gm[gs].artists, fa.Name)
		}
	}

	return
}

// popMax returns the genre with the highest score and removes it from the map.
func (gm genreMap) popMax() (*genre, error) {
	if len(gm) < 1 {
		return nil, errors.New("insufficient genres in genreMap")
	}

	max := &genre{
		artists: make(map[string]*spotify.FullArtist),
	}

	for _, g := range gm {
		if g.score() >= max.score() {
			max = g
		}
	}

	gm.deleteGenre(max)

	for _, a := range max.artists {
		gm.purgeArtist(a)
	}

	return max, nil
}

// popMaxTrend returns the trending genre with highest score and removes it
// from the map. If no trending genres exist, popMaxTrend returns nil.
func (gm genreMap) popMaxTrend() *genre {
	if len(gm) < 1 {
		return nil
	}

	g, err := gm.popMax()
	if err != nil {
		return nil
	}

	if !g.isTrend() {
		return nil
	}

	return g
}

// extractGenres returns the list of genres from the provided list of artists
// ordered from highest to lowest score.
func extractGenres(artists []spotify.FullArtist) ([]*genre, error) {
	gm, err := createGenreMap(artists)
	if err != nil {
		return nil, err
	}

	genres, err := processGenres(gm)
	if err != nil {
		return nil, err
	}

	return genres, nil
}

// createGenreMap returns a map of genres extracted from the provided
// artists.
func createGenreMap(artists []spotify.FullArtist) (genreMap, error) {
	if len(artists) < minArtistData {
		return nil, errInsufficientArtistGenres
	}

	gm := make(genreMap)
	for i, a := range artists {
		for _, g := range a.Genres {
			if _, ok := gm[g]; !ok {
				gm[g] = &genre{
					name:    g,
					artists: make(map[string]*spotify.FullArtist),
				}
			}
			gm[g].artists[a.Name] = &artists[i]
		}
	}

	return gm, nil
}

// processGenres returns a slice of trending genres in the order of highest
// score to lowest score.
func processGenres(gm genreMap) ([]*genre, error) {
	gs := make([]*genre, 0)
	for g := gm.popMaxTrend(); g != nil; g = gm.popMaxTrend() {
		gs = append(gs, g)
	}

	return gs, nil
}

// seeds returns a slice of seeds created with the provided genres.
func genreSeeds(gs []*genre) []spotify.Seeds {
	sds := make([]spotify.Seeds, 0, len(gs))
	for _, g := range gs {
		sds = append(sds, g.seed())
	}

	return sds
}

// sumScore returns the sum of all the scores for each of the provided genres.
func sumScore(gs []*genre) int {
	sum := 0
	for _, g := range gs {
		sum += g.score()
	}
	return sum
}
