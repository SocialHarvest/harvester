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
	"github.com/ChimeraCoder/anaconda"
	"github.com/SocialHarvest/harvester/lib/config"
	geohash "github.com/TomiHiltunen/geohash-golang"
	"github.com/tmaiaroto/geocoder"
	"log"
	"net/url"
	"time"
)

func NewTwitter(servicesConfig config.ServicesConfig) {
	anaconda.SetConsumerKey(servicesConfig.Twitter.ApiKey)
	anaconda.SetConsumerSecret(servicesConfig.Twitter.ApiSecret)
	services.twitter = anaconda.NewTwitterApi(servicesConfig.Twitter.AccessToken, servicesConfig.Twitter.AccessTokenSecret)
}

// If the territory has different keys to use
func NewTwitterTerritoryCredentials(territory string) {
	for _, t := range harvestConfig.Territories {
		if t.Name == territory {
			if t.Services.Twitter.ApiKey != "" && t.Services.Twitter.ApiSecret != "" && t.Services.Twitter.AccessToken != "" && t.Services.Twitter.AccessTokenSecret != "" {
				anaconda.SetConsumerKey(t.Services.Twitter.ApiKey)
				anaconda.SetConsumerSecret(t.Services.Twitter.ApiSecret)
				services.twitter = anaconda.NewTwitterApi(t.Services.Twitter.AccessToken, t.Services.Twitter.AccessTokenSecret)
			}
		}
	}
}

// Search for status updates and just pass the Tweet along (no special mapping required like FacebookPost{} because the Tweet struct is used across multiple API calls unlike Facebook)
// All "search" functions (and anything that gets data from an API) will now normalize the data, mapping it to a Social Harvest struct.
// This means there will be no way to get the original data from the service (back in the main app or from any other Go package that imports the harvester).
// This is fine because if someone wanted the original data, they could use packages like anaconda directly.
// What happens now is all data pulled from earch service's API will be sent to a channel (the harvester observer). However, this function should NOT be called in a go-subroutine though.
// We don't want to make multiple API calls in parallel (rate limits).
// NOTE: The number of items sent to the observer will be returned along with the last message's time and id. The main package can record this in the harvest logs/table.
// The harvester will not keep track of this information itself. Its only job is to gather data, send it to the channel and report back on how much was sent (and the last id/time). Period.
// It doens't care if the data is stored in a database, logged, or streamed out from an API. It just harvests and sends without looking or caring.
// Whereas previously it would be doing the db calls and logging, etc. This has now all been taken care of with the observer. All of these other processes simply subscribe and listen.
//
// Always passed in first (always): the territory name, and the position in the harvest (HarvestState) ... the rest are going to vary based on the API but typically are the query and options
// @return options(for pagination), count of items, last id, last time.
func TwitterSearch(territoryName string, harvestState config.HarvestState, query string, options url.Values) (url.Values, config.HarvestState) {
	searchResults, _ := services.twitter.GetSearch(query, options)
	// The cool thing about Twitter's API is that we have all the user data we need already. So we make less HTTP requests than when using Facebook's API.
	for _, tweet := range searchResults {
		//log.Println(tweet)
		//	log.Println("processing a tweet....")

		tweetCreatedTime, err := time.Parse(time.RubyDate, tweet.CreatedAt)
		// Only take tweets that have a time (and an ID from Facebook)
		if err == nil && len(tweet.IdStr) > 0 {
			harvestState.ItemsHarvested++
			// If this is the most recent tweet in the results, set it's date and id (to be returned) so we can continue where we left off in future harvests
			if harvestState.LastTime.IsZero() || tweetCreatedTime.Unix() > harvestState.LastTime.Unix() {
				harvestState.LastTime = tweetCreatedTime
				harvestState.LastId = tweet.IdStr
			}

			// determine gender
			var contributorGender = DetectGender(tweet.User.Name)

			// TODO: figure out type somehow...
			var contributorType = DetectContributorType(tweet.User.Name, contributorGender)

			var contributorCountry = ""
			var contributorState = ""
			var contributorCity = ""
			var contributorCounty = ""

			var statusLongitude = 0.0
			var statusLatitude = 0.0
			// TODO: is there a better way to do this? sheesh
			switch coordMap := tweet.Coordinates.(type) {
			case map[string]interface{}:
				for k, v := range coordMap {
					if k == "coordinates" {
						switch coords := v.(type) {
						case []interface{}:
							for i, c := range coords {
								switch cFloat := c.(type) {
								case float64:
									if i == 0 {
										statusLongitude = cFloat
									}
									if i == 1 {
										statusLatitude = cFloat
									}
									break
								}
							}
						}

					}
				}
				break
			}

			// Contributor location lookup (if no lat/lng was found on the message - try to reduce number of geocode lookups)
			contributorLat := 0.0
			contributorLng := 0.0
			if statusLatitude == 0.0 || statusLatitude == 0.0 {
				// Do not make a request for nothing (there are no 1 character locations either).
				if len(tweet.User.Location) > 1 {
					location, err := geocoder.GeocodeLocation(tweet.User.Location)
					if err == nil {
						contributorLat = location.LatLng.Lat
						contributorLng = location.LatLng.Lng
						contributorState = location.State
						contributorCity = location.City
						contributorCountry = location.CountryCode
						contributorCounty = location.County
					}
				}
				//contributorLat, contributorLng = Geocode(tweet.User.Location)
			} else {
				reverseLocation, geoErr := geocoder.ReverseGeocode(statusLatitude, statusLongitude)
				if geoErr == nil {
					contributorState = reverseLocation.State
					contributorCity = reverseLocation.City
					contributorCountry = reverseLocation.CountryCode
					contributorCounty = reverseLocation.County
				}
				// keep these, no need to change - might change accuracy, etc.
				contributorLat = statusLatitude
				contributorLng = statusLongitude
			}

			// Contributor geohash
			var contributorLocationGeoHash = geohash.Encode(contributorLat, contributorLng)
			// This is produced with empty lat/lng values - don't store it.
			if contributorLocationGeoHash == "7zzzzzzzzzzz" {
				contributorLocationGeoHash = ""
			}

			// Generate a harvest_id to avoid potential dupes (a unique index is placed on this field and all insert errors ignored).
			harvestId := GetHarvestMd5(tweet.IdStr + "twitter" + territoryName)

			message := config.SocialHarvestMessage{
				Time:                     tweetCreatedTime,
				HarvestId:                harvestId,
				Territory:                territoryName,
				Network:                  "twitter",
				ContributorId:            tweet.User.IdStr,
				ContributorScreenName:    tweet.User.ScreenName,
				ContributorName:          tweet.User.Name,
				ContributorLang:          tweet.User.Lang,
				ContributorLongitude:     contributorLng,
				ContributorLatitude:      contributorLat,
				ContributorGeohash:       contributorLocationGeoHash,
				ContributorCity:          contributorCity,
				ContributorState:         contributorState,
				ContributorCountry:       contributorCountry,
				ContributorCounty:        contributorCounty,
				ContributorVerified:      Btoi(tweet.User.Verified),
				ContributorFollowers:     tweet.User.FollowersCount,
				ContributorStatusesCount: int(tweet.User.StatusesCount),
				ContributorGender:        contributorGender,
				ContributorType:          contributorType,
				Message:                  tweet.Text,
				IsQuestion:               Btoi(IsQuestion(tweet.Text, harvestConfig.QuestionRegex)),
				MessageId:                tweet.IdStr,
				TwitterRetweetCount:      tweet.RetweetCount,
				TwitterFavoriteCount:     tweet.FavoriteCount,
			}
			// Send to the harvester observer
			go StoreHarvestedData(message)
			LogJson(message, "messages")

			// shared links
			if len(tweet.Entities.Urls) > 0 {
				for _, link := range tweet.Entities.Urls {
					if len(link.Url) > 0 {
						// Shared link harvest id has to be different because otherwise only one would be stored
						sharedLinkHarvestId := GetHarvestMd5(tweet.IdStr + "twitter" + territoryName + link.Expanded_url)

						linkHostName := ""
						pUrl, _ := url.Parse(link.Url)
						linkHostName = pUrl.Host

						// TODO: ADD contributor gender, contributor type
						sharedLink := config.SocialHarvestSharedLink{
							Time:                  tweetCreatedTime,
							HarvestId:             sharedLinkHarvestId,
							Territory:             territoryName,
							Network:               "twitter",
							MessageId:             tweet.IdStr,
							ContributorId:         tweet.User.IdStr,
							ContributorScreenName: tweet.User.ScreenName,
							ContributorName:       tweet.User.Name,
							ContributorLang:       tweet.User.Lang,
							ContributorType:       contributorType,
							ContributorGender:     contributorGender,
							ContributorLongitude:  contributorLng,
							ContributorLatitude:   contributorLat,
							ContributorGeohash:    contributorLocationGeoHash,
							ContributorCity:       contributorCity,
							ContributorState:      contributorState,
							ContributorCountry:    contributorCountry,
							ContributorCounty:     contributorCounty,
							Url:                   link.Url,
							ExpandedUrl:           link.Expanded_url,
							Host:                  linkHostName,
						}
						// Send to the harvester observer
						StoreHarvestedData(sharedLink)
						LogJson(sharedLink, "shared_links")
					}
				}
			}

			// more shared links (media entities)
			if len(tweet.Entities.Media) > 0 {
				for _, media := range tweet.Entities.Media {
					if len(media.Url) > 0 {
						sharedMediaHarvestId := GetHarvestMd5(tweet.IdStr + "twitter" + territoryName + media.Expanded_url)

						mediaHostName := ""
						pUrl, _ := url.Parse(media.Url)
						mediaHostName = pUrl.Host

						// TODO: ADD contributor gender, contributor type
						sharedMedia := config.SocialHarvestSharedLink{
							Time:                  tweetCreatedTime,
							HarvestId:             sharedMediaHarvestId,
							Territory:             territoryName,
							Network:               "twitter",
							MessageId:             tweet.IdStr,
							ContributorId:         tweet.User.IdStr,
							ContributorScreenName: tweet.User.ScreenName,
							ContributorName:       tweet.User.Name,
							ContributorLang:       tweet.User.Lang,
							ContributorType:       contributorType,
							ContributorGender:     contributorGender,
							ContributorLongitude:  contributorLng,
							ContributorLatitude:   contributorLat,
							ContributorGeohash:    contributorLocationGeoHash,
							ContributorCity:       contributorCity,
							ContributorState:      contributorState,
							ContributorCountry:    contributorCountry,
							ContributorCounty:     contributorCounty,
							Url:                   media.Url,
							ExpandedUrl:           media.Expanded_url,
							Host:                  mediaHostName,
							Type:                  media.Type,
							Source:                media.Media_url,
						}
						// Send to the harvester observer
						StoreHarvestedData(sharedMedia)
						LogJson(sharedMedia, "shared_links")
					}
				}
			}

			// hashtags
			if len(tweet.Entities.Hashtags) > 0 {
				for _, tag := range tweet.Entities.Hashtags {
					if len(tag.Text) > 0 {
						hashtagHarvestId := GetHarvestMd5(tweet.IdStr + "twitter" + territoryName + tag.Text)

						// TODO: ADD contributor gender, contributor type
						hashtag := config.SocialHarvestHashtag{
							Time:                  tweetCreatedTime,
							HarvestId:             hashtagHarvestId,
							Territory:             territoryName,
							Network:               "twitter",
							MessageId:             tweet.IdStr,
							ContributorId:         tweet.User.IdStr,
							ContributorScreenName: tweet.User.ScreenName,
							ContributorName:       tweet.User.Name,
							ContributorLang:       tweet.User.Lang,
							ContributorType:       contributorType,
							ContributorGender:     contributorGender,
							ContributorLongitude:  contributorLng,
							ContributorLatitude:   contributorLat,
							ContributorGeohash:    contributorLocationGeoHash,
							ContributorCity:       contributorCity,
							ContributorState:      contributorState,
							ContributorCountry:    contributorCountry,
							ContributorCounty:     contributorCounty,
							Tag:                   tag.Text,
						}
						// Send to the harvester observer
						StoreHarvestedData(hashtag)
						LogJson(hashtag, "hashtags")
					}
				}
			}

			// mentions
			if len(tweet.Entities.User_mentions) > 0 {
				for _, mentionedUser := range tweet.Entities.User_mentions {
					if len(mentionedUser.Id_str) > 0 {
						mentionHarvestId := GetHarvestMd5(tweet.IdStr + "twitter" + territoryName + mentionedUser.Id_str)

						// TODO: ADD contributor gender, contributor type
						// and mentioned user info (another api request)
						mention := config.SocialHarvestMention{
							Time:                  tweetCreatedTime,
							HarvestId:             mentionHarvestId,
							Territory:             territoryName,
							Network:               "twitter",
							MessageId:             tweet.IdStr,
							ContributorId:         tweet.User.IdStr,
							ContributorScreenName: tweet.User.ScreenName,
							ContributorName:       tweet.User.Name,
							ContributorLang:       tweet.User.Lang,
							ContributorType:       contributorType,
							ContributorGender:     contributorGender,
							ContributorLongitude:  contributorLng,
							ContributorLatitude:   contributorLat,
							ContributorGeohash:    contributorLocationGeoHash,

							MentionedId:         mentionedUser.Id_str,
							MentionedScreenName: mentionedUser.Screen_name,
							MentionedName:       mentionedUser.Name,
						}
						// Send to the harvester observer
						StoreHarvestedData(mention)
						LogJson(mention, "mentions")
					}
				}
			}

		} else {
			log.Println("Could not parse the time from the Tweet, so I'm throwing it away!")
			log.Println(err)
		}
	}

	return options, harvestState
}
