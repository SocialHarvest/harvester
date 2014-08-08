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
	"crypto/md5"
	"encoding/csv"
	"encoding/hex"
	"github.com/jasonwinn/geocoder"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// For determining gender, we use the US Census database
// https://www.census.gov/genealogy/www/data/1990surnames/names_files.html
// Note: we could also stistically guess ethnicity, https://www.census.gov/genealogy/www/data/2000surnames/index.html
// Frequency is the one we want. Cumulative frequency is in relation to all names in the database.
// So if there was a tie for example, "Pat" being both a male and female name...We could look at the cumulative to see if the Census saw more Pats who were male vs. female...
// This should be extremely rare and maybe not a great way to break ties, but works.
type UsCensusName struct {
	Name    string
	Freq    float64
	CumFreq float64
	Rank    int
}

var femaleNames = []UsCensusName{}
var maleNames = []UsCensusName{}

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

// Attempts to determine the contributor type (person, company, etc.) when not provided by a service API. In order to do this, we need a few values to test.
// TODO: More work on this...
func DetectContributorType(name string, gender int) string {
	// By default assume a person...Technically an actual person had to post the message right? =) ...and this is a safe bet.
	contributorType := "person"

	// If we have a gender then it's obviously a person (or a person on behalf of a company which is OK too)
	if gender == 1 || gender == -1 {
		contributorType = "person"
	}

	// Test the name, see if there's an LLC or Inc. or something ... Maybe get a CSV of well known brand/company names too
	r, _ := regexp.Compile(`llc|llp|company|limited|ltd\.|inc\.|\sinc$|corporation|corp\.|\scorp$|s\.a\.|\senterprises$|\sinternational$|\spartners$|associates|\sentertainment$|\sgroup$|\ssystems$|\ssoftware$|\smicrosystems$|\stechnologies$|\scommunications$|\snetworks$|\sindustries$|\spublishing$|\sgames$|\sfoundation$|\ssolutions$|\sholdings$|\sfinancial$`)
	if r.MatchString(strings.ToLower(name)) == true {
		contributorType = "company"
	}

	// TODO: maybe look at the URL and parse OG tags and such looking for a company name...

	return contributorType
}

// Detects gender based on US Census database, returns 0 for unknown, -1 for female, and 1 for male
func DetectGender(name string) int {

	firstName := ""
	nameParts := strings.Fields(name)
	if len(nameParts) > 0 {
		// All names in the data set are in uppercase
		firstName = strings.ToUpper(nameParts[0])
	}

	freqMale := 0.0
	freqFemale := 0.0
	for _, male := range maleNames {
		if male.Name == firstName {
			freqMale = male.Freq
		}
	}
	for _, female := range femaleNames {
		if female.Name == firstName {
			freqFemale = female.Freq
		}
	}

	if freqMale > freqFemale {
		return 1
	}
	if freqFemale > freqMale {
		return -1
	}

	return 0
}

// Geocodes using MapQuest API if available
func Geocode(locationQuery string) (float64, float64) {
	return geocoder.Geocode(locationQuery)
}

// Gets the final URL given a short URL (or one that has redirects)
func ExpandUrl(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		return ""
	}
	return resp.Request.URL.String()
}

// Simple boolean to integer
func Btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// For Gender CSV files --------------

// Load data from CSV files in order to detect gender. If new files are being used, call this again.
func NewGenderData(femaleFilename string, maleFilename string) {
	femaleNamesFile, err := os.Open(femaleFilename)
	if err != nil {
		// err is printable
		// elements passed are separated by space automatically
		log.Println("Error:", err)
	}
	// automatically call Close() at the end of current method
	defer femaleNamesFile.Close()
	femaleReader := csv.NewReader(femaleNamesFile)

	femaleReader.Comma = ','
	var fName UsCensusName
	for {
		err := unmarshalCensusData(femaleReader, &fName)
		if err == io.EOF {
			break
		}
		if err != nil {
			break
			//panic(err)
		}
		femaleNames = append(femaleNames, fName)
	}

	maleNamesFile, err := os.Open(maleFilename)
	if err != nil {
		// err is printable
		// elements passed are separated by space automatically
		log.Println("Error:", err)
	}
	// automatically call Close() at the end of current method
	defer maleNamesFile.Close()
	maleReader := csv.NewReader(maleNamesFile)

	maleReader.Comma = ','
	var mName UsCensusName
	for {
		err := unmarshalCensusData(maleReader, &mName)
		if err == io.EOF {
			break
		}
		if err != nil {
			//break
			panic(err)
		}
		maleNames = append(maleNames, mName)
	}

}

// Reads the census CSV data from files
func unmarshalCensusData(reader *csv.Reader, v interface{}) error {
	record, err := reader.Read()
	if err != nil {
		return err
	}
	s := reflect.ValueOf(v).Elem()
	if s.NumField() != len(record) {
		return &csvFieldMismatch{s.NumField(), len(record)}
	}
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		switch f.Type().String() {
		case "string":
			f.SetString(record[i])
		case "int":
			ival, err := strconv.ParseInt(record[i], 10, 0)
			if err != nil {
				return err
			}
			f.SetInt(ival)
		case "float64":
			fval, err := strconv.ParseFloat(record[i], 64)
			if err != nil {
				return err
			}
			f.SetFloat(fval)
		default:
			return &csvUnsupportedType{f.Type().String()}
		}
	}
	return nil
}

type csvFieldMismatch struct {
	expected, found int
}

func (e *csvFieldMismatch) Error() string {
	return "CSV line fields mismatch. Expected " + strconv.Itoa(e.expected) + " found " + strconv.Itoa(e.found)
}

type csvUnsupportedType struct {
	Type string
}

func (e *csvUnsupportedType) Error() string {
	return "Unsupported type: " + e.Type
}
