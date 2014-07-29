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

package main

import (
	"encoding/json"
	"flag"
	"github.com/SocialHarvest/harvester/lib/config"
	"github.com/SocialHarvest/harvester/lib/harvester"
	"github.com/ant0ine/go-json-rest/rest"
	"log"
	"net/http"
	"os"
	"strconv"
	//"sync"
	//"reflect"
)

var socialHarvest = config.SocialHarvest{}

// --------- Route functions

// Shows the harvest schedule as configured
func ShowSchedule(w rest.ResponseWriter, r *rest.Request) {

	socialHarvest.Schedule.Cron.AddFunc("0 5 * * * *", func() { log.Println("Every 5 minutes") }, "Another job every five min.")

	for _, item := range socialHarvest.Schedule.Cron.Entries() {
		log.Println(item.Name)
		log.Println(item.Next)

	}
}

func TestRoute(w rest.ResponseWriter, r *rest.Request) {

	w.WriteJson("foo")
}

// Main - initializes, configures, and sets routes for API
func main() {
	log.Println("harvester started")
	// Optionally allow a config JSON file to be passed via command line
	var confFile string
	flag.StringVar(&confFile, "conf", "social-harvest-conf.json", "Path to the Social Harvest configuration file.")
	flag.Parse()

	// Open the config JSON and decode it.
	file, _ := os.Open(confFile)
	decoder := json.NewDecoder(file)
	configuration := config.SocialHarvestConf{}
	err := decoder.Decode(&configuration)
	if err != nil {
		log.Println("error:", err)
	}
	//log.Println(configuration.Database)

	// Set the configuration, DB client, etc. so that it is available to other stuff.
	socialHarvest.Config = configuration
	socialHarvest.Database = config.NewDatabase(socialHarvest.Config)
	socialHarvest.Schedule = config.NewSchedule(socialHarvest.Config)
	socialHarvest.Writers = config.NewWriters(socialHarvest.Config)

	// Set up the data sources (social media APIs for now) and give them the writers they need (all for now)
	harvester.NewTwitter(socialHarvest)
	harvester.NewFacebook(socialHarvest)

	// Search Facebook public posts using keywords in Social Harvest config
	FacebookPublicMessagesByKeyword()
	// Search Facebook public feeds using account ids in Social Harvest config
	FacebookMessagesByAccount()

	/// TEST/debug
	//	log.Println(socialHarvest.config.Services.Twitter)

	//harvester.YoutubeVideoSearch("obama")
	///

	// The RESTful API server can be completely disabled by setting {"server":{"disabled": true}} in the config
	// NOTE: If this is done, main() returns and that means the schedule will not be processed. This is typically
	// for other packages that want to import Social Harvest. If a server is not desired, simply ensure whatever port
	// Social Harvest runs on is has appropriate firewall settings. Alternatively, we could prevent main() from returning,
	// but that would lead to a more confusing configuration.
	// TODO: Think about accepting command line arguments for adhoc harvesting.
	if !socialHarvest.Config.Server.Disabled {
		handler := rest.ResourceHandler{
			EnableRelaxedContentType: true,
		}
		/*
			// Route definitions
			handler.SetRoutes(
				rest.RouteObjectMethod("GET", "/test", &socialHarvest, "TestRoute"),
				rest.RouteObjectMethod("GET", "/schedule", &socialHarvest, "ShowSchedule"),
			)
		*/

		// Allow the port to be configured (we need it as a string, but let the config define an int)
		p := strconv.Itoa(socialHarvest.Config.Server.Port)
		// But if it can't be parsed (maybe wasn't set) then set it to 3000
		if p == "0" {
			p = "3000"
		}
		log.Println("Social Harvest listening on port " + p)
		http.ListenAndServe(":"+p, &handler)
	}

}
