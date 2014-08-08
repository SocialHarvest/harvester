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
	//"database/sql"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"sync"
	"time"
	"upper.io/db"
	"upper.io/db/mongo"
	"upper.io/db/mysql"
	"upper.io/db/postgresql"
)

type SocialHarvestDB struct {
	Settings db.Settings
	Type     string
	Session  *db.Database
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
	}
	result := col.Find(db.Cond{"network": network, "action": action, "value": value, "territory": territory}).Sort("-harvest_time")

	var lastHarvest SocialHarvestHarvest
	err = result.One(&lastHarvest)
	if err != nil {
		// log.Println(err)
	}

	lastHarvestTime = lastHarvest.LastTimeHarvested
	result.Close()

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
	}
	result := col.Find(db.Cond{"network": network, "action": action, "value": value, "territory": territory}).Sort("-harvest_time")

	var lastHarvest SocialHarvestHarvest
	err = result.One(&lastHarvest)
	if err != nil {
		log.Println(err)
	}

	lastHarvestId = lastHarvest.LastIdHarvested
	result.Close()

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

	// Remember to close the database session.
	//defer sess.Close()

	if err != nil {
		log.Fatalf("db.Open(): %q\n", err)
	}

	return sess
}
