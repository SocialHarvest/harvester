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
	"github.com/tmaiaroto/cron"
	"log"
)

type SocialHarvestSchedule struct {
	Cron *cron.Cron
}

var schedule = SocialHarvestSchedule{}

// Set up the schedule so it is accessible by others and start it
func NewSchedule(config SocialHarvestConf) *SocialHarvestSchedule {
	c := cron.New()

	c.AddFunc("@hourly", func() { log.Println("Every hour") })
	c.AddFunc("0 5 * * * *", func() { log.Println("Every 5 minutes") }, "Optional name here. Useful when inspecting.")

	// Set the initial schedule from config SocialHarvestConf

	c.Start()
	schedule.Cron = c
	return &schedule
}

func AddToSchedule() {

}

func ListSchedule() {
	for _, item := range schedule.Cron.Entries() {
		log.Println(item.Name)
		log.Println(item.Next)
	}
}

// TODO: add func that adds to the schedule... and pass it another fun to run on that schedule...
// Now those can be anonymous or defined on their own in server.go to be reused for other purposes.
