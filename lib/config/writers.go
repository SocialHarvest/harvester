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
	"github.com/tmaiaroto/log4go"
)

// All of the output log writers (fluentd or logstash picks up data from these logs, we'll rotate them and they could even be sent to S3, etc.)
// This right here is database agnostic magic (TODO: Allow this to be configured, perhaps use the XML configuration log4go provides)
type SocialHarvestWriters struct {
	MessagesWriter          log4go.Logger
	SharedLinksWriter       log4go.Logger
	MentionsWriter          log4go.Logger
	HashtagsWriter          log4go.Logger
	ContributorGrowthWriter log4go.Logger
	HarvestWriter           log4go.Logger
}

var writers = SocialHarvestWriters{}

// Initializes the default writers that log harvest data to files on disk
func NewWriters(config SocialHarvestConf) *SocialHarvestWriters {
	// Set up the file log writers (refactor this and the next block?)
	writers.MessagesWriter = make(log4go.Logger)
	writers.SharedLinksWriter = make(log4go.Logger)
	writers.MentionsWriter = make(log4go.Logger)
	writers.HashtagsWriter = make(log4go.Logger)
	writers.ContributorGrowthWriter = make(log4go.Logger)
	writers.HarvestWriter = make(log4go.Logger)

	// We don't need to log at all...
	if config.Logs.Directory != "dev/null" && config.Logs.Directory != "/dev/null" && config.Logs.Directory != "false" && config.Logs.Directory != "" {
		// TODO: Allow greater configuration for this (for rotation, etc.). Perhaps use the XML configuration log4go provides.
		// Messages (posts, status updates, comments, etc.)
		flw := log4go.NewFileLogWriter(config.Logs.Directory+"/messages.log", false)
		flw.SetRotateDaily(true)
		flw.SetFormat("%M")
		writers.MessagesWriter.AddFilter(config.Logs.Directory+"/messages.log", log4go.FINE, flw)
		// Shared Links (articles, media, and other sites)
		flwSl := log4go.NewFileLogWriter(config.Logs.Directory+"/shared_links.log", false)
		flwSl.SetRotateDaily(true)
		flwSl.SetFormat("%M")
		writers.SharedLinksWriter.AddFilter(config.Logs.Directory+"/shared_links.log", log4go.FINE, flwSl)
		// Mentions (who is being talked about, influencers, and conversations between users)
		flwM := log4go.NewFileLogWriter(config.Logs.Directory+"/mentions.log", false)
		flwM.SetRotateDaily(true)
		flwM.SetFormat("%M")
		writers.MentionsWriter.AddFilter(config.Logs.Directory+"/mentions.log", log4go.FINE, flwM)
		// Hashtags (what is being mentioned, tags, hashtags, keywords, etc.)
		flwH := log4go.NewFileLogWriter(config.Logs.Directory+"/hashtags.log", false)
		flwH.SetRotateDaily(true)
		flwH.SetFormat("%M")
		writers.HashtagsWriter.AddFilter(config.Logs.Directory+"/hashtags.log", log4go.FINE, flwH)
		// Contributor growth tracking
		flwCg := log4go.NewFileLogWriter(config.Logs.Directory+"/contributor_growth.log", false)
		flwCg.SetRotateDaily(true)
		flwCg.SetFormat("%M")
		writers.SharedLinksWriter.AddFilter(config.Logs.Directory+"/contributor_growth.log", log4go.FINE, flwCg)
		// Harvest log (last harvest info, to reduce duplicate requests for data, keep history, performance, etc.)
		flwHl := log4go.NewFileLogWriter(config.Logs.Directory+"/harvest.log", false)
		flwHl.SetRotateDaily(true)
		flwHl.SetFormat("%M")
		writers.HarvestWriter.AddFilter(config.Logs.Directory+"/harvest.log", log4go.FINE, flwHl)
	}

	return &writers
}
