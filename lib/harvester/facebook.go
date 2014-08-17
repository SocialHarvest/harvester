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

package harvester

import (
	"github.com/SocialHarvest/harvester/lib/config"
	geohash "github.com/TomiHiltunen/geohash-golang"
	fb "github.com/huandu/facebook"
	"github.com/tmaiaroto/geocoder"
	//"github.com/mitchellh/mapstructure"
	"log"
	"net/url"
	//"sync"
	"time"
)

type PagingResult struct {
	Next     string
	Previous string
}

type FacebookParams struct {
	IncludeEntities string
	Limit           string
	Count           string
	Type            string
	Lang            string
	Q               string
	AccessToken     string
	Until           string
	Since           string
	//Previous        string // Facebook uses __previous ...not sure if MakeParams() supports that...and not sure we even need to go backwards anyway.
	//Paging          *PagingResult
	//HasNextPage     bool
}

type MessageTag struct {
	Id   string
	Name string
	Type string
}

type FacebookPost struct {
	// "id" must exist in response. note the leading comma.
	Id   string `facebook:"id,required"`
	From struct {
		Id       string `facebook:"id"`
		Name     string `facebook:"name"`
		Category string `facebook:"category"`
	} `facebook:"from"`
	To struct {
		Id       string `facebook:"id"`
		Name     string `facebook:"name"`
		Category string `facebook:"category"`
	} `facebook:"to"`
	CreatedTime string `facebook:"created_time"`
	UpdatedTime string `facebook:"updated_time"`
	Message     string `facebook:"message"`
	Description string `facebook:"description"`
	Caption     string `facebook:"caption"`
	Picture     string `facebook:"picture"`
	Source      string `facebook:"source"`
	Link        string `facebook:"link"`
	Shares      struct {
		Count int `facebook:"count"`
	} `facebook:"shares"`
	Name string `facebook:"name"`
	// Should always be "post" right? No, Facebook also includes "status" and "link" and "photo" in there, even with the type param set to post. Seems like something changed/broke.
	Type string `facebook:"type"`
	// This can tell us if the user is posting from a mobile device...with some logic. Or just which client apps/SaaS' are most popular to post from (also true for Twitter and could be good data to have).
	Application struct {
		Name      string `facebook:"name"`
		Namespace string `facebook:"namespace"`
		Id        string `facebook:"id"`
	} `facebook:"application"`
	MessageTags map[string][]*MessageTag `facebook:"message_tags"`
	StoryTags   map[string][]*MessageTag `facebook:"story_tags"`
	Story       string                   `facebook:"story"`
	// Typically accompanies items of type photo.
	ObjectId string `facebook:"object_id"`
	// TODO Comments []struct{}
	// Comments have paging though. So care needs to be taken...Do we make more requests and get all comments? Do we limit?
	// What about API request limits?

	// This only exists on user/page /feed items...and it'll usually be "shared_story" but sometimes I've seen "mobile_status_update" ... which tells us the user is on a mobile device.
	// Is it important to keep? I don't know. Probably not right now.
	StatusType string `facebook:"status_type"`
}

// Facebook accounts can be for a user or a page
type FacebookAccount struct {
	// "id" must exist in response. note the leading comma.
	Id              string `facebook:"id,required"`
	About           string `facebook:"about"`
	Category        string `facebook:"category"`
	Checkins        int    `facebook:"checkins"`
	CompanyOverview string `facebook:"company_overview"`
	Description     string `facebook:"description"`
	Founded         string `facebook:"founded"`
	GeneralInfo     string `facebook:"general_info"`
	Likes           int    `facebook:"likes"`
	Link            string `facebook:"link"`
	Location        struct {
		Street    string  `facebook:"street"`
		City      string  `facebook:"city"`
		State     string  `facebook:"state"`
		Zip       string  `facebook:"zip"`
		Country   string  `facebook:"country"`
		Longitude float64 `facebook:"longitude"`
		Latitude  float64 `facebook:"latitude"`
	} `facebook:"location"`
	Name              string `facebook:"name"`
	Phone             string `facebook:"phone"`
	TalkingAboutCount int    `facebook:"talking_about_count"`
	WereHereCount     int    `facebook:"were_here_count"`
	Username          string `facebook:"username"`
	Website           string `facebook:"website"`
	Products          string `facebook:"products"`
	// User specific (the above is a mix of page and user)
	Gender    string `facebook:"gender"`
	Locale    string `facebook:"locale"`
	FirstName string `facebook:"first_name"`
	LastName  string `facebook:"last_name"`
}

// Set the appToken for future use (global)
func NewFacebook(servicesConfig config.ServicesConfig) {
	services.facebookAppToken = servicesConfig.Facebook.AppToken
}

// If the territory has a different appToken to use
func NewFacebookTerritoryCredentials(territory string) {
	for _, t := range harvestConfig.Territories {
		if t.Name == territory {
			if t.Services.Facebook.AppToken != "" {
				services.facebookAppToken = t.Services.Facebook.AppToken
			}
		}
	}
}

// Takes an array of Post structs and converts it to JSON and logs to file (to be picked up by Fluentd, Logstash, Ik, etc.)
func FacebookPostsOut(posts []FacebookPost, territoryName string) (int, string, time.Time) {
	var itemsHarvested = 0
	var latestId = ""
	var latestTime time.Time

	for _, post := range posts {
		postCreatedTime, err := time.Parse("2006-01-02T15:04:05-0700", post.CreatedTime)
		// Only take posts that have a time (and an ID from Facebook)
		if err == nil && len(post.Id) > 0 {
			itemsHarvested++
			// If this is the most recent post in the results, set it's date and id (to be returned) so we can continue where we left off in future harvests
			if latestTime.IsZero() || postCreatedTime.Unix() > latestTime.Unix() {
				latestTime = postCreatedTime
				latestId = post.Id
			}

			hostName := ""
			if len(post.Link) > 0 {
				pUrl, _ := url.Parse(post.Link)
				hostName = pUrl.Host
			}

			// Generate a harvest_id to avoid potential dupes (a unique index is placed on this field and all insert errors ignored).
			harvestId := GetHarvestMd5(post.Id + "facebook" + territoryName)
			//log.Println(harvestId)

			// contributor row (who created the message)
			// NOTE: This is synchronous...but that's ok because while I'd love to use channels and make a bunch of requests at once, there's rate limits from these APIs...
			// Plus the contributor info tells us a few things about the message, such as locale. Other series will use this data.
			var contributor = FacebookAccount{}
			contributor = FacebookGetUserInfo(post.From.Id)

			var contributorGender = 0
			if contributor.Gender == "male" {
				contributorGender = 1
			}
			if contributor.Gender == "female" {
				contributorGender = -1
			}

			var contributorName = contributor.Name
			if len(contributor.FirstName) > 0 {
				contributorName = contributor.FirstName + " " + contributor.LastName
			}

			var contributorType = "person"
			if len(contributor.CompanyOverview) > 0 || len(contributor.Founded) > 0 || len(contributor.Category) > 0 {
				contributorType = "company"
			}

			// Reverse code to get city, state, country, etc.
			var contributorCountry = ""
			var contributorState = ""
			var contributorCity = ""
			var contributorCounty = ""
			if contributor.Location.Latitude != 0.0 && contributor.Location.Latitude != 0.0 {
				reverseLocation, geoErr := geocoder.ReverseGeocode(contributor.Location.Latitude, contributor.Location.Longitude)
				if geoErr != nil {
					contributorState = reverseLocation.State
					contributorCity = reverseLocation.City
					contributorCountry = reverseLocation.CountryCode
					contributorCounty = reverseLocation.County
				}
			}

			// Geohash
			var locationGeoHash = geohash.Encode(contributor.Location.Latitude, contributor.Location.Longitude)
			// This is produced with empty lat/lng values - don't store it.
			if locationGeoHash == "7zzzzzzzzzzz" {
				locationGeoHash = ""
			}

			// TODO: Category (use a classifier in the future for this?)
			// message row
			messageRow := config.SocialHarvestMessage{
				Time:                  postCreatedTime,
				HarvestId:             harvestId,
				Territory:             territoryName,
				Network:               "facebook",
				MessageId:             post.Id,
				ContributorId:         post.From.Id,
				ContributorScreenName: post.From.Name,
				ContributorName:       contributorName,
				ContributorGender:     contributorGender,
				ContributorType:       contributorType,
				ContributorLang:       LocaleToLanguageISO(contributor.Locale),
				ContributorLongitude:  contributor.Location.Longitude,
				ContributorLatitude:   contributor.Location.Latitude,
				ContributorGeohash:    locationGeoHash,
				ContributorCity:       contributorCity,
				ContributorState:      contributorState,
				ContributorCountry:    contributorCountry,
				ContributorCounty:     contributorCounty,
				ContributorLikes:      contributor.Likes,
				Message:               post.Message,
				FacebookShares:        post.Shares.Count,
				Category:              contributor.Category,
				IsQuestion:            Btoi(IsQuestion(post.Message, harvestConfig.QuestionRegex)),
			}
			// Send to the harvester observer
			go StoreHarvestedData(messageRow)
			LogJson(messageRow, "messages")

			// shared links row
			// TODO: expand short urls (Facebook doesn't do it for us unfortunately)
			if len(post.Link) > 0 {
				sharedLinksRow := config.SocialHarvestSharedLink{
					Time:                  postCreatedTime,
					HarvestId:             harvestId,
					Territory:             territoryName,
					Network:               "facebook",
					MessageId:             post.Id,
					ContributorId:         post.From.Id,
					ContributorScreenName: post.From.Name,
					ContributorName:       contributorName,
					ContributorGender:     contributorGender,
					ContributorType:       contributorType,
					ContributorLang:       LocaleToLanguageISO(contributor.Locale),
					ContributorLongitude:  contributor.Location.Longitude,
					ContributorLatitude:   contributor.Location.Latitude,
					ContributorGeohash:    locationGeoHash,
					ContributorCity:       contributorCity,
					ContributorState:      contributorState,
					ContributorCountry:    contributorCountry,
					ContributorCounty:     contributorCounty,
					Type:                  post.Type,
					Preview:               post.Picture,
					Source:                post.Source,
					Url:                   post.Link,
					ExpandedUrl:           ExpandUrl(post.Link),
					Host:                  hostName,
				}
				// Send to the harvester observer
				go StoreHarvestedData(sharedLinksRow)
				LogJson(sharedLinksRow, "shared_links")
			}

			// mentions row (note the harvest id in the following - any post that has multiple subobjects to be stored separately will need a different harvest id, else only one of those subobjects would be stored)
			for _, tag := range post.StoryTags {
				for _, mention := range tag {
					// The harvest id is going to have to be a little different in this case too...Otherwise, we would only get one mention per post.
					storyTagsMentionHarvestId := GetHarvestMd5(post.Id + mention.Id + territoryName)

					// TODO: Keep an eye on this, it may add too many API requests...
					var mentionedContributor = FacebookAccount{}
					mentionedContributor = FacebookGetUserInfo(mention.Id)

					var mentionedGender = 0
					if mentionedContributor.Gender == "male" {
						mentionedGender = 1
					}
					if mentionedContributor.Gender == "female" {
						mentionedGender = -1
					}

					var mentionedName = mentionedContributor.Name
					if len(mentionedContributor.FirstName) > 0 {
						mentionedName = mentionedContributor.FirstName + " " + mentionedContributor.LastName
					}

					var mentionedType = "person"
					if len(mentionedContributor.CompanyOverview) > 0 || len(mentionedContributor.Founded) > 0 || len(mentionedContributor.Category) > 0 {
						mentionedType = "company"
					}

					var mentionedLocationGeoHash = geohash.Encode(mentionedContributor.Location.Latitude, mentionedContributor.Location.Longitude)
					// This is produced with empty lat/lng values - don't store it.
					if mentionedLocationGeoHash == "7zzzzzzzzzzz" {
						mentionedLocationGeoHash = ""
					}

					mentionRow := config.SocialHarvestMention{
						Time:                  postCreatedTime,
						HarvestId:             storyTagsMentionHarvestId,
						Territory:             territoryName,
						Network:               "facebook",
						MessageId:             post.Id,
						ContributorId:         post.From.Id,
						ContributorScreenName: post.From.Name,
						ContributorName:       contributorName,
						ContributorGender:     contributorGender,
						ContributorType:       contributorType,
						ContributorLongitude:  contributor.Location.Longitude,
						ContributorLatitude:   contributor.Location.Latitude,
						ContributorGeohash:    locationGeoHash,
						ContributorLang:       LocaleToLanguageISO(contributor.Locale),

						MentionedId:         mention.Id,
						MentionedScreenName: mention.Name,
						MentionedName:       mentionedName,
						MentionedGender:     mentionedGender,
						MentionedType:       mentionedType,
						MentionedLongitude:  mentionedContributor.Location.Longitude,
						MentionedLatitude:   mentionedContributor.Location.Latitude,
						MentionedGeohash:    mentionedLocationGeoHash,
						MentionedLang:       LocaleToLanguageISO(mentionedContributor.Locale),
					}
					// Send to the harvester observer
					go StoreHarvestedData(mentionRow)
					LogJson(mentionRow, "mentions")
				}
			}
			// Also try MessageTags (which exist on user and page feeds, whereas StoryTags are available on public posts search)
			for _, tag := range post.MessageTags {
				for _, mention := range tag {
					// Same here, the harvest id is going to have to be a little different in this case too...Otherwise, we would only get one mention per post.
					MessageTagsMentionHarvestId := GetHarvestMd5(post.Id + mention.Id + territoryName)

					// TODO: Keep an eye on this, it may add too many API requests...
					// TODO: this is repeated. don't repeat.
					var mentionedContributor = FacebookAccount{}
					mentionedContributor = FacebookGetUserInfo(mention.Id)

					var mentionedGender = 0
					if mentionedContributor.Gender == "male" {
						mentionedGender = 1
					}
					if mentionedContributor.Gender == "female" {
						mentionedGender = -1
					}

					var mentionedName = mentionedContributor.Name
					if len(mentionedContributor.FirstName) > 0 {
						mentionedName = mentionedContributor.FirstName + " " + mentionedContributor.LastName
					}

					var mentionedType = "person"
					if len(mentionedContributor.CompanyOverview) > 0 || len(mentionedContributor.Founded) > 0 || len(mentionedContributor.Category) > 0 {
						mentionedType = "company"
					}

					var mentionedLocationGeoHash = geohash.Encode(mentionedContributor.Location.Latitude, mentionedContributor.Location.Longitude)
					// This is produced with empty lat/lng values - don't store it.
					if mentionedLocationGeoHash == "7zzzzzzzzzzz" {
						mentionedLocationGeoHash = ""
					}

					mentionRow := config.SocialHarvestMention{
						Time:                  postCreatedTime,
						HarvestId:             MessageTagsMentionHarvestId,
						Territory:             territoryName,
						Network:               "facebook",
						MessageId:             post.Id,
						ContributorId:         post.From.Id,
						ContributorScreenName: post.From.Name,
						ContributorName:       contributorName,
						ContributorGender:     contributorGender,
						ContributorType:       contributorType,
						ContributorLongitude:  contributor.Location.Longitude,
						ContributorLatitude:   contributor.Location.Latitude,
						ContributorGeohash:    locationGeoHash,
						ContributorLang:       LocaleToLanguageISO(contributor.Locale),

						MentionedId:         mention.Id,
						MentionedScreenName: mention.Name,
						MentionedName:       mentionedName,
						MentionedGender:     mentionedGender,
						MentionedType:       mentionedType,
						MentionedLongitude:  mentionedContributor.Location.Longitude,
						MentionedLatitude:   mentionedContributor.Location.Latitude,
						MentionedGeohash:    mentionedLocationGeoHash,
						MentionedLang:       LocaleToLanguageISO(mentionedContributor.Locale),
					}
					// Send to the harvester observer
					go StoreHarvestedData(mentionRow)
					LogJson(mentionRow, "mentions")
				}
			}

		} else {
			log.Println("Could not parse the time from the Facebook post, so I'm throwing it away!")
			log.Println(err)
		}

	}

	// return the number of items harvested
	return itemsHarvested, latestId, latestTime
}

// -------------- API CALLS

// Searches public posts on Facebook
func FacebookSearch(params FacebookParams) ([]FacebookPost, FacebookParams) {
	// Get the access token from the configuration if not passed (shouldn't need to be passed, but can be)
	if params.AccessToken == "" {
		params.AccessToken = services.facebookAppToken
	}
	var fbParams = fb.MakeParams(params)

	// Get the results using the passed parameters including the search query
	var res = fb.Result{}
	res, _ = fb.Get("/search", fbParams)

	// Decode the results
	var posts []FacebookPost
	res.DecodeField("data", &posts)

	// Get additional pages
	var paging PagingResult
	res.DecodeField("paging", &paging)

	// Return a set of new params (so we don't have the access token coming back)
	newParams := FacebookParams{}
	newParams.Q = params.Q
	newParams.Type = params.Type
	newParams.Limit = params.Limit
	// Set the following to 0, this will help stop pagination. However if there are multiple pages, the values will not be returned as 0.
	newParams.Until = "0"
	// newParams.Since = "0"

	u, err := url.Parse(paging.Next)
	if err == nil {
		m, _ := url.ParseQuery(u.RawQuery)

		// Adjust the params if there are multiple pages.
		if until, ok := m["until"]; ok {
			newParams.Until = until[0]
		}

		//if since, ok := m["since"]; ok {
		//	newParams.Since = since[0]
		//}

		//log.Println(newParams)
	}

	return posts, newParams
}

// Gets the public posts for a given user or page id (or name actually)
func FacebookFeed(id string, params FacebookParams) ([]FacebookPost, FacebookParams) {
	// XBox page feed for example...
	// https://graph.facebook.com/xbox
	// 16547831022
	if params.AccessToken == "" {
		params.AccessToken = services.facebookAppToken
	}
	var fbParams = fb.MakeParams(params)

	res, _ := fb.Get("/"+id+"/feed", fbParams)

	var posts []FacebookPost
	res.DecodeField("data", &posts)

	// Get additional pages
	var paging PagingResult
	res.DecodeField("paging", &paging)

	// Return a set of new params (so we don't have the access token coming back)
	newParams := FacebookParams{}
	newParams.Limit = params.Limit
	// Set the following to 0, this will help stop pagination. However if there are multiple pages, the values will not be returned as 0.
	newParams.Until = "0"
	//newParams.Since = "0"

	u, err := url.Parse(paging.Next)
	if err == nil {
		m, _ := url.ParseQuery(u.RawQuery)
		// Adjust the params if there are multiple pages.
		if until, ok := m["until"]; ok {
			newParams.Until = until[0]
		}
		//if since, ok := m["since"]; ok {
		//	newParams.Since = since[0]
		//}

		//log.Println(newParams)
	}

	return posts, newParams
}

// Gets basic info about an account on Facebook
func FacebookGetUserInfo(id string) FacebookAccount {
	res, _ := fb.Get("/"+id, fb.Params{
		// This actually isn't required...Though I'm curious about rate limits. I can't find any real concrete numbers.
		// App tokens (which are counted as: app token + ip) have a pretty high limit for basic harvests. So I'm not overly concerned.
		// I imagine requests without access tokens are limited even more, so leave this in for now.
		"access_token": services.facebookAppToken,
	})

	var account FacebookAccount
	res.Decode(&account)

	return account
}
