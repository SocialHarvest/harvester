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
	"bytes"
	//"github.com/asaskevich/govalidator"
	influxdb "github.com/influxdb/influxdb/client"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"hash/crc64"
	"log"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type SocialHarvestDB struct {
	Postgres *sqlx.DB
	InfluxDB *influxdb.Client
	Series   []string
}

var database = SocialHarvestDB{}

// Optional settings table/collection holds Social Harvest configurations and configured dashboards for persistence and clustered servers it is more or less a key value store.
// Data is stored as JSON string. The Social Harvest config JSON string should easily map to the SocialHarvestConf struct. Other values could be for JavaScript on the front-end.
type Settings struct {
	Key      string    `json:"key" db:"key" bson:"key"`
	Value    string    `json:"value" db:"value" bson:"value"`
	Modified time.Time `json:"modified" db:"modified" bson:"modified"`
}

// Initializes the database and returns the client, setting it to `database.Postgres` in the current package scope
func NewDatabase(config SocialHarvestConf) *SocialHarvestDB {
	// A database is not required to use Social Harvest
	if config.Database.Type == "" {
		return &database
	}
	var err error

	// Now supporting Postgres OR InfluxDB
	// (for now...may add more in the future...the re-addition of InfluxDB is to satisfy performance curiosities, it may go away. Postgres will ALWAYS be supported.)
	// actually, if config.Database becomes an array, we can write to multiple databases...
	switch config.Database.Type {
	case "influxdb":
		config := &influxdb.ClientConfig{
			Host:       config.Database.Host + ":" + strconv.Itoa(config.Database.Port),
			Username:   config.Database.User,
			Password:   config.Database.Password,
			Database:   config.Database.Database,
			HttpClient: http.DefaultClient,
		}
		database.InfluxDB, err = influxdb.NewClient(config)
		if err != nil {
			log.Println(err)
			return &database
		}
	case "postgres":
	case "postgresql":
		// Note that sqlx just wraps database/sql and `database.Postgres` gets a sqlx.DB which is essentially a wrapped sql.DB
		database.Postgres, err = sqlx.Connect("postgres", "host="+config.Database.Host+" port="+strconv.Itoa(config.Database.Port)+" sslmode=disable dbname="+config.Database.Database+" user="+config.Database.User+" password="+config.Database.Password)
		if err != nil {
			log.Println(err)
			return &database
		}
	}

	// Keep a list of series (tables/collections/series - whatever the database calls them, we're going with series because we're really dealing with time with just about all our data)
	// These do relate to structures in lib/config/series.go
	database.Series = []string{"messages", "shared_links", "mentions", "hashtags", "contributor_growth"}

	return &database
}

// Saves a settings key/value (Social Harvest config or dashboard settings, etc. - anything that needs configuration data can optionally store it using this function)
func (database *SocialHarvestDB) SaveSettings(settingsRow Settings) {
	if len(settingsRow.Key) > 0 {

		var count int
		err := database.Postgres.Get(&count, "SELECT count(*) FROM settings;")
		if err != nil {
			log.Println(err)
			return
		}

		// If it already exists, update
		if count > 0 {
			tx, err := database.Postgres.Beginx()
			if err != nil {
				log.Println(err)
				return
			}
			tx.MustExec("UPDATE settings SET value = $1 WHERE key = $2", settingsRow.Value, settingsRow.Key)
			tx.Commit()

		} else {
			// Otherwise, save new
			tx, err := database.Postgres.Beginx()
			if err != nil {
				log.Println(err)
				return
			}
			tx.NamedExec("INSERT INTO settings (key, value, modified) VALUES (:key, :value, :modified)", settingsRow)
			tx.Commit()
		}
	}
	return
}

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
	if database.Postgres != nil {
		database.Postgres.Get(&lastHarvest, "SELECT * FROM harvest WHERE network = $1 AND action = $2 AND value = $3 AND territory = $4", network, action, value, territory)
	}
	// log.Println(lastHarvest)
	lastHarvestTime = lastHarvest.LastTimeHarvested
	return lastHarvestTime
}

// Gets the last harvest id for a given task, param, and network.
func (database *SocialHarvestDB) GetLastHarvestId(territory string, network string, action string, value string) string {
	lastHarvestId := ""
	var lastHarvest SocialHarvestHarvest
	if database.Postgres != nil {
		database.Postgres.Get(&lastHarvest, "SELECT * FROM harvest WHERE network = $1 AND action = $2 AND value = $3 AND territory = $4", network, action, value, territory)
	}
	lastHarvestId = lastHarvest.LastIdHarvested
	return lastHarvestId
}

// For InfluxDB. This hash (crc64 checksum) should not easily repeat itself with the time field. Time is to the second in most cases, hashing the message id (id_str for Twitter and Facebook's Id values are strings) should avoid dupes just in case a message is processed twice.
func MakeSequenceHash(hash string) uint64 {
	crcTable := crc64.MakeTable(crc64.ECMA)
	hashBytes := []byte(hash)
	return crc64.Checksum(hashBytes, crcTable)
}

// Stores a harvested row of data into the configured database.
func (database *SocialHarvestDB) StoreRow(row interface{}) {
	// A database connection is not required to use Social Harvest (could be logging to file)
	if database.Postgres == nil && database.InfluxDB == nil {
		// log.Println("There appears to be no database connection.")
		return
	}

	// The downside to not using upper.io/db or something like it is that INSERT statements incur technical debt.
	// There will be a maintenance burden in keeping the field names up to date.
	// ...and values have to be in the right order, maintaining this in a repeated fashion leads to spelling mistakes, etc. All the reasons I HATE dealing with SQL...But oh well.

	var err error

	if database.Postgres != nil {
		// Check if valid type to store and determine the proper table/collection based on it
		switch row.(type) {
		case SocialHarvestMessage:
			_, err = database.Postgres.NamedExec("INSERT INTO messages (time, harvest_id, territory, network, message_id, contributor_id, contributor_screen_name, contributor_name, contributor_gender, contributor_type, contributor_longitude, contributor_latitude, contributor_geohash, contributor_lang, contributor_country, contributor_city, contributor_state, contributor_county, contributor_likes, contributor_statuses_count, contributor_listed_count, contributor_followers, contributor_verified, message, is_question, category, facebook_shares, twitter_retweet_count, twitter_favorite_count, like_count, google_plus_reshares, google_plus_ones) VALUES (:time, :harvest_id, :territory, :network, :message_id, :contributor_id, :contributor_screen_name, :contributor_name, :contributor_gender, :contributor_type, :contributor_longitude, :contributor_latitude, :contributor_geohash, :contributor_lang, :contributor_country, :contributor_city, :contributor_state, :contributor_county, :contributor_likes, :contributor_statuses_count, :contributor_listed_count, :contributor_followers, :contributor_verified, :message, :is_question, :category, :facebook_shares, :twitter_retweet_count, :twitter_favorite_count, :like_count, :google_plus_reshares, :google_plus_ones);", row)
			if err != nil {
				//log.Println(err)
			} else {
				//log.Println("Successful insert")
			}
		case SocialHarvestSharedLink:
			_, err = database.Postgres.NamedExec("INSERT INTO shared_links (time, harvest_id, territory, network, message_id, contributor_id, contributor_screen_name, contributor_name, contributor_gender, contributor_type, contributor_longitude, contributor_latitude, contributor_geohash, contributor_lang, contributor_country, contributor_city, contributor_state, contributor_county, type, preview, source, url, expanded_url, host) VALUES (:time, :harvest_id, :territory, :network, :message_id, :contributor_id, :contributor_screen_name, :contributor_name, :contributor_gender, :contributor_type, :contributor_longitude, :contributor_latitude, :contributor_geohash, :contributor_lang, :contributor_country, :contributor_city, :contributor_state, :contributor_county, :type, :preview, :source, :url, :expanded_url, :host);", row)
			if err != nil {
				//log.Println(err)
			}
		case SocialHarvestMention:
			_, err = database.Postgres.NamedExec("INSERT INTO mentions (time, harvest_id, territory, network, message_id, contributor_id, contributor_screen_name, contributor_name, contributor_gender, contributor_type, contributor_longitude, contributor_latitude, contributor_geohash, contributor_lang, mentioned_id, mentioned_screen_name, mentioned_name, mentioned_gender, mentioned_type, mentioned_longitude, mentioned_latitude, mentioned_geohash, mentioned_lang) VALUES (:time, :harvest_id, :territory, :network, :message_id, :contributor_id, :contributor_screen_name, :contributor_name, :contributor_gender, :contributor_type, :contributor_longitude, :contributor_latitude, :contributor_geohash, :contributor_lang, :mentioned_id, :mentioned_screen_name, :mentioned_name, :mentioned_gender, :mentioned_type, :mentioned_longitude, :mentioned_latitude, :mentioned_geohash, :mentioned_lang);", row)
			if err != nil {
				//log.Println(err)
			}
		case SocialHarvestHashtag:
			_, err = database.Postgres.NamedExec("INSERT INTO hashtags (time, harvest_id, territory, network, message_id, tag, keyword, contributor_id, contributor_screen_name, contributor_name, contributor_gender, contributor_type, contributor_longitude, contributor_latitude, contributor_geohash, contributor_lang, contributor_country, contributor_city, contributor_state, contributor_county) VALUES (:time, :harvest_id, :territory, :network, :message_id, :tag, :keyword, :contributor_id, :contributor_screen_name, :contributor_name, :contributor_gender, :contributor_type, :contributor_longitude, :contributor_latitude, :contributor_geohash, :contributor_lang, :contributor_country, :contributor_city, :contributor_state, :contributor_county);", row)
			if err != nil {
				//log.Println(err)
			}
		case SocialHarvestContributorGrowth:
			_, err = database.Postgres.NamedExec(`INSERT INTO contributor_growth (
			time, 
			harvest_id, 
			territory, 
			network, 
			contributor_id, 
			likes,
			talking_about,
			were_here, 
			checkins, 
			views, 
			status_updates, 
			listed, 
			favorites, 
			followers, 
			following,
			plus_ones, 
			comments
		) VALUES (
			:time, 
			:harvest_id, 
			:territory, 
			:network, 
			:contributor_id, 
			:likes, 
			:talking_about,
			:were_here, 
			:checkins, 
			:views, 
			:status_updates, 
			:listed, 
			:favorites, 
			:followers,
			:following,
			:plus_ones, 
			:comments
		);`, row)
			if err != nil {
				// log.Println(err)
			}
		case SocialHarvestHarvest:
			_, err = database.Postgres.NamedExec("INSERT INTO harvest (territory, network, action, value, last_time_harvested, last_id_harvested, items_harvested, harvest_time) VALUES (:territory, :network, :action, :value, :last_time_harvested, :last_id_harvested, :items_harvested, :harvest_time);", row)
			if err != nil {
				//log.Println(err)
			}
		default:
			// log.Println("trying to store unknown collection")
		}
	}

	if database.InfluxDB != nil {
		for i := 0; i < 500; i++ {

			v := reflect.ValueOf(row)
			values := make([]interface{}, v.NumField())
			for i := 0; i < v.NumField(); i++ {
				switch v.Field(i).Interface().(type) {
				case time.Time:
					// InfluxDB wants time as float
				//	timeField := v.Field(i).Interface()
				//	values[i] = float64(timeField.(time.Time).Unix())
				default:
					values[i] = v.Field(i).Interface()
				}
			}
			// If we have a HarvestId, convert it to a sequence_number for InfluxDB (helps prevent dupes)
			harvestId := reflect.ValueOf(row).FieldByName("HarvestId")
			if harvestId.IsValid() {
				//sequenceHash := MakeSequenceHash(harvestId.String() + strconv.Itoa(i))
				//values = append(values, sequenceHash)
			}
			points := [][]interface{}{}
			points = append(points, values)
			var series = []*influxdb.Series{}

			switch row.(type) {
			case SocialHarvestMessage:
				message := &influxdb.Series{
					Name:    "messages",
					Columns: []string{"harvest_id", "territory", "network", "message_id", "contributor_id", "contributor_screen_name", "contributor_name", "contributor_gender", "contributor_type", "contributor_longitude", "contributor_latitude", "contributor_geohash", "contributor_lang", "contributor_country", "contributor_city", "contributor_state", "contributor_county", "contributor_likes", "contributor_statuses_count", "contributor_listed_count", "contributor_followers", "contributor_verified", "message", "is_question", "category", "facebook_shares", "twitter_retweet_count", "twitter_favorite_count", "like_count", "google_plus_reshares", "google_plus_ones"},
					Points:  points,
				}
				series = append(series, message)
				if err := database.InfluxDB.WriteSeries(series); err != nil {
					log.Println(err)
				} else {
					log.Println("inserted 500 records?")
					log.Println(len(series))
				}
			// case SocialHarvestSharedLink:
			// 	sharedLink := &influxdb.Series{
			// 		Name:    "shared_links",
			// 		Columns: []string{"time", "harvest_id", "territory", "network", "message_id", "contributor_id", "contributor_screen_name", "contributor_name", "contributor_gender", "contributor_type", "contributor_longitude", "contributor_latitude", "contributor_geohash", "contributor_lang", "contributor_country", "contributor_city", "contributor_state", "contributor_county", "type", "preview", "source", "url", "expanded_url", "host", "sequence_number"},
			// 		Points:  points,
			// 	}
			// 	series = append(series, sharedLink)
			// 	if err := database.InfluxDB.WriteSeries(series); err != nil {
			// 		log.Println(err)
			// 	}
			// case SocialHarvestMention:
			// 	mention := &influxdb.Series{
			// 		Name:    "mentions",
			// 		Columns: []string{"time", "harvest_id", "territory", "network", "message_id", "contributor_id", "contributor_screen_name", "contributor_name", "contributor_gender", "contributor_type", "contributor_longitude", "contributor_latitude", "contributor_geohash", "contributor_lang", "mentioned_id", "mentioned_screen_name", "mentioned_name", "mentioned_gender", "mentioned_type", "mentioned_longitude", "mentioned_latitude", "mentioned_geohash", "mentioned_lang", "sequence_number"},
			// 		Points:  points,
			// 	}
			// 	series = append(series, mention)
			// 	if err := database.InfluxDB.WriteSeries(series); err != nil {
			// 		log.Println(err)
			// 	}
			// case SocialHarvestHashtag:
			// 	hashtag := &influxdb.Series{
			// 		Name:    "hashtags",
			// 		Columns: []string{"time", "harvest_id", "territory", "network", "message_id", "tag", "keyword", "contributor_id", "contributor_screen_name", "contributor_name", "contributor_gender", "contributor_type", "contributor_longitude", "contributor_latitude", "contributor_geohash", "contributor_lang", "contributor_country", "contributor_city", "contributor_state", "contributor_county", "sequence_number"},
			// 		Points:  points,
			// 	}
			// 	series = append(series, hashtag)
			// 	if err := database.InfluxDB.WriteSeries(series); err != nil {
			// 		log.Println(err)
			// 	}
			// case SocialHarvestContributorGrowth:
			// 	growth := &influxdb.Series{
			// 		Name:    "contributor_growth",
			// 		Columns: []string{"time", "harvest_id", "territory", "network", "contributor_id", "likes", "talking_about", "were_here", "checkins", "views", "status_updates", "listed", "favorites", "followers", "following", "plus_ones", "comments", "sequence_number"},
			// 		Points:  points,
			// 	}
			// 	series = append(series, growth)
			// 	if err := database.InfluxDB.WriteSeries(series); err != nil {
			// 		log.Println(err)
			// 	}
			// case SocialHarvestHarvest:
			// 	harvest := &influxdb.Series{
			// 		Name:    "harvest",
			// 		Columns: []string{"territory", "network", "action", "value", "last_time_harvested", "last_id_harvested", "items_harvested", "harvest_time"},
			// 		Points:  points,
			// 	}
			// 	series = append(series, harvest)
			// 	if err := database.InfluxDB.WriteSeries(series); err != nil {
			// 		log.Println(err)
			// 	}
			default:
				// log.Println("trying to store unknown collection")
			}
		}
	}

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

	//database.Postgres.Get(&lastHarvest, "SELECT * FROM harvest WHERE network = $1 AND action = $2 AND value = $3 AND territory = $4", network, action, value, territory)
	//
	// OR...
	//tx, err := database.Postgres.Beginx()
	// if err != nil {
	// 	log.Println(err)
	// 	return
	// }
	// tx.NamedExec("INSERT INTO settings (key, value, modified) VALUES (:key, :value, :modified)", settingsRow)
	// tx.Commit()

	var buffer bytes.Buffer
	buffer.WriteString("SELECT COUNT(*) FROM ")
	buffer.WriteString(sanitizedQueryParams.Series)
	//buffer.WriteString(" WHERE 1=1")

	condCount := 0

	// optional date range (can have either or both)
	if sanitizedQueryParams.From != "" {
		buffer.WriteString(" WHERE time >=:timeFrom")
		condCount++

		// buffer.WriteString(" AND ")
		// buffer.WriteString("time >= ")
		// buffer.WriteString(sanitizedQueryParams.From)
	}
	if sanitizedQueryParams.To != "" {
		if condCount > 0 {
			buffer.WriteString(" AND time<=:timeTo")
		} else {
			buffer.WriteString(" WHERE time<=:timeTo")
		}
		condCount++

		// buffer.WriteString(" AND ")
		// buffer.WriteString("time <= ")
		// buffer.WriteString(sanitizedQueryParams.To)
	}
	if sanitizedQueryParams.Network != "" {
		//multiple - but i think this was for upper.io/db ... we ultimately want it as a string so why split to only join somewhere else again?
		//networkMultiple := strings.Split(sanitizedQueryParams.Network, ",")

		// singular
		// conditions["network"] = sanitizedQueryParams.Network

		if condCount > 0 {
			buffer.WriteString(" AND network IN(:network)")
		} else {
			buffer.WriteString(" WHERE network IN(:network)")
		}
		condCount++

		// buffer.WriteString(" AND ")
		// buffer.WriteString("network IN(")
		// buffer.WriteString(sanitizedQueryParams.Network)
		// buffer.WriteString(")")
	}
	if sanitizedQueryParams.Territory != "" {
		if condCount > 0 {
			buffer.WriteString(" AND territory=:territory")
		} else {
			buffer.WriteString(" WHERE territory=:territory")
		}
		condCount++

		// buffer.WriteString(" AND ")
		// buffer.WriteString("territory = ")
		// buffer.WriteString(sanitizedQueryParams.Territory)
	}

	if sanitizedQueryParams.Field != "" && fieldValue != "" {
		if condCount > 0 {
			buffer.WriteString(" AND ")
			buffer.WriteString(sanitizedQueryParams.Field)
			buffer.WriteString("=")
			buffer.WriteString(fieldValue)
		}

		// buffer.WriteString(" AND ")
		// buffer.WriteString(sanitizedQueryParams.Field)
		// buffer.WriteString(" = ")
		// buffer.WriteString(fieldValue)
	}

	sqlQuery := buffer.String()
	buffer.Reset()

	log.Println(sqlQuery)

	data := struct {
		TimeFrom  string `db:timeFrom`
		TimeTo    string `db:timeTo`
		Network   string `db:network`
		Territory string `db:territory`
		//FieldValue interface{} `db:fieldValue`
	}{
		TimeFrom:  sanitizedQueryParams.From,
		TimeTo:    sanitizedQueryParams.To,
		Network:   sanitizedQueryParams.Network,
		Territory: sanitizedQueryParams.Territory,
		//FieldValue: fieldValue,
	}
	rows, err := database.Postgres.NamedQuery(sqlQuery, data)
	if err != nil {
		// log.Println(err)
		return count
	}
	for rows.Next() {
		err := rows.StructScan(&count)
		if err != nil {
			// log.Println(err)
		}
	}

	return count
}

// Allows the messages series to be queried in some general ways.
func (database *SocialHarvestDB) Messages(queryParams CommonQueryParams, conds MessageConditions) ([]SocialHarvestMessage, uint64, uint, uint) {
	sanitizedQueryParams := SanitizeCommonQueryParams(queryParams)
	var results = []SocialHarvestMessage{}
	var total uint64

	return results, total, sanitizedQueryParams.Skip, sanitizedQueryParams.Limit
}
