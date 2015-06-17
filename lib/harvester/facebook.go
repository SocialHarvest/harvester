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
	geohash "github.com/SocialHarvestVendors/geohash-golang"
	"github.com/SocialHarvestVendors/go-querystring/query"
	//"github.com/mitchellh/mapstructure"
	"bytes"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"net/url"
	//"sync"
	"time"
)

type PagingResult struct {
	Next     string `json:"next" url:"next"`
	Previous string `json:"previous" url:"previous"`
}

type FacebookParams struct {
	IncludeEntities string `url:"include_entities,omitempty"`
	Limit           string `url:"limit,omitempty"`
	Count           string `url:"count,omitempty"`
	Type            string `url:"type,omitempty"`
	Lang            string `url:"lang,omitempty"`
	Q               string `url:"q,omitempty"`
	AccessToken     string `url:"access_token,omitempty"`
	Until           string `url:"until,omitempty"`
	Since           string `url:"since,omitempty"`
	//Previous        string // Facebook uses __previous ...not sure if MakeParams() supports that...and not sure we even need to go backwards anyway.
	//Paging          *PagingResult
	//HasNextPage     bool
}

type MessageTag struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type FacebookPost struct {
	// "id" must exist in response. note the leading comma.
	Id   string `json:"id,required"`
	From struct {
		Id       string `json:"id"`
		Name     string `json:"name"`
		Category string `json:"category"`
	} `json:"from"`
	To struct {
		Data []struct {
			Id       string `json:"id"`
			Name     string `json:"name"`
			Category string `json:"category"`
		} `json:"data"`
	} `json:"to"`
	CreatedTime string `json:"created_time"`
	UpdatedTime string `json:"updated_time"`
	Message     string `json:"message"`
	Description string `json:"description"`
	Caption     string `json:"caption"`
	Picture     string `json:"picture"`
	Source      string `json:"source"`
	Link        string `json:"link"`
	Shares      struct {
		Count int `json:"count"`
	} `json:"shares"`
	Name string `json:"name"`
	// Should always be "post" right? No, facebook also includes "status" and "link" and "photo" in there, even with the type param set to post. Seems like something changed/broke.
	Type string `json:"type"`
	// This can tell us if the user is posting from a mobile device...with some logic. Or just which client apps/SaaS' are most popular to post from (also true for Twitter and could be good data to have).
	Application struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		Id        string `json:"id"`
	} `json:"application"`
	MessageTags map[string][]*MessageTag `json:"message_tags"`
	StoryTags   map[string][]*MessageTag `json:"story_tags"`
	Story       string                   `json:"story"`
	// Typically accompanies items of type photo.
	ObjectId string `json:"object_id"`
	// TODO Comments []struct{}
	// Comments have paging though. So care needs to be taken...Do we make more requests and get all comments? Do we limit?
	// What about API request limits?

	// This only exists on user/page /feed items...and it'll usually be "shared_story" but sometimes I've seen "mobile_status_update" ... which tells us the user is on a mobile device.
	// Is it important to keep? I don't know. Probably not right now.
	StatusType string `json:"status_type"`
}

// Facebook accounts can be for a user or a page
type FacebookAccount struct {
	// "id" must exist in response. note the leading comma.
	Id              string `json:"id,required"`
	About           string `json:"about"`
	Category        string `json:"category"`
	Checkins        int    `json:"checkins"`
	CompanyOverview string `json:"company_overview"`
	Description     string `json:"description"`
	Founded         string `json:"founded"`
	GeneralInfo     string `json:"general_info"`
	Likes           int    `json:"likes"`
	Link            string `json:"link"`
	Location        struct {
		Street    string  `json:"street"`
		City      string  `json:"city"`
		State     string  `json:"state"`
		Zip       string  `json:"zip"`
		Country   string  `json:"country"`
		Longitude float64 `json:"longitude"`
		Latitude  float64 `json:"latitude"`
	} `json:"location"`
	Name              string `json:"name"`
	Phone             string `json:"phone"`
	TalkingAboutCount int    `json:"talking_about_count"`
	WereHereCount     int    `json:"were_here_count"`
	Username          string `json:"username"`
	Website           string `json:"website"`
	Products          string `json:"products"`
	// User specific (the above is a mix of page and user)
	Gender    string `json:"gender"`
	Locale    string `json:"locale"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

var fbToken string

var fbHttpClient *http.Client
var fbGraphApiBaseUrl = "https://graph.facebook.com/"

// Set the appToken for future use (global)
func NewFacebook(servicesConfig config.ServicesConfig) {
	fbToken = servicesConfig.Facebook.AppToken

	fbHttpClient = &http.Client{
		Transport: &TimeoutTransport{
			Transport: http.Transport{
				Dial: func(netw, addr string) (net.Conn, error) {
					//log.Printf("dial to %s://%s", netw, addr)
					return net.Dial(netw, addr) // Regular ass dial.
				},
				DisableKeepAlives: true,
				//DisableCompression: true,
			},
			// Facebook's payload (especially with 100 results) will take a little while to download
			RoundTripTimeout: time.Second * 10,
		},
	}
}

// If the territory has a different appToken to use
func NewFacebookTerritoryCredentials(territory string) {
	for _, t := range harvestConfig.Territories {
		if t.Name == territory {
			if t.Services.Facebook.AppToken != "" {
				// TODO: This actually should be passed on each harvest. Because otherwise it'd overwrite the harvest wide token.

				services.facebookAppToken = t.Services.Facebook.AppToken
			}
		}
	}
}

// Takes an array of Post structs and converts it to JSON and logs to file (to be picked up by Fluentd, Logstash, Ik, etc.)
func FacebookPostsOut(posts []FacebookPost, territoryName string, params FacebookParams) (int, string, time.Time) {
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
			contributor = FacebookGetUserInfo(post.From.Id, params)

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
			var contributorRegion = ""
			var contributorCity = ""
			var contributorCityPopulation = int32(0)
			// This isn't always available with Geobed information and while many counties will be, they still need to be decoded with the Geonames data set (id numbers to string names).
			// When Geobed updates, then Social Harvest can add county information in again. "State" (US state) has also changed to "Region" due to the data sets being used.
			// A little consistency has been lost, but geocoding is all internal now. Not a bad trade off.
			// var contributorCounty = ""
			if contributor.Location.Latitude != 0.0 && contributor.Location.Latitude != 0.0 {
				reverseLocation := services.geocoder.ReverseGeocode(contributor.Location.Latitude, contributor.Location.Longitude)
				contributorRegion = reverseLocation.Region
				contributorCity = reverseLocation.City
				contributorCountry = reverseLocation.Country
				contributorCityPopulation = reverseLocation.Population
				// contributorCounty = reverseLocation.County
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
				Time:                      postCreatedTime,
				HarvestId:                 harvestId,
				Territory:                 territoryName,
				Network:                   "facebook",
				MessageId:                 post.Id,
				ContributorId:             post.From.Id,
				ContributorScreenName:     post.From.Name,
				ContributorName:           contributorName,
				ContributorGender:         contributorGender,
				ContributorType:           contributorType,
				ContributorLang:           LocaleToLanguageISO(contributor.Locale),
				ContributorLongitude:      contributor.Location.Longitude,
				ContributorLatitude:       contributor.Location.Latitude,
				ContributorGeohash:        locationGeoHash,
				ContributorCity:           contributorCity,
				ContributorCityPopulation: contributorCityPopulation,
				ContributorRegion:         contributorRegion,
				ContributorCountry:        contributorCountry,
				ContributorLikes:          contributor.Likes,
				Message:                   post.Message,
				FacebookShares:            post.Shares.Count,
				Category:                  contributor.Category,
				Sentiment:                 services.sentimentAnalyzer.Classify(post.Message),
				IsQuestion:                Btoi(IsQuestion(post.Message, harvestConfig.QuestionRegex)),
			}
			StoreHarvestedData(messageRow)
			LogJson(messageRow, "messages")

			// Keywords are stored on the same collection as hashtags - but under a `keyword` field instead of `tag` field as to not confuse the two.
			// Limit to words 4 characters or more and only return 8 keywords. This could greatly increase the database size if not limited.
			keywords := GetKeywords(post.Message, 4, 8)
			if len(keywords) > 0 {
				for _, keyword := range keywords {
					if keyword != "" {
						keywordHarvestId := GetHarvestMd5(post.Id + "facebook" + territoryName + keyword)

						// Again, keyword share the same series/table/collection
						hashtag := config.SocialHarvestHashtag{
							Time:                      postCreatedTime,
							HarvestId:                 keywordHarvestId,
							Territory:                 territoryName,
							Network:                   "facebook",
							MessageId:                 post.Id,
							ContributorId:             post.From.Id,
							ContributorScreenName:     post.From.Name,
							ContributorName:           contributorName,
							ContributorGender:         contributorGender,
							ContributorType:           contributorType,
							ContributorLang:           LocaleToLanguageISO(contributor.Locale),
							ContributorLongitude:      contributor.Location.Longitude,
							ContributorLatitude:       contributor.Location.Latitude,
							ContributorGeohash:        locationGeoHash,
							ContributorCity:           contributorCity,
							ContributorCityPopulation: contributorCityPopulation,
							ContributorRegion:         contributorRegion,
							ContributorCountry:        contributorCountry,
							Keyword:                   keyword,
						}
						StoreHarvestedData(hashtag)
						LogJson(hashtag, "hashtags")
					}
				}
			}

			// shared links row
			// TODO: expand short urls (Facebook doesn't do it for us unfortunately)
			if len(post.Link) > 0 {
				sharedLinksRow := config.SocialHarvestSharedLink{
					Time:                      postCreatedTime,
					HarvestId:                 harvestId,
					Territory:                 territoryName,
					Network:                   "facebook",
					MessageId:                 post.Id,
					ContributorId:             post.From.Id,
					ContributorScreenName:     post.From.Name,
					ContributorName:           contributorName,
					ContributorGender:         contributorGender,
					ContributorType:           contributorType,
					ContributorLang:           LocaleToLanguageISO(contributor.Locale),
					ContributorLongitude:      contributor.Location.Longitude,
					ContributorLatitude:       contributor.Location.Latitude,
					ContributorGeohash:        locationGeoHash,
					ContributorCity:           contributorCity,
					ContributorCityPopulation: contributorCityPopulation,
					ContributorRegion:         contributorRegion,
					ContributorCountry:        contributorCountry,
					Type:                      post.Type,
					Preview:                   post.Picture,
					Source:                    post.Source,
					Url:                       post.Link,
					ExpandedUrl:               ExpandUrl(post.Link),
					Host:                      hostName,
				}
				StoreHarvestedData(sharedLinksRow)
				LogJson(sharedLinksRow, "shared_links")
			}

			// mentions row (note the harvest id in the following - any post that has multiple subobjects to be stored separately will need a different harvest id, else only one of those subobjects would be stored)
			for _, tag := range post.StoryTags {
				for _, mention := range tag {
					// The harvest id is going to have to be a little different in this case too...Otherwise, we would only get one mention per post.
					storyTagsMentionHarvestId := GetHarvestMd5(post.Id + mention.Id + territoryName)

					// TODO: Keep an eye on this, it may add too many API requests...
					var mentionedContributor = FacebookAccount{}
					mentionedContributor = FacebookGetUserInfo(mention.Id, params)

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
					StoreHarvestedData(mentionRow)
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
					mentionedContributor = FacebookGetUserInfo(mention.Id, params)

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
					StoreHarvestedData(mentionRow)
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
func FacebookSearch(territoryName string, harvestState config.HarvestState, params FacebookParams) (FacebookParams, config.HarvestState) {
	// Look for access_token override, if not present, use default fbToken from config
	if params.AccessToken == "" {
		params.AccessToken = fbToken
	}
	// If that happens to be empty, just return.
	if params.AccessToken == "" {
		return params, harvestState
	}

	// Concatenate and build the searchUrl
	var buffer bytes.Buffer
	buffer.WriteString(fbGraphApiBaseUrl)
	buffer.WriteString("/search?")

	// convert struct to querystring params
	v, err := query.Values(params)
	if err != nil {
		return params, harvestState
	}
	buffer.WriteString(v.Encode())
	searchUrl := buffer.String()
	buffer.Reset()

	// set up the request
	req, err := http.NewRequest("GET", searchUrl, nil)
	if err != nil {
		return params, harvestState
	}
	// doo it
	resp, err := fbHttpClient.Do(req)
	if err != nil {
		return params, harvestState
	}
	defer resp.Body.Close()

	// now to parse response, store and contine along.
	data := struct {
		Posts  []FacebookPost `json:"data"`
		Paging struct {
			Previous string `json:"previous"`
			Next     string `json:"next"`
		} `json:"paging"`
	}{}

	dec := json.NewDecoder(resp.Body)
	dec.Decode(&data)
	//log.Println(data)
	// close the response now, we don't need it anymore - otherwise it'll stay open while we write to the database. best to close it.
	resp.Body.Close()

	// parse the querystring of "next" so we can get the "until" value for params
	if data.Paging.Next != "" {
		u, err := url.Parse(data.Paging.Next)
		if err == nil {
			m, _ := url.ParseQuery(u.RawQuery)
			if _, ok := m["until"]; ok {
				params.Until = m["until"][0]
			} else {
				// By setting this empty, we'll know not to loop again. This is up to date and should be the last request for this harvest.
				params.Until = ""
			}
		} else {
			// log.Println(err)
		}
	}

	// Only attempt to store if we have some results.
	if len(data.Posts) > 0 {
		// Save, then return updated params and harvest state for next round (if there is another one)
		harvestState.ItemsHarvested, harvestState.LastId, harvestState.LastTime = FacebookPostsOut(data.Posts, territoryName, params)
	}

	return params, harvestState
}

// Gets the public posts for a given user or page id (or name actually)
func FacebookFeed(territoryName string, harvestState config.HarvestState, account string, params FacebookParams) (FacebookParams, config.HarvestState) {
	// XBox page feed for example...
	// https://graph.facebook.com/xbox
	// 16547831022

	// Look for access_token override, if not present, use default fbToken from config
	if params.AccessToken == "" {
		params.AccessToken = fbToken
	}
	// If that happens to be empty, just return.
	if params.AccessToken == "" {
		return params, harvestState
	}

	var buffer bytes.Buffer
	buffer.WriteString(fbGraphApiBaseUrl)
	buffer.WriteString(account)
	buffer.WriteString("/feed?")

	// convert struct to querystring params
	v, err := query.Values(params)
	if err != nil {
		return params, harvestState
	}
	buffer.WriteString(v.Encode())
	feedUrl := buffer.String()
	buffer.Reset()

	// set up the request
	req, err := http.NewRequest("GET", feedUrl, nil)
	if err != nil {
		return params, harvestState
	}
	// doo it
	resp, err := fbHttpClient.Do(req)
	if err != nil {
		return params, harvestState
	}
	defer resp.Body.Close()

	// now to parse response, store and contine along.
	data := struct {
		Posts  []FacebookPost `json:"data"`
		Paging struct {
			Previous string `json:"previous"`
			Next     string `json:"next"`
		} `json:"paging"`
	}{}

	dec := json.NewDecoder(resp.Body)
	dec.Decode(&data)
	//log.Println(data)
	// close the response now, we don't need it anymore - otherwise it'll stay open while we write to the database. best to close it.
	resp.Body.Close()

	// parse the querystring of "next" so we can get the "until" value for params
	if data.Paging.Next != "" {
		u, err := url.Parse(data.Paging.Next)
		if err == nil {
			m, _ := url.ParseQuery(u.RawQuery)
			if _, ok := m["until"]; ok {
				params.Until = m["until"][0]
			} else {
				// By setting this empty, we'll know not to loop again. This is up to date and should be the last request for this harvest.
				params.Until = ""
			}
		} else {
			// log.Println(err)
		}
	}

	// Only attempt to store if we have some results.
	if len(data.Posts) > 0 {
		// Save, then return updated params and harvest state for next round (if there is another one)
		harvestState.ItemsHarvested, harvestState.LastId, harvestState.LastTime = FacebookPostsOut(data.Posts, territoryName, params)
	}

	return params, harvestState
}

// Gets basic info about an account on Facebook
func FacebookGetUserInfo(id string, params FacebookParams) FacebookAccount {
	var account FacebookAccount

	if id != "" {
		var buffer bytes.Buffer
		buffer.WriteString(fbGraphApiBaseUrl)
		buffer.WriteString(id)
		buffer.WriteString("?")

		// convert struct to querystring params (for now, only pass the access_token, the other stuff doesn't matter for our use here)
		userInfoParams := FacebookParams{AccessToken: params.AccessToken}
		v, err := query.Values(userInfoParams)
		if err != nil {
			return account
		}
		buffer.WriteString(v.Encode())
		userInfoUrl := buffer.String()
		buffer.Reset()

		// set up the request
		req, err := http.NewRequest("GET", userInfoUrl, nil)
		if err != nil {
			return account
		}
		// doo it
		resp, err := fbHttpClient.Do(req)
		if err != nil {
			return account
		}
		defer resp.Body.Close()

		dec := json.NewDecoder(resp.Body)
		dec.Decode(&account)
		// close the response now, we don't need it anymore - it should close right after this anyway because of the defer... but these calls are atomic. so just to be safe.
		// i've had too much trouble with unclosed http requests. i'm happy to be paranoid and safe rather than sorry, because i've been sorry and sore before.
		resp.Body.Close()
	}

	return account
}

// Harvests Facebook account details to track changes in likes, etc. (only for public pages)
func FacebookAccountDetails(territoryName string, account string) {
	params := FacebookParams{}
	contributor := FacebookGetUserInfo(account, params)
	now := time.Now()
	// The harvest id in this case will be unique by time / account / network / territory, since there is no post id or anything else like that
	harvestId := GetHarvestMd5(account + now.String() + "facebook" + territoryName)

	row := config.SocialHarvestContributorGrowth{
		Time:          now,
		HarvestId:     harvestId,
		Territory:     territoryName,
		Network:       "facebook",
		ContributorId: contributor.Id,
		Likes:         contributor.Likes,
		TalkingAbout:  contributor.TalkingAboutCount,
		WereHere:      contributor.WereHereCount,
		Checkins:      contributor.Checkins,
	}
	StoreHarvestedData(row)
	LogJson(row, "contributor_growth")
	return
}
