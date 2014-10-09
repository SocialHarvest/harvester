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
	"code.google.com/p/google-api-go-client/googleapi/transport"
	"code.google.com/p/google-api-go-client/plus/v1"
	"github.com/SocialHarvest/harvester/lib/config"
	geohash "github.com/TomiHiltunen/geohash-golang"
	"github.com/tmaiaroto/geocoder"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func NewGooglePlus(servicesConfig config.ServicesConfig) {
	client := &http.Client{
		Transport: &transport.APIKey{Key: servicesConfig.Google.ServerKey},
	}
	plusService, err := plus.New(client)
	if err == nil {
		services.googlePlus = plusService
	} else {
		log.Println(err)
	}
}

// If the territory has different keys to use
func NewGooglePlusTerritoryCredentials(territory string) {
	for _, t := range harvestConfig.Territories {
		if t.Name == territory {
			if t.Services.Google.ServerKey != "" {
				client := &http.Client{
					Transport: &transport.APIKey{Key: t.Services.Google.ServerKey},
				}
				plusService, err := plus.New(client)
				if err == nil {
					services.googlePlus = plusService
				} else {
					log.Println(err)
				}
			}
		}
	}
}

// Gets Google+ activities (posts) by searching for a keyword.
func GooglePlusActivitySearch(territoryName string, harvestState config.HarvestState, query string, options url.Values) (url.Values, config.HarvestState) {
	limit, lErr := strconv.ParseInt(options.Get("count"), 10, 64)
	if lErr != nil {
		limit = 20
	}
	if limit > 20 {
		limit = 20
	}
	// If there's a next page token, it'll be used to continue to the next page for this harvest
	nextPageToken := options.Get("nextPageToken")

	activities, err := services.googlePlus.Activities.Search(query).MaxResults(limit).PageToken(nextPageToken).Do()
	if err == nil {
		// Passed back to whatever called this function, so it can continue with the next page.
		options.Set("nextPageToken", activities.NextPageToken)

		for _, item := range activities.Items {

			itemCreatedTime, err := time.Parse(time.RFC3339, item.Published)
			// Only take instagrams that have a time
			if err == nil && len(item.Id) > 0 {
				harvestState.ItemsHarvested++
				// If this is the most recent tweet in the results, set it's date and id (to be returned) so we can continue where we left off in future harvests
				if harvestState.LastTime.IsZero() || itemCreatedTime.Unix() > harvestState.LastTime.Unix() {
					harvestState.LastTime = itemCreatedTime
					harvestState.LastId = item.Id
				}

				// Generate a harvest_id to avoid potential dupes (a unique index is placed on this field and all insert errors ignored).
				harvestId := GetHarvestMd5(item.Id + "googlePlus" + territoryName)

				// contributor row (who created the message)
				// NOTE: This is synchronous...but that's ok because while I'd love to use channels and make a bunch of requests at once, there's rate limits from these APIs...
				// Plus the contributor info tells us a few things about the message, such as locale. Other series will use this data.
				contributor, err := services.googlePlus.People.Get(item.Actor.Id).Do()
				if err != nil {
					log.Println(err)
					return options, harvestState
				}

				var contributorGender = 0
				if contributor.Gender == "male" {
					contributorGender = 1
				}
				if contributor.Gender == "female" {
					contributorGender = -1
				}
				var contributorType = DetectContributorType(item.Actor.DisplayName, contributorGender)
				contributorLanguage := LocaleToLanguageISO(contributor.Language)

				var itemLat = 0.0
				var itemLng = 0.0
				// Reverse code to get city, state, country, etc.
				var contributorCountry = ""
				var contributorState = ""
				var contributorCity = ""
				var contributorCounty = ""
				if item.Location != nil && item.Location.Position != nil {
					if item.Location.Position.Latitude != 0.0 && item.Location.Position.Longitude != 0.0 {
						itemLat = item.Location.Position.Latitude
						itemLng = item.Location.Position.Longitude
						reverseLocation, geoErr := geocoder.ReverseGeocode(item.Location.Position.Latitude, item.Location.Position.Longitude)
						if geoErr == nil {
							contributorState = reverseLocation.State
							contributorCity = reverseLocation.City
							contributorCountry = reverseLocation.CountryCode
							contributorCounty = reverseLocation.County
						}
					}
				}

				// Geohash
				var locationGeoHash = geohash.Encode(itemLat, itemLng)
				// This is produced with empty lat/lng values - don't store it.
				if locationGeoHash == "7zzzzzzzzzzz" {
					locationGeoHash = ""
				}

				// message row
				messageRow := config.SocialHarvestMessage{
					Time:                  itemCreatedTime,
					HarvestId:             harvestId,
					Territory:             territoryName,
					Network:               "googlePlus",
					MessageId:             item.Id,
					ContributorId:         item.Actor.Id,
					ContributorScreenName: item.Actor.DisplayName,
					ContributorName:       item.Actor.DisplayName,
					ContributorGender:     contributorGender,
					ContributorType:       contributorType,
					ContributorLang:       contributorLanguage,
					ContributorLongitude:  itemLng,
					ContributorLatitude:   itemLat,
					ContributorGeohash:    locationGeoHash,
					ContributorCity:       contributorCity,
					ContributorState:      contributorState,
					ContributorCountry:    contributorCountry,
					ContributorCounty:     contributorCounty,
					Message:               item.Object.Content,
					IsQuestion:            Btoi(IsQuestion(item.Object.OriginalContent, harvestConfig.QuestionRegex)),
					GooglePlusReshares:    item.Object.Resharers.TotalItems,
					GooglePlusOnes:        item.Object.Plusoners.TotalItems,
				}
				StoreHarvestedData(messageRow)
				LogJson(messageRow, "messages")

				// Keywords are stored on the same collection as hashtags - but under a `keyword` field instead of `tag` field as to not confuse the two.
				// Limit to words 4 characters or more and only return 8 keywords. This could greatly increase the database size if not limited.
				keywords := GetKeywords(item.Object.OriginalContent, 4, 8)
				if len(keywords) > 0 {
					for _, keyword := range keywords {
						if keyword != "" {
							keywordHarvestId := GetHarvestMd5(item.Id + "googlePlus" + territoryName + keyword)

							// Again, keyword share the same series/table/collection
							hashtag := config.SocialHarvestHashtag{
								Time:                  itemCreatedTime,
								HarvestId:             keywordHarvestId,
								Territory:             territoryName,
								Network:               "googlePlus",
								MessageId:             item.Id,
								ContributorId:         item.Actor.Id,
								ContributorScreenName: item.Actor.DisplayName,
								ContributorName:       item.Actor.DisplayName,
								ContributorGender:     contributorGender,
								ContributorType:       contributorType,
								ContributorLang:       contributorLanguage,
								ContributorLongitude:  itemLng,
								ContributorLatitude:   itemLat,
								ContributorGeohash:    locationGeoHash,
								ContributorCity:       contributorCity,
								ContributorState:      contributorState,
								ContributorCountry:    contributorCountry,
								ContributorCounty:     contributorCounty,
								Keyword:               keyword,
							}
							StoreHarvestedData(hashtag)
							LogJson(hashtag, "hashtags")
						}
					}
				}

				if len(item.Object.Attachments) > 0 {
					for _, attachment := range item.Object.Attachments {
						hostName := ""
						if len(attachment.Url) > 0 {
							pUrl, _ := url.Parse(attachment.Url)
							hostName = pUrl.Host
						}

						previewImg := ""
						if attachment.Image != nil {
							previewImg = attachment.Image.Url
						}
						fullImg := ""
						if attachment.FullImage != nil {
							fullImg = attachment.FullImage.Url
						}

						sharedLinksRow := config.SocialHarvestSharedLink{
							Time:                  itemCreatedTime,
							HarvestId:             harvestId,
							Territory:             territoryName,
							Network:               "googlePlus",
							MessageId:             item.Id,
							ContributorId:         item.Actor.Id,
							ContributorScreenName: item.Actor.DisplayName,
							ContributorName:       item.Actor.DisplayName,
							ContributorGender:     contributorGender,
							ContributorType:       contributorType,
							ContributorLang:       contributorLanguage,
							ContributorLongitude:  itemLng,
							ContributorLatitude:   itemLat,
							ContributorGeohash:    locationGeoHash,
							ContributorCity:       contributorCity,
							ContributorState:      contributorState,
							ContributorCountry:    contributorCountry,
							ContributorCounty:     contributorCounty,
							Type:                  attachment.ObjectType,
							Preview:               previewImg,
							Source:                fullImg,
							Url:                   attachment.Url,
							ExpandedUrl:           ExpandUrl(attachment.Url),
							Host:                  hostName,
						}
						StoreHarvestedData(sharedLinksRow)
						LogJson(sharedLinksRow, "shared_links")
					}
				}

			}
		}
	} else {
		log.Println(err)
	}

	return options, harvestState
}

// Gets public Google+ activities (posts) by account.
func GooglePlusActivityByAccount(territoryName string, harvestState config.HarvestState, account string, options url.Values) (url.Values, config.HarvestState) {
	limit, lErr := strconv.ParseInt(options.Get("count"), 10, 64)
	if lErr != nil {
		limit = 100
	}
	if limit > 100 {
		limit = 100
	}
	// If there's a next page token, it'll be used to continue to the next page for this harvest
	nextPageToken := options.Get("nextPageToken")

	activities, err := services.googlePlus.Activities.List(account, "public").MaxResults(limit).PageToken(nextPageToken).Do()
	if err == nil {
		// Passed back to whatever called this function, so it can continue with the next page.
		options.Set("nextPageToken", activities.NextPageToken)

		for _, item := range activities.Items {

			itemCreatedTime, err := time.Parse(time.RFC3339, item.Published)
			// Only take instagrams that have a time
			if err == nil && len(item.Id) > 0 {
				harvestState.ItemsHarvested++
				// If this is the most recent tweet in the results, set it's date and id (to be returned) so we can continue where we left off in future harvests
				if harvestState.LastTime.IsZero() || itemCreatedTime.Unix() > harvestState.LastTime.Unix() {
					harvestState.LastTime = itemCreatedTime
					harvestState.LastId = item.Id
				}

				// Generate a harvest_id to avoid potential dupes (a unique index is placed on this field and all insert errors ignored).
				harvestId := GetHarvestMd5(item.Id + "googlePlus" + territoryName)

				// contributor row (who created the message)
				// NOTE: This is synchronous...but that's ok because while I'd love to use channels and make a bunch of requests at once, there's rate limits from these APIs...
				// Plus the contributor info tells us a few things about the message, such as locale. Other series will use this data.
				contributor, err := services.googlePlus.People.Get(item.Actor.Id).Do()
				if err != nil {
					log.Println(err)
					return options, harvestState
				}

				var contributorGender = 0
				if contributor.Gender == "male" {
					contributorGender = 1
				}
				if contributor.Gender == "female" {
					contributorGender = -1
				}
				var contributorType = DetectContributorType(item.Actor.DisplayName, contributorGender)
				contributorLanguage := LocaleToLanguageISO(contributor.Language)

				var itemLat = 0.0
				var itemLng = 0.0
				// Reverse code to get city, state, country, etc.
				var contributorCountry = ""
				var contributorState = ""
				var contributorCity = ""
				var contributorCounty = ""
				if item.Location != nil && item.Location.Position != nil {
					if item.Location.Position.Latitude != 0.0 && item.Location.Position.Longitude != 0.0 {
						itemLat = item.Location.Position.Latitude
						itemLng = item.Location.Position.Longitude
						reverseLocation, geoErr := geocoder.ReverseGeocode(item.Location.Position.Latitude, item.Location.Position.Longitude)
						if geoErr == nil {
							contributorState = reverseLocation.State
							contributorCity = reverseLocation.City
							contributorCountry = reverseLocation.CountryCode
							contributorCounty = reverseLocation.County
						}
					}
				}

				// Geohash
				var locationGeoHash = geohash.Encode(itemLat, itemLng)
				// This is produced with empty lat/lng values - don't store it.
				if locationGeoHash == "7zzzzzzzzzzz" {
					locationGeoHash = ""
				}

				// message row
				messageRow := config.SocialHarvestMessage{
					Time:                  itemCreatedTime,
					HarvestId:             harvestId,
					Territory:             territoryName,
					Network:               "googlePlus",
					MessageId:             item.Id,
					ContributorId:         item.Actor.Id,
					ContributorScreenName: item.Actor.DisplayName,
					ContributorName:       item.Actor.DisplayName,
					ContributorGender:     contributorGender,
					ContributorType:       contributorType,
					ContributorLang:       contributorLanguage,
					ContributorLongitude:  itemLng,
					ContributorLatitude:   itemLat,
					ContributorGeohash:    locationGeoHash,
					ContributorCity:       contributorCity,
					ContributorState:      contributorState,
					ContributorCountry:    contributorCountry,
					ContributorCounty:     contributorCounty,
					Message:               item.Object.Content,
					IsQuestion:            Btoi(IsQuestion(item.Object.OriginalContent, harvestConfig.QuestionRegex)),
					GooglePlusReshares:    item.Object.Resharers.TotalItems,
					GooglePlusOnes:        item.Object.Plusoners.TotalItems,
				}
				StoreHarvestedData(messageRow)
				LogJson(messageRow, "messages")

				if len(item.Object.Attachments) > 0 {
					for _, attachment := range item.Object.Attachments {
						hostName := ""
						if len(attachment.Url) > 0 {
							pUrl, _ := url.Parse(attachment.Url)
							hostName = pUrl.Host
						}

						previewImg := ""
						if attachment.Image != nil {
							previewImg = attachment.Image.Url
						}
						fullImg := ""
						if attachment.FullImage != nil {
							fullImg = attachment.FullImage.Url
						}

						sharedLinksRow := config.SocialHarvestSharedLink{
							Time:                  itemCreatedTime,
							HarvestId:             harvestId,
							Territory:             territoryName,
							Network:               "googlePlus",
							MessageId:             item.Id,
							ContributorId:         item.Actor.Id,
							ContributorScreenName: item.Actor.DisplayName,
							ContributorName:       item.Actor.DisplayName,
							ContributorGender:     contributorGender,
							ContributorType:       contributorType,
							ContributorLang:       contributorLanguage,
							ContributorLongitude:  itemLng,
							ContributorLatitude:   itemLat,
							ContributorGeohash:    locationGeoHash,
							ContributorCity:       contributorCity,
							ContributorState:      contributorState,
							ContributorCountry:    contributorCountry,
							ContributorCounty:     contributorCounty,
							Type:                  attachment.ObjectType,
							Preview:               previewImg,
							Source:                fullImg,
							Url:                   attachment.Url,
							ExpandedUrl:           ExpandUrl(attachment.Url),
							Host:                  hostName,
						}
						StoreHarvestedData(sharedLinksRow)
						LogJson(sharedLinksRow, "shared_links")
					}
				}

			}
		}
	} else {
		log.Println(err)
	}

	return options, harvestState
}

// Harvests Google+ account details to track changes in followers, etc. (NOTE: Pages can't currently be tracked by the existing API, it's invite only)
func GooglePlusAccountDetails(territoryName string, account string) {
	contributor, err := services.googlePlus.People.Get(account).Do()
	if err == nil {
		now := time.Now()
		// The harvest id in this case will be unique by time / account / network / territory, since there is no post id or anything else like that
		harvestId := GetHarvestMd5(account + now.String() + "googlePlus" + territoryName)

		row := config.SocialHarvestContributorGrowth{
			Time:          now,
			HarvestId:     harvestId,
			Territory:     territoryName,
			Network:       "googlePlus",
			ContributorId: contributor.Id,
			Followers:     int(contributor.CircledByCount),
			PlusOnes:      int(contributor.PlusOneCount),
		}
		StoreHarvestedData(row)
		LogJson(row, "contributor_growth")
	}
	return
}
