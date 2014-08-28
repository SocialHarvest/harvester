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

	for _, territory := range socialHarvest.Config.Harvest.Territories {
		// If different credentials were set for the territory, this will find and set them
		// TODO: Change this. Always pass the credentials for overrides. OR, always set them back on each harvest (harvests do happen one by one).
		// NOTE: Pass to the params strut. simple.
		//harvester.NewFacebookTerritoryCredentials(territory.Name)

		for _, keyword := range territory.Content.Keywords {
			// Reset Until and Since, just in case. The for loop below could have changed it for the next keyword.
			params.Until = ""
			params.Since = ""
			// Set some other params
			params.Type = "post"
			params.Q = keyword
			//log.Print("Searching for: " + keyword)
			if territory.Limits.ResultsPerPage != "" {
				params.Limit = territory.Limits.ResultsPerPage
			} else {
				params.Limit = "100"
			}

			harvestState := config.HarvestState{
				LastId:         "",
				LastTime:       time.Now(),
				PagesHarvested: 1,
				ItemsHarvested: 0,
			}

			// Limit to 10 pages max. Anything more will simply take too long and cause issues.
			maxPages := territory.Limits.MaxResultsPages
			if maxPages == 0 {
				maxPages = 10
			}

			// Fetch X pages of results
			for i := 0; i < maxPages; i++ {
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

				// No need to pass the keyword to this function by itself, it's set in params.Q because it's not part of the API URL path, it's in the querystring.
				updatedParams, updatedHarvestState := harvester.FacebookSearch(territory.Name, harvestState, params)
				params = updatedParams
				harvestState = updatedHarvestState
				log.Println("harvested a page of results from facebook")

				// Always save this on each page. Then if something crashes for some reason during a harvest of several pages, we can pick up where we left off. Rather than starting over again.
				socialHarvest.Database.SetLastHarvestTime(territory.Name, "facebook", "FacebookPublicMessagesByKeyword", keyword, harvestState.LastTime, harvestState.LastId, harvestState.ItemsHarvested)

				// We also avoid using "break" because the for loop is now based on number of pages to harvest.
				// But this could lead to harvesting pages taht don't exist, so we should still "break" in that case.
				// Since every call to FacebookFeed() should return with a new Until value, we'll look to see if it's empty. If so, it was the latest page of results from FB. Break the loop.
				if params.Until == "" {
					// log.Println("completed search - ran into page limit set by config")
					break
				}
			}

		}
	}
	return
}

// Harvest Facebook publicly accessible posts from a specific account (user or page)
func FacebookMessagesByAccount() {
	params := harvester.FacebookParams{}

	for _, territory := range socialHarvest.Config.Harvest.Territories {
		// If different credentials were set for the territory, this will find and set them
		// TODO: Change this. Always pass the credentials for overrides. OR, always set them back on each harvest (harvests do happen one by one).
		// NOTE: Pass to the params strut. simple.
		//harvester.NewFacebookTerritoryCredentials(territory.Name)

		for _, account := range territory.Accounts.Facebook {
			// Reset Until and Since, just in case. The for loop below could have changed it for the next account.
			params.Until = ""
			params.Since = ""
			if territory.Limits.ResultsPerPage != "" {
				params.Limit = territory.Limits.ResultsPerPage
			} else {
				params.Limit = "100"
			}

			harvestState := config.HarvestState{
				LastId:         "",
				LastTime:       time.Now(),
				PagesHarvested: 1,
				ItemsHarvested: 0,
			}

			// Limit to 10 pages max. Anything more will simply take too long and cause issues.
			maxPages := territory.Limits.MaxResultsPages
			if maxPages == 0 {
				maxPages = 10
			}

			// Fetch X pages of results
			for i := 0; i < maxPages; i++ {
				lastHarvestTime := socialHarvest.Database.GetLastHarvestTime(territory.Name, "facebook", "FacebookMessagesByAccount", account)
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

				updatedParams, updatedHarvestState := harvester.FacebookFeed(territory.Name, harvestState, account, params)
				params = updatedParams
				harvestState = updatedHarvestState
				// log.Println("harvested a page of results from facebook")

				// Always save this on each page. Then if something crashes for some reason during a harvest of several pages, we can pick up where we left off. Rather than starting over again.
				socialHarvest.Database.SetLastHarvestTime(territory.Name, "facebook", "FacebookMessagesByAccount", account, harvestState.LastTime, harvestState.LastId, harvestState.ItemsHarvested)

				// We also avoid using "break" because the for loop is now based on number of pages to harvest.
				// But this could lead to harvesting pages taht don't exist, so we should still "break" in that case.
				// Since every call to FacebookFeed() should return with a new Until value, we'll look to see if it's empty. If so, it was the latest page of results from FB. Break the loop.
				if params.Until == "" {
					// log.Println("completed search - no more pages of results")
					break
				}
			}

		}
	}
	return
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

				// TODO: Make like Facebook and look at maxpages...
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
					//log.Println("harvested a page of results from twitter")

					// TODO: Look at facebook and do something similar for the following
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
							// log.Println("done for " + keyword + "(twitter)")
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
	log.Println("done twitter public message search")
	return
}

// Searches Instagram for media by territory keyword criteria (first needs to get tags)
func InstagramMediaByKeyword() {
	for _, territory := range socialHarvest.Config.Harvest.Territories {
		// If different credentials were set for the territory, this will find and set them
		harvester.NewInstagramTerritoryCredentials(territory.Name)

		// First find the top tag for the keyword (basically, try to convert keywords into tags) - though this can be disabled, per territory, by configuration.
		// The default is going to be false, so it will use keywords and lookup a tag for each (setting true would only use defined Instagram tags from the config).
		if len(territory.Content.Keywords) > 0 {
			for _, keyword := range territory.Content.Keywords {
				if !territory.Content.Options.OnlyUseInstagramTags {
					keywordTag := harvester.InstagramFindTags(keyword)
					// The following isn't too awesome for memory usage, but the slices should be small
					territory.Content.InstagramTags = append(territory.Content.InstagramTags, keywordTag)

				}
			}
			// Remove any duplicates (again, not great for memory)
			m := map[string]bool{}
			deDuped := []string{}
			for _, v := range territory.Content.InstagramTags {
				if _, seen := m[v]; !seen {
					deDuped = append(deDuped, v)
					m[v] = true
				}
			}
			territory.Content.InstagramTags = deDuped
		}

		// Build params for search
		params := url.Values{}
		// Search all keywords
		if len(territory.Content.InstagramTags) > 0 {
			for _, tag := range territory.Content.InstagramTags {
				// log.Print("Searching for: " + tag)

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
				// log.Println(harvestState)

				// Limit to 10 pages max. Anything more will simply take too long and cause issues.
				maxPages := territory.Limits.MaxResultsPages
				if maxPages == 0 {
					maxPages = 10
				}

				// Fetch X pages of results
				for i := 0; i < maxPages; i++ {
					lastHarvestId := socialHarvest.Database.GetLastHarvestId(territory.Name, "instagram", "InstagramMediaByKeyword", tag)
					params.Set("max_tag_id", lastHarvestId)

					updatedParams, updatedHarvestState := harvester.InstagramSearch(territory.Name, harvestState, tag, params)
					params = updatedParams
					harvestState = updatedHarvestState
					// log.Println("harvested a page of results from instagram")

					// Always save this on each page. Then if something crashes for some reason during a harvest of several pages, we can pick up where we left off. Rather than starting over again.
					socialHarvest.Database.SetLastHarvestTime(territory.Name, "instagram", "InstagramMediaByKeyword", tag, harvestState.LastTime, harvestState.LastId, harvestState.ItemsHarvested)

					// We also avoid using "break" because the for loop is now based on number of pages to harvest.
					// But this could lead to harvesting pages taht don't exist, so we should still "break" in that case.
					// Since every call to FacebookFeed() should return with a new Until value, we'll look to see if it's empty. If so, it was the latest page of results from FB. Break the loop.
					// TODO: Find equivelant here
					if params.Get("max_tag_id") == "" {
						// log.Println("completed search - no more pages of results")
						break
					}
				}
			}
		}

	}
}

// If specific Instagram tags were passed in the territory criteria, then those will be used directly
func InstagramMediaByTag() {
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
