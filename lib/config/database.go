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
	"bytes"
	"database/sql"
	//"github.com/asaskevich/govalidator"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"upper.io/db"
	"upper.io/db/mongo"
	"upper.io/db/mysql"
	"upper.io/db/postgresql"
	"upper.io/db/util/sqlutil"
)

type SocialHarvestDB struct {
	Settings db.Settings
	Type     string
	Session  db.Database
	Series   []string
}

var database = SocialHarvestDB{}

// Initializes the database and returns the client (NOTE: In the future, this *may* be interchangeable for another database)
func NewDatabase(config SocialHarvestConf) *SocialHarvestDB {
	database.Type = config.Database.Type
	database.Settings = db.Settings{
		Host:     config.Database.Host,
		Port:     config.Database.Port,
		Database: config.Database.Database,
		User:     config.Database.User,
		Password: config.Database.Password,
	}

	// Keep a list of series (tables/collections/series - whatever the database calls them, we're going with series because we're really dealing with time with just about all our data)
	// These do relate to structures in lib/config/series.go
	database.Series = []string{"messages", "shared_links", "mentions", "hashtags", "contributor_growth"}

	// Set some indicies
	SetupIndicies()

	// Keep a session for queries (writers have their own) - main.go will defer the close of this session.
	database.Session = database.GetSession()

	return &database
}

// We'll want to set a unique index on "harvest_id" to mitigate dupes up front so we don't need to worry when querying later (and so those queries execute faster)
func SetupIndicies() {
	harvestIdCollections := []string{"messages", "shared_links", "mentions", "hashtags"}
	switch database.Type {
	case "mongodb":
		sess, err := db.Open(mongo.Adapter, database.Settings)
		defer sess.Close()
		if err == nil {
			drv := sess.Driver().(*mgo.Session)
			db := drv.DB(database.Settings.Database)
			for _, v := range harvestIdCollections {
				col := db.C(v)
				index := mgo.Index{
					Key:      []string{"harvest_id"},
					Unique:   true,
					DropDups: true,
					Sparse:   true,
				}

				err := col.EnsureIndex(index)
				if err != nil {
					log.Println(err)
				}

			}
		}
		break
	}
}

// For some reason empty documents are being saved in MongoDB when there are duplicate key errors.
// If the unique index is not sparse then things save fine, otherwise once one empty document gets saved, it blocks others from saving.
// So the unique index needs to be sparse to allow this null value. This is only happening with MongoDB. The SQL databases don't have
// any empty records. While I'm not in love with this hack, I'll live with it for now.
func (database *SocialHarvestDB) RemoveEmpty(collection string) {
	// TODO: Fix whatever is wrong
	switch database.Type {
	case "mongodb":
		sess, err := db.Open(mongo.Adapter, database.Settings)
		defer sess.Close()
		if err == nil {
			drv := sess.Driver().(*mgo.Session)
			db := drv.DB(database.Settings.Database)
			col := db.C(collection)
			// _ could instead be set to return an info struct that would have number of docs removed, etc. - I don't care about this right now.
			_, removeErr := col.RemoveAll(bson.M{"harvest_id": bson.M{"$exists": false}})
			if removeErr != nil {
				log.Println(removeErr)
			}
		}
		break
	}
}

// Sets the last harvest time for a given action, value, network set.
// For example: "facebook" "publicPostsByKeyword" "searchKeyword" 1402260944
// We can use the time to pass to future searches, in Facebook's case, an "until" param
// that tells Facebook to not give us anything before the last harvest date...assuming we
// already have it for that particular search query. Multiple params separated by colon.
func (database *SocialHarvestDB) SetLastHarvestTime(territory string, network string, action string, value string, lastTimeHarvested time.Time, lastIdHarvested string, itemsHarvested int) {
	lastHarvestRow := SocialHarvestHarvest{territory, network, action, value, lastTimeHarvested, lastIdHarvested, itemsHarvested, time.Now()}

	log.Println(lastTimeHarvested)
	// Create a wait group to manage the goroutines (don't think we need concurrency for this particular call since it's not as frequent, but keep for now purely to be compatible with the StoreRow() function).
	var waitGroup sync.WaitGroup
	dbSession := database.GetSession()
	waitGroup.Add(1)
	go database.StoreRow(lastHarvestRow, &waitGroup, dbSession)

	// Wait for the query to complete.
	waitGroup.Wait()
}

// Gets the last harvest time for a given action, value, and network (NOTE: This doesn't necessarily need to have been set, it could be empty...check with time.IsZero()).
func (database *SocialHarvestDB) GetLastHarvestTime(territory string, network string, action string, value string) time.Time {
	var lastHarvestTime time.Time

	sess := database.GetSession()
	defer sess.Close()
	col, err := sess.Collection("harvest")
	if err != nil {
		log.Fatalf("sess.Collection(): %q\n", err)
		return lastHarvestTime
	}
	result := col.Find(db.Cond{"network": network, "action": action, "value": value, "territory": territory}).Sort("-harvest_time")
	defer result.Close()

	var lastHarvest SocialHarvestHarvest
	err = result.One(&lastHarvest)
	if err != nil {
		log.Println(err)
		return lastHarvestTime
	}

	lastHarvestTime = lastHarvest.LastTimeHarvested

	return lastHarvestTime
}

// Gets the last harvest id for a given task, param, and network.
func (database *SocialHarvestDB) GetLastHarvestId(territory string, network string, action string, value string) string {
	lastHarvestId := ""

	sess := database.GetSession()
	defer sess.Close()
	col, err := sess.Collection("harvest")
	if err != nil {
		log.Println(err)
		return lastHarvestId
	}
	result := col.Find(db.Cond{"network": network, "action": action, "value": value, "territory": territory}).Sort("-harvest_time")
	defer result.Close()

	var lastHarvest SocialHarvestHarvest
	err = result.One(&lastHarvest)
	if err != nil {
		log.Println(err)
		return lastHarvestId
	}

	lastHarvestId = lastHarvest.LastIdHarvested

	return lastHarvestId
}

// Stores a harvested row of data into the configured database.
func (database *SocialHarvestDB) StoreRow(row interface{}, waitGroup *sync.WaitGroup, dbSession db.Database) {
	// Decrement the wait group count so the program knows this
	// has been completed once the goroutine exits.
	defer waitGroup.Done()

	// Request a socket connection from the session to process our query.
	// Close the session when the goroutine exits and put the connection back
	// into the pool.
	sessionCopy, err := dbSession.Clone()
	if err != nil {
		log.Println(err)
	}
	defer sessionCopy.Close()

	// TODO: change to collection to series for consistency - it's a little confusing in some areas because Mongo uses "collection" (but SQL of course is table so...)
	collection := ""

	// Check if valid type to store and determine the proper table/collection based on it
	switch row.(type) {
	case SocialHarvestMessage:
		collection = SeriesCollections["SocialHarvestMessage"]
	case SocialHarvestSharedLink:
		collection = SeriesCollections["SocialHarvestSharedLink"]
	case SocialHarvestMention:
		collection = SeriesCollections["SocialHarvestMention"]
	case SocialHarvestHashtag:
		collection = SeriesCollections["SocialHarvestHashtag"]
	case SocialHarvestContributorGrowth:
		collection = SeriesCollections["SocialHarvestContributorGrowth"]
	case SocialHarvestHarvest:
		collection = SeriesCollections["SocialHarvestHarvest"]
	default:
		// log.Println("trying to store unknown collection")
	}
	//log.Println("saving to collection: " + collection)

	//col, colErr := dbSession.Collection(collection)
	col, colErr := sessionCopy.Collection(collection)
	if colErr != nil {
		log.Fatalf("sessionCopy.Collection(): %q\n", colErr)
	}

	if collection != "" {
		// Save
		_, appendErr := col.Append(row)
		if appendErr != nil {
			// this would log a bunch of errors on duplicate entries (not too many, but enough to be annoying)
			//log.Println(appendErr)
		}
	} else {
		log.Println("trying to store to an unknown collection")
	}

}

func (database *SocialHarvestDB) GetSession() db.Database {
	// Figure out which database is being used
	var dbAdapter = ""
	switch database.Type {
	case "mongodb":
		dbAdapter = mongo.Adapter
		break
	case "postgresql":
		dbAdapter = postgresql.Adapter
		break
	case "mysql":
	case "mariadb":
		dbAdapter = mysql.Adapter
		break
	}

	// If one is even being used, connect to it and store the data
	sess, err := db.Open(dbAdapter, database.Settings)

	// Remember to close the database session. - call this from where ever GetSession() is being called.
	// Closing it here would be a problem for another function =)
	//defer sess.Close()

	if err != nil {
		log.Fatalf("db.Open(): %q\n", err)
	}

	return sess
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
	Limit     int    `json:"limit,omitempty"`
	Series    string `json:"series,omitempty"`
}

type ResultCount struct {
	Count    int    `json:"count"`
	TimeFrom string `json:"timeFrom"`
	TimeTo   string `json:"timeTo"`
}

type ResultAggregateCount struct {
	Count int    `json:"count"`
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
	Total    int                                 `json:"total"`
}

// Sanitizes common query params to prevent SQL injection and to ensure proper formatting, etc.
func SanitizeCommonQueryParams(params CommonQueryParams) CommonQueryParams {
	sanitizedParams := CommonQueryParams{}

	// Just double check it's positive
	if params.Limit > 0 {
		sanitizedParams.Limit = params.Limit
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

	switch database.Type {
	case "mongodb":

		break
	case "postgresql", "mysql", "mariadb":
		// The following query should work for pretty much any SQL database (at least any we're supporting)
		var err error
		var rows *sql.Rows
		//var row *sql.Row
		var drv *sql.DB
		drv = database.Session.Driver().(*sql.DB)

		// First get the overall total number of records
		var buffer bytes.Buffer
		buffer.WriteString("SELECT COUNT(*) AS count FROM ")
		buffer.WriteString(sanitizedQueryParams.Series)
		buffer.WriteString(" WHERE territory = '")
		buffer.WriteString(sanitizedQueryParams.Territory)
		buffer.WriteString("'")
		// optional date range (can have either or both)
		if sanitizedQueryParams.From != "" {
			buffer.WriteString(" AND time >= '")
			buffer.WriteString(sanitizedQueryParams.From)
			buffer.WriteString("'")
		}
		if sanitizedQueryParams.To != "" {
			buffer.WriteString(" AND time <= '")
			buffer.WriteString(sanitizedQueryParams.To)
			buffer.WriteString("'")
		}

		err = drv.QueryRow(buffer.String()).Scan(&total)
		buffer.Reset()
		if err != nil {
			log.Println(err)
			return fieldCounts, total
		}

		for _, field := range fields {
			if len(field) > 0 {
				buffer.Reset()
				buffer.WriteString("SELECT COUNT(")
				buffer.WriteString(field)
				buffer.WriteString(") AS count,")
				buffer.WriteString(field)
				buffer.WriteString(" AS value")
				buffer.WriteString(" FROM ")
				buffer.WriteString(sanitizedQueryParams.Series)
				buffer.WriteString(" WHERE territory = '")
				buffer.WriteString(sanitizedQueryParams.Territory)
				buffer.WriteString("'")

				// optional date range (can have either or both)
				if sanitizedQueryParams.From != "" {
					buffer.WriteString(" AND time >= '")
					buffer.WriteString(sanitizedQueryParams.From)
					buffer.WriteString("'")
				}
				if sanitizedQueryParams.To != "" {
					buffer.WriteString(" AND time <= '")
					buffer.WriteString(sanitizedQueryParams.To)
					buffer.WriteString("'")
				}

				buffer.WriteString(" GROUP BY ")
				buffer.WriteString(field)

				buffer.WriteString(" ORDER BY count DESC")

				// optional limit (in this case I don't know why one would use it - a date range would be a better limiter)
				if sanitizedQueryParams.Limit > 0 {
					buffer.WriteString(" LIMIT ")
					buffer.WriteString(strconv.FormatInt(int64(sanitizedQueryParams.Limit), 10))
				}

				rows, err = drv.Query(buffer.String())
				buffer.Reset()
				if err != nil {
					log.Println(err)
					continue
				}
				// Close the pointer
				defer rows.Close()

				var valueCounts []ResultAggregateCount
				if err = sqlutil.FetchRows(rows, &valueCounts); err != nil {
					log.Println(err)
					continue
				}

				count := map[string][]ResultAggregateCount{}
				count[field] = valueCounts

				fieldCount := ResultAggregateFields{Count: count, TimeFrom: sanitizedQueryParams.From, TimeTo: sanitizedQueryParams.To, Total: total.Count}
				fieldCounts = append(fieldCounts, fieldCount)
			}
		}
		// harmless to call again to make sure, but use defer above in case something unexpected happens in the loop and it returns, etc.
		rows.Close()

		break
	}

	return fieldCounts, total
}

// Returns total number of records for a given territory and series. Optional conditions for network, field/value, and date range. This is just a simple COUNT().
// However, since it accepts a date range, it could be called a few times to get a time series graph.
func (database *SocialHarvestDB) Count(queryParams CommonQueryParams, fieldValue string) ResultCount {
	sanitizedQueryParams := SanitizeCommonQueryParams(queryParams)
	var count = ResultCount{Count: 0, TimeFrom: sanitizedQueryParams.From, TimeTo: sanitizedQueryParams.To}

	switch database.Type {
	case "mongodb":

		break
	case "postgresql", "mysql", "mariadb":
		// The following query should work for pretty much any SQL database (at least any we're supporting)
		var err error
		var drv *sql.DB
		drv = database.Session.Driver().(*sql.DB)

		var buffer bytes.Buffer
		buffer.WriteString("SELECT COUNT(*) AS count FROM ")
		buffer.WriteString(sanitizedQueryParams.Series)
		buffer.WriteString(" WHERE territory = '")
		buffer.WriteString(sanitizedQueryParams.Territory)
		buffer.WriteString("'")

		// optional date range (can have either or both)
		if sanitizedQueryParams.From != "" {
			buffer.WriteString(" AND time >= '")
			buffer.WriteString(sanitizedQueryParams.From)
			buffer.WriteString("'")
		}
		if sanitizedQueryParams.To != "" {
			buffer.WriteString(" AND time <= '")
			buffer.WriteString(sanitizedQueryParams.To)
			buffer.WriteString("'")
		}

		// Because we're accepting user inuput, use a prepared statement. Sanitizing fieldValue could also be done in the future perhaps (if needed).
		// The problem with prepared statements everywhere is that we can't put the tables through them. So only a few places will we be able to use them.
		// Here is one though.
		if sanitizedQueryParams.Field != "" && fieldValue != "" {
			buffer.WriteString(" AND ")
			buffer.WriteString(sanitizedQueryParams.Field)
			// Must everything be so different?
			if database.Type == "postgresql" {
				buffer.WriteString(" = $1")
			} else {
				buffer.WriteString(" = ?")
			}
		}

		// Again for the network
		if sanitizedQueryParams.Network != "" {
			buffer.WriteString(" AND network")
			// Must everything be so different?
			if database.Type == "postgresql" {
				buffer.WriteString(" = $2")
			} else {
				buffer.WriteString(" = ?")
			}
		}

		// log.Println(buffer.String())
		stmt, err := drv.Prepare(buffer.String())
		buffer.Reset()
		if err != nil {
			log.Println(err)
			return count
		}
		// TODO: There has to be a better way to do this.... This is pretty stupid, but I can't see how to pass a variable number of args.
		// Prepare() would need to perhaps keep track of how many it was expecting I guess...
		if fieldValue != "" && sanitizedQueryParams.Network == "" {
			err = stmt.QueryRow(fieldValue).Scan(&count)
			if err != nil {
				log.Println(err)
			}
		} else if fieldValue != "" && sanitizedQueryParams.Network != "" {
			err = stmt.QueryRow(fieldValue, sanitizedQueryParams.Network).Scan(&count)
			if err != nil {
				log.Println(err)
			}
		} else if fieldValue == "" && sanitizedQueryParams.Network != "" {
			err = stmt.QueryRow(sanitizedQueryParams.Network).Scan(&count)
			if err != nil {
				log.Println(err)
			}
		}

		break
	}

	return count
}
