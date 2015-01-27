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

var appVersion = "0.15.0-alpha"
var confFile string
var socialHarvest = config.SocialHarvest{}

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
	res.Links["reload"] = config.HypermediaLink{
		Href: "/config/reload{?original}",
	}
	res.Links["write"] = config.HypermediaLink{
		Href: "/config/write",
	}
	res.Data["config"] = socialHarvest.Config.Harvest
	res.Success()
	w.WriteJson(res.End())
}

// Reloads the configuration from the available file on disk
func ReloadSocialHarvestConfig(w rest.ResponseWriter, r *rest.Request) {
	res := config.NewHypermediaResource()
	res.Links["self"] = config.HypermediaLink{
		Href: "/config/reload{?original}",
	}
	res.Links["read"] = config.HypermediaLink{
		Href: "/config/read",
	}
	res.Links["write"] = config.HypermediaLink{
		Href: "/config/write",
	}

	// If the original configuration (passed via --conf flag or the default social-harvest-conf.json) is desired, pass true to setConfig() by looking
	// for an "?original=true" in the route. By default, it will look for an updated config in the "sh-data" directory.
	queryParams := r.URL.Query()
	original := false
	if len(queryParams["original"]) > 0 {
		original = true
		res.Meta.Message = "Original configuration loaded."
	}
	setConfig(original)

	// Return the updated config
	res.Data["config"] = socialHarvest.Config.Harvest
	res.Success()
	w.WriteJson(res.End())
}

// Writes a new JSON configuration file into "sh-data" (a reload should be called afer this unless there's an error) so that the original is preserved
func WriteSocialHarvestConfig(w rest.ResponseWriter, r *rest.Request) {
	res := config.NewHypermediaResource()
	res.Links["self"] = config.HypermediaLink{
		Href: "/config/write",
	}
	res.Links["read"] = config.HypermediaLink{
		Href: "/config/read",
	}
	res.Links["reload"] = config.HypermediaLink{
		Href: "/config/reload{?original}",
	}

	// TODO: Take JSON from request and create new SocialHarvestConf struct with it.
	// Then save to disk in "sh-data" path.
	// Validate it? Aside from being able to convert it to a struct... Make sure it has a certain number of fields?

	var c = config.SocialHarvestConf{}
	err := r.DecodeJsonPayload(&c)
	if err != nil {
		//rest.Error(w, err.Error(), http.StatusInternalServerError)
		//return
		res.Meta.Message = "Invalid configuration."
		w.WriteJson(res.End())
	}
	if config.SaveConfig(c) {
		res.Success()
	}

	w.WriteJson(res.End())
}

// Returns information about the currently configured database, if it's reachable, etc.
func DatabaseInfo(w rest.ResponseWriter, r *rest.Request) {
	res := config.NewHypermediaResource()
	res.Links["database:info"] = config.HypermediaLink{
		Href: "/database/info",
	}

	if socialHarvest.Database.Postgres != nil {
		res.Data["type"] = "postgres"
		// SELECT * FROM has_database_privilege('username', 'database', 'connect');
		// var r struct {
		// 	hasAccess string `db:"has_database_privilege" json:"has_database_privilege"`
		// }
		//err := socialHarvest.Database.Postgres.Get(&r, "SELECT * FROM has_database_privilege("+socialHarvest.Config.Database.User+", "+socialHarvest.Config.Database.Database+", 'connect')")
		//res.Data["r"] = r
		//res.Data["err"] = err
		res.Data["hasAccess"] = socialHarvest.Database.HasAccess()
	}
	if socialHarvest.Database.InfluxDB != nil {
		res.Data["type"] = "infxludb"
	}

	res.Data["configuredType"] = socialHarvest.Config.Database.Type

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
		for _, key := range socialHarvest.Config.HarvesterServer.AuthKeys {
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
	// NOTE: For now the schedule will always be set by an entire config reload, but in the future allowing the schedule to be updated without an entire config reload would be nice.
	// TODO: ^^^^
	for _, territory := range socialHarvest.Config.Harvest.Territories {
		if territory.Schedule.Everything.Accounts != "" {
			socialHarvest.Schedule.Cron.AddFunc(territory.Schedule.Everything.Accounts, HarvestAllAccounts, "Harvesting all accounts - "+territory.Schedule.Everything.Accounts)
		}
		if territory.Schedule.Everything.Content != "" {
			socialHarvest.Schedule.Cron.AddFunc(territory.Schedule.Everything.Content, HarvestAllContent, "Harvesting all content - "+territory.Schedule.Everything.Content)
		}
	}

	// Set cron tasks for creating partitions in Postgres
}

// Helper function to get the name of a function (primarily used to show scheduled tasks)
func getFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

// Sets (or updates) the configuration for the application. If true is passed, then it won't look for an updated config in the "sh-data" path. The original config will be used instead.
func setConfig(original bool) error {
	var err error
	var f *os.File
	// First try to load the config file from the "sh-data" path where it would be if it was updated via the API. Unless the original config is to be loaded.
	if !original {
		f, err = os.Open("./sh-data/social-harvest-conf.json")
		if !os.IsExist(err) {
			// If that fails, go back to the original "confFile" path given at run time using the "--conf" flag (or the default value).
			// In this case, the config may have been updated on disk and perhaps someone wanted to reload the config without restarting the application.
			// Of course there is no reload config from the command line. This call must be made from the RESTful API for now.
			f, err = os.Open(confFile)
		}
	} else {
		f, err = os.Open(confFile)
	}

	decoder := json.NewDecoder(f)
	c := config.SocialHarvestConf{}
	err = decoder.Decode(&c)
	if err != nil {
		log.Println("config decode error:", err)
		return err
	}
	socialHarvest.Config = c

	// Setup Bugsnag (first), profiling, etc.
	if socialHarvest.Config.Debug.Bugsnag.ApiKey != "" {
		bugsnag.Configure(bugsnag.Configuration{
			APIKey:          socialHarvest.Config.Debug.Bugsnag.ApiKey,
			ReleaseStage:    socialHarvest.Config.Debug.Bugsnag.ReleaseStage,
			ProjectPackages: []string{"main", "github.com/SocialHarvest/harvester/*"},
			AppVersion:      appVersion,
		})
	}

	// Check the data directory, copy data needed by the harvester for various analysis of harvested data.
	// Note: The data will only be copied if it doesn't exist already.
	// TODO: Allow configuration to define this data and when that updates...new files get put into place.
	config.CheckDataDir()
	config.CopyTrainingData()

	// Continue configuration
	socialHarvest.Database = config.NewDatabase(socialHarvest.Config)
	socialHarvest.Schedule = config.NewSchedule(socialHarvest.Config)

	// this gets the configuration and the database. TODO: Make database optional
	harvester.New(socialHarvest.Config, socialHarvest.Database)
	// Load new gender data from CSV files for detecting gender (this is callable so it can be changed during runtime)
	// TODO: Considerations with an asset system.
	harvester.NewGenderData("./sh-data/census-female-names.csv", "./sh-data/census-male-names.csv")

	// Set the initial schedule (can be changed via API if available)
	setInitialSchedule()

	return err
}

// Main - initializes, configures, and sets routes for API
func main() {
	// Optionally allow a config JSON file to be passed via command line
	flag.StringVar(&confFile, "conf", "social-harvest-conf.json", "Path to the Social Harvest configuration file.")
	flag.Parse()

	// Set the configuration, DB client, etc. so that it is available to other stuff.
	cErr := setConfig(false)
	if cErr != nil {
		log.Fatalln("Failed to load the harvester configuration.")
	}
	// NOTE: A database is optional for Social Harvest (harvested data can be logged for use with Fluentd for example)
	if socialHarvest.Database.Postgres != nil {
		defer socialHarvest.Database.Postgres.Close()
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

	// Immedate calls to use for testing during development
	// Search Facebook public posts using keywords in Social Harvest config
	go FacebookPublicMessagesByKeyword()
	//Search Facebook public feeds using account ids in Social Harvest config
	go FacebookMessagesByAccount()
	//Search Twitter using keywords in Social Harvest config
	go TwitterPublicMessagesByKeyword()
	go TwitterPublicMessagesByAccount()
	// Search Instagram
	go InstagramMediaByKeyword()
	go GooglePlusActivitieByKeyword()
	go GooglePlusActivitieByAccount()
	go HarvestAllContent()
	go HarvestAllAccounts()

	//TODO: Continue with this...
	//socialHarvest.Database.CreatePartitionTable("messages")

	// The RESTful API harvester server can be completely disabled by setting {"harvesterServer":{"disabled": true}} in the config.
	// NOTE: The actual API server (if running) can not be updated (port changes, etc.) without the harvester application being restarted.
	// TODO: Think about accepting command line arguments for adhoc harvesting (useful if the server is disabled because main() will return then).
	if !socialHarvest.Config.HarvesterServer.Disabled {
		restMiddleware := []rest.Middleware{}

		// If additional origins were allowed for CORS, handle them
		if len(socialHarvest.Config.HarvesterServer.Cors.AllowedOrigins) > 0 {
			restMiddleware = append(restMiddleware,
				&rest.CorsMiddleware{
					RejectNonCorsRequests: false,
					OriginValidator: func(origin string, request *rest.Request) bool {
						for _, allowedOrigin := range socialHarvest.Config.HarvesterServer.Cors.AllowedOrigins {
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
		if len(socialHarvest.Config.HarvesterServer.AuthKeys) > 0 {
			restMiddleware = append(restMiddleware,
				&BasicAuthMw{
					Realm: "Social Harvest (harvester) API",
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
			&rest.Route{"POST", "/config/write", WriteSocialHarvestConfig},
			&rest.Route{"GET", "/config/reload", ReloadSocialHarvestConfig},
			&rest.Route{"GET", "/database/info", DatabaseInfo},
			&rest.Route{"GET", "/territory/list", TerritoryList},
		)
		if err != nil {
			log.Fatal(err)
		}

		// Allow the port to be configured (we need it as a string, but let the config define an int)
		p := strconv.Itoa(socialHarvest.Config.HarvesterServer.Port)
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
