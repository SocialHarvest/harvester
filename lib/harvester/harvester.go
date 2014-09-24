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
	"code.google.com/p/google-api-go-client/plus/v1"
	"github.com/ChimeraCoder/anaconda"
	"github.com/SocialHarvest/harvester/lib/config"
	"github.com/carbocation/go-instagram/instagram"
	"github.com/tmaiaroto/geocoder"
	"net"
	"net/http"
	"time"
)

type harvesterServices struct {
	twitter          *anaconda.TwitterApi
	facebookAppToken string
	instagram        *instagram.Client
	googlePlus       *plus.Service
}

var harvestConfig = config.HarvestConfig{}
var services = harvesterServices{}
var socialHarvestDB *config.SocialHarvestDB
var httpClient *http.Client

// Sets up a new harvester with the given configuration (which is comprised of several "services")
func New(configuration config.SocialHarvestConf, database *config.SocialHarvestDB) {
	harvestConfig = configuration.Harvest
	// Now set up all the services with the configuration
	NewTwitter(configuration.Services)
	NewFacebook(configuration.Services)
	NewInstagram(configuration.Services)
	NewGooglePlus(configuration.Services)
	NewGeocoder(configuration.Services)

	// StoreHarvestedData() needs this now
	socialHarvestDB = database

	// Internal logging (log4go became problematic for concurrency and I've found a better solution in less than 100 lines now anyway)
	NewLoggers(configuration.Logs.Directory)

	// Set up an http.Client for a variety of uses including expanding shortened URLs.
	httpClient = &http.Client{
		Transport: &TimeoutTransport{
			Transport: http.Transport{
				Dial: func(netw, addr string) (net.Conn, error) {
					//log.Printf("dial to %s://%s", netw, addr)
					return net.Dial(netw, addr) // Regular ass dial.
				},
			},
			// RoundTripTimeout: time.Millisecond * 200, // <--- what the author had
			// RoundTripTimeout: time.Nanosecond * 10, // <--- still was completing requests in this amount of time (holy smokes that's fast)!
			// I'm going to go a little tiny bit longer because I don't know what kind of machine this will run on.
			// Though the geocoding service is fast and the payload small, so requests should be fast.
			//RoundTripTimeout: time.Millisecond * 300,
			RoundTripTimeout: time.Second * 5,
		},
	}
}

// Sets the API key from configuration (or possibly Social Harvest API)
func NewGeocoder(servicesConfiguration config.ServicesConfig) {
	if servicesConfiguration.MapQuest.ApplicationKey != "" {
		geocoder.NewGeocoder()
		geocoder.SetAPIKey(servicesConfiguration.MapQuest.ApplicationKey)
	}
}

// Rather than using an observer, just call this function instead (the observer was causing memory leaks)
// TODO: Look back into channels in the future because I like the idea of pub/sub. In the future it could expand into something useful.
// The thing I don't like (and why I used the observer) is passing all the configuration stuff around.
func StoreHarvestedData(message interface{}) {
	// Write to database (if configured)
	socialHarvestDB.StoreRow(message)
}
