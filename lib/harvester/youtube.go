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
	"code.google.com/p/google-api-go-client/youtube/v3"
	//"encoding/json"
	"log"
	"net/http"
)

// TODO: Get from database/config file.
// NOTE: this is shared among all the .go files, so the name changed here. can't re-define it.
// I believe the files are processed alphabetically...but we'll use a method to look up the keys and manage all that in one spot.
const youtubeDeveloperKey = "AIzaSyA1EklkNQ-G9fqkCHEUuv4ERFJqysIUNTg"

// YOUTUBE HARVESTER
// All done with public API access key. Easy. Awesome. Pretty much all we need...

type VideoResponse struct {
	YtVideoId string `json:"yt_video_id"`
	Title     string `json:"title"`
}

type VideosResponse struct {
	Videos []VideoResponse `json:"videos"`
}

func YoutubeVideoSearch(query string) *youtube.VideoListResponse {

	client := &http.Client{
		Transport: &transport.APIKey{Key: youtubeDeveloperKey},
	}

	service, err := youtube.New(client)

	if err != nil {
		log.Fatal(err)
	}

	// SEARCH
	//fmt.Println(query)
	/*
	   call := service.Search.List("id,snippet").
	       Q(query + " live").Type("video").MaxResults(25)

	   response, err := call.Do()
	   if err != nil {
	       log.Fatal(err)
	   }

	   videos := make([]VideoResponse, 0)

	   for _, item := range response.Items {
	       log.Print(item.Snippet)
	       video := VideoResponse{item.Id.VideoId, item.Snippet.Title}
	       videos = append(videos, video)
	       //log.Print(video)
	   }

	   videoStruct := VideosResponse{videos}

	   jsonData, err := json.Marshal(videoStruct)

	   if err != nil {
	       log.Fatal(err)
	   }

	   return jsonData
	*/

	// VIDEO STATISTICS & INFO
	// https://developers.google.com/youtube/v3/docs/videos/list#try-it
	// id,snippet,statistics <-- among the most basic/useful. views, likes/dislikes, etc.
	// status <-- maybe useful. tells if the video can be embedded... and if public stats are even viewable
	// contentDetails <-- 2d, 3d ? HD? sd... duration etc.
	// player <-- the actual embed code...we can build an embed code given the id, so not too useful (unless they vary for some reason and this is easier or just convenience)

	// https://developers.google.com/freebase/v1/topic-overview
	// topicDetails provides Freebase topics! VERY cool. Showing a division of topics for a given query might be a nice statistic.
	// Especially when it comes to secondary topics... ie. "Obama" sure you're talking about the president, but what other topics are popular?

	// NOTE: The "chart" filter... only one value for now: mostPopular – Return the most popular videos for the specified content region and video category.
	// This can be very handy. Note the "category" in this case is not Freebase (can be found on video info with snippet).
	// https://developers.google.com/youtube/v3/docs/videoCategories/list#try-it  <-- 20 for example is "Gaming"

	// Also note: it is possible to request info for multiple videos in a single request.
	/*
	   call := service.Videos.List("id,snippet,statistics").
	       Id("R24CUMBBTbU")

	   response, err := call.Do()
	   if err != nil {
	       log.Fatal(err)
	   }

	   videos := make([]VideoResponse, 0)

	   for _, item := range response.Items {
	       log.Print(item.Statistics)
	   }

	   if err != nil {
	       log.Fatal(err)
	   }
	*/

	// TOP VIDEOS FOR "Gaming" exmaple

	call := service.Videos.List("id,snippet,statistics").
		Chart("mostPopular").VideoCategoryId("20")

	response, err := call.Do()
	if err != nil {
		log.Fatal(err)
	}

	for _, item := range response.Items {
		//		log.Print(item.Snippet.Title)
		log.Print(item.Statistics)
	}

	if err != nil {
		log.Fatal(err)
	}

	return response
	// VIDEO COMMENTS *very handy for sentiment and such* (also Google+ user ids...maybe a filter can get more details...who knows)
	// For now, API v3 doesn't have comments (though API v2 has been deprecated as of March 4th, 2014)
	// There is an RSS feed from API v2 which is public. yay. https://gdata.youtube.com/feeds/api/videos/R24CUMBBTbU/comments?orderby=published
	// Annoying we now need to bring in an RSS parser...maybe API v3 will have all this soon. I hope.

	// There's also some stuff for channels which might be useful for a more specific type of analytics (for companies like Machinima for example)
	// Though the analytics api has reports for channels...which may also require auth. So to start with, definitely don't worry about all that.

}
