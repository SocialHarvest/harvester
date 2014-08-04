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
 *
 * Otherwise, instead of a few insert errors failing silently, we'd have to resort to more expensive GROUP queries or other aggregations.
 *
 * Some other conventions and requirements:
 *  - Key names are to have underscores (to support potential case insentive databases, so "harvest_id" and not "harvestId")
 *  - All data series structures here will have simple values. Integer, Float, String, etc. (uint64 may not be fully supported by every database and no arrays or objects)
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
 * The bottom line: While we could have each harvester could return its own struct -> JSON, we need to standardize the data.
 * Why? Unknown data stores. Schema-less is awesome, but we can't count on everyone using it.
 *
 * Exceptions: Social Harvest will allow data to be filtered (modified going in). One easy way to do this is through Fluentd.
 * However, additional methods will be made available in the future. More on this later.
 */

// Where to store this stuff (log file, collection, and table names)
var SeriesCollections = map[string]string{
	"SocialHarvestMessage":     "messages",
	"SocialHarvestSharedLink":  "shared_links",
	"SocialHarvestSharedMedia": "shared_media",
	"SocialHarvestMention":     "mentions",
	"SocialHarvestContributor": "contributors",
	"SocialHarvestQuestion":    "questions",
	"SocialHarvestHarvest":     "harvest",
}

// Posts, status updates, comments, etc.
type SocialHarvestMessage struct {
	Time                        time.Time `json:"time" db:"time" bson:"time"`
	HarvestId                   string    `json:"harvest_id" db:"harvest_id" bson:"harvest_id"`
	Territory                   string    `json:"territory" db:"territory" bson:"territory"`
	Network                     string    `json:"network" db:"network" bson:"network"`
	MessageId                   string    `json:"message_id" db:"message_id" bson:"message_id"`
	ContributorId               string    `json:"contributor_id" db:"contributor_id" bson:"contributor_id"`
	ContributorScreenName       string    `json:"contributor_screen_name" db:"contributor_screen_name" bson:"contributor_screen_name"`
	ContributorFacebookCategory string    `json:"contributor_facebook_category" db:"contributor_facebook_category" bson:"contributor_facebook_category"`
	IsoLanguageCode             string    `json:"iso_language_code" db:"iso_language_code" bson:"iso_language_code"`
	Longitude                   float64   `json:"longitude" db:"longitude" bson:"longitude"`
	Latitude                    float64   `json:"latitude" db:"latitude" bson:"latitude"`
	Geohash                     string    `json:"geohash" db:"geohash" bson:"geohash"`
	Message                     string    `json:"message" db:"message" bson:"message"`
	FacebookShares              int       `json:"facebook_shares" db:"facebook_shares" bson:"facebook_shares"`
	// TODO: when we start gathering from Twitter, etc. add in twitter_mentions, etc.
}

type SocialHarvestQuestion struct {
	Time                  time.Time `json:"time" db:"time" bson:"time"`
	HarvestId             string    `json:"harvest_id" db:"harvest_id" bson:"harvest_id"`
	Territory             string    `json:"territory" db:"territory" bson:"territory"`
	Network               string    `json:"network" db:"network" bson:"network"`
	ContributorId         string    `json:"contributor_id" db:"contributor_id" bson:"contributor_id"`
	ContributorScreenName string    `json:"contributor_screen_name" db:"contributor_screen_name" bson:"contributor_screen_name"`
	IsoLanguageCode       string    `json:"iso_language_code" db:"iso_language_code" bson:"iso_language_code"`
	Longitude             float64   `json:"longitude" db:"longitude" bson:"longitude"`
	Latitude              float64   `json:"latitude" db:"latitude" bson:"latitude"`
	Geohash               string    `json:"geohash" db:"geohash" bson:"geohash"`
	MessageId             string    `json:"message_id" db:"message_id" bson:"message_id"`
	Message               string    `json:"message" db:"message" bson:"message"`
}

// Shared URLs include everything (TODO: Possibly scrape those pages to get extra information to get semantic data being discussed/shared for a particular territory)
type SocialHarvestSharedLink struct {
	Time                        time.Time `json:"time" db:"time" bson:"time"`
	HarvestId                   string    `json:"harvest_id" db:"harvest_id" bson:"harvest_id"`
	Territory                   string    `json:"territory" db:"territory" bson:"territory"`
	Network                     string    `json:"network" db:"network" bson:"network"`
	MessageId                   string    `json:"message_id" db:"message_id" bson:"message_id"`
	ContributorId               string    `json:"contributor_id" db:"contributor_id" bson:"contributor_id"`
	ContributorScreenName       string    `json:"contributor_screen_name" db:"contributor_screen_name" db:"contributor_screen_name"`
	ContributorFacebookCategory string    `json:"contributor_facebook_category" db:"contributor_facebook_category" bson:"contributor_facebook_category"`
	Url                         string    `json:"url" db:"url" bson:"url"`
	ExpandedUrl                 string    `json:"expanded_url" db:"expanded_url" bson:"expanded_url"`
	Host                        string    `json:"host" db:"host" bson:"host"`
	FacebookShares              int       `json:"facebook_shares" db:"facebook_shares" bson:"facebook_shares"`
}

// Images, videos, and the like
type SocialHarvestSharedMedia struct {
	Time                        time.Time `json:"time" db:"time" bson:"time"`
	HarvestId                   string    `json:"harvest_id" db:"harvest_id" bson:"harvest_id"`
	Territory                   string    `json:"territory" db:"territory" bson:"territory"`
	Network                     string    `json:"network" db:"network" bson:"network"`
	MessageId                   string    `json:"message_id" db:"message_id" bson:"message_id"`
	ContributorId               string    `json:"contributor_id" db:"contributor_id" bson:"contributor_id"`
	ContributorScreenName       string    `json:"contributor_screen_name" db:"contributor_screen_name" db:"contributor_screen_name"`
	ContributorFacebookCategory string    `json:"contributor_facebook_category" db:"contributor_facebook_category" bson:"contributor_facebook_category"`
	Type                        string    `json:"type" db:"type" bson:"type"`
	Preview                     string    `json:"preview" db:"preview" bson:"preview"`
	Source                      string    `json:"source" db:"source" bson:"source"`
	Url                         string    `json:"url" db:"url" bson:"url"`
	ExpandedUrl                 string    `json:"expanded_url" db:"expanded_url" bson:"expanded_url"`
	Host                        string    `json:"host" db:"host" bson:"host"`
}

// When contributors mention other contributors (and from where - useful for customer base for example)
type SocialHarvestMention struct {
	Time                        time.Time `json:"time" db:"time" bson:"time"`
	HarvestId                   string    `json:"harvest_id" db:"harvest_id" bson:"harvest_id"`
	Territory                   string    `json:"territory" db:"territory" bson:"territory"`
	Network                     string    `json:"network" db:"network" bson:"network"`
	MessageId                   string    `json:"message_id" db:"message_id" bson:"message_id"`
	ContributorId               string    `json:"contributor_id" db:"contributor_id" bson:"contributor_id"`
	ContributorScreenName       string    `json:"contributor_screen_name" db:"contributor_screen_name" db:"contributor_screen_name"`
	ContributorFacebookCategory string    `json:"contributor_facebook_category" db:"contributor_facebook_category" bson:"contributor_facebook_category"`
	MentionedScreenName         string    `json:"mentioned_screen_name" db:"mentioned_screen_name" bson:"mentioned_screen_name"`
	MentionedId                 string    `json:"mentioned_id" db:"mentioned_id" bson:"mentioned_id"`
	MentionedType               string    `json:"mentioned_type" db:"mentioned_type" bson:"mentioned_type"`
	Longitude                   float64   `json:"longitude" db:"longitude" bson:"longitude"`
	Latitude                    float64   `json:"latitude" db:"latitude" bson:"latitude"`
	Geohash                     string    `json:"geohash" db:"geohash" bson:"geohash"`
	IsoLanguageCode             string    `json:"iso_language_code" db:"iso_language_code" bson:"iso_language_code"`
}

// As much information about contributors (users, pages, YouTube channels?) that can be gathered
type SocialHarvestContributor struct {
	Time                        time.Time `json:"time" db:"time" bson:"time"`
	HarvestId                   string    `json:"harvest_id" db:"harvest_id" bson:"harvest_id"`
	Territory                   string    `json:"territory" db:"territory" bson:"territory"`
	Network                     string    `json:"network" db:"network" bson:"network"`
	ContributorId               string    `json:"contributor_id" db:"contributor_id" bson:"contributor_id"`
	ContributorScreenName       string    `json:"contributor_screen_name" db:"contributor_screen_name" db:"contributor_screen_name"`
	ContributorFacebookCategory string    `json:"contributor_facebook_category" db:"contributor_facebook_category" bson:"contributor_facebook_category"`
	IsoLanguageCode             string    `json:"iso_language_code" db:"iso_language_code" bson:"iso_language_code"`
	Gender                      int       `json:"gender" db:"gender" bson:"gender"`
	// This should include first and last where possible (I see no need, currently, to separate first and last name - name can be company name too)
	Name string `json:"name"`

	// There's going to be a lot of social network specific fields...Any network not having data for the field can pass empty values.
	// Remember, contributors can be users OR pages when it comes to Facebook.
	About             string  `json:"about" db:"about" bson:"about"`
	Checkins          int     `json:"checkins" db:"checkins" bson:"checkins"`
	CompanyOverview   string  `json:"company_overview" db:"company_overview" bson:"company_overview"`
	Description       string  `json:"description" db:"description" bson:"description"`
	Founded           string  `json:"founded" db:"founded" bson:"founded"`
	GeneralInfo       string  `json:"general_info" db:"general_info" bson:"general_info"`
	Likes             int     `json:"likes" db:"likes" bson:"likes"`
	Link              string  `json:"link" db:"link" bson:"link"` // link typically to the account on the network
	Street            string  `json:"street" db:"street" bson:"street"`
	City              string  `json:"city" db:"city" bson:"city"`
	State             string  `json:"state" db:"state" bson:"state"`
	Zip               string  `json:"zip" db:"zip" bson:"zip"`
	Country           string  `json:"country" db:"country" bson:"country"`
	Longitude         float64 `json:"longitude" db:"longitude" bson:"longitude"`
	Latitude          float64 `json:"latitude" db:"latitude" bson:"latitude"`
	Geohash           string  `json:"geohash" db:"geohash" bson:"geohash"`
	Phone             string  `json:"phone" db:"phone" bson:"phone"`
	TalkingAboutCount int     `json:"talking_about_count" db:"talking_about_count" bson:"talking_about_count"`
	WereHereCount     int     `json:"were_here_count" db:"were_here_count" bson:"were_here_count"`
	Url               string  `json:"url" db:"url" bson:"url"` // "Website" in Facebook's API - this is a link out to the person or company's web site
	Products          string  `json:"products" db:"products" bson:"products"`
}

// Changes in growth and reach over time for a contributor.
// In theory, an API call could always be made to get details about the contributor, so it need not exist in the contributors collection...Though it likely will.
// It would be interesting to track all of this for every contributor discovered, but API rate limits restrict us from doing that.
type SocialHarvestContributorGrowth struct {
	Time              time.Time `json:"time" db:"time" bson:"time"`
	HarvestId         string    `json:"harvest_id" db:"harvest_id" bson:"harvest_id"`
	Territory         string    `json:"territory" db:"territory" bson:"territory"`
	Network           string    `json:"network" db:"network" bson:"network"`
	ContributorId     string    `json:"contributor_id" db:"contributor_id" bson:"contributor_id"`
	Likes             int       `json:"likes" db:"likes" bson:"likes"`
	TalkingAboutCount int       `json:"talking_about_count" db:"talking_about_count" bson:"talking_about_count"`
	WereHereCount     int       `json:"were_here_count" db:"were_here_count" bson:"were_here_count"`
	Checkins          int       `json:"checkins" db:"checkins" bson:"checkins"`
	Followers         int       `json:"followers" db:"followers" bson:"followers"`
	Views             int       `json:"views" db:"views" bson:"views"`
	Subscribers       int       `json:"subscribers" db:"subscribers" bson:"subscribers"`
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
