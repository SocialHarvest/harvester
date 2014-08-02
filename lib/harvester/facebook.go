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
	//"github.com/mitchellh/mapstructure"
	"encoding/json"
	//"log"
	"net/url"
	"sync"
	"time"
)

type Facebook struct {
	appToken      string
	socialHarvest config.SocialHarvest
}

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

var facebook = Facebook{}

// Set the appToken for future use (global)
func NewFacebook(sh config.SocialHarvest) {
	facebook.appToken = sh.Config.Services.Facebook.AppToken
	facebook.socialHarvest = sh
}

// If the territory has a different appToken to use
func NewFacebookTerritoryCredentials(territory string) {
	for _, t := range facebook.socialHarvest.Config.Harvest.Territories {
		if t.Name == territory {
			if t.Services.Facebook.AppToken != "" {
				facebook.appToken = t.Services.Facebook.AppToken
			}
		}
	}
}

// Takes an array of Post structs and converts it to JSON and logs to file (to be picked up by Fluentd, Logstash, Ik, etc.)
func FacebookPostsOut(posts []FacebookPost, territory ...string) (int, string, time.Time) {
	// Save the territory name in its own column for starters, but also use it in part for the generation of the harvest_id (a unique-ish identifier).
	var territoryName string
	if len(territory) > 0 {
		territoryName = territory[0]
	}

	var itemsHarvested = 0
	var latestId = ""
	var latestTime time.Time

	// Create a wait group to manage the goroutines.
	var waitGroup sync.WaitGroup

	dbSession := facebook.socialHarvest.Database.GetSession()

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
			harvestId := facebook.socialHarvest.Database.GetHarvestMd5(post.Id + "facebook" + territoryName)
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

			// Geohash
			var locationGeoHash = geohash.Encode(contributor.Location.Latitude, contributor.Location.Longitude)
			// This is produced with empty lat/lng values - don't store it.
			if locationGeoHash == "7zzzzzzzzzzz" {
				locationGeoHash = ""
			}

			// The harvest id here has to be different. It must include the account (user/page) id so we don't have unnecessarily redundant data being stored/logged.
			// Technically, contributor data can change over time, but for now we're going to assume we only need to retrieve it once and that it won't (drastically) change.
			// TODO: Think about upserts for contributors...Though is the history important? Maybe only for accounts that are being tracked...
			contributorharvestId := facebook.socialHarvest.Database.GetHarvestMd5(post.From.Id + "facebook" + territoryName)
			//log.Println(contributorharvestId)

			contributorRow := config.SocialHarvestContributor{}
			contributorRow.Time = postCreatedTime
			contributorRow.HarvestId = contributorharvestId
			contributorRow.Territory = territoryName
			contributorRow.Network = "facebook"
			contributorRow.ContributorId = post.From.Id
			contributorRow.ContributorScreenName = post.From.Name
			contributorRow.ContributorFacebookCategory = post.From.Category
			contributorRow.IsoLanguageCode = LocaleToLanguageISO(contributor.Locale)
			contributorRow.Gender = contributorGender
			contributorRow.Name = contributorName
			contributorRow.About = contributor.About
			contributorRow.Checkins = contributor.Checkins
			contributorRow.CompanyOverview = contributor.CompanyOverview
			contributorRow.Description = contributor.Description
			contributorRow.Founded = contributor.Founded
			contributorRow.GeneralInfo = contributor.GeneralInfo
			contributorRow.Likes = contributor.Likes
			contributorRow.Link = contributor.Link
			// Address (and lat/lon) is flattened coming from Facebook's object
			contributorRow.Street = contributor.Location.Street
			contributorRow.City = contributor.Location.City
			contributorRow.State = contributor.Location.State
			contributorRow.Zip = contributor.Location.Zip
			contributorRow.Country = contributor.Location.Country
			contributorRow.Longitude = contributor.Location.Longitude
			contributorRow.Latitude = contributor.Location.Latitude
			contributorRow.Geohash = locationGeoHash
			contributorRow.Phone = contributor.Phone
			contributorRow.TalkingAboutCount = contributor.TalkingAboutCount
			contributorRow.WereHereCount = contributor.WereHereCount
			contributorRow.Url = contributor.Website
			contributorRow.Products = contributor.Products

			// write contributors out
			contrib, messageMarshalErr := json.Marshal(contributorRow)
			if messageMarshalErr == nil {
				facebook.socialHarvest.Writers.ContributorsWriter.Info(string(contrib))
			}
			waitGroup.Add(1)
			go facebook.socialHarvest.Database.StoreRow(contributorRow, &waitGroup, dbSession)

			// message row
			messageRow := config.SocialHarvestMessage{}
			messageRow.Time = postCreatedTime
			messageRow.HarvestId = harvestId
			messageRow.Territory = territoryName
			messageRow.Network = "facebook"
			messageRow.MessageId = post.Id
			messageRow.ContributorId = post.From.Id
			messageRow.ContributorScreenName = post.From.Name
			messageRow.ContributorFacebookCategory = post.From.Category
			messageRow.IsoLanguageCode = LocaleToLanguageISO(contributor.Locale)
			messageRow.Longitude = contributor.Location.Longitude
			messageRow.Latitude = contributor.Location.Latitude
			messageRow.Geohash = locationGeoHash
			messageRow.FacebookShares = post.Shares.Count
			// write messages out
			message, messageMarshalErr := json.Marshal(messageRow)
			if messageMarshalErr == nil {
				//log.Println(string(message))
				facebook.socialHarvest.Writers.MessagesWriter.Info(string(message))
			}
			waitGroup.Add(1)
			go facebook.socialHarvest.Database.StoreRow(messageRow, &waitGroup, dbSession)

			// question row (if message is a question)
			if IsQuestion(post.Message, facebook.socialHarvest.Config.Harvest.QuestionRegex) == true {
				questionRow := config.SocialHarvestQuestion{}
				questionRow.Time = postCreatedTime
				questionRow.HarvestId = harvestId
				questionRow.Territory = territoryName
				questionRow.Network = "facebook"
				questionRow.MessageId = post.Id
				questionRow.ContributorId = post.From.Id
				questionRow.ContributorScreenName = post.From.Name
				questionRow.IsoLanguageCode = LocaleToLanguageISO(contributor.Locale)
				questionRow.Longitude = contributor.Location.Longitude
				questionRow.Latitude = contributor.Location.Latitude
				questionRow.Geohash = locationGeoHash
				questionRow.Message = post.Message
				// write questions out
				question, messageMarshalErr := json.Marshal(questionRow)
				if messageMarshalErr == nil {
					facebook.socialHarvest.Writers.MessagesWriter.Info(string(question))
				}
				waitGroup.Add(1)
				go facebook.socialHarvest.Database.StoreRow(questionRow, &waitGroup, dbSession)
			}

			// shared links row
			// TODO: expand short urls (Facebook doesn't do it for us unfortunately)
			if len(post.Link) > 0 {
				sharedLinksRow := config.SocialHarvestSharedLink{}
				sharedLinksRow.Time = postCreatedTime
				sharedLinksRow.HarvestId = harvestId
				sharedLinksRow.Territory = territoryName
				sharedLinksRow.Network = "facebook"
				sharedLinksRow.MessageId = post.Id
				sharedLinksRow.ContributorId = post.From.Id
				sharedLinksRow.ContributorScreenName = post.From.Name
				sharedLinksRow.ContributorFacebookCategory = post.From.Category
				sharedLinksRow.Url = post.Link
				sharedLinksRow.Host = hostName
				sharedLinksRow.FacebookShares = post.Shares.Count
				// write shared links out
				sharedLink, sharedLinkMarshalErr := json.Marshal(sharedLinksRow)
				if sharedLinkMarshalErr == nil {
					facebook.socialHarvest.Writers.SharedLinksWriter.Info(string(sharedLink))
				}
				waitGroup.Add(1)
				go facebook.socialHarvest.Database.StoreRow(sharedLinksRow, &waitGroup, dbSession)
			}

			// shared media row
			if post.Type == "video" || post.Type == "photo" {
				sharedMediaRow := config.SocialHarvestSharedMedia{}
				sharedMediaRow.Time = postCreatedTime
				sharedMediaRow.HarvestId = harvestId
				sharedMediaRow.Territory = territoryName
				sharedMediaRow.Network = "facebook"
				sharedMediaRow.MessageId = post.Id
				sharedMediaRow.ContributorId = post.From.Id
				sharedMediaRow.ContributorScreenName = post.From.Name
				sharedMediaRow.ContributorFacebookCategory = post.From.Category
				sharedMediaRow.Type = post.Type
				sharedMediaRow.Preview = post.Picture
				sharedMediaRow.Source = post.Source
				sharedMediaRow.Url = post.Link
				sharedMediaRow.Host = hostName
				// write shared media out
				sharedMedia, sharedMediaMarshalErr := json.Marshal(sharedMediaRow)
				if sharedMediaMarshalErr == nil {
					facebook.socialHarvest.Writers.SharedMediaWriter.Info(string(sharedMedia))
				}
				waitGroup.Add(1)
				go facebook.socialHarvest.Database.StoreRow(sharedMediaRow, &waitGroup, dbSession)
			}

			// mentions row (note the harvest id in the following - any post that has multiple subobjects to be stored separately will need a different harvest id, else only one of those subobjects would be stored)
			for _, tag := range post.StoryTags {
				for _, mention := range tag {
					// The harvest id is going to have to be a little different in this case too...Otherwise, we would only get one mention per post.
					storyTagsMentionHarvestId := facebook.socialHarvest.Database.GetHarvestMd5(post.Id + mention.Id + territoryName)

					mentionRow := config.SocialHarvestMention{}
					mentionRow.Time = postCreatedTime
					mentionRow.HarvestId = storyTagsMentionHarvestId
					mentionRow.Territory = territoryName
					mentionRow.Network = "facebook"
					mentionRow.MessageId = post.Id
					mentionRow.ContributorId = post.From.Id
					mentionRow.ContributorScreenName = post.From.Name
					mentionRow.ContributorFacebookCategory = post.From.Category
					mentionRow.MentionedScreenName = mention.Name
					mentionRow.MentionedId = mention.Id
					mentionRow.MentionedType = mention.Type
					// write mentions out
					mention, mentionMarshalErr := json.Marshal(mentionRow)
					if mentionMarshalErr == nil {
						facebook.socialHarvest.Writers.MentionsWriter.Info(string(mention))
					}
					waitGroup.Add(1)
					go facebook.socialHarvest.Database.StoreRow(mentionRow, &waitGroup, dbSession)

				}
			}
			// Also try MessageTags (which exist on user and page feeds, whereas StoryTags are available on public posts search)
			for _, tag := range post.MessageTags {
				for _, mention := range tag {
					// Same here, the harvest id is going to have to be a little different in this case too...Otherwise, we would only get one mention per post.
					MessageTagsMentionHarvestId := facebook.socialHarvest.Database.GetHarvestMd5(post.Id + mention.Id + territoryName)

					mentionRow := config.SocialHarvestMention{}
					mentionRow.Time = postCreatedTime
					mentionRow.HarvestId = MessageTagsMentionHarvestId
					mentionRow.Territory = territoryName
					mentionRow.Network = "facebook"
					mentionRow.MessageId = post.Id
					mentionRow.ContributorId = post.From.Id
					mentionRow.ContributorScreenName = post.From.Name
					mentionRow.ContributorFacebookCategory = post.From.Category
					mentionRow.MentionedScreenName = mention.Name
					mentionRow.MentionedId = mention.Id
					mentionRow.MentionedType = mention.Type
					messageRow.IsoLanguageCode = LocaleToLanguageISO(contributor.Locale)
					messageRow.Longitude = contributor.Location.Longitude
					messageRow.Latitude = contributor.Location.Latitude
					messageRow.Geohash = locationGeoHash
					// write mentions out
					mention, mentionMarshalErr := json.Marshal(mentionRow)
					if mentionMarshalErr == nil {
						facebook.socialHarvest.Writers.MentionsWriter.Info(string(mention))
					}
					waitGroup.Add(1)
					go facebook.socialHarvest.Database.StoreRow(mentionRow, &waitGroup, dbSession)

				}
			}

		}

	}

	// Wait for all the queries to complete.
	waitGroup.Wait()

	// Remove any empty saves (weird bug)
	facebook.socialHarvest.Database.RemoveEmpty("contributors")
	facebook.socialHarvest.Database.RemoveEmpty("messages")
	facebook.socialHarvest.Database.RemoveEmpty("shared_links")
	facebook.socialHarvest.Database.RemoveEmpty("shared_media")
	facebook.socialHarvest.Database.RemoveEmpty("mentions")

	// return the number of items harvested
	return itemsHarvested, latestId, latestTime
}

// -------------- API CALLS

// Searches public posts on Facebook
func FacebookSearch(params FacebookParams) ([]FacebookPost, FacebookParams) {
	// Get the access token from the configuration if not passed (shouldn't need to be passed, but can be)
	if params.AccessToken == "" {
		params.AccessToken = facebook.appToken
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
		params.AccessToken = facebook.appToken
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
		"access_token": facebook.appToken,
	})

	var account FacebookAccount
	res.Decode(&account)

	return account
}
