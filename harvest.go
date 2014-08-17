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
	//"encoding/json"
	"github.com/SocialHarvest/harvester/lib/config"
	"github.com/SocialHarvest/harvester/lib/harvester"
	"log"
	"net/url"
	"strconv"
	//"sync"
	"time"
)

// All functions for harvesting should follow the same format.
// <Network>PublicMessagesBy<Criteria>
// <Network>MessagesBy<Criteria>
// <Network>GrowthBy<Criteria>
// Criteria is typically going to be "Keyword" and "Account" but there may be other values in the future.
// The following functions are intended to be scheduled (and perhaps even called by other packages or via the RESTful API).
// ...and that way, the functions that are going to save to the database, log, etc. won't interfere.

// TODO: Look into: http://labix.org/pipe
// Pipe everything(?) through the specified scripts/commands before saving and writing to log files.
// This was the original thinking for filters, but Fluentd may be enough? If this is introduced, it will be in the future (lower priority).

// Harvest Facebook publicly accessible posts by searching keyword criteria
func FacebookPublicMessagesByKeyword() {
	params := harvester.FacebookParams{}
	posts := []harvester.FacebookPost{}

	for _, territory := range socialHarvest.Config.Harvest.Territories {
		// If different credentials were set for the territory, this will find and set them
		harvester.NewFacebookTerritoryCredentials(territory.Name)

		if len(territory.Content.Options.Lang) > 0 {
			params.Lang = territory.Content.Options.Lang
		}

		// Search all keywords
		if len(territory.Content.Keywords) > 0 {
			for _, keyword := range territory.Content.Keywords {
				//log.Print("Searching for: " + keyword)
				params.Type = "post"
				params.Q = keyword
				// A globally set limit in the Social Harvest config (or default of "100")
				if territory.Limits.ResultsPerPage != "" {
					params.Limit = territory.Limits.ResultsPerPage
				} else {
					params.Limit = "100"
				}

				lastIdHarvested := ""
				pagesHarvested := 1
				itemsHarvested := 0
				lastTimeHarvested := time.Now() // set something...but this should be changed by the data being harvested

				// Fetch all pages (it keeps going until there are no more, but that could be problematic for API rate limits - so in the Social Harvest config, a limit can be put on number of pages returned)
				for {
					// Note: The "since" seems to get removed in the "next" pagination link.
					// It would have worked perfectly and stopped if they held on to it as a limiter. Now, we need to hold on to it in the harvester and watch.
					// When results start coming in that have a time older than this "since" value - break the loop (also note, configuration can limit pages too).
					// However. If nothing has truly been posted since the last harvest, then no results will be returned when passing "since" and that will help a little.
					// So always pass it. Since we only get the "next" page, we don't need to change it (and it does help particularly with account feeds).
					lastHarvestTime := socialHarvest.Database.GetLastHarvestTime(territory.Name, "facebook", "FacebookPublicMessagesByKeyword", keyword)
					sinceStr := ""
					if !lastHarvestTime.IsZero() {
						sinceTimeUnix := lastHarvestTime.Unix()
						if sinceTimeUnix > 0 {
							sinceStr = strconv.FormatInt(sinceTimeUnix, 10)
						}
					}
					if sinceStr != "" {
						params.Since = sinceStr
					}

					// TODO: Refactor. We shouldn't need this second FacebookPostsOut() function.
					// I know that it was done because there were two different search API endpoints and both returned different responses.
					// So rather than some duplicate code everything was first mapped to a "post" and then that post was mapped to a social harvest message...Something like that.
					// Refactor it though. For some reason Twitter is so sparkly clean. I want them all to be like that.
					posts, params = harvester.FacebookSearch(params)
					// Process the data (number of items, the latest item's time and id will be returned so the next harvest can pick up where it left off)
					items, lastId, lastTime := harvester.FacebookPostsOut(posts, territory.Name)
					itemsHarvested += items
					lastIdHarvested = lastId
					lastTimeHarvested = lastTime

					// Just check Until, Since would be for going backwards in the pagination which we don't need to do here...
					if params.Until == "0" {
						log.Println("done for " + keyword)
						// Save the last harvest time for this task with the keyword
						socialHarvest.Database.SetLastHarvestTime(territory.Name, "facebook", "FacebookPublicMessagesByKeyword", keyword, lastTimeHarvested, lastIdHarvested, itemsHarvested)
						break
					}

					// Limit the number of pages of results if specified in the Social Harvest Config
					if territory.Limits.MaxResultsPages != 0 {
						if pagesHarvested >= territory.Limits.MaxResultsPages {
							log.Println("done for " + keyword)
							socialHarvest.Database.SetLastHarvestTime(territory.Name, "facebook", "FacebookPublicMessagesByKeyword", keyword, lastTimeHarvested, lastIdHarvested, itemsHarvested)
							break
						}
					}
					// Count pages of results
					pagesHarvested++
				}
			}
		}
	}
}

// Harvest Facebook publicly accessible posts from a specific account (user or page)
func FacebookMessagesByAccount() {
	params := harvester.FacebookParams{}
	posts := []harvester.FacebookPost{}

	for _, territory := range socialHarvest.Config.Harvest.Territories {
		if len(territory.Accounts.Facebook) > 0 {
			// If different credentials were set for the territory, this will find and set them
			harvester.NewFacebookTerritoryCredentials(territory.Name)

			//log.Print(territory.Accounts.Facebook)
			for _, account := range territory.Accounts.Facebook {
				//log.Print("Getting feed for: " + account)
				// A globally set limit in the Social Harvest config (or default of "100")
				if territory.Limits.ResultsPerPage != "" {
					params.Limit = territory.Limits.ResultsPerPage
				} else {
					params.Limit = "100"
				}

				lastIdHarvested := ""
				itemsHarvested := 0
				pagesHarvested := 1
				lastTimeHarvested := time.Now() // set something...but this should be changed by the data being harvested
				for {
					// Pass the last harvest time in for "since" to help avoid duplicate data in API requests
					// (helps a lot more in this case since individual accounts don't post as frequently as the masses in a public post search).
					lastHarvestTime := socialHarvest.Database.GetLastHarvestTime(territory.Name, "facebook", "FacebookMessagesByAccount", account)
					if !lastHarvestTime.IsZero() {
						params.Since = strconv.FormatInt(lastHarvestTime.Unix(), 10)
					}

					posts, params = harvester.FacebookFeed(account, params)
					items, lastId, lastTime := harvester.FacebookPostsOut(posts, territory.Name)
					itemsHarvested += items
					lastIdHarvested = lastId
					lastTimeHarvested = lastTime

					// Just check Until, Since would be for going backwards in the pagination which we don't need to do here...
					if params.Until == "0" {
						log.Println("done for " + account)
						// Save the last harvest time for this task with the keyword
						socialHarvest.Database.SetLastHarvestTime(territory.Name, "facebook", "FacebookMessagesByAccount", account, lastTimeHarvested, lastIdHarvested, itemsHarvested)
						break
					}

					// Limit the number of pages of results if specified in the Social Harvest config
					if territory.Limits.MaxResultsPages != 0 {
						if pagesHarvested >= territory.Limits.MaxResultsPages {
							log.Println("done for " + account)
							socialHarvest.Database.SetLastHarvestTime(territory.Name, "facebook", "FacebookMessagesByAccount", account, lastTimeHarvested, lastIdHarvested, itemsHarvested)
							break
						}
					}
					// Count pages of results
					pagesHarvested++
				}

			}
		}
	}
}

// Track Facebook account changes for public pages (without extended permissions, we can't determine personal account growth/number of friends)
func FacebookGrowthByAccount() {

}

// Searches Twitter for status updates by territory keyword criteria
func TwitterPublicMessagesByKeyword() {
	for _, territory := range socialHarvest.Config.Harvest.Territories {
		// If different credentials were set for the territory, this will find and set them
		harvester.NewTwitterTerritoryCredentials(territory.Name)

		// Build params for search
		params := url.Values{}
		params.Set("include_entities", "true")
		if len(territory.Content.Options.Lang) > 0 {
			params.Set("lang", territory.Content.Options.Lang)
		}
		if len(territory.Content.Options.TwitterGeocode) > 0 {
			params.Set("geocode", territory.Content.Options.TwitterGeocode)
		}

		// Search all keywords
		if len(territory.Content.Keywords) > 0 {
			for _, keyword := range territory.Content.Keywords {
				log.Print("Searching for: " + keyword)

				// A globally set limit in the Social Harvest config (or default of "100")
				if territory.Limits.ResultsPerPage != "" {
					params.Set("count", territory.Limits.ResultsPerPage)
				} else {
					params.Set("count", "100")
				}

				// Keep track of the last id harvested, the number of items harvested, etc. This information will be returend from `harvester.TwitterSearch()`
				// on each call in the loop. We'll just keep incrementing the items and overwriting the last id and time. This information then gets saved to the harvest series.
				// So then on the next harvest, we can see where we left off so we don't request the same data again from the API. This doesn't guarantee the prevention of dupes
				// of course, but it does decrease unnecessary API calls which helps with rate limiting and efficiency.
				harvestState := config.HarvestState{
					LastId:         "",
					LastTime:       time.Now(),
					PagesHarvested: 1,
					ItemsHarvested: 0,
				}

				// Fetch all pages (it keeps going until there are no more, but that could be problematic for API rate limits - so in the Social Harvest config, a limit can be put on number of pages returned)
				for {
					// Note: The "since" seems to get removed in the "next" pagination link.
					// It would have worked perfectly and stopped if they held on to it as a limiter. Now, we need to hold on to it in the harvester and watch.
					// When results start coming in that have a time older than this "since" value - break the loop (also note, configuration can limit pages too).
					// However. If nothing has truly been posted since the last harvest, then no results will be returned when passing "since" and that will help a little.
					// So always pass it. Since we only get the "next" page, we don't need to change it (and it does help particularly with account feeds).
					lastHarvestId := socialHarvest.Database.GetLastHarvestId(territory.Name, "twitter", "TwitterPublicMessagesByKeyword", keyword)
					if lastHarvestId != "" {
						params.Set("since_id", lastHarvestId)
					}

					updatedParams, updatedHarvestState := harvester.TwitterSearch(territory.Name, harvestState, keyword, params)
					params = updatedParams
					harvestState = updatedHarvestState
					log.Println("harvested a page of results")

					// Just check Until, Since would be for going backwards in the pagination which we don't need to do here...
					/*
						if params.Until == "0" {
							log.Println("done for " + keyword)
							// Save the last harvest time for this task with the keyword
							socialHarvest.Database.SetLastHarvestTime(territory.Name, "twitter", "TwitterPublicMessagesByKeyword", keyword, harvestState.LastTime, harvestState.LastId, harvestState.ItemsHarvested)
							break
						}
					*/

					// Limit the number of pages of results if specified in the Social Harvest Config
					if territory.Limits.MaxResultsPages != 0 {
						if harvestState.PagesHarvested >= territory.Limits.MaxResultsPages {
							log.Println("done for " + keyword + "(twitter)")
							socialHarvest.Database.SetLastHarvestTime(territory.Name, "twitter", "TwitterPublicMessagesByKeyword", keyword, harvestState.LastTime, harvestState.LastId, harvestState.ItemsHarvested)
							break
						}
					}
					// Count pages of results
					harvestState.PagesHarvested++
				}
			}
		}
	}
}

// Simply calls every other function here, harvesting everything
func HarvestAll() {
	//FacebookPublicMessagesByKeyword()
	//FacebookMessagesByAccount()

	//FacebookGrowthByAccount()
}

// Calls all harvest functions that gather content
func HarvestAllContent() {
	go FacebookPublicMessagesByKeyword()
	go FacebookMessagesByAccount()
	go TwitterPublicMessagesByKeyword()
}

// Calls all harvest functions that gather information about account changes/growth
func HarvestAllAccounts() {
	//FacebookGrowthByAccount()
}

/*
// Stores harvested messages by subscribing to the harvester observable "SocialHarvestMessage" event and storing those messages in the configured database and out to log file
func StoreMessage() {
	var waitGroup sync.WaitGroup
	//dbSession := socialHarvest.Database.GetSession()
	// we have a session already...the StoreRow() function copies it. no need to keep opening connections.
	// it could likely lead to issues over time (from what I'm seeing, I'm guessing this was the case).

	streamCh := make(chan interface{})
	harvester.Subscribe("SocialHarvestMessage", streamCh)
	for {
		message := <-streamCh

		// Log (if configured)
		jsonMsg, err := json.Marshal(message)
		if err == nil {
			socialHarvest.Writers.MessagesWriter.Info(string(jsonMsg))
		}

		// Write to database (if configured)
		waitGroup.Add(1)
		go socialHarvest.Database.StoreRow(message, &waitGroup, socialHarvest.Database.Session)
		// Wait for all the queries to complete.
		waitGroup.Wait()
	}
}

// Stores harvested mentions by subscribing to the harvester observable "SocialHarvestMention" event and storing those messages in the configured database and out to log file
func StoreMention() {
	var waitGroup sync.WaitGroup
	//dbSession := socialHarvest.Database.GetSession()

	streamCh := make(chan interface{})
	harvester.Subscribe("SocialHarvestMention", streamCh)
	for {
		message := <-streamCh

		// Log (if configured)
		jsonMsg, err := json.Marshal(message)
		if err == nil {
			socialHarvest.Writers.MentionsWriter.Info(string(jsonMsg))
		}

		// Write to database (if configured)
		waitGroup.Add(1)
		go socialHarvest.Database.StoreRow(message, &waitGroup, socialHarvest.Database.Session)
		// Wait for all the queries to complete.
		waitGroup.Wait()
	}
}

// Stores harvested shared links by subscribing to the harvester observable "SocialHarvestSharedLink" event and storing those messages in the configured database and out to log file
func StoreSharedLink() {
	var waitGroup sync.WaitGroup
	//dbSession := socialHarvest.Database.GetSession()

	streamCh := make(chan interface{})
	harvester.Subscribe("SocialHarvestSharedLink", streamCh)
	for {
		message := <-streamCh

		// Log (if configured)
		jsonMsg, err := json.Marshal(message)
		if err == nil {
			socialHarvest.Writers.SharedLinksWriter.Info(string(jsonMsg))
		}

		// Write to database (if configured)
		waitGroup.Add(1)
		go socialHarvest.Database.StoreRow(message, &waitGroup, socialHarvest.Database.Session)
		// Wait for all the queries to complete.
		waitGroup.Wait()
	}
}

// Stores harvested hashtags by subscribing to the harvester observable "SocialHarvestHasthag" event and storing those messages in the configured database and out to log file
func StoreHashtag() {
	var waitGroup sync.WaitGroup
	//dbSession := socialHarvest.Database.GetSession()

	streamCh := make(chan interface{})
	harvester.Subscribe("SocialHarvestHashtag", streamCh)
	for {
		message := <-streamCh

		// Log (if configured)
		jsonMsg, err := json.Marshal(message)
		if err == nil {
			socialHarvest.Writers.HashtagsWriter.Info(string(jsonMsg))
		}

		// Write to database (if configured)
		waitGroup.Add(1)
		go socialHarvest.Database.StoreRow(message, &waitGroup, socialHarvest.Database.Session)
		// Wait for all the queries to complete.
		waitGroup.Wait()
	}
}

// Stores harvested info about user/account change by subscribing to the harvester observable "SocialHarvestContributorGrowth" event and storing those messages in the configured database and out to log file
func StoreContributorGrowth() {
	var waitGroup sync.WaitGroup
	//dbSession := socialHarvest.Database.GetSession()

	streamCh := make(chan interface{})
	harvester.Subscribe("SocialHarvestContributorGrowth", streamCh)
	for {
		message := <-streamCh

		// Log (if configured)
		jsonMsg, err := json.Marshal(message)
		if err == nil {
			socialHarvest.Writers.ContributorGrowthWriter.Info(string(jsonMsg))
		}

		// Write to database (if configured)
		waitGroup.Add(1)
		go socialHarvest.Database.StoreRow(message, &waitGroup, socialHarvest.Database.Session)
		// Wait for all the queries to complete.
		waitGroup.Wait()
	}
}
*/
