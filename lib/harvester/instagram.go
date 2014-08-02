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
	"github.com/SocialHarvest/harvester/lib/config"
	//geohash "github.com/TomiHiltunen/geohash-golang"
	"github.com/carbocation/go-instagram/instagram"
	"log"
	//"sync"
	//"time"
)

type Instagram struct {
	api           *instagram.Client
	socialHarvest config.SocialHarvest
}

var instagramClient = Instagram{}

// Set the client for future use
func NewInstagram(sh config.SocialHarvest) {
	instagramClient.socialHarvest = sh
	instagramClient.api = instagram.NewClient(nil)
	instagramClient.api.ClientID = sh.Config.Services.Instagram.ClientId
}

func ByTag() {
	// There is another API endpoint that searches tags... Keywords could be used to find tags and use those here....
	// But for now, manual tagging may be better. Separate in config? Maybe not, because tags go in without the #. They are essentially keywords.

	opt := &instagram.Parameters{Count: 3}
	media, next, err := instagramClient.api.Tags.RecentMedia("cats", opt)

	if err == nil {
		for _, item := range media {
			log.Println(item)
		}

		log.Println(next)
	}
}
