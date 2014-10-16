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
	"github.com/mitchellh/mapstructure"
	"hash/crc64"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"time"
)

type SocialHarvestDB struct {
	Postgres *sqlx.DB
	InfluxDB *influxdb.Client
	Series   []string
	Schema   struct {
		Compact bool `json:"compact"`
	}
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

	// Holds some options that will adjust the schema
	database.Schema = config.Schema

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
	case "postgres", "postgresql":
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
// TODO: Maybe just make this update the JSON file OR save to some sort of localstore so the settings don't go into the database where data is harvested
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
// TODO: Support InfluxDB
func (database *SocialHarvestDB) GetLastHarvestTime(territory string, network string, action string, value string) time.Time {
	var lastHarvestTime time.Time
	var lastHarvest SocialHarvestHarvest
	if database.Postgres != nil {
		database.Postgres.Get(&lastHarvest, "SELECT * FROM harvest WHERE network = $1 AND action = $2 AND value = $3 AND territory = $4", network, action, value, territory)
	}
	if database.InfluxDB != nil {
		var buffer bytes.Buffer
		buffer.WriteString("SELECT * FROM harvest WHERE network = '")
		buffer.WriteString(network)
		buffer.WriteString("' AND action = '")
		buffer.WriteString(action)
		buffer.WriteString("' AND value = '")
		buffer.WriteString(value)
		buffer.WriteString("' AND territory = '")
		buffer.WriteString(territory)
		buffer.WriteString("' LIMIT 1")
		query := buffer.String()
		buffer.Reset()

		result, err := database.InfluxDB.Query(query)
		if err == nil {
			if len(result) > 0 {
				if len(result[0].Points) > 0 {
					mappedRecord := map[string]interface{}{}
					keys := result[0].Columns
					// Just one in this case (see LIMIT 1 above)
					record := result[0].Points[0]
					for i := 0; i < len(keys); i++ {
						mappedRecord[keys[i]] = record[i]
					}
					// mapstructure is a very handy package in this case...
					err = mapstructure.Decode(mappedRecord, &lastHarvest)
					if err != nil {
						log.Println(err)
					}
				}
			}
		}
	}

	// log.Println(lastHarvest)
	lastHarvestTime = lastHarvest.LastTimeHarvested
	return lastHarvestTime
}

// Gets the last harvest id for a given task, param, and network.
// TODO: Support InfluxDB
func (database *SocialHarvestDB) GetLastHarvestId(territory string, network string, action string, value string) string {
	lastHarvestId := ""
	var lastHarvest SocialHarvestHarvest
	if database.Postgres != nil {
		database.Postgres.Get(&lastHarvest, "SELECT * FROM harvest WHERE network = $1 AND action = $2 AND value = $3 AND territory = $4", network, action, value, territory)
	}
	if database.InfluxDB != nil {
		var buffer bytes.Buffer
		buffer.WriteString("SELECT * FROM harvest WHERE network = '")
		buffer.WriteString(network)
		buffer.WriteString("' AND action = '")
		buffer.WriteString(action)
		buffer.WriteString("' AND value = '")
		buffer.WriteString(value)
		buffer.WriteString("' AND territory = '")
		buffer.WriteString(territory)
		buffer.WriteString("' LIMIT 1")
		query := buffer.String()
		buffer.Reset()

		result, err := database.InfluxDB.Query(query)
		if err == nil {
			if len(result) > 0 {
				if len(result[0].Points) > 0 {
					mappedRecord := map[string]interface{}{}
					keys := result[0].Columns
					// Just one in this case (see LIMIT 1 above)
					record := result[0].Points[0]
					for i := 0; i < len(keys); i++ {
						mappedRecord[keys[i]] = record[i]
					}
					// mapstructure is a very handy package in this case...
					err = mapstructure.Decode(mappedRecord, &lastHarvest)
					if err != nil {
						log.Println(err)
					}
				}
			}
		}
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

// Returns data in a series of points for use with InfluxDB, optionally filtering which fields end up in the series.
func MakeInfluxRow(row interface{}, fields []string) [][]interface{} {
	// The values
	v := reflect.ValueOf(row)
	// The type (which let's us get the struct field names)
	vT := reflect.TypeOf(row)
	values := make([]interface{}, v.NumField())
	for i := 0; i < v.NumField(); i++ {
		switch v.Field(i).Interface().(type) {
		case time.Time:
			// InfluxDB wants time as int64, ms precision.
			timeField := v.Field(i).Interface()
			values[i] = (timeField.(time.Time).Unix() * 1000)
		default:
			if len(fields) > 0 {
				for _, fieldVal := range fields {
					// If the field from the `row` is listed in `fields` then keep it.
					if fieldVal == vT.Field(i).Tag.Get("json") {
						values[i] = v.Field(i).Interface()
					}
				}
			} else {
				// If `fields` is empty, return all of the fields.
				values[i] = v.Field(i).Interface()
			}
		}
	}
	// TODO: Maybe make the sequence_hash optional by looking for it in fields if there have been fields specified? or another argument
	// We likely will always want a sequence_number generated to avoid dupes...But it's theoretically possible that we would want it to get automaticalled assigned
	// by InfluxDB somewhere, since this MakeInfluxRow() could be called for a variety of reasons aside from harvested data storage.

	// If we have a HarvestId, convert it to a sequence_number for InfluxDB (helps prevent dupes)
	harvestId := reflect.ValueOf(row).FieldByName("HarvestId")
	if harvestId.IsValid() {
		sequenceHash := MakeSequenceHash(harvestId.String())
		values = append(values, sequenceHash)
	}
	points := [][]interface{}{}
	points = append(points, values)

	return points
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

	// The following will insert the data into the supported databases. Certain series will contain more or less data depending on configuration.
	// Compact storage reduces the number of fields stored on series and assumes the database supports JOINs (or is making some other query) to get the data from the `messages` series.
	// This saves on disk space, but increases query complexity. Full flat storage / expanded schema. This uses more disk space, but the queries should be faster.

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
			if database.Schema.Compact {
				_, err = database.Postgres.NamedExec("INSERT INTO shared_links (time, harvest_id, territory, network, message_id, contributor_id, type, preview, source, url, expanded_url, host) VALUES (:time, :harvest_id, :territory, :network, :message_id, :contributor_id, :type, :preview, :source, :url, :expanded_url, :host);", row)
			} else {
				_, err = database.Postgres.NamedExec("INSERT INTO shared_links (time, harvest_id, territory, network, message_id, contributor_id, contributor_screen_name, contributor_name, contributor_gender, contributor_type, contributor_longitude, contributor_latitude, contributor_geohash, contributor_lang, contributor_country, contributor_city, contributor_state, contributor_county, type, preview, source, url, expanded_url, host) VALUES (:time, :harvest_id, :territory, :network, :message_id, :contributor_id, :contributor_screen_name, :contributor_name, :contributor_gender, :contributor_type, :contributor_longitude, :contributor_latitude, :contributor_geohash, :contributor_lang, :contributor_country, :contributor_city, :contributor_state, :contributor_county, :type, :preview, :source, :url, :expanded_url, :host);", row)
			}
			if err != nil {
				//log.Println(err)
			}
		case SocialHarvestMention:
			if database.Schema.Compact {
				_, err = database.Postgres.NamedExec("INSERT INTO mentions (time, harvest_id, territory, network, message_id, contributor_id, mentioned_id, mentioned_screen_name, mentioned_name, mentioned_gender, mentioned_type, mentioned_longitude, mentioned_latitude, mentioned_geohash, mentioned_lang) VALUES (:time, :harvest_id, :territory, :network, :message_id, :contributor_id, :mentioned_id, :mentioned_screen_name, :mentioned_name, :mentioned_gender, :mentioned_type, :mentioned_longitude, :mentioned_latitude, :mentioned_geohash, :mentioned_lang);", row)
			} else {
				_, err = database.Postgres.NamedExec("INSERT INTO mentions (time, harvest_id, territory, network, message_id, contributor_id, contributor_screen_name, contributor_name, contributor_gender, contributor_type, contributor_longitude, contributor_latitude, contributor_geohash, contributor_lang, mentioned_id, mentioned_screen_name, mentioned_name, mentioned_gender, mentioned_type, mentioned_longitude, mentioned_latitude, mentioned_geohash, mentioned_lang) VALUES (:time, :harvest_id, :territory, :network, :message_id, :contributor_id, :contributor_screen_name, :contributor_name, :contributor_gender, :contributor_type, :contributor_longitude, :contributor_latitude, :contributor_geohash, :contributor_lang, :mentioned_id, :mentioned_screen_name, :mentioned_name, :mentioned_gender, :mentioned_type, :mentioned_longitude, :mentioned_latitude, :mentioned_geohash, :mentioned_lang);", row)
			}
			if err != nil {
				//log.Println(err)
			}
		case SocialHarvestHashtag:
			if database.Schema.Compact {
				_, err = database.Postgres.NamedExec("INSERT INTO hashtags (time, harvest_id, territory, network, message_id, tag, keyword, contributor_id) VALUES (:time, :harvest_id, :territory, :network, :message_id, :tag, :keyword, :contributor_id);", row)
			} else {
				_, err = database.Postgres.NamedExec("INSERT INTO hashtags (time, harvest_id, territory, network, message_id, tag, keyword, contributor_id, contributor_screen_name, contributor_name, contributor_gender, contributor_type, contributor_longitude, contributor_latitude, contributor_geohash, contributor_lang, contributor_country, contributor_city, contributor_state, contributor_county) VALUES (:time, :harvest_id, :territory, :network, :message_id, :tag, :keyword, :contributor_id, :contributor_screen_name, :contributor_name, :contributor_gender, :contributor_type, :contributor_longitude, :contributor_latitude, :contributor_geohash, :contributor_lang, :contributor_country, :contributor_city, :contributor_state, :contributor_county);", row)
			}
			if err != nil {
				//log.Println(err)
			}
		case SocialHarvestContributorGrowth:
			_, err = database.Postgres.NamedExec(`INSERT INTO contributor_growth (
			time, harvest_id, territory, network, contributor_id, likes, talking_about, were_here, checkins, views, status_updates, listed, favorites, followers, following, plus_ones, comments) VALUES (:time, :harvest_id, :territory, :network, :contributor_id, :likes, :talking_about, :were_here, :checkins, :views, :status_updates, :listed, :favorites, :followers, :following, :plus_ones, :comments);`, row)
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
		var series = []*influxdb.Series{}
		switch row.(type) {
		case SocialHarvestMessage:
			message := &influxdb.Series{
				Name:    "messages",
				Columns: []string{"time", "harvest_id", "territory", "network", "message_id", "contributor_id", "contributor_screen_name", "contributor_name", "contributor_gender", "contributor_type", "contributor_longitude", "contributor_latitude", "contributor_geohash", "contributor_lang", "contributor_country", "contributor_city", "contributor_state", "contributor_county", "contributor_likes", "contributor_statuses_count", "contributor_listed_count", "contributor_followers", "contributor_verified", "message", "is_question", "category", "facebook_shares", "twitter_retweet_count", "twitter_favorite_count", "like_count", "google_plus_reshares", "google_plus_ones", "sequence_number"},
				Points:  MakeInfluxRow(row, []string{}),
			}
			series = append(series, message)
			if err := database.InfluxDB.WriteSeries(series); err != nil {
				//log.Println(err)
			}
		case SocialHarvestSharedLink:
			sharedLink := &influxdb.Series{}
			if database.Schema.Compact {
				sharedLink = &influxdb.Series{
					Name:    "shared_links",
					Columns: []string{"time", "harvest_id", "territory", "network", "message_id", "contributor_id", "type", "preview", "source", "url", "expanded_url", "host", "sequence_number"},
					Points:  MakeInfluxRow(row, []string{"time", "harvest_id", "territory", "network", "message_id", "contributor_id", "type", "preview", "source", "url", "expanded_url", "host"}),
				}
			} else {
				sharedLink = &influxdb.Series{
					Name:    "shared_links",
					Columns: []string{"time", "harvest_id", "territory", "network", "message_id", "contributor_id", "contributor_screen_name", "contributor_name", "contributor_gender", "contributor_type", "contributor_longitude", "contributor_latitude", "contributor_geohash", "contributor_lang", "contributor_country", "contributor_city", "contributor_state", "contributor_county", "type", "preview", "source", "url", "expanded_url", "host", "sequence_number"},
					Points:  MakeInfluxRow(row, []string{}),
				}
			}

			series = append(series, sharedLink)
			if err := database.InfluxDB.WriteSeries(series); err != nil {
				//log.Println(err)
			}
		case SocialHarvestMention:
			mention := &influxdb.Series{}
			if database.Schema.Compact {
				mention = &influxdb.Series{
					Name:    "mentions",
					Columns: []string{"time", "harvest_id", "territory", "network", "message_id", "contributor_id", "mentioned_id", "mentioned_screen_name", "mentioned_name", "mentioned_gender", "mentioned_type", "mentioned_longitude", "mentioned_latitude", "mentioned_geohash", "mentioned_lang", "sequence_number"},
					Points:  MakeInfluxRow(row, []string{"time", "harvest_id", "territory", "network", "message_id", "contributor_id", "mentioned_id", "mentioned_screen_name", "mentioned_name", "mentioned_gender", "mentioned_type", "mentioned_longitude", "mentioned_latitude", "mentioned_geohash", "mentioned_lang", "sequence_number"}),
				}
			} else {
				mention = &influxdb.Series{
					Name:    "mentions",
					Columns: []string{"time", "harvest_id", "territory", "network", "message_id", "contributor_id", "contributor_screen_name", "contributor_name", "contributor_gender", "contributor_type", "contributor_longitude", "contributor_latitude", "contributor_geohash", "contributor_lang", "mentioned_id", "mentioned_screen_name", "mentioned_name", "mentioned_gender", "mentioned_type", "mentioned_longitude", "mentioned_latitude", "mentioned_geohash", "mentioned_lang", "sequence_number"},
					Points:  MakeInfluxRow(row, []string{}),
				}
			}

			series = append(series, mention)
			if err := database.InfluxDB.WriteSeries(series); err != nil {
				//log.Println(err)
			}
		case SocialHarvestHashtag:
			hashtag := &influxdb.Series{}
			if database.Schema.Compact {
				hashtag = &influxdb.Series{
					Name:    "hashtags",
					Columns: []string{"time", "harvest_id", "territory", "network", "message_id", "tag", "keyword", "contributor_id", "sequence_number"},
					Points:  MakeInfluxRow(row, []string{"time", "harvest_id", "territory", "network", "message_id", "tag", "keyword", "contributor_id"}),
				}
			} else {
				hashtag = &influxdb.Series{
					Name:    "hashtags",
					Columns: []string{"time", "harvest_id", "territory", "network", "message_id", "tag", "keyword", "contributor_id", "contributor_screen_name", "contributor_name", "contributor_gender", "contributor_type", "contributor_longitude", "contributor_latitude", "contributor_geohash", "contributor_lang", "contributor_country", "contributor_city", "contributor_state", "contributor_county", "sequence_number"},
					Points:  MakeInfluxRow(row, []string{}),
				}
			}
			series = append(series, hashtag)
			if err := database.InfluxDB.WriteSeries(series); err != nil {
				//log.Println(err)
			}
		case SocialHarvestContributorGrowth:
			growth := &influxdb.Series{
				Name:    "contributor_growth",
				Columns: []string{"time", "harvest_id", "territory", "network", "contributor_id", "likes", "talking_about", "were_here", "checkins", "views", "status_updates", "listed", "favorites", "followers", "following", "plus_ones", "comments", "sequence_number"},
				Points:  MakeInfluxRow(row, []string{}),
			}
			series = append(series, growth)
			if err := database.InfluxDB.WriteSeries(series); err != nil {
				//log.Println(err)
			}
		case SocialHarvestHarvest:
			harvest := &influxdb.Series{
				Name:    "harvest",
				Columns: []string{"territory", "network", "action", "value", "last_time_harvested", "last_id_harvested", "items_harvested", "harvest_time"},
				Points:  MakeInfluxRow(row, []string{}),
			}
			series = append(series, harvest)
			if err := database.InfluxDB.WriteSeries(series); err != nil {
				//log.Println(err)
			}
		default:
			// log.Println("trying to store unknown collection")
		}
	}

}
