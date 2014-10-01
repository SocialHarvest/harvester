package harvester

// import (
// 	"github.com/SocialHarvest/harvester/lib/config"
// 	//geohash "github.com/TomiHiltunen/geohash-golang"
// 	//"github.com/stretchr/testify/assert"
// 	//"reflect"
// 	"github.com/jmoiron/sqlx"
// 	_ "github.com/lib/pq"
// 	. "gopkg.in/check.v1"
// 	"log"
// 	"strconv"
// 	"testing"
// 	"time"
// )

// // Hook up gocheck into the "go test" runner.
// func Test(t *testing.T) { TestingT(t) }

// type FacebookSuite struct {
// 	config      config.SocialHarvestConf
// 	testMessage config.SocialHarvestMessage
// }

// var _ = Suite(&FacebookSuite{})

// // NOTE: A test Postgres database is required, called: socialharvest-test
// // TODO: Maybe add a separate JSON config for testing - will need to test both MySQL and Postgres...and others?
// var schema = `
// DROP TABLE IF EXISTS "settings";
// CREATE TABLE "settings" (
// 	"key" varchar(150) COLLATE "default",
// 	"value" text COLLATE "default",
// 	"modified" timestamp(6) NOT NULL
// )
// WITH (OIDS=FALSE);

// DROP TABLE IF EXISTS "harvest";
// CREATE TABLE "harvest" (
// 	"territory" varchar(150) COLLATE "default",
// 	"network" varchar(75) COLLATE "default",
// 	"action" varchar(255) COLLATE "default",
// 	"value" text COLLATE "default",
// 	"last_time_harvested" timestamp(6) NULL,
// 	"last_id_harvested" varchar(255) COLLATE "default",
// 	"items_harvested" int4,
// 	"harvest_time" timestamp(6) NOT NULL
// )
// WITH (OIDS=FALSE);

// DROP TABLE IF EXISTS "messages";
// CREATE TABLE "messages" (
// 	"time" timestamp(6) NULL,
// 	"harvest_id" varchar(255) NOT NULL COLLATE "default",
// 	"territory" varchar(255) COLLATE "default",
// 	"network" varchar(75) COLLATE "default",
// 	"contributor_id" varchar(255) COLLATE "default",
// 	"contributor_screen_name" varchar(255) COLLATE "default",
// 	"facebook_shares" int4,
// 	"message_id" varchar(255) COLLATE "default",
// 	"message" text COLLATE "default",
// 	"contributor_name" varchar(255) COLLATE "default",
// 	"contributor_gender" int2,
// 	"contributor_type" varchar(100) COLLATE "default",
// 	"contributor_longitude" float8,
// 	"contributor_latitude" float8,
// 	"contributor_geohash" varchar(100) COLLATE "default",
// 	"contributor_lang" varchar(8) COLLATE "default",
// 	"contributor_likes" int4,
// 	"contributor_statuses_count" int4,
// 	"contributor_listed_count" int4,
// 	"contributor_followers" int4,
// 	"contributor_verified" int2,
// 	"is_question" int2,
// 	"category" varchar(100) COLLATE "default",
// 	"twitter_retweet_count" int4,
// 	"twitter_favorite_count" int4,
// 	"like_count" int4,
// 	"google_plus_reshares" int4,
// 	"google_plus_ones" int4,
// 	"contributor_country" varchar(6) COLLATE "default",
// 	"contributor_city" varchar(75) COLLATE "default",
// 	"contributor_state" varchar(50) COLLATE "default",
// 	"contributor_county" varchar(75) COLLATE "default"
// )
// WITH (OIDS=FALSE);

// ALTER TABLE "messages" ADD CONSTRAINT "messages_harvest_id_unique" UNIQUE ("harvest_id") NOT DEFERRABLE INITIALLY IMMEDIATE;
// `

// func (s *FacebookSuite) SetUpSuite(c *C) {
// 	// s.config = config.SocialHarvestConf{}
// 	// s.config.Database.Type = "postgres"
// 	// s.config.Database.Host = "localhost"
// 	// s.config.Database.Port = 5432
// 	// s.config.Database.Database = "socialharvest-test"
// 	// s.config.Database.User = "tom"
// 	// s.config.Database.Password = ""

// 	// TODO: DO NOT CHECK THIS IN. We will need some sort of test config JSON I guess...
// 	s.config.Services = config.ServicesConfig{}

// 	s.testMessage = config.SocialHarvestMessage{}
// 	s.testMessage.Time = time.Now()
// 	s.testMessage.HarvestId = "foobarhash"
// 	s.testMessage.Territory = "foo"
// 	s.testMessage.Network = "facebook"
// 	s.testMessage.MessageId = "12345"
// 	s.testMessage.ContributorId = "09876"
// 	s.testMessage.ContributorScreenName = "Happy Poster"
// 	s.testMessage.ContributorName = "Happy LastName"
// 	s.testMessage.ContributorGender = 1
// 	s.testMessage.ContributorType = "person"
// 	s.testMessage.ContributorLongitude = -122.152927
// 	s.testMessage.ContributorLatitude = 37.446485
// 	s.testMessage.ContributorGeohash = "9q9jh2fvw6xz"
// 	s.testMessage.ContributorLang = "en"
// 	s.testMessage.ContributorCountry = "US"
// 	s.testMessage.ContributorCity = "Palo Alto"
// 	s.testMessage.ContributorState = "California"
// 	s.testMessage.ContributorCounty = "Santa Clara"
// 	s.testMessage.Message = "foobar"
// 	s.testMessage.IsQuestion = 0

// 	// Actually connect to a live test database. This is all more of an integration test.. And testing this would be to test sqlx... But it's nice to have an actual test database setup.
// 	// It's more easily lets us catch issues like connection limits (even if each server has different limits and is configured differently, it still helps) and even helps with benchmarking.
// 	// Of course the test environment may not be setup like the production environment, but this can still give us some ideas.
// 	// db, err := sqlx.Connect(s.config.Database.Type, "host="+s.config.Database.Host+" port="+strconv.Itoa(s.config.Database.Port)+" sslmode=disable dbname="+s.config.Database.Database+" user="+s.config.Database.User+" password="+s.config.Database.Password)
// 	// if err != nil {
// 	// 	c.Error(err)
// 	// 	c.Fail()
// 	// }

// 	// Setup the schema for tests.
// 	//db.MustExec(schema)
// }

// func (s *FacebookSuite) TestNewFacebook(c *C) {
// 	NewFacebook(s.config.Services)
// }

// func (s *FacebookSuite) TestFacebookSearch(c *C) {
// 	harvestState := config.HarvestState{
// 		LastId:         "",
// 		LastTime:       time.Now(),
// 		PagesHarvested: 1,
// 		ItemsHarvested: 0,
// 	}
// 	params := FacebookParams{}

// 	newParams, newState := FacebookSearch("foo", harvestState, params)
// 	log.Println(newParams)
// 	log.Println(newState)
// }

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
