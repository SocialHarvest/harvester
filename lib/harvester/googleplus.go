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
	//"encoding/json"

	"log"
	"net/http"
)

// TODO: Get from database/config file.
const developerKey = "AIzaSyA1EklkNQ-G9fqkCHEUuv4ERFJqysIUNTg"

func GooglePlusActivitySearch() {

	client := &http.Client{
		Transport: &transport.APIKey{Key: developerKey},
	}
	plusService, err := plus.New(client)

	if err != nil {
		log.Fatal(err)
	}

	// GOOGLE+
	// Activities (posts from people)
	// https://developers.google.com/+/api/latest/activities/search#try-it

	// Comments on Activities (replies to what people post by other people)
	// https://developers.google.com/+/api/latest/comments/list
	// example id: z12bgt3ixljacjnkl23xsjuzokn5jthuf

	// With both comments and activities we get an "actor" object with "displayName": "Denise E. Bodinski" for example...
	// No gender info, demographics, etc. but we can filter that through the gender filter which compares against US Census DB of male/female names...
	// Or technically, make another API call though we need to see what public info is available.

	// https://developers.google.com/+/api/latest/people/get#try-it
	// 113544226688149802325 for example is Richard Branson
	// This is actually an awesome API call. It DOES have gender. It also has any other associated social media accounts (Twitter, etc.)
	// Also organizations the actor is associated with too (school, work). <-- this is interesting because we can, with some effort, look up geo location data based on these things in some cases.
	// There is sometimes a "placesLived" array of objects with a "primary" flag which would give us a better idea for geolocation. Still not lat/lon =( Have to look that up AND it may not be a clean field. ie. "Earth" and "Mars" may be ok like Twitter allows.
	// Occupation and skills too...The data is sparse, but there is actually quite a GOOD bit of demographic info.
	// Occupation and skills are bad ass magic demographic data.

	// This works with a server key....But I had to first try from the developer console and click the OAuth 2.0 switch which prompted me for permission... Weird user experience/setup instructions.
	// We may need to create pages with OAuth for users to approve everything (which we can store in InfluxDB). There was going to be a config page anyway.

	person, err := plusService.People.Get("113544226688149802325").Do()
	if err != nil {
		log.Fatal(err)
	}
	log.Print(person.DisplayName)

}
