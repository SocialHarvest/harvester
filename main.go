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
	"github.com/advancedlogic/GoOse"
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
	"strings"
	"time"
)

var socialHarvest = config.SocialHarvest{}

var harvestChannel = make(chan interface{})

// --------- Route functions (maybe move into various go files for organization)

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

// --------- API: Territory end points ---------

// Territory aggregates (gender, language, etc.) shows a breakdown and count of various values and their percentage of total
func TerritoryAggregateData(w rest.ResponseWriter, r *rest.Request) {
	res := setTerritoryLinks("territory:aggregate")

	territory := r.PathParam("territory")
	series := r.PathParam("series")
	queryParams := r.URL.Query()

	timeFrom := ""
	if len(queryParams["from"]) > 0 {
		timeFrom = queryParams["from"][0]
	}
	timeTo := ""
	if len(queryParams["to"]) > 0 {
		timeTo = queryParams["to"][0]
	}
	network := ""
	if len(queryParams["network"]) > 0 {
		timeTo = queryParams["network"][0]
	}

	limit := 0
	if len(queryParams["limit"]) > 0 {
		parsedLimit, err := strconv.Atoi(queryParams["limit"][0])
		if err == nil {
			limit = parsedLimit
		}
	}

	fields := []string{}
	if len(queryParams["fields"]) > 0 {
		fields = strings.Split(queryParams["fields"][0], ",")
		// trim any white space
		for i, val := range fields {
			fields[i] = strings.Trim(val, " ")
		}
	}

	if territory != "" && series != "" && len(fields) > 0 {
		params := config.CommonQueryParams{
			Series:    series,
			Territory: territory,
			Network:   network,
			From:      timeFrom,
			To:        timeTo,
			Limit:     uint(limit),
		}

		var total config.ResultCount
		res.Data["aggregate"], total = socialHarvest.Database.FieldCounts(params, fields)
		res.Data["total"] = total.Count
		res.Success()
	}

	w.WriteJson(res.End())
}

// This works practically the same way as TerritoryAggregateData, only instead it is a streaming API response that returns multiple slices of time.
// This makes light work of a potential mount of data and streams the response back to the client for progressive loading. For example, a time series
// graph can be drawn using JavaScript in an animated sort of fashion. As more data came in, the chart would change. For smaller ranges this should
// return quite quickly and perhaps not even be necessary...But for large date ranges (or lots of data) this could work around performance issues.
// Alternative to this, would be storing aggregate report data in a database somewhere, but that limits the use of the "resolution" option.
// This method makes for greater flexibility and less database storage. Though it may need to be re-evaluated in the future if there proves to be
// too much data, resulting in slow loads.
// NOTE: This returns sparse data and an aggregate count of all field values, so it may be difficult to parse on the front-end.
func TerritoryTimeSeriesAggregateData(w rest.ResponseWriter, r *rest.Request) {
	territory := r.PathParam("territory")
	series := r.PathParam("series")
	queryParams := r.URL.Query()

	timeFrom := ""
	if len(queryParams["from"]) > 0 {
		timeFrom = queryParams["from"][0]
	}
	timeTo := ""
	if len(queryParams["to"]) > 0 {
		timeTo = queryParams["to"][0]
	}
	network := ""
	if len(queryParams["network"]) > 0 {
		network = queryParams["network"][0]
	}

	// likely not used here
	limit := 0
	if len(queryParams["limit"]) > 0 {
		parsedLimit, err := strconv.Atoi(queryParams["limit"][0])
		if err == nil {
			limit = parsedLimit
		}
	}

	// in minutes
	resolution := 0
	if len(queryParams["resolution"]) > 0 {
		parsedResolution, err := strconv.Atoi(queryParams["resolution"][0])
		if err == nil {
			resolution = parsedResolution
		}
	}
	// TODO: limit the resolution? 5, 10, 15, 30, 45, 60, 180, 1440 etc.? (minutes, hour, 3 hours, day, etc.)
	// TODO: maybe also add timezone? (this would require changes all over to date picker, other queries, etc.)

	fields := []string{}
	if len(queryParams["fields"]) > 0 {
		fields = strings.Split(queryParams["fields"][0], ",")
		// trim any white space
		for i, val := range fields {
			fields[i] = strings.Trim(val, " ")
		}
	}

	//log.Println(resolution)
	if resolution != 0 && territory != "" && series != "" && len(fields) > 0 {
		// only accepting days for now - not down to minutes or hours (yet)
		tF, _ := time.Parse("2006-01-02", timeFrom)
		tT, _ := time.Parse("2006-01-02", timeTo)

		timeRange := tT.Sub(tF)
		//totalRangeMinutes := int(timeRange.Minutes())
		periodsInRange := int(timeRange.Minutes() / float64(resolution))

		params := config.CommonQueryParams{
			Series:    series,
			Territory: territory,
			Network:   network,
			From:      timeFrom,
			To:        timeTo,
			Limit:     uint(limit),
		}

		w.Header().Set("Content-Type", "application/json")
		var aggregate []config.ResultAggregateFields
		//var total config.ResultCount
		for i := 0; i < periodsInRange; i++ {
			params.From = tF.Format("2006-01-02 15:04:05")
			tF = tF.Add(time.Duration(resolution) * time.Minute)
			params.To = tF.Format("2006-01-02 15:04:05")

			aggregate, _ = socialHarvest.Database.FieldCounts(params, fields)
			w.WriteJson(aggregate)
			w.(http.ResponseWriter).Write([]byte("\n"))
			// Flush the buffer to client immediately
			// (for most cases, this stream will be quick and short - just how we like it. for the more crazy requests, it may take a little while and that's ok too)
			w.(http.Flusher).Flush()
		}
	}
}

// Returns a simple count based on various conditions.
func TerritoryCountData(w rest.ResponseWriter, r *rest.Request) {
	res := setTerritoryLinks("territory:count")

	territory := r.PathParam("territory")
	series := r.PathParam("series")
	field := r.PathParam("field")
	queryParams := r.URL.Query()

	timeFrom := ""
	if len(queryParams["from"]) > 0 {
		timeFrom = queryParams["from"][0]
	}
	timeTo := ""
	if len(queryParams["to"]) > 0 {
		timeTo = queryParams["to"][0]
	}
	fieldValue := ""
	if len(queryParams["fieldValue"]) > 0 {
		fieldValue = queryParams["fieldValue"][0]
	}
	network := ""
	if len(queryParams["network"]) > 0 {
		network = queryParams["network"][0]
	}

	params := config.CommonQueryParams{
		Series:    series,
		Territory: territory,
		Field:     field,
		Network:   network,
		From:      timeFrom,
		To:        timeTo,
	}

	var count config.ResultCount
	count = socialHarvest.Database.Count(params, fieldValue)
	res.Data["count"] = count.Count
	res.Meta.From = count.TimeFrom
	res.Meta.To = count.TimeTo

	res.Success()
	w.WriteJson(res.End())
}

// Returns a simple count based on various conditions in a streaming time series.
func TerritoryTimeseriesCountData(w rest.ResponseWriter, r *rest.Request) {
	territory := r.PathParam("territory")
	series := r.PathParam("series")
	field := r.PathParam("field")
	queryParams := r.URL.Query()

	timeFrom := ""
	if len(queryParams["from"]) > 0 {
		timeFrom = queryParams["from"][0]
	}
	timeTo := ""
	if len(queryParams["to"]) > 0 {
		timeTo = queryParams["to"][0]
	}
	fieldValue := ""
	if len(queryParams["fieldValue"]) > 0 {
		fieldValue = queryParams["fieldValue"][0]
	}
	network := ""
	if len(queryParams["network"]) > 0 {
		network = queryParams["network"][0]
	}

	params := config.CommonQueryParams{
		Series:    series,
		Territory: territory,
		Field:     field,
		Network:   network,
		From:      timeFrom,
		To:        timeTo,
	}

	// in minutes
	resolution := 0
	if len(queryParams["resolution"]) > 0 {
		parsedResolution, err := strconv.Atoi(queryParams["resolution"][0])
		if err == nil {
			resolution = parsedResolution
		}
	}

	if resolution != 0 && territory != "" && series != "" {
		// only accepting days for now - not down to minutes or hours (yet)
		tF, _ := time.Parse("2006-01-02", timeFrom)
		tT, _ := time.Parse("2006-01-02", timeTo)

		timeRange := tT.Sub(tF)
		//totalRangeMinutes := int(timeRange.Minutes())
		periodsInRange := int(timeRange.Minutes() / float64(resolution))

		w.Header().Set("Content-Type", "application/json")
		var count config.ResultCount
		for i := 0; i < periodsInRange; i++ {
			params.From = tF.Format("2006-01-02 15:04:05")
			tF = tF.Add(time.Duration(resolution) * time.Minute)
			params.To = tF.Format("2006-01-02 15:04:05")

			count = socialHarvest.Database.Count(params, fieldValue)
			w.WriteJson(count)
			w.(http.ResponseWriter).Write([]byte("\n"))
			// Flush the buffer to client immediately
			// (for most cases, this stream will be quick and short - just how we like it. for the more crazy requests, it may take a little while and that's ok too)
			w.(http.Flusher).Flush()
		}
	}

}

// API: Returns the messages (paginated) for a territory with the ability to filter by question or not, etc.
func TerritoryMessages(w rest.ResponseWriter, r *rest.Request) {
	res := setTerritoryLinks("territory:messages")

	territory := r.PathParam("territory")
	queryParams := r.URL.Query()

	timeFrom := ""
	if len(queryParams["from"]) > 0 {
		timeFrom = queryParams["from"][0]
	}
	timeTo := ""
	if len(queryParams["to"]) > 0 {
		timeTo = queryParams["to"][0]
	}
	network := ""
	if len(queryParams["network"]) > 0 {
		network = queryParams["network"][0]
	}
	// Limit and Skip
	limit := uint(100)
	if len(queryParams["limit"]) > 0 {
		l, lErr := strconv.ParseUint(queryParams["limit"][0], 10, 64)
		if lErr == nil {
			limit = uint(l)
		}
		if limit > 100 {
			limit = 100
		}
		if limit < 1 {
			limit = 1
		}
	}
	skip := uint(0)
	if len(queryParams["skip"]) > 0 {
		sk, skErr := strconv.ParseUint(queryParams["skip"][0], 10, 64)
		if skErr == nil {
			skip = uint(sk)
		}
		if skip < 0 {
			skip = 0
		}
	}
	// Always passed as field,direction (dashes aren't allowed)
	sort := "time,desc"
	if len(queryParams["sort"]) > 0 {
		sort = queryParams["sort"][0]
	}

	// Build the conditions
	var conditions = config.MessageConditions{}

	// For a LIKE% match (MongoDb regex)
	if len(queryParams["search"]) > 0 {
		conditions.Search = queryParams["search"][0]
	}

	// Condition for questions
	if len(queryParams["questions"]) > 0 {
		conditions.IsQuestion = 1
	}
	// Gender condition
	if len(queryParams["gender"]) > 0 {
		conditions.Gender = queryParams["gender"][0]
	}
	// Language condition
	if len(queryParams["lang"]) > 0 {
		conditions.Lang = queryParams["lang"][0]
	}
	// Country condition
	if len(queryParams["country"]) > 0 {
		conditions.Country = queryParams["country"][0]
	}
	// Geohash condition (nearby)
	if len(queryParams["geohash"]) > 0 {
		conditions.Geohash = queryParams["geohash"][0]
	}

	params := config.CommonQueryParams{
		Series:    "messages",
		Territory: territory,
		Network:   network,
		From:      timeFrom,
		To:        timeTo,
		Limit:     limit,
		Skip:      skip,
		Sort:      sort,
	}

	messages, total, skip, limit := socialHarvest.Database.Messages(params, conditions)
	res.Data["messages"] = messages
	res.Data["total"] = total
	res.Data["limit"] = limit
	res.Data["skip"] = skip

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
	res.Links["territory:count"] = config.HypermediaLink{
		Href: "/territory/count/{territory}/{series}/{field}{?from,to,network,fieldValue}",
	}
	res.Links["territory:timeseries-count"] = config.HypermediaLink{
		Href: "/territory/timeseries/count/{territory}/{series}/{field}{?from,to,network,fieldValue}",
	}
	res.Links["territory:aggregate"] = config.HypermediaLink{
		Href: "/territory/aggregate/{territory}/{series}{?from,to,network,fields}",
	}
	res.Links["territory:timeseries-aggregate"] = config.HypermediaLink{
		Href: "/territory/timeseries/aggregate/{territory}/{series}{?from,to,network,fields,resolution}",
	}
	res.Links["territory:messages"] = config.HypermediaLink{
		Href: "/territory/messages/{territory}{?from,to,limit,skip,network,lang,country,geohash,gender,questions,sort,search}",
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

// --------- API: Utility end points ---------

// Retrieves information to provide a summary about a give URL, specifically articles/blog posts.
// TODO: Make this more robust (more details, videos, etc.). Some of this may eventually also go into the harvest.
// TODO: Likely fork this package and add in some of the things I did for Virality Score in order to get even more data.
func LinkDetails(w rest.ResponseWriter, r *rest.Request) {
	res := config.NewHypermediaResource()
	res.Links["self"] = config.HypermediaLink{
		Href: "/link/details{?url}",
	}

	queryParams := r.URL.Query()
	if len(queryParams["url"]) > 0 {
		g := goose.New()
		article := g.ExtractFromUrl(queryParams["url"][0])

		res.Data["title"] = article.Title
		res.Data["published"] = article.PublishDate
		res.Data["favicon"] = article.MetaFavicon
		res.Data["domain"] = article.Domain
		res.Data["description"] = article.MetaDescription
		res.Data["keywords"] = article.MetaKeywords
		res.Data["content"] = article.CleanedText
		res.Data["url"] = article.FinalUrl
		res.Data["image"] = article.TopImage
		res.Data["movies"] = article.Movies
		res.Success()
	}

	w.WriteJson(res.End())
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
	appVersion := "0.10.0-preview"

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
	if socialHarvest.Database.Session != nil {
		defer socialHarvest.Database.Session.Close()
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
	//go HarvestAllContent()
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
			// NOTE: The routes with "timeseries" are streams.
			// Simple aggregates for a territory
			//&rest.Route{"GET", "/territory/aggregate/:territory/:series", TerritoryAggregateData},
			//&rest.Route{"GET", "/territory/timeseries/aggregate/:territory/:series", TerritoryTimeSeriesAggregateData},
			// Simple counts for a territory
			//&rest.Route{"GET", "/territory/count/:territory/:series/:field", TerritoryCountData},
			//&rest.Route{"GET", "/territory/timeseries/count/:territory/:series/:field", TerritoryTimeseriesCountData},
			// Messages for a territory
			//&rest.Route{"GET", "/territory/messages/:territory", TerritoryMessages},

			// FOR TESTING. TODO: REMOVE
			//&rest.Route{"GET", "/test", TestQueries},

			// A utility route to help get some details about any given external web page
			&rest.Route{"GET", "/link/details", LinkDetails},
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
		if socialHarvest.Config.Debug.Bugsnag.ApiKey != "" {
			log.Println(http.ListenAndServe(":"+p, bugsnag.Handler(&handler)))
		} else {
			log.Fatal(http.ListenAndServe(":"+p, &handler))
		}
	}
}

// TODO REMOVE ME
func TestQueries(w rest.ResponseWriter, r *rest.Request) {
	queryParams := r.URL.Query()

	timeFrom := ""
	if len(queryParams["from"]) > 0 {
		timeFrom = queryParams["from"][0]
	}
	timeTo := ""
	if len(queryParams["to"]) > 0 {
		timeTo = queryParams["to"][0]
	}

	network := ""
	if len(queryParams["network"]) > 0 {
		network = queryParams["network"][0]
	}

	// likely not used here
	limit := 0
	if len(queryParams["limit"]) > 0 {
		parsedLimit, err := strconv.Atoi(queryParams["limit"][0])
		if err == nil {
			limit = parsedLimit
		}
	}

	territory := "SocialMedia"
	series := "shared_links"

	// in minutes (one day by default)
	resolution := 1440
	if len(queryParams["resolution"]) > 0 {
		parsedResolution, err := strconv.Atoi(queryParams["resolution"][0])
		if err == nil {
			resolution = parsedResolution
		}
	}
	// TODO: limit the resolution? 5, 10, 15, 30, 45, 60, 180, 1440 etc.? (minutes, hour, 3 hours, day, etc.)
	// TODO: maybe also add timezone? (this would require changes all over to date picker, other queries, etc.)

	//log.Println(resolution)
	if resolution != 0 && territory != "" && series != "" {
		// only accepting days for now - not down to minutes or hours (yet)
		tF, _ := time.Parse("2006-01-02", timeFrom)
		tT, _ := time.Parse("2006-01-02", timeTo)

		timeRange := tT.Sub(tF)
		//totalRangeMinutes := int(timeRange.Minutes())
		periodsInRange := int(timeRange.Minutes() / float64(resolution))

		params := config.CommonQueryParams{
			Series:    series,
			Territory: territory,
			Network:   network,
			From:      timeFrom,
			To:        timeTo,
			Limit:     uint(limit),
		}

		//w.Header().Set("Content-Type", "application/json")

		for i := 0; i < periodsInRange; i++ {
			params.From = tF.Format("2006-01-02 15:04:05")
			tF = tF.Add(time.Duration(resolution) * time.Minute)
			params.To = tF.Format("2006-01-02 15:04:05")

			result := socialHarvest.Database.TestQueries(params)

			w.WriteJson(result)
			w.(http.ResponseWriter).Write([]byte("\n"))
			// Flush the buffer to client immediately
			// (for most cases, this stream will be quick and short - just how we like it. for the more crazy requests, it may take a little while and that's ok too)
			w.(http.Flusher).Flush()
		}
	}

	//w.WriteJson(result)
	//w.(http.ResponseWriter).Write([]byte("\n"))
	// Flush the buffer to client immediately
	// (for most cases, this stream will be quick and short - just how we like it. for the more crazy requests, it may take a little while and that's ok too)
	//w.(http.Flusher).Flush()
}
