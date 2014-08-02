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
	"github.com/SocialHarvest/harvester/lib/harvester"
	"log"
	"strconv"
	"time"
)

// All functions for harvesting should follow the same format.
// <Network>PublicMessagesBy<Criteria>
// <Network>MessagesBy<Criteria>
// <Network>GrowthBy<Criteria>
// Criteria is typically going to be "Keyword" and "Account" but there may be other values in the future.
// The following functions are intended to be scheduled (and perhaps even called by other packages or via the RESTful API).

// TODO: Look into: http://labix.org/pipe
// Pipe everything(?) through the specified scripts/commands before saving and writing to log files.
// This was the original thinking for filters, but Fluentd may be enough? If this is introduced, it will be in the future (lower priority).

// Harvest Facebook publicly accessible posts by searching keyword criteria
func FacebookPublicMessagesByKeyword() {
	log.Println("getting facebook public messages by keyword")
	params := harvester.FacebookParams{}
	posts := []harvester.FacebookPost{}

	for _, territory := range socialHarvest.Config.Harvest.Territories {
		if len(territory.Content.Lang) > 0 {
			params.Lang = territory.Content.Lang
		}
		// If different credentials were set for the territory, this will find and set them
		harvester.NewFacebookTerritoryCredentials(territory.Name)

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
					lastHarvestTime := socialHarvest.Database.GetLastHarvestTime(territory.Name, "facebook", "FacebookPublicPostsByKeyword", keyword)
					if !lastHarvestTime.IsZero() {
						params.Since = strconv.FormatInt(lastHarvestTime.Unix(), 10)
					}

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
						socialHarvest.Database.SetLastHarvestTime(territory.Name, "facebook", "FacebookPublicPostsByKeyword", keyword, lastTimeHarvested, lastIdHarvested, itemsHarvested)
						break
					}

					// Limit the number of pages of results if specified in the Social Harvest Config
					if territory.Limits.MaxResultsPages != 0 {
						if pagesHarvested >= territory.Limits.MaxResultsPages {
							log.Println("done for " + keyword)
							socialHarvest.Database.SetLastHarvestTime(territory.Name, "facebook", "FacebookPublicPostsByKeyword", keyword, lastTimeHarvested, lastIdHarvested, itemsHarvested)
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
					lastHarvestTime := socialHarvest.Database.GetLastHarvestTime(territory.Name, "facebook", "FacebookPostsBssyAccount", account)
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
						socialHarvest.Database.SetLastHarvestTime(territory.Name, "facebook", "FacebookFeedsByAccount", account, lastTimeHarvested, lastIdHarvested, itemsHarvested)
						break
					}

					// Limit the number of pages of results if specified in the Social Harvest config
					if territory.Limits.MaxResultsPages != 0 {
						if pagesHarvested >= territory.Limits.MaxResultsPages {
							log.Println("done for " + account)
							socialHarvest.Database.SetLastHarvestTime(territory.Name, "facebook", "FacebookFeedsByAccount", account, lastTimeHarvested, lastIdHarvested, itemsHarvested)
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

// Simply calls every other function here, harvesting everything
func HarvestAll() {
	FacebookPublicMessagesByKeyword()
	FacebookMessagesByAccount()

	FacebookGrowthByAccount()
}

// Calls all harvest functions that gather content
func HarvestAllContent() {
	FacebookPublicMessagesByKeyword()
	FacebookMessagesByAccount()
}

// Calls all harvest functions that gather information about account changes/growth
func HarvestAllAccounts() {
	FacebookGrowthByAccount()
}
