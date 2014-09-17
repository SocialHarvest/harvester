// Social Harvest is a social media analytics platform.
//     Copyright (C) 2014 Tom Maiaroto, Shift8Creative, LLC (http://www.socialharvest.io)
//
//     This program is free software: you can redistribute it and/or modify
//     it under the terms of the GNU General Public License as published by
//     the Free Software Foundation, either version 3 of the License, or
//     (at your option) any later version.
//
//     This program is distributed in the hope that it will be useful,
//     but WITHOUT ANY WARRANTY; without even the implied warranty of
//     MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//     GNU General Public License for more details.
//
//     You should have received a copy of the GNU General Public License
//     along with this program.  If not, see <http://www.gnu.org/licenses/>.

package config

import (
	//"net/http"
	//"bytes"
	//"database/sql"
	//"github.com/asaskevich/govalidator"
	//"database/sql"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type SocialHarvestDB struct {
	Type    string
	Session *sqlx.DB
	Series  []string
}

var database = SocialHarvestDB{}

// Optional settings table/collection holds Social Harvest configurations and configured dashboards for persistence and clustered servers it is more or less a key value store.
// Data is stored as JSON string. The Social Harvest config JSON string should easily map to the SocialHarvestConf struct. Other values could be for JavaScript on the front-end.
type Settings struct {
	Key      string    `json:"key" db:"key" bson:"key"`
	Value    string    `json:"value" db:"value" bson:"value"`
	Modified time.Time `json:"modified" db:"modified" bson:"modified"`
}

// Initializes the database and returns the client (NOTE: In the future, this *may* be interchangeable for another database)
func NewDatabase(config SocialHarvestConf) *SocialHarvestDB {

	db, err := sqlx.Connect(config.Database.Type, "host="+config.Database.Host+" port="+strconv.Itoa(config.Database.Port)+" sslmode=disable dbname="+config.Database.Database+" user="+config.Database.User+" password="+config.Database.Password)
	if err != nil {
		log.Fatalln(err)
	}

	// Keep a list of series (tables/collections/series - whatever the database calls them, we're going with series because we're really dealing with time with just about all our data)
	// These do relate to structures in lib/config/series.go
	database.Series = []string{"messages", "shared_links", "mentions", "hashtags", "contributor_growth"}

	// Keep a session for queries (writers have their own) - main.go will defer the close of this session.
	database.Session = db

	return &database
}

// // Saves a settings key/value (Social Harvest config or dashboard settings, etc. - anything that needs configuration data can optionally store it using this function)
// func (database *SocialHarvestDB) SaveSettings(settingsRow Settings, dbSession sqlx.DB) {
// 	if len(settingsRow.Key) > 0 {
// 		col, colErr := dbSession.Collection("settings")
// 		if colErr != nil {
// 			//log.Fatalf("sessionCopy.Collection(): %q\n", colErr)
// 			log.Printf("dbSession.Collection(%s): %q\n", "settings", colErr)
// 			return
// 		}

// 		// If it already exists, update
// 		res := col.Find(db.Cond{"key": settingsRow.Key})
// 		count, findErr := res.Count()
// 		if findErr != nil {
// 			log.Println(findErr)
// 		}
// 		if count > 0 {
// 			updateErr := res.Update(settingsRow)
// 			if updateErr != nil {
// 				log.Println(updateErr)
// 			}
// 		} else {
// 			// Otherwise, save new
// 			_, appendErr := col.Append(settingsRow)
// 			if appendErr != nil {
// 				// this would log a bunch of errors on duplicate entries (not too many, but enough to be annoying)
// 				//log.Println(appendErr)
// 			}
// 		}
// 	}
// 	return
// }

// Sets the last harvest time for a given action, value, network set.
// For example: "facebook" "publicPostsByKeyword" "searchKeyword" 1402260944
// We can use the time to pass to future searches, in Facebook's case, an "until" param
// that tells Facebook to not give us anything before the last harvest date...assuming we
// already have it for that particular search query. Multiple params separated by colon.
func (database *SocialHarvestDB) SetLastHarvestTime(territory string, network string, action string, value string, lastTimeHarvested time.Time, lastIdHarvested string, itemsHarvested int) {
	lastHarvestRow := SocialHarvestHarvest{
		Territory:         territory,
		Network:           network,
		Action:            action,
		Value:             value,
		LastTimeHarvested: lastTimeHarvested,
		LastIdHarvested:   lastIdHarvested,
		ItemsHarvested:    itemsHarvested,
		HarvestTime:       time.Now(),
	}

	//log.Println(lastTimeHarvested)

	database.StoreRow(lastHarvestRow)
}

// Gets the last harvest time for a given action, value, and network (NOTE: This doesn't necessarily need to have been set, it could be empty...check with time.IsZero()).
func (database *SocialHarvestDB) GetLastHarvestTime(territory string, network string, action string, value string) time.Time {
	var lastHarvestTime time.Time
	var lastHarvest SocialHarvestHarvest
	database.Session.Get(&lastHarvest, "SELECT * FROM harvest WHERE network = $1 AND action = $2 AND value = $3 AND territory = $4", network, action, value, territory)
	log.Println(lastHarvest)
	lastHarvestTime = lastHarvest.LastTimeHarvested
	return lastHarvestTime
}

// Gets the last harvest id for a given task, param, and network.
func (database *SocialHarvestDB) GetLastHarvestId(territory string, network string, action string, value string) string {
	lastHarvestId := ""
	var lastHarvest SocialHarvestHarvest
	database.Session.Get(&lastHarvest, "SELECT * FROM harvest WHERE network = $1 AND action = $2 AND value = $3 AND territory = $4", network, action, value, territory)
	lastHarvestId = lastHarvest.LastIdHarvested
	return lastHarvestId
}

// Stores a harvested row of data into the configured database.
func (database *SocialHarvestDB) StoreRow(row interface{}) {
	tx := database.Session.MustBegin()

	// The downside to not using upper.io/db or something like it is that INSERT statements incur technical debt.
	// There will be a maintenance burden in keeping the field names up to date.
	// ...and values have to be in the right order, maintaining this in a repeated fashion leads to spelling mistakes, etc. All the reasons I HATE dealing with SQL...But oh well.

	// Check if valid type to store and determine the proper table/collection based on it
	switch row.(type) {
	case SocialHarvestMessage:
		tx.NamedExec("INSERT INTO messages (time, harvest_id, territory, network, message_id, contributor_id, contributor_screen_name, contributor_name, contributor_gender, contributor_type, contributor_longitude, contributor_latitude, contributor_geohash, contributor_lang, contributor_country, contributor_city, contributor_state, contributor_county, contributor_likes, contributor_statuses_count, contributor_listed_count, contributor_followers, contributor_verified, message, is_question, category, facebook_shares, twitter_retweet_count, twitter_favorite_count, like_count, google_plus_reshares, google_plus_ones) VALUES (:time, :harvest_id, :territory, :network, :message_id, :contributor_id, :contributor_screen_name, :contributor_name, :contributor_gender, :contributor_type, :contributor_longitude, :contributor_latitude, :contributor_geohash, :contributor_lang, :contributor_country, :contributor_city, :contributor_state, :contributor_county, :contributor_likes, :contributor_statuses_count, :contributor_listed_count, :contributor_followers, :contributor_verified, :message, :is_question, :category, :facebook_shares, :twitter_retweet_count, :twitter_favorite_count, :like_count, :google_plus_reshares, :google_plus_ones)", &row)
	case SocialHarvestSharedLink:
		//collection = SeriesCollections["SocialHarvestSharedLink"]
	case SocialHarvestMention:
		//collection = SeriesCollections["SocialHarvestMention"]
	case SocialHarvestHashtag:
		//collection = SeriesCollections["SocialHarvestHashtag"]
	case SocialHarvestContributorGrowth:
		//collection = SeriesCollections["SocialHarvestContributorGrowth"]
	case SocialHarvestHarvest:
		tx.NamedExec("INSERT INTO harvest (territory, network, action, value, last_time_harvested, last_id_harvested, items_harvested, harvest_time) VALUES (:territory, :network, :action, :value, :last_time_harvested, :last_id_harvested, :items_harvested, :harvest_time)", &row)
	default:
		// log.Println("trying to store unknown collection")
	}

	tx.Commit()
}

// -------- GETTING STUFF BACK OUT ------------
// Note: We're a little stuck in the ORM and prepared statement department because our queries need to be pretty flexible.
// Table names are dynamic in some cases (rules out prepared statements) and we have special functions and "AS" keywords all over,
// so most ORMs are out because they are designed for basic CRUD. Upper.io wasn't the most robust ORM either, but it supported quite
// a few databases and worked well for the writes. The reading was always going to be a challenge. We luck out a little bit with using
// the CommonQueryParams struct because we know the Limit, for example, must be an int and therefore is sanitized already.
// Sanitizing data won't be so bad though because we're only allowing a limited amount of user input to begin with.

// Some common parameters to make passing them around a bit easier
type CommonQueryParams struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Territory string `json:"territory"`
	Network   string `json:"network,omitempty"`
	Field     string `json:"field,omitempty"`
	Limit     uint   `json:"limit,omitempty"`
	Series    string `json:"series,omitempty"`
	Skip      uint   `json:"skip,omitempty"`
	Sort      string `json:"sort,omitempty"`
}

type ResultCount struct {
	Count    uint64 `json:"count"`
	TimeFrom string `json:"timeFrom"`
	TimeTo   string `json:"timeTo"`
}

type ResultAggregateCount struct {
	Count uint64 `json:"count"`
	Value string `json:"value"`
}

type ResultAggregateAverage struct {
	Average int    `json:"average"`
	Value   string `json:"value"`
}

type ResultAggregateFields struct {
	Count    map[string][]ResultAggregateCount   `json:"counts,omitempty"`
	Average  map[string][]ResultAggregateAverage `json:"averages,omitempty"`
	TimeFrom string                              `json:"timeFrom"`
	TimeTo   string                              `json:"timeTo"`
	Total    uint64                              `json:"total"`
}

type MessageConditions struct {
	Gender     string `json:"contributor_gender,omitempty"`
	Lang       string `json:"contributor_lang,omitempty"`
	Country    string `json:"contributor_country,omitempty"`
	IsQuestion int    `json:"is_question,omitempty"`
	Geohash    string `json:"contributor_geohash,omitempty"`
	Search     string `json:"search,omitempty"`
}

// Sanitizes common query params to prevent SQL injection and to ensure proper formatting, etc.
func SanitizeCommonQueryParams(params CommonQueryParams) CommonQueryParams {
	sanitizedParams := CommonQueryParams{}

	// Just double check it's positive
	if params.Limit > 0 {
		sanitizedParams.Limit = params.Limit
	}
	if params.Skip > 0 {
		sanitizedParams.Skip = params.Skip
	}

	// Prepared statements not so good when we let users dynamically chose the table to query (neither are any of the ORMs for Golang either unfortunately).
	// Only allow tables speicfied in the series slice to be used in a query.
	for _, v := range database.Series {
		if params.Series == v {
			sanitizedParams.Series = params.Series
		}
	}

	// Territory names can included spaces and are alphanumeric
	pattern := `(?i)[A-z0-9\s]`
	r, _ := regexp.Compile(pattern)
	if r.MatchString(params.Territory) {
		sanitizedParams.Territory = params.Territory
	}

	// Field (column) names and Network names can contain letters, numbers, and underscores
	pattern = `(?i)[A-z0-9\_]`
	r, _ = regexp.Compile(pattern)
	if r.MatchString(params.Field) {
		sanitizedParams.Field = params.Field
	}
	r, _ = regexp.Compile(pattern)
	if r.MatchString(params.Network) {
		sanitizedParams.Network = params.Network
	}

	// Sort can contain letters, numbers, underscores, and commas
	pattern = `(?i)[A-z0-9\_\,]`
	r, _ = regexp.Compile(pattern)
	if r.MatchString(params.Sort) {
		sanitizedParams.Sort = params.Sort
	}

	// to/from are dates and there's only certain characters necessary there too. Fore xample, something like 2014-08-08 12:00:00 is all we need.
	// TODO: Maybe timezone too? All dates should be UTC so there may really be no need.
	// Look for anything other than numbers, a single dash, colons, and spaces. Then also trim a dash at the end of the string in case. It's an invalid query really, but let it work still (for now).
	pattern = `\-{2,}|\"|\'|[A-z]|\#|\;|\*|\!|\\|\/|\(|\)|\|`
	r, _ = regexp.Compile(pattern)
	if !r.MatchString(params.To) {
		sanitizedParams.To = strings.Trim(params.To, "-")
	}
	if !r.MatchString(params.From) {
		sanitizedParams.From = strings.Trim(params.From, "-")
	}

	//log.Println(sanitizedParams)
	return sanitizedParams
}

// Groups fields values and returns a count of occurences
func (database *SocialHarvestDB) FieldCounts(queryParams CommonQueryParams, fields []string) ([]ResultAggregateFields, ResultCount) {
	var fieldCounts []ResultAggregateFields
	var total ResultCount
	sanitizedQueryParams := SanitizeCommonQueryParams(queryParams)
	total.TimeTo = sanitizedQueryParams.To
	total.TimeFrom = sanitizedQueryParams.From

	return fieldCounts, total
}

// Returns total number of records for a given territory and series. Optional conditions for network, field/value, and date range. This is just a simple COUNT().
// However, since it accepts a date range, it could be called a few times to get a time series graph.
func (database *SocialHarvestDB) Count(queryParams CommonQueryParams, fieldValue string) ResultCount {
	sanitizedQueryParams := SanitizeCommonQueryParams(queryParams)
	var count = ResultCount{Count: 0, TimeFrom: sanitizedQueryParams.From, TimeTo: sanitizedQueryParams.To}

	return count
}

// Allows the messages series to be queried in some general ways.
func (database *SocialHarvestDB) Messages(queryParams CommonQueryParams, conds MessageConditions) ([]SocialHarvestMessage, uint64, uint, uint) {
	sanitizedQueryParams := SanitizeCommonQueryParams(queryParams)
	var results = []SocialHarvestMessage{}
	var total uint64

	return results, total, sanitizedQueryParams.Skip, sanitizedQueryParams.Limit
}
