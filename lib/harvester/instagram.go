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
	//"encoding/json"
	"bytes"
	"github.com/SocialHarvest/harvester/lib/config"
	geohash "github.com/SocialHarvestVendors/geohash-golang"
	"github.com/SocialHarvestVendors/go-instagram/instagram"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	//"sync"
)

// Use our own http.Client for Instagram (has timeouts and such)
var instagramHttpClient *http.Client

// Set the client for future use
func NewInstagram(servicesConfig config.ServicesConfig) {
	instagramHttpClient = &http.Client{
		Transport: &TimeoutTransport{
			Transport: http.Transport{
				Dial: func(netw, addr string) (net.Conn, error) {
					//log.Printf("dial to %s://%s", netw, addr)
					return net.Dial(netw, addr) // Regular ass dial.
				},
			},
			// A payload with a bunch of results will take a little while to download
			RoundTripTimeout: time.Second * 10,
		},
	}

	// NOTE: Can change this back to nil for the default client. See how the custom client goes (not sure what the default is being used, did the package create one? Or default Go?).
	services.instagram = instagram.NewClient(instagramHttpClient)
	services.instagram.ClientID = servicesConfig.Instagram.ClientId
}

// If the territory has different keys to use
func NewInstagramTerritoryCredentials(territory string) {
	for _, t := range harvestConfig.Territories {
		if t.Name == territory {
			if t.Services.Instagram.ClientId != "" {
				services.instagram.ClientID = t.Services.Instagram.ClientId
			}
		}
	}
}

// Get recent Instagram for media related to specific tags on Instagram
func InstagramSearch(territoryName string, harvestState config.HarvestState, tag string, options url.Values) (url.Values, config.HarvestState) {
	count, err := strconv.ParseUint(options.Get("count"), 10, 64)
	if err != nil {
		count = 100
	}
	opt := &instagram.Parameters{Count: count}

	// If there is a starting point (pagination / pick up where last harvest left off)
	if options.Get("max_tag_id") != "" {
		opt.MinID = options.Get("min_tag_id")
	}

	media, next, err := services.instagram.Tags.RecentMedia(tag, opt)

	if err == nil {
		for _, item := range media {
			instagramCreatedTime := time.Unix(0, item.CreatedTime*int64(time.Second))
			// Only take instagrams that have a time
			if err == nil && len(item.ID) > 0 {
				harvestState.ItemsHarvested++
				// If this is the most recent tweet in the results, set it's date and id (to be returned) so we can continue where we left off in future harvests
				if harvestState.LastTime.IsZero() || instagramCreatedTime.Unix() > harvestState.LastTime.Unix() {
					harvestState.LastTime = instagramCreatedTime
					harvestState.LastId = item.ID
				}

				// determine gender
				var contributorGender = DetectGender(item.User.FullName)

				// Figure out type (based on if a gender could be detected, name, etc.)
				var contributorType = DetectContributorType(item.User.FullName, contributorGender)

				var contributorCountry = ""
				var contributorRegion = ""
				var contributorCity = ""
				var contributorCityPopulation = int32(0)

				var statusLongitude = 0.0
				var statusLatitude = 0.0
				if item.Location != nil {
					statusLatitude = item.Location.Latitude
					statusLongitude = item.Location.Longitude
				}

				// Contributor location lookup (if no lat/lng was found on the message - try to reduce number of geocode lookups)
				contributorLat := 0.0
				contributorLng := 0.0
				if statusLatitude != 0.0 && statusLatitude != 0.0 {
					reverseLocation := services.geocoder.ReverseGeocode(statusLatitude, statusLongitude)
					contributorRegion = reverseLocation.Region
					contributorCity = reverseLocation.City
					contributorCityPopulation = reverseLocation.Population
					contributorCountry = reverseLocation.Country

					// They don't provide user location of any sort, so use the status lat/lng.
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
				harvestId := GetHarvestMd5(item.ID + "instagram" + territoryName)

				// Retrieve the contributor for the "counts" info (everything else is actually already given with the media - kinda sad to even have to make this request)
				var contributor, contributorErr = services.instagram.Users.Get(item.User.ID)
				contributorFollowedByCount := 0
				contributorMediaCount := 0
				if contributorErr == nil {
					contributorFollowedByCount = contributor.Counts.FollowedBy
					contributorMediaCount = contributor.Counts.Media
				}

				caption := ""
				isQuestion := 0
				if item.Caption != nil {
					caption = item.Caption.Text
					isQuestion = Btoi(IsQuestion(caption, harvestConfig.QuestionRegex))
				}

				message := config.SocialHarvestMessage{
					Time:                      instagramCreatedTime,
					HarvestId:                 harvestId,
					Territory:                 territoryName,
					Network:                   "instagram",
					ContributorId:             item.User.ID,
					ContributorScreenName:     item.User.Username,
					ContributorName:           item.User.FullName,
					ContributorLongitude:      contributorLng,
					ContributorLatitude:       contributorLat,
					ContributorGeohash:        contributorLocationGeoHash,
					ContributorCity:           contributorCity,
					ContributorCityPopulation: contributorCityPopulation,
					ContributorRegion:         contributorRegion,
					ContributorCountry:        contributorCountry,
					ContributorFollowers:      contributorFollowedByCount,
					ContributorStatusesCount:  contributorMediaCount,
					ContributorGender:         contributorGender,
					ContributorType:           contributorType,
					Message:                   caption,
					Sentiment:                 services.sentimentAnalyzer.Classify(caption),
					IsQuestion:                isQuestion,
					MessageId:                 item.ID,
					LikeCount:                 item.Likes.Count,
				}
				// Send to the harvester observer
				go StoreHarvestedData(message)
				LogJson(message, "messages")

				// Keywords are stored on the same collection as hashtags - but under a `keyword` field instead of `tag` field as to not confuse the two.
				// Limit to words 4 characters or more and only return 8 keywords. This could greatly increase the database size if not limited.
				keywords := GetKeywords(caption, 4, 8)
				if len(keywords) > 0 {
					for _, keyword := range keywords {
						if keyword != "" {
							keywordHarvestId := GetHarvestMd5(item.ID + "instagram" + territoryName + keyword)

							// Again, keyword share the same series/table/collection
							hashtag := config.SocialHarvestHashtag{
								Time:                      instagramCreatedTime,
								HarvestId:                 keywordHarvestId,
								Territory:                 territoryName,
								Network:                   "instagram",
								MessageId:                 item.ID,
								ContributorId:             item.User.ID,
								ContributorScreenName:     item.User.Username,
								ContributorName:           item.User.FullName,
								ContributorLongitude:      contributorLng,
								ContributorLatitude:       contributorLat,
								ContributorGeohash:        contributorLocationGeoHash,
								ContributorCity:           contributorCity,
								ContributorCityPopulation: contributorCityPopulation,
								ContributorRegion:         contributorRegion,
								ContributorCountry:        contributorCountry,
								ContributorGender:         contributorGender,
								ContributorType:           contributorType,
								Keyword:                   keyword,
							}
							StoreHarvestedData(hashtag)
							LogJson(hashtag, "hashtags")
						}
					}
				}

				// shared links (the media in Instagram's case...for data query and aggregation reasons, we aren't treating media as part of the message)
				// though, less confusing is Instagram's own API which provides a "link" field (and they are always also the expanded version)
				linkHostName := ""
				pUrl, _ := url.Parse(item.Link)
				linkHostName = pUrl.Host

				// This changes depending on the Type
				preview := ""
				source := ""
				if item.Type == "video" {
					preview = item.Videos.LowResolution.URL
					source = item.Videos.StandardResolution.URL
				}
				if item.Type == "image" {
					preview = item.Images.Thumbnail.URL
					source = item.Images.StandardResolution.URL
				}

				sharedLink := config.SocialHarvestSharedLink{
					Time:                      instagramCreatedTime,
					HarvestId:                 harvestId,
					Territory:                 territoryName,
					Network:                   "instagram",
					MessageId:                 item.ID,
					ContributorId:             item.User.ID,
					ContributorScreenName:     item.User.Username,
					ContributorName:           item.User.FullName,
					ContributorLongitude:      contributorLng,
					ContributorLatitude:       contributorLat,
					ContributorGeohash:        contributorLocationGeoHash,
					ContributorCity:           contributorCity,
					ContributorCityPopulation: contributorCityPopulation,
					ContributorRegion:         contributorRegion,
					ContributorCountry:        contributorCountry,
					ContributorGender:         contributorGender,
					ContributorType:           contributorType,
					Url:                       item.Link,
					ExpandedUrl:               item.Link,
					Host:                      linkHostName,
					Type:                      item.Type,
					Preview:                   preview,
					Source:                    source,
				}
				// Send to the harvester observer
				StoreHarvestedData(sharedLink)
				LogJson(sharedLink, "shared_links")

				// hashtags
				if len(item.Tags) > 0 {
					for _, tag := range item.Tags {
						if len(tag) > 0 {
							hashtagHarvestId := GetHarvestMd5(item.ID + "instagram" + territoryName + tag)

							// TODO: ADD contributor gender, contributor type
							hashtag := config.SocialHarvestHashtag{
								Time:                      instagramCreatedTime,
								HarvestId:                 hashtagHarvestId,
								Territory:                 territoryName,
								Network:                   "instagram",
								MessageId:                 item.ID,
								ContributorId:             item.User.ID,
								ContributorScreenName:     item.User.Username,
								ContributorName:           item.User.FullName,
								ContributorLongitude:      contributorLng,
								ContributorLatitude:       contributorLat,
								ContributorGeohash:        contributorLocationGeoHash,
								ContributorCity:           contributorCity,
								ContributorCityPopulation: contributorCityPopulation,
								ContributorRegion:         contributorRegion,
								ContributorCountry:        contributorCountry,
								ContributorGender:         contributorGender,
								ContributorType:           contributorType,
								Tag:                       tag,
							}
							// Send to the harvester observer
							StoreHarvestedData(hashtag)
							LogJson(hashtag, "hashtags")
						}
					}
				}

			} else {
				log.Println("Could not parse the time from the Instagram, so I'm throwing it away!")
				log.Println(err)
			}

			// Set it, but it won't be used to make requests in the future
			if instagramCreatedTime.Unix() > harvestState.LastTime.Unix() {
				harvestState.LastTime = instagramCreatedTime
			}
		}

		// This is where the id will come from (like Facebook) to be passed back in updated harvestState
		if next.NextMaxID != "" {
			harvestState.LastId = next.NextMaxID
		}
		// ...and always set it for the params, so the loop can get the next page (and if empty string, it should stop)
		options.Set("max_tag_id", next.NextMaxID)
	}

	return options, harvestState
}

// Try to find tags based on a keyword (just return one for now, that's all we need for our purposes)
func InstagramFindTags(keyword string) string {
	tag := ""

	// first, remove all spaces from the keyword because it won't return tags if there are any and sometimes phrases are tags
	// TODO: Maybe take words and shorten them to search for tags ie. "to" might be "2" - people often make tags short
	// then search a few ways and use the tag with the highest MediaCount
	var buffer bytes.Buffer
	keywordPieces := strings.Split(keyword, " ")
	if len(keywordPieces) > 0 {
		for _, piece := range keywordPieces {
			buffer.WriteString(piece)
		}
		keyword = buffer.String()
		buffer.Reset()
	}

	media, _, err := services.instagram.Tags.Search(keyword)
	if err == nil {
		/*for _, item := range media {
			log.Println(item)
		}*/
		if media != nil && len(media) > 0 {
			tag = media[0].Name
		}
	}

	return tag
}

// Harvests Instagram account details to track changes in followers, etc.
func InstagramAccountDetails(territoryName string, account string) {
	contributor, err := services.instagram.Users.Get(account)
	if err == nil {
		now := time.Now()
		// The harvest id in this case will be unique by time / account / network / territory, since there is no post id or anything else like that
		harvestId := GetHarvestMd5(account + now.String() + "instagram" + territoryName)

		row := config.SocialHarvestContributorGrowth{
			Time:          now,
			HarvestId:     harvestId,
			Territory:     territoryName,
			Network:       "instagram",
			ContributorId: contributor.ID,
			Followers:     contributor.Counts.FollowedBy,
			Following:     contributor.Counts.Follows,
			StatusUpdates: contributor.Counts.Media,
		}
		StoreHarvestedData(row)
		LogJson(row, "contributor_growth")
	}
	return
}
