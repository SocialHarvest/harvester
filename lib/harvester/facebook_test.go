package harvester

import (
	//"github.com/SocialHarvest/harvester/lib/config"
	//geohash "github.com/TomiHiltunen/geohash-golang"
	//"github.com/stretchr/testify/assert"
	//"reflect"
	"testing"
)

// TODO: Configure test databases and logs and such?

// TODO: Add tests. The following are for reference/example...But this has completely changed and needs to be updated.
func TestNewFacebook(t *testing.T) {
}

/*
var servicesCfg = config.ServicesConfig{}

func TestNewFacebook(t *testing.T) {
	servicesCfg.Facebook.AppToken = "1234567890"
	NewFacebook(servicesCfg)

	expected := "1234567890"
	assert.Equal(t, services.facebookAppToken, expected, "SocialHarvestConf.Services.Facebook.AppToken should be "+expected)
}

func TestNewFacebookTerritoryCredentials(t *testing.T) {
	servicesCfg.Facebook.AppToken = "1234567890"

	territoryServices := config.ServicesConfig{}
	territoryServices.Facebook.AppToken = "override"

	NewFacebook(servicesCfg)

	expected := "1234567890"
	assert.Equal(t, services.facebookAppToken, expected, "SocialHarvestConf.Services.Facebook.AppToken should be "+expected)

	// TODO: Actually test the override

}

// This is neat, but do benchmark tests make sense? A lot of these functions make HTTP requests, so tat's gonna vary and even if we stop the timer for that...
// ...Which we can't without re-writing code...I don't know. I'm more interested in seeing benchmarks for things like the gender lookup and any other data filtering.
func Benchmark_NewFacebook(b *testing.B) {
	servicesCfg.Facebook.AppToken = "1234567890"

	for i := 0; i < b.N; i++ { //use b.N for looping
		NewFacebook(servicesCfg)
	}
}

// This might make more sense...but the placement not so much...Maybe a separate (single?) benchmark test file for a variety of things...
func Benchmark_Geohash(b *testing.B) {
	for i := 0; i < b.N; i++ { //use b.N for looping
		geohash.Encode(30.2500, 97.7500)
	}
}

func TestFacebookSearch(t *testing.T) {
	params := FacebookParams{}
	params.Q = "obama"
	params.Limit = "2"
	params.Type = "post"

	posts, newParams := FacebookSearch(params)

	actual := newParams.Until
	expected := "0"
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("returned FacebookParams.Until (expected) %s != %s (actual)", expected, actual)
	}

	actual = newParams.Q
	expected = "obama"
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("returned FacebookParams.Q (expected) %s != %s (actual)", expected, actual)
	}

	actual = newParams.Limit
	expected = "2"
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("returned FacebookParams.Limit (expected) %s != %s (actual)", expected, actual)
	}

	actual = newParams.Type
	expected = "post"
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("returned FacebookParams.Type (expected) %s != %s (actual)", expected, actual)
	}

	if !reflect.DeepEqual(reflect.TypeOf(posts).Kind(), reflect.Slice) {
		t.Error("returned []FacebookPost did not return a Slice")
	}
}

func TestFacebookFeed(t *testing.T) {
	params := FacebookParams{}
	params.Limit = "2"

	posts, newParams := FacebookFeed("socialharvest", params)

	actual := newParams.Until
	expected := "0"
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("returned FacebookParams.Until (expected) %s != %s (actual)", expected, actual)
	}

	actual = newParams.Limit
	expected = "2"
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("returned FacebookParams.Limit (expected) %s != %s (actual)", expected, actual)
	}

	if !reflect.DeepEqual(reflect.TypeOf(posts).Kind(), reflect.Slice) {
		t.Error("returned []FacebookPost did not return a Slice")
	}
}

func TestFacebookGetUserInfo(t *testing.T) {
	account := FacebookGetUserInfo("socialharvest")

	actual := account.Username
	expected := ""
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("returned FacebookAccount.Username (expected) %s != %s (actual)", expected, actual)
	}
}
*/
