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
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
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
	HarvesterServer struct {
		Port int `json:"port"`
		Cors struct {
			AllowedOrigins []string `json:"allowedOrigins"`
		} `json:"cors"`
		AuthKeys []string `json:"authKeys"`
		Disabled bool     `json:"disabled"`
	} `json:"harvesterServer"`
	ReporterServer struct {
		Port int `json:"port"`
		Cors struct {
			AllowedOrigins []string `json:"allowedOrigins"`
		} `json:"cors"`
		AuthKeys []string `json:"authKeys"`
		Disabled bool     `json:"disabled"`
	} `json:"reporterServer"`
	Database struct {
		Type          string `json:"type"`
		Host          string `json:"host"`
		Port          int    `json:"port"`
		Socket        string `json:"socket"`
		User          string `json:"user"`
		Password      string `json:"password"`
		Database      string `json:"database"`
		RetentionDays int    `json:"retentionDays"`
		PartitionDays int    `json:"partitionDays"`
	} `json:"database"`
	Schema struct {
		Compact bool `json:"compact"`
	} `json:"schema"`
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

// Checks to ensure the data directory exists and is writable. It will be created if not. Config and training data go into this directory.
func CheckDataDir() {
	_, err := os.Stat("./sh-data")
	if err != nil {
		if os.IsNotExist(err) {
			os.Mkdir("./sh-data", 0766)
		}
	}
}

// Copies default or configured training data to `sh-data` if it isn't there already.
func CopyTrainingData() {
	// NOTE: In the future, these will also come from the config so users can use their own data files instead of the defaults.
	// Additionally, there may be training data coming from S3 or other locations (as to not put large files in the GitHub repo).
	files := []map[string]string{
		{"url": "https://raw.githubusercontent.com/SocialHarvest/harvester/master/data/census-female-names.csv", "path": "./sh-data/census-female-names.csv"},
		{"url": "https://raw.githubusercontent.com/SocialHarvest/harvester/master/data/census-male-names.csv", "path": "./sh-data/census-male-names.csv"},
		{"url": "https://raw.githubusercontent.com/SocialHarvest/harvester/master/data/keyword-stop-list.txt", "path": "./sh-data/keyword-stop-list.txt"},
	}

	// TODO: Ultimately, the FileInfo should be checked for size/modification time, etc.
	// Really, an entire "asset" system should be built. Though I'd like to keep it very simple.
	for _, f := range files {
		_, err := os.Stat(f["path"])
		if err != nil {
			if os.IsNotExist(err) {
				log.Println(f["path"] + " does not exist, downloading...")
				out, oErr := os.Create(f["path"])
				defer out.Close()
				if oErr == nil {
					r, rErr := http.Get(f["url"])
					defer r.Body.Close()
					if rErr == nil {
						_, nErr := io.Copy(out, r.Body)
						if nErr != nil {
							log.Println("Failed to copy data file, it will be tried again on next application start.")
							// remove file so another attempt can be made, should something fail
							err = os.Remove(f["path"])
						}
						r.Body.Close()
					}
					out.Close()
				} else {
					log.Println(oErr)
				}
			}
		}
	}
	return
}

// Saves the current configuration to the shared data directory on disk as to not overwrite the original. Unless removed, this will be used should the application be restarted (overwriting the default config).
func SaveConfig(c SocialHarvestConf, f ...string) bool {
	fN := "social-harvest-conf.json"
	if len(f) > 0 {
		fN = f[0]
	}
	// Set the full file path (default) - always store in "sh-data"
	fP := "./sh-data/" + fN
	// If the configuration says to use a different path
	b, err := json.Marshal(c)
	if err == nil {
		err = ioutil.WriteFile(fP, b, 0766)
		if err != nil {
			return false
		}
		return true
	}
	return false
}
