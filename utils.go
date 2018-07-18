package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"math/rand"
	"time"

	"github.com/zmb3/spotify"
)

// simpleToFull returns a list of simple tracks corresponding to the provided
// full tracks.
func simpleToFull(full ...spotify.FullTrack) []spotify.SimpleTrack {
	simple := make([]spotify.SimpleTrack, 0)
	for _, f := range full {
		new := spotify.SimpleTrack{
			Artists: f.Artists,
			ID:      f.ID,
			Name:    f.Name,
		}
		simple = append(simple, new)
	}
	return simple
}

// boundedInt returns num unless num is greater than max, in which case max is
// returned.
func boundedInt(num int, max int) int {
	if num > max {
		return max
	}
	return num
}

// shuffleTracks returns returns a shuffled list of the provided tracks.
func shuffleTracks(old []spotify.SimpleTrack) []spotify.SimpleTrack {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	new := make([]spotify.SimpleTrack, len(old))
	p := r.Perm(len(old))
	for i, j := range p {
		new[i] = old[j]
	}
	return new
}

// extractArtistIDs returns a list of IDs corresponding to the artists of the
// provided tracks.
func extractArtistIDs(tracks []spotify.SimpleTrack) []spotify.ID {
	IDs := make([]spotify.ID, 0)
	for _, t := range tracks {
		IDs = append(IDs, t.Artists[0].ID)
	}

	return IDs
}

// extractTrackIDs returns a list of IDs corresponding to the provided tracks.
func extractTrackIDs(tracks []spotify.SimpleTrack) []spotify.ID {
	IDs := make([]spotify.ID, 0)
	for _, t := range tracks {
		IDs = append(IDs, t.ID)
	}

	return IDs
}

// trackSeed returns a seed using up to 5 of the provided tracks.
func trackSeed(tracks []spotify.SimpleTrack) spotify.Seeds {
	sd := spotify.Seeds{}

	for _, t := range tracks {
		if len(sd.Tracks) > maxSeedInput {
			break
		}
		sd.Tracks = append(sd.Tracks, t.ID)
	}

	return sd
}

// trackSeeds returns a list of seeds using up to 5 of the provided tracks per
// seed.
func trackSeeds(tracks []spotify.SimpleTrack) []spotify.Seeds {
	sds := make([]spotify.Seeds, 0)

	i := 0
	for i <= len(tracks)-maxSeedInput {
		sds = append(sds, trackSeed(tracks[i:i+maxSeedInput]))
		i += maxSeedInput
	}
	if i < len(tracks) {
		sds = append(sds, trackSeed(tracks[i:len(tracks)]))
	}

	return sds
}

// concatBuf concatenates two strings inside a byte buffer
func concatBuf(a, b string) bytes.Buffer {
	var buf bytes.Buffer
	buf.WriteString(a)
	buf.WriteString(b)
	return buf
}

// hash returns the HMAC hash of the provided slice of bytes using SHA-256.
func hash(b []byte) ([]byte, error) {
	mac := hmac.New(sha256.New, hashKey)
	_, err := mac.Write(b)
	if err != nil {
		return nil, err
	}
	h := mac.Sum(nil)
	return h, nil
}
