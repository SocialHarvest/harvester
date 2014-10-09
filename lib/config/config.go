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

type SocialHarvest struct {
	Config   SocialHarvestConf
	Schedule *SocialHarvestSchedule
	Database *SocialHarvestDB
}

type HarvestState struct {
	LastId         string
	LastTime       time.Time
	PagesHarvested int
	ItemsHarvested int
}

type Harvest struct{}

// The configuration structure mapping from JSON
type SocialHarvestConf struct {
	Server struct {
		Port int `json:"port"`
		Cors struct {
			AllowedOrigins []string `json:"allowedOrigins"`
		} `json:"cors"`
		AuthKeys []string `json:"authKeys"`
		Disabled bool     `json:"disabled"`
	} `json:"server"`
	Database struct {
		Type     string `json:"type"`
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Socket   string `json:"socket"`
		User     string `json:"user"`
		Password string `json:"password"`
		Database string `json:"database"`
	} `json:"database"`
	Logs struct {
		Directory string `json:"directory"`
	} `json:"logs"`
	Debug struct {
		WebProfile bool `json:"webProfile"`
		Bugsnag    struct {
			ApiKey       string `json:"apiKey"`
			ReleaseStage string `json:"releaseStage"`
		} `json:"bugsnag"`
	} `json:"debug"`
	Services ServicesConfig `json:"services"`
	Harvest  HarvestConfig  `json:"harvest"`
}

type HarvestConfig struct {
	QuestionRegex string `json:"questionRegex"`
	Territories   []struct {
		Services ServicesConfig `json:"-"`
		Name     string         `json:"name"`
		Content  struct {
			Options struct {
				KeepMessage          bool   `json:"keepMessage"`
				Lang                 string `json:"lang"`
				TwitterGeocode       string `json:"twitterGeocode"`
				OnlyUseInstagramTags bool   `json:"onlyUseInstagramTags"`
			} `json:"options"`
			Keywords      []string `json:"keywords"`
			Urls          []string `json:"urls"`
			InstagramTags []string `json:"instagramTags"`
		} `json:"content"`
		Accounts struct {
			Twitter    []string `json:"twitter"`
			Facebook   []string `json:"facebook"`
			GooglePlus []string `json:"googlePlus"`
			YouTube    []string `json:"youTube"`
			Instagram  []string `json:"instagram"`
		} `json:"accounts"`
		Schedule struct {
			Everything struct {
				Content  string `json:"content"`
				Accounts string `json:"accounts"`
				Streams  string `json:"streams"`
			} `json:"everything"`
			Twitter struct {
				Content  string `json:"content"`
				Accounts string `json:"accounts"`
				Streams  string `json:"streams"`
			} `json:"twitter"`
			Facebook struct {
				Content  string `json:"content"`
				Accounts string `json:"accounts"`
				Streams  string `json:"streams"`
			} `json:"facebook"`
			GooglePlus struct {
				Content  string `json:"content"`
				Accounts string `json:"accounts"`
			} `json:"googlePlus"`
			YouTube struct {
				Content  string `json:"content"`
				Accounts string `json:"accounts"`
				Streams  string `json:"streams"`
			} `json:"youTube"`
		} `json:"schedule"`
		Limits struct {
			MaxResultsPages int    `json:"maxResultsPages"`
			ResultsPerPage  string `json:"resultsPerPage"`
		} `json:"limits"`
	} `json:"territories"`
}

type ServicesConfig struct {
	Twitter struct {
		ApiKey            string `json:"apiKey"`
		ApiSecret         string `json:"apiSecret"`
		AccessToken       string `json:"accessToken"`
		AccessTokenSecret string `json:"accessTokenSecret"`
	} `json:"twitter"`
	Facebook struct {
		AppId     string `json:"appId"`
		AppSecret string `json:"appSecret"`
		AppToken  string `json:"appToken"`
	} `json:"facebook"`
	Google struct {
		ServerKey string `json:"serverKey"`
	} `json:"google"`
	Instagram struct {
		ClientId     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
	} `json:"instagram"`
	MapQuest struct {
		ApplicationKey string `json:"applicationKey"`
	} `json:"mapQuest"`
}
