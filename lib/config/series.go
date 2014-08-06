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

package config

import (
	"time"
)

/**
 * The following structures represent Social Harvest's base data schema.
 *
 * Most series will have a time and a harvest id. This is for two reasons:
 * 1. Most of the data being gathered is time sensitive and will need to be queried by date range (that's why we use the word "series" to describe the collections, tables, etc.)
 * 2. The harvest id is simply a concatenation of whatever item id (from the harvested network), network name, and territory name to help with duplicate data
 *
 * Duplicate data is completely unavoidable unfortunately. For example, the same Facebook post could come up in two different keyword searches.
 * However, it is ok to have the same messages appear in multiple territories.
 *
 * IMPORTANT: Databases should have a unique index on this "harvest_id" field.
 * Otherwise, instead of a few insert errors failing silently, we'd have to resort to more expensive GROUP queries or other aggregations.
 * This makes dupes less of an issue.
 *
 * Other data is duplicated too across series in some cases. For example, location data. Sometimes this is done for tracking slight changes and
 * other times for convenience in the data schema. It could make for a simpler query for example (might avoid a few expensives JOINs).
 *
 * Some other conventions and requirements:
 *  - Key names are to have underscores (to support potential case insentive databases, so "harvest_id" and not "harvestId")
 *  - All data series structures here will have simple values. Integer, Float, String, etc. (uint64 may not be fully supported by every database and obviously no arrays or objects)
 *  - Boolean values will be converted to int 0 or 1
 *  - Gender values are stored as (signed) int. -1 for female, 1 for male, 0 for unknown (this is to allow simple math to determine if there are more of a certain gender, etc.)
 *  - Sentiment also will be stored in number format for much the same reason
 *  - id values will be strings regardless of what the harvested source returns (avoid uint64 and keep schema consistent)
 *
 * NOTE: lat/lng can come from the contributor data. It will be assumed the contributor is posting from their primary location, but sometimes posts can carry location based
 * data with them and that will take priority for accuracy. ie. mobile devices can report a more specific location within a city that a company page may not provide...
 * Also, we will try to get contributor location based on reverse geo lookup when no lat/lng is provided. This too can be shared back to the message data.
 * This is for convenience (less JOINs or multiple queries). Technically, the message need not have location.
 *
 * The bottom line: While we could have each harvester return its own struct -> JSON, we need to standardize the data.
 * Why? Unknown data stores. Schema-less is awesome, but we can't count on everyone using it.
 * It would also be nice to standardize common things across social networks.
 *
 * Exceptions: Social Harvest will allow data to be filtered (modified going in). One easy way to do this is through Fluentd.
 * However, additional methods will be made available in the future. This could make getting data back out challenging. More on this later.
 */

// Where to store this stuff (log file, collection, and table names)
var SeriesCollections = map[string]string{
	"SocialHarvestMessage":           "messages",
	"SocialHarvestSharedLink":        "shared_links",
	"SocialHarvestMention":           "mentions",
	"SocialHarvestContributorGrowth": "contributor_growth",
	"SocialHarvestHarvest":           "harvest",
}

// Posts, status updates, comments, etc.
type SocialHarvestMessage struct {
	Time      time.Time `json:"time" db:"time" bson:"time"`
	HarvestId string    `json:"harvest_id" db:"harvest_id" bson:"harvest_id"`
	Territory string    `json:"territory" db:"territory" bson:"territory"`
	Network   string    `json:"network" db:"network" bson:"network"`
	MessageId string    `json:"message_id" db:"message_id" bson:"message_id"`
	// contributor information (some transient information, we take note at the time of the message - can help with a contributor's influence at the time of message - or we can track how certain messages helped a contributor gain influence - OR we can say only show me messages from contributors who have X followers, etc.)
	ContributorId              string  `json:"contributor_id" db:"contributor_id" bson:"contributor_id"`
	ContributorScreenName      string  `json:"contributor_screen_name" db:"contributor_screen_name" bson:"contributor_screen_name"`
	ContributorName            string  `json:"contributor_name" db:"contributor_name" bson:"contributor_name"`
	ContributorGender          int     `json:"contributor_gender" db:"contributor_gender" bson:"contributor_gender"`
	ContributorType            string  `json:"contributor_type" db:"contributor_type" bson:"contributor_type"`
	ContributorLongitude       float64 `json:"contributor_longitude" db:"contributor_longitude" bson:"contributor_longitude"`
	ContributorLatitude        float64 `json:"contributor_latitude" db:"contributor_latitude" bson:"contributor_latitude"`
	ContributorGeohash         string  `json:"contributor_geohash" db:"contributor_geohash" bson:"contributor_geohash"`
	ContributorIsoLanguageCode string  `json:"contributor_iso_language_code" db:"contributor_iso_language_code" bson:"contributor_iso_language_code"`

	// Stateful data that changes, think about the value of having it...maybe remove it... API calls can always be made to get this current info.
	// But this kinda gives a user an idea for influencers (at the harvest time at least). So while it's definitely dated...It could be used as a
	// decent filter, ie. only show users who have over a million followers, etc.
	ContributorLikes         int `json:"contributor_likes" db:"contributor_likes" bson:"contributor_likes"`
	ContributorStatusesCount int `json:"contributor_statuses_count" db:"contributor_statuses_count" bson:"contributor_statuses_count"`
	ContributorListedCount   int `json:"contributor_listed_count" db:"contributor_listed_count" bson:"contributor_listed_count"`
	ContributorFollowers     int `json:"contributor_followers" db:"contributor_followers" bson:"contributor_followers"`
	// This value is technically stateful, but can be treated as stateless because it doesn't really get revoked and change back...
	ContribtuorVerified int `json:"contributor_verified" db:"contributor_verified" bson:"contributor_verified"` // Twitter for sure, but I think other networks too?

	// this is all about the message (location and language may actually differ from the contributor normal values or may be the exact same... leave for now)
	// i can see a case where people want to understand if users are tweeting from their home address or not. plus we geocode cities and such. so a contributor can
	// be from "Austin, TX" but tweet in another state or from different and distinct places in the same city. we want to see that movement. it can be visualized
	// much like mentions can. and it also serves as a location detail with a fallback to the user's assumed location (which may be less accurate).
	Longitude       float64 `json:"longitude" db:"longitude" bson:"longitude"`
	Latitude        float64 `json:"latitude" db:"latitude" bson:"latitude"`
	Geohash         string  `json:"geohash" db:"geohash" bson:"geohash"`
	IsoLanguageCode string  `json:"iso_language_code" db:"iso_language_code" bson:"iso_language_code"`
	Message         string  `json:"message" db:"message" bson:"message"`
	IsQuestion      int     `json:"is_question" db:"is_question" bson:"is_question"`
	Category        string  `json:"category" db:"category" bson:"category"`
	// Note these values are at the time of harvest. it may be confusing enough to not need these values stored...but how long can we track each message? API rate limits...
	// TODO: Maybe remove these? (think on it) also these technically don't need prefixes because we have the "network" field.
	FacebookShares       int `json:"facebook_shares" db:"facebook_shares" bson:"facebook_shares"`
	TwitterRetweetCount  int `json:"twitter_retweet_count" db:"twitter_retweet_count" bson:"twitter_retweet_count"`
	TwitterFavoriteCount int `json:"twitter_favorite_count" db:"twitter_favorite_count" bson:"twitter_favorite_count"`
}

// Shared URLs. The "type" will tell us if it's media (video, photo, etc.) or HTML. It's more about content type. Not necessarily "blog" or something.
// TODO: Possibly scrape those pages to get extra information to get semantic data being discussed/shared for a particular territory. This would enrich things like "type" ...
type SocialHarvestSharedLink struct {
	Time                  time.Time `json:"time" db:"time" bson:"time"`
	HarvestId             string    `json:"harvest_id" db:"harvest_id" bson:"harvest_id"`
	Territory             string    `json:"territory" db:"territory" bson:"territory"`
	Network               string    `json:"network" db:"network" bson:"network"`
	MessageId             string    `json:"message_id" db:"message_id" bson:"message_id"`
	ContributorId         string    `json:"contributor_id" db:"contributor_id" bson:"contributor_id"`
	ContributorScreenName string    `json:"contributor_screen_name" db:"contributor_screen_name" db:"contributor_screen_name"`
	ContributorName       string    `json:"contributor_name" db:"contributor_name" db:"contributor_name"`
	ContributorGender     int       `json:"contributor_gender" db:"contributor_gender" bson:"contributor_gender"`
	ContributorType       string    `json:"contributor_type" db:"contributor_type" bson:"contributor_type"`
	Type                  string    `json:"type" db:"type" bson:"type"`
	Preview               string    `json:"preview" db:"preview" bson:"preview"`
	Source                string    `json:"source" db:"source" bson:"source"`
	Url                   string    `json:"url" db:"url" bson:"url"`
	ExpandedUrl           string    `json:"expanded_url" db:"expanded_url" bson:"expanded_url"`
	Host                  string    `json:"host" db:"host" bson:"host"`
}

// When contributors mention other contributors (and from where - useful for tracking customer base for example). This series tells a good story visually (hopefully on a map).
// Note: "Type" is directly applicable to Facebook (users vs pages), but we can expand upon this (we have a network value too). So things like "business" or "product" can be added.
// This would be helpful if a user wanted to filter for any companies being mentioned on Twitter for example. Despite Twitter not having a "type" ... This would require a special
// process on the data of course, but that's ok. It's set to do that now. We can expand upon it from there. A case could be made for even more fields here, but this is ok for now.
// Yes there is repeated information that doesn't change (like gender, etc.) but that's also ok. It may require more storage in the database, but it makes for a more efficient query.
type SocialHarvestMention struct {
	Time      time.Time `json:"time" db:"time" bson:"time"`
	HarvestId string    `json:"harvest_id" db:"harvest_id" bson:"harvest_id"`
	Territory string    `json:"territory" db:"territory" bson:"territory"`
	Network   string    `json:"network" db:"network" bson:"network"`
	MessageId string    `json:"message_id" db:"message_id" bson:"message_id"`

	ContributorId              string  `json:"contributor_id" db:"contributor_id" bson:"contributor_id"`
	ContributorScreenName      string  `json:"contributor_screen_name" db:"contributor_screen_name" db:"contributor_screen_name"`
	ContributorName            string  `json:"contributor_name" db:"contributor_name" db:"contributor_name"`
	ContributorGender          int     `json:"contributor_gender" db:"contributor_gender" bson:"contributor_gender"`
	ContributorType            string  `json:"contributor_type" db:"contributor_type" bson:"contributor_type"`
	ContributorLongitude       float64 `json:"contributor_longitude" db:"contributor_longitude" bson:"contributor_longitude"`
	ContributorLatitude        float64 `json:"contributor_latitude" db:"contributor_latitude" bson:"contributor_latitude"`
	ContributorGeohash         string  `json:"contributor_geohash" db:"contributor_geohash" bson:"contributor_geohash"`
	ContributorIsoLanguageCode string  `json:"contributor_iso_language_code" db:"contributor_iso_language_code" bson:"contributor_iso_language_code"`

	MentionedId              string  `json:"mentioned_id" db:"mentioned_id" bson:"mentioned_id"`
	MentionedScreenName      string  `json:"mentioned_screen_name" db:"mentioned_screen_name" bson:"mentioned_screen_name"`
	MentionedName            string  `json:"mentioned_name" db:"mentioned_name" bson:"mentioned_name"`
	MentionedGender          int     `json:"mentioned_gender" db:"mentioned_gender" bson:"mentioned_gender"`
	MentionedType            string  `json:"mentioned_type" db:"mentioned_type" bson:"mentioned_type"`
	MentionedLongitude       float64 `json:"mentioned_longitude" db:"mentioned_longitude" bson:"mentioned_longitude"`
	MentionedLatitude        float64 `json:"mentioned_latitude" db:"mentioned_latitude" bson:"mentioned_latitude"`
	MentionedGeohash         string  `json:"mentioned_geohash" db:"mentioned_geohash" bson:"mentioned_geohash"`
	MentionedIsoLanguageCode string  `json:"mentioned_iso_language_code" db:"mentioned_iso_language_code" bson:"mentioned_iso_language_code"`
}

// Changes in growth and reach over time for a contributor.
// It would be interesting to track all of this for every contributor discovered, but API rate limits restrict us from doing that.
// So this will only track for accounts under the "accounts" section of the harvest configuration.
// NOTE: contributor details (like location, about, website url, etc.) can be obtained when necessary via the service's API on the front-end. A lot of that data changes.
type SocialHarvestContributorGrowth struct {
	Time      time.Time `json:"time" db:"time" bson:"time"`
	HarvestId string    `json:"harvest_id" db:"harvest_id" bson:"harvest_id"`
	Territory string    `json:"territory" db:"territory" bson:"territory"`
	Network   string    `json:"network" db:"network" bson:"network"`
	// We can look up additional contributor details (like name, location, website URL, etc.) via service API calls as needed. It doesn't change often.
	// So storing in the database would really be wasteful.
	ContributorId string `json:"contributor_id" db:"contributor_id" bson:"contributor_id"`
	// NOTE: No need to prefix fields with network...because we have the network field. Unused fields will simply be empty.
	// It is also possible for networks to share fields (if they use the same semantics / have the same kind of data).

	// Facebook specific (mostly)
	Likes             int `json:"likes" db:"likes" bson:"likes"`
	TalkingAboutCount int `json:"talking_about_count" db:"talking_about_count" bson:"talking_about_count"`
	WereHereCount     int `json:"were_here_count" db:"were_here_count" bson:"were_here_count"`
	Checkins          int `json:"checkins" db:"checkins" bson:"checkins"`

	// Youtube (mostly)
	Views       int `json:"views" db:"views" bson:"views"`
	Subscribers int `json:"subscribers" db:"subscribers" bson:"subscribers"`

	// Twitter specific (mostly)
	StatusesCount int `json:"statuses_count" db:"statuses_count" bson:"statuses_count"`
	ListedCount   int `json:"listed_count" db:"listed_count" bson:"listed_count"`
	Followers     int `json:"followers" db:"followers" bson:"followers"`
	Following     int `json:"following" db:"following" bson:"following"`
}

// Used for efficiently harvesting (help avoid gathering duplicate data), running through paginated results from APIs, as well as information about harvester performance.
type SocialHarvestHarvest struct {
	Territory         string    `json:"territory" db:"territory" bson:"territory"`
	Network           string    `json:"network" db:"network" bson:"network"`
	Action            string    `json:"action" db:"action" bson:"action"`
	Value             string    `json:"value" db:"value" bson:"value"`
	LastTimeHarvested time.Time `json:"last_time_harvested" db:"last_time_harvested" bson:"last_time_harvested"`
	LastIdHarvested   string    `json:"last_id_harvested" db:"last_id_harvested" bson:"last_id_harvested"`
	ItemsHarvested    int       `json:"items_harvested" db:"items_harvested" bson:"items_harvested"`
	HarvestTime       time.Time `json:"harvest_time" db:"harvest_time" bson:"harvest_time"`
}
