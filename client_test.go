package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/zmb3/spotify"
)

// assertError checks if the provided error and expected
// error string signal the same error. If they do not, the
// given test will fail. If they do, the test continues as normal.
func assertError(t *testing.T, err error, expErr string) {
	if err != nil {
		if err.Error() != expErr {
			t.Fatalf("Expected error '%v', got '%v'", expErr, err.Error())
		}
	} else {
		if expErr != "" {
			t.Fatalf("Expected error '%v', got nil error", expErr)
		}
	}
	return
}

type testClient struct{}

func (ts testClient) CurrentUser() (*spotify.PrivateUser, error) {
	u := spotify.PrivateUser{}
	err := loadFile("test_data/current_user.json", &u)
	if err != nil {
		return nil, err
	}

	return &u, nil
}

func (ts testClient) CurrentUsersTopArtistsOpt(opt *spotify.Options) (*spotify.FullArtistPage, error) {
	a := spotify.FullArtistPage{}
	err := loadFile("test_data/current_users_top_artists.json", &a)
	if err != nil {
		return nil, err
	}
	a.Artists = a.Artists[:*opt.Limit]
	return &a, nil
}

func (ts testClient) CurrentUsersTopTracksOpt(opt *spotify.Options) (*spotify.FullTrackPage, error) {
	t := spotify.FullTrackPage{}
	err := loadFile("test_data/current_users_top_tracks.json", &t)
	if err != nil {
		return nil, err
	}
	t.Tracks = t.Tracks[:*opt.Limit]
	return &t, nil
}

func (ts testClient) PlayerRecentlyPlayedOpt(opt *spotify.RecentlyPlayedOptions) ([]spotify.RecentlyPlayedItem, error) {
	rp := spotify.RecentlyPlayedResult{}
	err := loadFile("test_data/player_recently_played.json", &rp)
	if err != nil {
		return nil, err
	}
	rp.Items = rp.Items[:opt.Limit]
	return rp.Items, nil
}

func (ts testClient) GetRecommendations(spotify.Seeds, *spotify.TrackAttributes, *spotify.Options) (*spotify.Recommendations, error) {
	recs := spotify.Recommendations{}
	err := loadFile("test_data/get_recommendations.json", &recs)
	if err != nil {
		return nil, err
	}
	return &recs, nil
}

func (ts testClient) CreatePlaylistForUser(string, string, bool) (*spotify.FullPlaylist, error) {
	pl := spotify.FullPlaylist{}
	err := loadFile("test_data/create_playlist_for_user.json", &pl)
	if err != nil {
		return nil, err
	}
	return &pl, nil
}

func (ts testClient) AddTracksToPlaylist(string, spotify.ID, ...spotify.ID) (string, error) {
	resp := struct {
		SnapshotID string `json:"snapshot_id"`
	}{}
	err := loadFile("test_data/add_tracks_to_playlist.json", &resp)
	if err != nil {
		return "", err
	}
	return resp.SnapshotID, nil
}

func (ts testClient) GetArtists(IDs ...spotify.ID) ([]*spotify.FullArtist, error) {
	a := struct {
		Artists []*spotify.FullArtist `json:"artists"`
	}{}
	err := loadFile("test_data/get_artists.json", &a)
	if err != nil {
		return nil, err
	}
	a.Artists = a.Artists[:len(IDs)]
	return a.Artists, nil
}

func loadFile(name string, dest interface{}) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	dec.Decode(&dest)
	return nil
}

func newTestClient() spotClient {
	return spotClient{c: testClient{}}
}

func TestUser(t *testing.T) {
	c := newTestClient()

	u, err := c.User()
	if err != nil {
		t.Fatal(err)
	}

	expURI := spotify.URI("spotify:user:wizzler")
	actURI := u.URI
	if actURI != expURI {
		t.Fatalf("Expected URI '%s' got '%s'", expURI, actURI)
	}

	expTotal := uint(3829)
	actTotal := u.Followers.Count
	if actTotal != expTotal {
		t.Fatalf("Expected total of %d got %d", expTotal, actTotal)
	}
}

func TestTopArtistsVar(t *testing.T) {
	var artistTests = []struct {
		Name   string
		Num    int
		Time   []string
		ExpLen int
		ExpErr string
	}{
		{"Single time input", 20, []string{"short"}, 20, ""},
		{"Multiple time input", 20, []string{"long", "short", "medium"}, 60, ""},
		{"Invalid time input", 20, []string{"test", "short"}, 0, errTimeInvalid.Error()},
		{"Invalid low num", -10, []string{"short"}, 0, errNumRange.Error()},
		{"Invalid high num", 100, []string{"short"}, 0, errNumRange.Error()},
		{"No time input", 20, []string{}, 0, errTimeMissing.Error()},
	}
	for _, tt := range artistTests {
		t.Run(tt.Name, func(t *testing.T) {
			c := newTestClient()

			ta, err := c.TopArtistsVar(tt.Num, tt.Time...)
			assertError(t, err, tt.ExpErr)

			if tt.ExpErr != "" {
				return
			}

			el := tt.ExpLen
			al := len(ta)
			if al != el {
				t.Fatalf("Expected length of %d, got %d", el, al)
			}

			en := "Radiohead"
			an := ta[0].Name
			if an != en {
				t.Fatalf("Expected name '%s', got '%s'", en, an)
			}
		})
	}
}

func TestTopArtists(t *testing.T) {
	var artistTests = []struct {
		Name   string
		Num    int
		Time   string
		ExpErr string
	}{
		{"Short time input", 20, "short", ""},
		{"Medium time input", 20, "medium", ""},
		{"Long time input", 20, "long", ""},
		{"Invalid low num", 0, "short", errNumRange.Error()},
		{"Invalid high num", 51, "short", errNumRange.Error()},
		{"Invalid time", 20, "wrong", errTimeInvalid.Error()},
	}
	for _, tt := range artistTests {
		t.Run(tt.Name, func(t *testing.T) {
			c := newTestClient()

			ta, err := c.TopArtists(tt.Num, tt.Time)
			assertError(t, err, tt.ExpErr)

			if tt.ExpErr != "" {
				return
			}

			el := 20
			al := len(ta)
			if al != el {
				t.Fatalf("Expected length of %d, got %d", el, al)
			}

			en := "Radiohead"
			an := ta[0].Name
			if an != en {
				t.Fatalf("Expected name '%s', got '%s'", en, an)
			}
		})
	}
}

func TestTopTracksVar(t *testing.T) {
	var trackTests = []struct {
		Name   string
		Num    int
		Time   []string
		ExpLen int
		ExpErr string
	}{
		{"Single time input", 20, []string{"short"}, 20, ""},
		{"Multiple time input", 20, []string{"long", "short", "medium"}, 60, ""},
		{"Invalid time input", 20, []string{"test", "short"}, 0, errTimeInvalid.Error()},
		{"Invalid low num", -10, []string{"short"}, 0, errNumRange.Error()},
		{"Invalid high num", 100, []string{"short"}, 0, errNumRange.Error()},
		{"No time input", 20, []string{}, 0, errTimeMissing.Error()},
	}
	for _, tt := range trackTests {
		t.Run(tt.Name, func(t *testing.T) {
			c := newTestClient()

			top, err := c.TopTracksVar(tt.Num, tt.Time...)
			assertError(t, err, tt.ExpErr)

			if tt.ExpErr != "" {
				return
			}

			el := tt.ExpLen
			al := len(top)
			if al != el {
				t.Fatalf("Expected length of %d, got %d", el, al)
			}

			en := "Ammunition"
			an := top[0].Name
			if an != en {
				t.Fatalf("Expected name '%s', got '%s'", en, an)
			}

			eID := spotify.ID("2HoDr1yVksl19omOhc1zWy")
			aID := top[1].ID
			if aID != eID {
				t.Fatalf("Expected ID '%s', got '%s'", eID, aID)
			}
		})
	}
}

func TestTopTracks(t *testing.T) {
	var trackTests = []struct {
		Name   string
		Num    int
		Time   string
		ExpErr string
	}{
		{"Short time input", 25, "short", ""},
		{"Medium time input", 25, "medium", ""},
		{"Long time input", 25, "long", ""},
		{"Invalid low num", 0, "short", errNumRange.Error()},
		{"Invalid high num", 51, "short", errNumRange.Error()},
		{"Invalid time", 25, "wrong", errTimeInvalid.Error()},
	}
	for _, tt := range trackTests {
		t.Run(tt.Name, func(t *testing.T) {
			c := newTestClient()

			top, err := c.TopTracks(tt.Num, tt.Time)
			assertError(t, err, tt.ExpErr)

			if tt.ExpErr != "" {
				return
			}

			el := 25
			al := len(top)
			if al != el {
				t.Fatalf("Expected length of %d, got %d", el, al)
			}

			en := "Ammunition"
			an := top[0].Name
			if an != en {
				t.Fatalf("Expected name '%s', got '%s'", en, an)
			}

			eID := spotify.ID("2HoDr1yVksl19omOhc1zWy")
			aID := top[1].ID
			if aID != eID {
				t.Fatalf("Expected ID '%s', got '%s'", eID, aID)
			}
		})
	}
}

func TestRecentTracks(t *testing.T) {
	var trackTests = []struct {
		Name   string
		Num    int
		ExpLen int
		ExpErr string
	}{
		{"Valid num", 25, 25, ""},
		{"Invalid low num", -15, 0, errNumRange.Error()},
		{"Invalid high num", 100, 0, errNumRange.Error()},
		{"Invalid zero num", 0, 0, errNumRange.Error()},
	}

	for _, tt := range trackTests {
		t.Run(tt.Name, func(t *testing.T) {
			c := newTestClient()

			rt, err := c.RecentTracks(tt.Num)
			assertError(t, err, tt.ExpErr)

			if tt.ExpErr != "" {
				return
			}

			el := tt.ExpLen
			al := len(rt)
			if al != el {
				t.Fatalf("Expected length %d, got %d", el, al)
			}

			en := "Black Nostaljack AKA Come On"
			an := rt[0].Name
			if an != en {
				t.Fatalf("Expected name '%s', got '%s'", en, an)
			}

			eURI := spotify.URI("spotify:track:1s92LwFTivD2f9o0s2hb78")
			aURI := rt[1].URI
			if aURI != eURI {
				t.Fatalf("Expected URI '%s', got '%s'", eURI, aURI)
			}
		})
	}
}

func TestRecentArtists(t *testing.T) {
	var artistTests = []struct {
		Name   string
		Num    int
		ExpLen int
		ExpErr string
	}{
		{"Valid num", 15, 15, ""},
		{"Invalid low num", -30, 0, errNumRange.Error()},
		{"Invalid high num", 60, 0, errNumRange.Error()},
		{"Invalid zero num", 0, 0, errNumRange.Error()},
	}

	for _, tt := range artistTests {
		t.Run(tt.Name, func(t *testing.T) {
			c := newTestClient()

			ra, err := c.RecentArtists(tt.Num)
			assertError(t, err, tt.ExpErr)

			if tt.ExpErr != "" {
				return
			}

			el := tt.ExpLen
			al := len(ra)
			if al != el {
				t.Fatalf("Expected length %d, got %d", el, al)
			}

			en := "Radiohead"
			an := ra[0].Name
			if an != en {
				t.Fatalf("Expected name '%s', got '%s'", en, an)
			}

			eURI := spotify.URI("spotify:artist:3yY2gUcIsjMr8hjo51PoJ8")
			aURI := ra[1].URI
			if aURI != eURI {
				t.Fatalf("Expected URI '%s', got '%s'", eURI, aURI)
			}
		})
	}
}

func TestRecommendation(t *testing.T) {
	c := newTestClient()

	recs, err := c.Recommendation(spotify.Seeds{}, 10)
	if err != nil {
		t.Fatal(err)
	}

	el := 10
	al := len(recs)
	if al != el {
		t.Fatalf("Expected length of %d, got %d", el, al)
	}

	en := recs[9].Name
	an := "Appalachian Spring: Moderato - Coda"
	if an != en {
		t.Fatalf("Expected name %s, got %s", en, an)
	}
}

func TestPlaylist(t *testing.T) {
	var playlistTests = []struct {
		Name   string
		PName  string
		IDs    []spotify.ID
		ExpErr string
	}{
		{"Single ID", "MyPlaylist", []spotify.ID{"id1"}, ""},
		{"Multiple IDs", "MyPlaylist", []spotify.ID{"id1", "id2", "id3", "id4"}, ""},
		{"No IDs", "MyPlaylist", []spotify.ID{}, errPlaylistTracksMissing.Error()},
		{"No name", "", []spotify.ID{"id1", "id2", "id3"}, ""},
	}

	for _, tt := range playlistTests {
		t.Run(tt.Name, func(t *testing.T) {
			c := newTestClient()

			pl, err := c.Playlist(tt.PName, tt.IDs)
			assertError(t, err, tt.ExpErr)

			if tt.ExpErr != "" {
				return
			}

			eURI := spotify.URI("spotify:user:zbergquist99:playlist:7I6yjOAxMq4qsvzgqxw7aU")
			aURI := pl.URI
			if aURI != eURI {
				t.Fatalf("Expected name '%s', got '%s'", eURI, aURI)
			}

			ep := true
			ap := pl.IsPublic
			if ap != ep {
				t.Fatalf("Expected public bool '%t', got '%t'", ep, ap)
			}
		})
	}
}
