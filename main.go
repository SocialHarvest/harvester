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
	"github.com/fatih/color"
	"log"
	"net/http"
	"os"
	"strconv"
	//"sync"
	_ "net/http/pprof"
	"reflect"
	"runtime"
	"time"
)

var socialHarvest = config.SocialHarvest{}

var harvestChannel = make(chan interface{})

// --------- Route functions

// Shows the harvest schedule as currently configured
func ShowSchedule(w rest.ResponseWriter, r *rest.Request) {
	res := config.NewHypermediaResource()
	res.AddCurie("schedule", "/docs/rels/{rel}", true)

	res.Links["self"] = config.HypermediaLink{
		Href: "/schedule/read",
	}
	res.Links["schedule:add"] = config.HypermediaLink{
		Href: "/schedule/add",
	}
	res.Links["schedule:delete"] = config.HypermediaLink{
		Href:      "/schedule/delete/{id}",
		Templated: true,
	}

	jobs := []map[string]interface{}{}
	for _, item := range socialHarvest.Schedule.Cron.Entries() {
		m := make(map[string]interface{})
		m["id"] = item.Id
		m["name"] = item.Name
		m["next"] = item.Next
		m["prev"] = item.Prev
		m["job"] = getFunctionName(item.Job)
		jobs = append(jobs, m)
	}
	res.Data["totalJobs"] = len(jobs)
	res.Data["jobs"] = jobs

	res.Success()
	w.WriteJson(res.End("There are " + strconv.Itoa(len(jobs)) + " jobs scheduled."))
}

// Shows the current harvester configuration
func ShowSocialHarvestConfig(w rest.ResponseWriter, r *rest.Request) {
	res := config.NewHypermediaResource()
	res.Links["self"] = config.HypermediaLink{
		Href: "/config/read",
	}
	res.Data["config"] = socialHarvest.Config.Harvest
	res.Success()
	w.WriteJson(res.End())
}

// Streams stuff
func StreamTwitter(w rest.ResponseWriter, r *rest.Request) {

	// TODO: allow this stream to be filtered...
	// we can have event names like: SocialHarvestMessage:twitter
	// or by territory, SocialHarvestMessage:territoryName
	// or both, SocialHarvestMessage:territoryName:twitter
	// (or alter the observer to take and pass back more arguments)
	// then we can simply use switches to ensure the proper messages are being put into WriteJson
	// i think we can also use select{} too...

	streamCh := make(chan interface{})
	harvester.Subscribe("SocialHarvestMessage", streamCh)
	// harvester.Subscribe("sub1", streamCh) // this seemingly had no affect...(can only subscribe to one event per channel) which means we will need to have multiple channels here
	// and that means select{} is going to be our filter. of course we could merge the data or call w.WriteJson() multiple times that's fine too.
	// but selecting the right channel may be more efficient if set up properly.
	for {
		data := <-streamCh
		//fmt.Printf("sub3: %v\n", data)

		w.Header().Set("Content-Type", "application/json")
		w.WriteJson(data)
		w.(http.ResponseWriter).Write([]byte("\n"))
		w.(http.Flusher).Flush()
		time.Sleep(time.Duration(1) * time.Second)
	}

	/*
		member := socialHarvest.Broadcasters.Contributor.Join()

		for {
			message := member.Recv()
			//message := <-member.In
			w.Header().Set("Content-Type", "application/json")
			w.WriteJson(message)
			w.(http.ResponseWriter).Write([]byte("\n"))
			// Flush the buffer to client
			w.(http.Flusher).Flush()
			// wait a second between messages (though we don't have to!)
			// note that long after the broadcasted messages occured this will still be going, so theoretically it's possible to get a big pile up...though messages do timeout)
			time.Sleep(time.Duration(1) * time.Second)
		}
	*/
}

// --------- Initial schedule

// Set the initial schedule entries from config SocialHarvestConf
func setInitialSchedule() {
	for _, territory := range socialHarvest.Config.Harvest.Territories {
		if territory.Schedule.Everything.Accounts != "" {
			socialHarvest.Schedule.Cron.AddFunc(territory.Schedule.Everything.Accounts, HarvestAllAccounts, "Harvesting all content - "+territory.Schedule.Everything.Accounts)
		}
		if territory.Schedule.Everything.Content != "" {
			socialHarvest.Schedule.Cron.AddFunc(territory.Schedule.Everything.Accounts, HarvestAllAccounts, "Harvesting all content - "+territory.Schedule.Everything.Accounts)
		}

	}
}

// Helper function to get the name of a function (primarily used to show scheduled tasks)
func getFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

// Main - initializes, configures, and sets routes for API
func main() {
	/*
		runtime.SetBlockProfileRate(1)
		// Start another profile server
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	*/

	color.Cyan(" ____             _       _   _   _                           _  ")
	color.Cyan(`/ ___|  ___   ___(_) __ _| | | | | | __ _ _ ____   _____  ___| |_ ®`)
	color.Cyan("\\___ \\ / _ \\ / __| |/ _` | | | |_| |/ _` | '__\\ \\ / / _ \\/ __| __|")
	color.Cyan(" ___) | (_) | (__| | (_| | | |  _  | (_| | |   \\ V /  __/\\__ \\ |_ ")
	color.Cyan("|____/ \\___/ \\___|_|\\__,_|_| |_| |_|\\__,_|_|    \\_/ \\___||___/\\__|")
	//	color.Cyan("                                                                  ")
	color.Yellow("_____________________________________________version 0.4.0-preview")
	color.Cyan("   ")

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

	// Set the configuration, DB client, etc. so that it is available to other stuff.
	socialHarvest.Config = configuration
	socialHarvest.Database = config.NewDatabase(socialHarvest.Config)
	socialHarvest.Schedule = config.NewSchedule(socialHarvest.Config)
	socialHarvest.Writers = config.NewWriters(socialHarvest.Config)

	// Set up all the channels used by Social Harvest
	//socialHarvest.Broadcasters = config.NewBroadcasters()

	// Set up the data sources (social media APIs for now) and give them the writers (and database connection) they need (all for now)
	//harvester.NewTwitter(socialHarvest)
	//harvester.NewFacebook(socialHarvest)
	//harvester.NewInstagram(socialHarvest)
	//
	// TODO: See about only passing the part of the config needed (Services)
	// We don't need to pass the entire configuration (port, server, passwords, etc. lots of stuff will come to be in there), but we do need all the API tokens and any territroy API token overrides.
	// We might need some other harvest settings, likely not the schedule though. But it's ok to pass anyway. TODO: Think about breaking this down farther.
	harvester.New(socialHarvest.Config.Harvest, socialHarvest.Config.Services)

	// Set the initial schedule (can be changed via API if available)
	setInitialSchedule()

	// Immedate calls to use for testing during development
	// Search Facebook public posts using keywords in Social Harvest config
	//go FacebookPublicMessagesByKeyword()
	// Search Facebook public feeds using account ids in Social Harvest config
	//go FacebookMessagesByAccount()
	// Search Twitter using keywords in Social Harvest config
	//go TwitterPublicMessagesByKeyword()

	// TODO: Maybe the configuration can specify which data to store? I don't know why anyone would want to restrict what's being stored, but who knows...
	// Plus, this would only prevent storage/logging. The data would still be harvested. ... Maybe also a StoreAll() function? Note that all of these should be gosubroutines.
	go StoreMessage()
	go StoreMention()
	go StoreSharedLink()
	// TODO: add hashtags and contributor growth too.

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
		err := handler.SetRoutes(
			&rest.Route{"GET", "/schedule/read", ShowSchedule},
			&rest.Route{"GET", "/config/read", ShowSocialHarvestConfig},
			&rest.Route{"GET", "/stream/twitter", StreamTwitter},
		)
		if err != nil {
			log.Fatal(err)
		}

		// Allow the port to be configured (we need it as a string, but let the config define an int)
		p := strconv.Itoa(socialHarvest.Config.Server.Port)
		// But if it can't be parsed (maybe wasn't set) then set it to 3000
		if p == "0" {
			p = "3000"
		}
		log.Println("Social Harvest API listening on port " + p)
		log.Fatal(http.ListenAndServe(":"+p, &handler))
	}
}
