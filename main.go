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
	"github.com/bugsnag/bugsnag-go"
	"github.com/fatih/color"
	"log"
	"net/http"
	//_ "net/http/pprof"
	"os"
	"reflect"
	"runtime"
	"strconv"
)

var socialHarvest = config.SocialHarvest{}

var harvestChannel = make(chan interface{})

// --------- Route functions for the harvester API (which allows for the configuration of the harvester after it is up and running as well as various statistics about the harvester)

// API: Shows the harvest schedule as currently configured
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

// API: Shows the current harvester configuration
func ShowSocialHarvestConfig(w rest.ResponseWriter, r *rest.Request) {
	res := config.NewHypermediaResource()
	res.Links["self"] = config.HypermediaLink{
		Href: "/config/read",
	}
	res.Data["config"] = socialHarvest.Config.Harvest
	res.Success()
	w.WriteJson(res.End())
}

// API: Territory list returns all currently configured territories and their settings
func TerritoryList(w rest.ResponseWriter, r *rest.Request) {
	res := setTerritoryLinks("territory:list")
	res.Data["territories"] = socialHarvest.Config.Harvest.Territories
	res.Success()
	w.WriteJson(res.End())
}

// Sets the hypermedia response "_links" section with all of the routes we have defined for territories.
func setTerritoryLinks(self string) *config.HypermediaResource {
	res := config.NewHypermediaResource()
	res.Links["territory:list"] = config.HypermediaLink{
		Href: "/territory/list",
	}

	selfedRes := config.NewHypermediaResource()
	for link, _ := range res.Links {
		if link == self {
			selfedRes.Links["self"] = res.Links[link]
		} else {
			selfedRes.Links[link] = res.Links[link]
		}
	}
	return selfedRes
}

// --------- API Basic Auth Middleware (valid keys are defined in the Social Harvest config, there are no roles or anything)
type BasicAuthMw struct {
	Realm string
	Key   string
}

func (bamw *BasicAuthMw) MiddlewareFunc(handler rest.HandlerFunc) rest.HandlerFunc {
	return func(writer rest.ResponseWriter, request *rest.Request) {

		authHeader := request.Header.Get("Authorization")
		log.Println(authHeader)
		if authHeader == "" {
			queryParams := request.URL.Query()
			if len(queryParams["apiKey"]) > 0 {
				bamw.Key = queryParams["apiKey"][0]
			} else {
				bamw.unauthorized(writer)
				return
			}
		} else {
			bamw.Key = authHeader
		}

		keyFound := false
		for _, key := range socialHarvest.Config.Server.AuthKeys {
			if bamw.Key == key {
				keyFound = true
			}
		}

		if !keyFound {
			bamw.unauthorized(writer)
			return
		}

		handler(writer, request)
	}
}

func (bamw *BasicAuthMw) unauthorized(writer rest.ResponseWriter) {
	writer.Header().Set("WWW-Authenticate", "Basic realm="+bamw.Realm)
	rest.Error(writer, "Not Authorized", http.StatusUnauthorized)
}

// --------- Initial schedule

// Set the initial schedule entries from config SocialHarvestConf
func setInitialSchedule() {

	for _, territory := range socialHarvest.Config.Harvest.Territories {
		if territory.Schedule.Everything.Accounts != "" {
			socialHarvest.Schedule.Cron.AddFunc(territory.Schedule.Everything.Accounts, HarvestAllAccounts, "Harvesting all accounts - "+territory.Schedule.Everything.Accounts)
		}
		if territory.Schedule.Everything.Content != "" {
			socialHarvest.Schedule.Cron.AddFunc(territory.Schedule.Everything.Content, HarvestAllContent, "Harvesting all content - "+territory.Schedule.Everything.Content)
		}

	}
}

// Helper function to get the name of a function (primarily used to show scheduled tasks)
func getFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

// Main - initializes, configures, and sets routes for API
func main() {
	appVersion := "0.11.0-preview"

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

	// Setup Bugsnag (first), profiling, etc.
	if socialHarvest.Config.Debug.Bugsnag.ApiKey != "" {
		bugsnag.Configure(bugsnag.Configuration{
			APIKey:          socialHarvest.Config.Debug.Bugsnag.ApiKey,
			ReleaseStage:    socialHarvest.Config.Debug.Bugsnag.ReleaseStage,
			ProjectPackages: []string{"main", "github.com/SocialHarvest/harvester/*"},
			AppVersion:      appVersion,
		})
	}

	// Debug - do not compile with this
	// runtime.SetBlockProfileRate(1)
	// // Start a profile server so information can be viewed using a web browser
	// go func() {
	// 	log.Println(http.ListenAndServe("localhost:6060", nil))
	// }()

	// Banner (would appear twice if it came before bugsnag for some reason)
	color.Cyan(" ____             _       _   _   _                           _  ")
	color.Cyan(`/ ___|  ___   ___(_) __ _| | | | | | __ _ _ ____   _____  ___| |_ ®`)
	color.Cyan("\\___ \\ / _ \\ / __| |/ _` | | | |_| |/ _` | '__\\ \\ / / _ \\/ __| __|")
	color.Cyan(" ___) | (_) | (__| | (_| | | |  _  | (_| | |   \\ V /  __/\\__ \\ |_ ")
	color.Cyan("|____/ \\___/ \\___|_|\\__,_|_| |_| |_|\\__,_|_|    \\_/ \\___||___/\\__|")
	//	color.Cyan("                                                                  ")
	color.Yellow("_____________________________________________version " + appVersion)
	color.Cyan("   ")

	// Continue configuration
	socialHarvest.Database = config.NewDatabase(socialHarvest.Config)
	// NOTE: A database is optional for Social Harvest (harvested data can be logged for use with Fluentd for example)
	if socialHarvest.Database.Postgres != nil {
		defer socialHarvest.Database.Postgres.Close()
	}
	socialHarvest.Schedule = config.NewSchedule(socialHarvest.Config)

	// this gets the configuration and the database. TODO: Make database optional
	harvester.New(socialHarvest.Config, socialHarvest.Database)
	// Load new gender data from CSV files for detecting gender (this is callable so it can be changed during runtime)
	// TODO: Think about being able to post more gender statistics via the API to add to the data set...
	harvester.NewGenderData("data/census-female-names.csv", "data/census-male-names.csv")

	// Set the initial schedule (can be changed via API if available)
	setInitialSchedule()

	// Immedate calls to use for testing during development
	// Search Facebook public posts using keywords in Social Harvest config
	//go FacebookPublicMessagesByKeyword()
	// Search Facebook public feeds using account ids in Social Harvest config
	//go FacebookMessagesByAccount()
	// Search Twitter using keywords in Social Harvest config
	//go TwitterPublicMessagesByKeyword()
	//go TwitterPublicMessagesByAccount()
	//  Search Instagram
	//go InstagramMediaByKeyword()
	//go GooglePlusActivitieByKeyword()
	//go GooglePlusActivitieByAccount()
	go HarvestAllContent()
	//go HarvestAllAccounts()

	//harvester.YoutubeVideoSearch("obama")
	///

	// The RESTful API server can be completely disabled by setting {"server":{"disabled": true}} in the config
	// NOTE: If this is done, main() returns and that means the schedule will not be processed. This is typically
	// for other packages that want to import Social Harvest. If a server is not desired, simply ensure whatever port
	// Social Harvest runs on is has appropriate firewall settings. Alternatively, we could prevent main() from returning,
	// but that would lead to a more confusing configuration.
	// TODO: Think about accepting command line arguments for adhoc harvesting.
	if !socialHarvest.Config.Server.Disabled {
		restMiddleware := []rest.Middleware{}

		// If additional origins were allowed for CORS, handle them
		if len(socialHarvest.Config.Server.Cors.AllowedOrigins) > 0 {
			restMiddleware = append(restMiddleware,
				&rest.CorsMiddleware{
					RejectNonCorsRequests: false,
					OriginValidator: func(origin string, request *rest.Request) bool {
						for _, allowedOrigin := range socialHarvest.Config.Server.Cors.AllowedOrigins {
							// If the request origin matches one of the allowed origins, return true
							if origin == allowedOrigin {
								return true
							}
						}
						return false
					},
					AllowedMethods: []string{"GET", "POST", "PUT"},
					AllowedHeaders: []string{
						"Accept", "Content-Type", "X-Custom-Header", "Origin"},
					AccessControlAllowCredentials: true,
					AccessControlMaxAge:           3600,
				},
			)
		}
		// If api keys are defined, setup basic auth (any key listed allows full access, there are no roles for now, this is just very basic auth)
		if len(socialHarvest.Config.Server.AuthKeys) > 0 {
			restMiddleware = append(restMiddleware,
				&BasicAuthMw{
					Realm: "Social Harvest API",
					Key:   "",
				},
			)
		}

		handler := rest.ResourceHandler{
			EnableRelaxedContentType: true,
			PreRoutingMiddlewares:    restMiddleware,
		}
		err := handler.SetRoutes(
			&rest.Route{"GET", "/schedule/read", ShowSchedule},
			&rest.Route{"GET", "/config/read", ShowSocialHarvestConfig},
			&rest.Route{"GET", "/territory/list", TerritoryList},
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
		log.Println("Social Harvest (harvester) API listening on port " + p)
		if socialHarvest.Config.Debug.Bugsnag.ApiKey != "" {
			log.Println(http.ListenAndServe(":"+p, bugsnag.Handler(&handler)))
		} else {
			log.Fatal(http.ListenAndServe(":"+p, &handler))
		}
	}
}
