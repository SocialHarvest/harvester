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
	"bitbucket.org/tmaiaroto/go-social-harvest/lib/config"
	"github.com/ChimeraCoder/anaconda"
	"log"
	"net/url"
	//"net/http"
)

type Twitter struct {
	api           *anaconda.TwitterApi
	socialHarvest config.SocialHarvest
}

var twitter = Twitter{}

func InitTwitter(sh config.SocialHarvest) {
	anaconda.SetConsumerKey(sh.Config.Services.Twitter.ApiKey)
	anaconda.SetConsumerSecret(sh.Config.Services.Twitter.ApiSecret)
	twitter.api = anaconda.NewTwitterApi(sh.Config.Services.Twitter.AccessToken, sh.Config.Services.Twitter.AccessTokenSecret)
	twitter.socialHarvest = sh
}

func SearchTwitter(query string, options url.Values) string {
	//v := url.Values{}
	//v.Set("count", "30")
	searchResult, _ := twitter.api.GetSearch("golang", options)
	for _, tweet := range searchResult {
		log.Println(tweet.Text)
	}

	return "end twitter search"
}
