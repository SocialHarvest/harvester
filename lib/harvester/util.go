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
	//"github.com/jasonwinn/geocoder"
	"crypto/md5"
	"encoding/hex"
	"log"
	"regexp"
	"strings"
)

// Turns the harvest id into an md5 string (a simple concatenation would work but some databases such as MySQL have a limit on unique key values so md5 fits without worry)
func GetHarvestMd5(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

// Simple method for converting locale values like "en_US" (or even en-US) to ISO 639-1 (which would just be "en")
func LocaleToLanguageISO(code ...string) string {
	// I think it's just safe to take the first two characters, no?

	iso639 := ""
	if len(code) > 0 {
		underscoreLocale := strings.Split(code[0], "_")
		if underscoreLocale[0] != "" {
			iso639 = underscoreLocale[0]
		}
		dashLocale := strings.Split(code[0], "_")
		if dashLocale[0] != "" {
			iso639 = dashLocale[0]
		}
	}

	return iso639
}

// Detects questions in messages
func IsQuestion(text string, regexString ...string) bool {
	// Default question regex matches strings with a question mark if it has letters before it and a space or new line afterward.
	// Does not match URLs with querystrings, etc.
	pattern := `(?i)\w\?(\s|\n)`

	// TODO: Maybe use a classifier or neural network to determine liklihood of question...I think regex is enough though...And it doesn't require training.
	// Rhetorical questions are going to be the gotchya's...But I imagine humans weed through questions to determine which to (maybe) answer anyway.
	// A specific regex for matching questions can be defined in the Social Harvest config.
	// For example: (?i)\w\?(\s|\n)|i.*(would|like|need|have|love).*to know\sabout[^a-zA-Z]
	// Of course it's in JSON so it needs to be escaped. The above example would look like:
	// "questionRegex": "(?i)\\w\\?(\\s|\\n)|i.*(would|like|need|have|love).*to know about[^a-zA-Z]",
	if len(regexString) > 0 {
		pattern = regexString[0]
	}

	r, rErr := regexp.Compile(pattern)
	if rErr != nil {
		log.Println("There is a problem with the question regex.")
		return false
	}

	return r.MatchString(text)
}

// Detects gender based on US Census database
func DetectGender() {
	// TODO
}

// Geocodes using MapQuest API if available
func Geocode() {
	// TODO: Also think about using https://github.com/kellydunn/golang-geo
	// That package will allow the use of google Maps API or Open Street Maps API (though MapQuest has a higher rate limit)

}
