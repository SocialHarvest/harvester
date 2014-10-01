package config

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	//"reflect"
	. "gopkg.in/check.v1"
	"strconv"
	"testing"
	"time"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type DatabaseSuite struct {
	config      SocialHarvestConf
	testMessage SocialHarvestMessage
}

var _ = Suite(&DatabaseSuite{})

// NOTE: A test Postgres database is required, called: socialharvest-test
// TODO: Maybe add a separate JSON config for testing - will need to test both MySQL and Postgres...and others?
var schema = `
DROP TABLE IF EXISTS "settings";
CREATE TABLE "settings" (
	"key" varchar(150) COLLATE "default",
	"value" text COLLATE "default",
	"modified" timestamp(6) NOT NULL
)
WITH (OIDS=FALSE);

DROP TABLE IF EXISTS "harvest";
CREATE TABLE "harvest" (
	"territory" varchar(150) COLLATE "default",
	"network" varchar(75) COLLATE "default",
	"action" varchar(255) COLLATE "default",
	"value" text COLLATE "default",
	"last_time_harvested" timestamp(6) NULL,
	"last_id_harvested" varchar(255) COLLATE "default",
	"items_harvested" int4,
	"harvest_time" timestamp(6) NOT NULL
)
WITH (OIDS=FALSE);

DROP TABLE IF EXISTS "messages";
CREATE TABLE "messages" (
	"time" timestamp(6) NULL,
	"harvest_id" varchar(255) NOT NULL COLLATE "default",
	"territory" varchar(255) COLLATE "default",
	"network" varchar(75) COLLATE "default",
	"contributor_id" varchar(255) COLLATE "default",
	"contributor_screen_name" varchar(255) COLLATE "default",
	"facebook_shares" int4,
	"message_id" varchar(255) COLLATE "default",
	"message" text COLLATE "default",
	"contributor_name" varchar(255) COLLATE "default",
	"contributor_gender" int2,
	"contributor_type" varchar(100) COLLATE "default",
	"contributor_longitude" float8,
	"contributor_latitude" float8,
	"contributor_geohash" varchar(100) COLLATE "default",
	"contributor_lang" varchar(8) COLLATE "default",
	"contributor_likes" int4,
	"contributor_statuses_count" int4,
	"contributor_listed_count" int4,
	"contributor_followers" int4,
	"contributor_verified" int2,
	"is_question" int2,
	"category" varchar(100) COLLATE "default",
	"twitter_retweet_count" int4,
	"twitter_favorite_count" int4,
	"like_count" int4,
	"google_plus_reshares" int4,
	"google_plus_ones" int4,
	"contributor_country" varchar(6) COLLATE "default",
	"contributor_city" varchar(75) COLLATE "default",
	"contributor_state" varchar(50) COLLATE "default",
	"contributor_county" varchar(75) COLLATE "default"
)
WITH (OIDS=FALSE);

ALTER TABLE "messages" ADD CONSTRAINT "messages_harvest_id_unique" UNIQUE ("harvest_id") NOT DEFERRABLE INITIALLY IMMEDIATE;
`

func (s *DatabaseSuite) SetUpSuite(c *C) {
	s.config = SocialHarvestConf{}
	s.config.Database.Type = "postgres"
	s.config.Database.Host = "localhost"
	s.config.Database.Port = 5432
	s.config.Database.Database = "socialharvest-test"
	s.config.Database.User = "tom"
	s.config.Database.Password = ""

	s.testMessage = SocialHarvestMessage{}
	s.testMessage.Time = time.Now()
	s.testMessage.HarvestId = "foobarhash"
	s.testMessage.Territory = "foo"
	s.testMessage.Network = "facebook"
	s.testMessage.MessageId = "12345"
	s.testMessage.ContributorId = "09876"
	s.testMessage.ContributorScreenName = "Happy Poster"
	s.testMessage.ContributorName = "Happy LastName"
	s.testMessage.ContributorGender = 1
	s.testMessage.ContributorType = "person"
	s.testMessage.ContributorLongitude = -122.152927
	s.testMessage.ContributorLatitude = 37.446485
	s.testMessage.ContributorGeohash = "9q9jh2fvw6xz"
	s.testMessage.ContributorLang = "en"
	s.testMessage.ContributorCountry = "US"
	s.testMessage.ContributorCity = "Palo Alto"
	s.testMessage.ContributorState = "California"
	s.testMessage.ContributorCounty = "Santa Clara"
	s.testMessage.Message = "foobar"
	s.testMessage.IsQuestion = 0

	// Actually connect to a live test database. This is all more of an integration test.. And testing this would be to test sqlx... But it's nice to have an actual test database setup.
	// It's more easily lets us catch issues like connection limits (even if each server has different limits and is configured differently, it still helps) and even helps with benchmarking.
	// Of course the test environment may not be setup like the production environment, but this can still give us some ideas.
	db, err := sqlx.Connect(s.config.Database.Type, "host="+s.config.Database.Host+" port="+strconv.Itoa(s.config.Database.Port)+" sslmode=disable dbname="+s.config.Database.Database+" user="+s.config.Database.User+" password="+s.config.Database.Password)
	if err != nil {
		c.Error(err)
		c.Fail()
	}

	// Setup the schema for tests.
	db.MustExec(schema)
}

func (s *DatabaseSuite) TestNewDatabase(c *C) {
	db := NewDatabase(s.config)
	c.Assert(db.Session.DriverName(), Equals, "postgres")
}

func (s *DatabaseSuite) TestSaveSettings(c *C) {
	settings := Settings{
		Key:      "foo",
		Value:    "bar",
		Modified: time.Now(),
	}

	db := NewDatabase(s.config)

	// new
	db.SaveSettings(settings)
	var count int
	err := db.Session.Get(&count, "SELECT count(*) FROM settings;")
	if err != nil {
		c.Error(err)
	} else {
		c.Assert(count, Equals, 1)
	}

	// update
	settings.Value = "updated"
	db.SaveSettings(settings)
	var u Settings
	err = db.Session.Get(&u, "SELECT * FROM settings WHERE key = $1", settings.Key)
	if err != nil {
		c.Error(err)
	} else {
		c.Assert(u.Value, Equals, "updated")
	}
}

func (s *DatabaseSuite) TestLastHarvest(c *C) {
	lastTime := time.Now().Round(time.Second)
	lastHarvestData := SocialHarvestHarvest{
		Territory:         "foo",
		Network:           "facebook",
		Action:            "publicPostsByKeyword",
		Value:             "bar",
		LastTimeHarvested: lastTime,
		LastIdHarvested:   "12345",
		ItemsHarvested:    5,
		HarvestTime:       time.Now(),
	}

	db := NewDatabase(s.config)

	db.SetLastHarvestTime(lastHarvestData.Territory, lastHarvestData.Network, lastHarvestData.Action, lastHarvestData.Value, lastHarvestData.LastTimeHarvested, lastHarvestData.LastIdHarvested, lastHarvestData.ItemsHarvested)

	lastHarvestTime := db.GetLastHarvestTime("foo", "facebook", "publicPostsByKeyword", "bar")
	// Note: the timezone is going to be different based on the system/database configuration
	c.Assert(lastHarvestTime.Minute(), Equals, lastTime.Minute())
	c.Assert(lastHarvestTime.Second(), Equals, lastTime.Second())

	lastHarvestId := db.GetLastHarvestId("foo", "facebook", "publicPostsByKeyword", "bar")
	c.Assert(lastHarvestId, Equals, lastHarvestData.LastIdHarvested)
}

// This didn't seem to work, but running a regular benchmark using Go's native benchmark stuff, I saw: 625010 ns/op (over 1400 records per second) on a local db
// Faster than real world, but fast enough even when it's a fraction of that.
func (s *DatabaseSuite) TestStoreRow(c *C) {
	db := NewDatabase(s.config)

	db.StoreRow(s.testMessage)

	var m SocialHarvestMessage
	err := db.Session.Get(&m, "SELECT * FROM messages WHERE harvest_id = $1", s.testMessage.HarvestId)
	if err != nil {
		c.Error(err)
	} else {
		c.Assert(m.Message, Equals, s.testMessage.Message)
	}
}
