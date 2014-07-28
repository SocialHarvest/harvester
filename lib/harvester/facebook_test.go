package harvester

import (
	"bitbucket.org/tmaiaroto/go-social-harvest/lib/config"
	geohash "github.com/TomiHiltunen/geohash-golang"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

// TODO: Configure test databases and logs and such?

var testCfg = config.SocialHarvestConf{}
var testShCfg = config.SocialHarvest{}

func TestInitFacebook(t *testing.T) {
	testCfg.Services.Facebook.AppToken = "1234567890"
	testShCfg.Config = testCfg
	InitFacebook(testShCfg)

	expected := "1234567890"
	assert.Equal(t, facebook.appToken, expected, "SocialHarvestConf.Services.Facebook.AppToken should be "+expected)
}

// This is neat, but do benchmark tests make sense? A lot of these functions make HTTP requests, so tat's gonna vary and even if we stop the timer for that...
// ...Which we can't without re-writing code...I don't know. I'm more interested in seeing benchmarks for things like the gender lookup and any other data filtering.
func Benchmark_InitFacebook(b *testing.B) {
	testCfg.Services.Facebook.AppToken = "1234567890"
	testShCfg.Config = testCfg

	for i := 0; i < b.N; i++ { //use b.N for looping
		InitFacebook(testShCfg)
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
