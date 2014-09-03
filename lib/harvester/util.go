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
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
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

// Gets the final URL given a short URL (or one that has redirects)
func ExpandUrl(url string) string {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ""
	}
	resp, err := httpClient.Do(req)
	// resp, err := http.Get(url)
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

func GetKeywords(text string, minSize int, limit int) []string {
	var keywords []string

	// strip out all URLs
	rUrl := regexp.MustCompile(`(?i)http(|s)\:\/\/.+?(\s|$)`)
	text = rUrl.ReplaceAllString(text, " ")

	// strip out all hashtags and mentions
	rH := regexp.MustCompile(`(?i)(\#|\@|\+).+?(\s|$)`)
	text = rH.ReplaceAllString(text, " ")

	//words := strings.Fields(text)
	// I don't know if strings.Fields() is faster or not, but even if it is, we only have to perform more regex anyway (to remove punctuation and so on).
	r := regexp.MustCompile(`(?i)[A-z\']+`)
	words := r.FindAllString(text, -1)

	i := 0
	for _, v := range words {
		if i < limit {
			if len(v) >= minSize && !IsStopKeyword(strings.ToLower(v)) {
				keywords = append(keywords, strings.ToLower(v))
			}
		}
		i++
	}
	return keywords
}

// A list of stop words for keyword extraction (also available under data/keyword-stop-list.txt - mostly, more were added after testing)
func IsStopKeyword(word string) bool {
	// var stopWords [2571]string
	var stopWords = map[string]string{`abans`: ``, `aber`: ``, `able`: ``, `about`: ``, `above`: ``, `abst`: ``, `acaba`: ``, `accordance`: ``, `according`: ``, `accordingly`: ``, `acerca`: ``, `across`: ``, `actually`: ``, `added`: ``, `aderton`: ``, `adertonde`: ``, `adesso`: ``, `adjö`: ``, `affected`: ``, `affecting`: ``, `affects`: ``, `after`: ``, `afterwards`: ``, `again`: ``, `against`: ``, `agora`: ``, `aiemmin`: ``, `aint`: ``, `ain't`: ``, `aika`: ``, `aikaa`: ``, `aikaan`: ``, `aikaisemmin`: ``, `aikaisin`: ``, `aikajen`: ``, `aikana`: ``, `aikoina`: ``, `aikoo`: ``, `aikovat`: ``, `aina`: ``, `ainakaan`: ``, `ainakin`: ``, `ainoa`: ``, `ainoat`: ``, `aiomme`: ``, `aion`: ``, `aiotte`: ``, `aist`: ``, `aivan`: ``, `ajan`: ``, `alas`: ``, `albo`: ``, `aldrig`: ``, `alemmas`: ``, `algmas`: ``, `algun`: ``, `alguna`: ``, `algunas`: ``, `algunes`: ``, `alguno`: ``, `algunos`: ``, `alguns`: ``, `algún`: ``, `alkuisin`: ``, `alkuun`: ``, `alla`: ``, `allas`: ``, `alle`: ``, `allo`: ``, `allora`: ``, `allt`: ``, `alltid`: ``, `alltså`: ``, `almost`: ``, `aloitamme`: ``, `aloitan`: ``, `aloitat`: ``, `aloitatte`: ``, `aloitattivat`: ``, `aloitettava`: ``, `aloitettevaksi`: ``, `aloitettu`: ``, `aloitimme`: ``, `aloitin`: ``, `aloitit`: ``, `aloititte`: ``, `aloittaa`: ``, `aloittamatta`: ``, `aloitti`: ``, `aloittivat`: ``, `alone`: ``, `along`: ``, `alors`: ``, `already`: ``, `also`: ``, `alta`: ``, `although`: ``, `altmýþ`: ``, `altre`: ``, `altri`: ``, `altro`: ``, `altý`: ``, `aluksi`: ``, `alussa`: ``, `alusta`: ``, `always`: ``, `ambdós`: ``, `ambos`: ``, `among`: ``, `amongst`: ``, `ampleamos`: ``, `anar`: ``, `anche`: ``, `ancora`: ``, `andet`: ``, `andra`: ``, `andras`: ``, `andre`: ``, `annan`: ``, `annat`: ``, `annettavaksi`: ``, `annetteva`: ``, `annettu`: ``, `announce`: ``, `another`: ``, `antaa`: ``, `antamatta`: ``, `ante`: ``, `antes`: ``, `antoi`: ``, `anybody`: ``, `anyhow`: ``, `anymore`: ``, `anyone`: ``, `anything`: ``, `anyway`: ``, `anyways`: ``, `anywhere`: ``, `aoua`: ``, `apontar`: ``, `apparently`: ``, `approximately`: ``, `aquel`: ``, `aquela`: ``, `aquelas`: ``, `aquele`: ``, `aqueles`: ``, `aquell`: ``, `aquellas`: ``, `aquelles`: ``, `aquellos`: ``, `aquells`: ``, `aqui`: ``, `aquí`: ``, `arbeid`: ``, `aren`: ``, `arent`: ``, `arise`: ``, `around`: ``, `arriba`: ``, `artonde`: ``, `artonn`: ``, `asia`: ``, `asiaa`: ``, `asian`: ``, `asiasta`: ``, `asiat`: ``, `aside`: ``, `asioiden`: ``, `asioihin`: ``, `asioita`: ``, `ask`: ``, `asks`: ``, `asking`: ``, `asti`: ``, `atras`: ``, `atrás`: ``, `auch`: ``, `aucuns`: ``, `aussi`: ``, `auth`: ``, `autre`: ``, `available`: ``, `avant`: ``, `avec`: ``, `avere`: ``, `aveva`: ``, `avevano`: ``, `avoir`: ``, `avuksi`: ``, `avulla`: ``, `avun`: ``, `avutta`: ``, `away`: ``, `awfully`: ``, `back`: ``, `bajo`: ``, `bakom`: ``, `bana`: ``, `bara`: ``, `bardzo`: ``, `bastant`: ``, `bastante`: ``, `bazý`: ``, `became`: ``, `because`: ``, `become`: ``, `becomes`: ``, `becoming`: ``, `been`: ``, `before`: ``, `beforehand`: ``, `begge`: ``, `begin`: ``, `beginning`: ``, `beginnings`: ``, `begins`: ``, `behind`: ``, `behöva`: ``, `behövas`: ``, `behövde`: ``, `behövt`: ``, `being`: ``, `believe`: ``, `belki`: ``, `below`: ``, `benden`: ``, `beni`: ``, `benim`: ``, `beside`: ``, `besides`: ``, `beslut`: ``, `beslutat`: ``, `beslutit`: ``, `between`: ``, `beyond`: ``, `bien`: ``, `biol`: ``, `biri`: ``, `birkaç`: ``, `birkez`: ``, `birþey`: ``, `birþeyi`: ``, `bist`: ``, `bizden`: ``, `bizi`: ``, `bizim`: ``, `bland`: ``, `blev`: ``, `blir`: ``, `blivit`: ``, `bort`: ``, `borta`: ``, `both`: ``, `bout`: ``, `brief`: ``, `briefly`: ``, `bruke`: ``, `buna`: ``, `bunda`: ``, `bundan`: ``, `bunu`: ``, `bunun`: ``, `buono`: ``, `bäst`: ``, `bättre`: ``, `båda`: ``, `bådas`: ``, `cada`: ``, `came`: ``, `caminho`: ``, `cant`: ``, `can't`: ``, `cannot`: ``, `cause`: ``, `causes`: ``, `cela`: ``, `certain`: ``, `certainly`: ``, `ceux`: ``, `chaque`: ``, `ciebie`: ``, `cierta`: ``, `ciertas`: ``, `cierto`: ``, `ciertos`: ``, `cima`: ``, `cinque`: ``, `come`: ``, `coming`: ``, `comes`: ``, `comme`: ``, `comment`: ``, `como`: ``, `comprare`: ``, `comprido`: ``, `conhecido`: ``, `consecutivi`: ``, `consecutivo`: ``, `consegueixo`: ``, `conseguim`: ``, `conseguimos`: ``, `conseguir`: ``, `consigo`: ``, `consigue`: ``, `consigueix`: ``, `consigueixen`: ``, `consigueixes`: ``, `consiguen`: ``, `consigues`: ``, `contain`: ``, `containing`: ``, `contains`: ``, `corrente`: ``, `cosa`: ``, `could`: ``, `couldnt`: ``, `csak`: ``, `cual`: ``, `cuando`: ``, `dadurch`: ``, `dagar`: ``, `dagarna`: ``, `dagen`: ``, `daha`: ``, `daher`: ``, `dahi`: ``, `daleko`: ``, `dalt`: ``, `dans`: ``, `darum`: ``, `dass`: ``, `date`: ``, `debaixo`: ``, `dedans`: ``, `defa`: ``, `dehors`: ``, `dein`: ``, `deine`: ``, `delen`: ``, `della`: ``, `dello`: ``, `denne`: ``, `dentro`: ``, `depuis`: ``, `deras`: ``, `deres`: ``, `desde`: ``, `deshalb`: ``, `desligado`: ``, `dess`: ``, `dessen`: ``, `detta`: ``, `dette`: ``, `deux`: ``, `deve`: ``, `devem`: ``, `deverá`: ``, `devo`: ``, `devrait`: ``, `didn't`: ``, `dies`: ``, `dieser`: ``, `dieses`: ``, `different`: ``, `dina`: ``, `dins`: ``, `direita`: ``, `disse`: ``, `ditt`: ``, `diye`: ``, `dizer`: ``, `dlaczego`: ``, `dlatego`: ``, `dobrze`: ``, `doch`: ``, `dock`: ``, `does`: ``, `doesn't`: ``, `doing`: ``, `dois`: ``, `doit`: ``, `doksan`: ``, `dokuz`: ``, `dokąd`: ``, `dont`: ``, `don't`: ``, `donc`: ``, `donde`: ``, `done`: ``, `doppio`: ``, `dort`: ``, `down`: ``, `downwards`: ``, `dość`: ``, `droite`: ``, `durch`: ``, `during`: ``, `dużo`: ``, `dwaj`: ``, `dwie`: ``, `dwoje`: ``, `dzisiaj`: ``, `dziś`: ``, `därför`: ``, `début`: ``, `dört`: ``, `each`: ``, `ecco`: ``, `edelle`: ``, `edelleen`: ``, `edellä`: ``, `edeltä`: ``, `edemmäs`: ``, `edes`: ``, `edessä`: ``, `edestä`: ``, `effect`: ``, `efter`: ``, `eftersom`: ``, `ehkä`: ``, `eight`: ``, `eighty`: ``, `eikä`: ``, `eilen`: ``, `eine`: ``, `einem`: ``, `einen`: ``, `einer`: ``, `eines`: ``, `either`: ``, `eivät`: ``, `eles`: ``, `elfte`: ``, `ellas`: ``, `elle`: ``, `ellei`: ``, `elleivät`: ``, `ellemme`: ``, `ellen`: ``, `eller`: ``, `elles`: ``, `ellet`: ``, `ellette`: ``, `elli`: ``, `ellos`: ``, `ells`: ``, `else`: ``, `elsewhere`: ``, `elva`: ``, `emme`: ``, `empleais`: ``, `emplean`: ``, `emplear`: ``, `empleas`: ``, `empleo`: ``, `encima`: ``, `encore`: ``, `ending`: ``, `enemmän`: ``, `eneste`: ``, `enhver`: ``, `eniten`: ``, `enkel`: ``, `enkelt`: ``, `enkla`: ``, `enligt`: ``, `ennen`: ``, `enough`: ``, `enquanto`: ``, `ensi`: ``, `ensimmäinen`: ``, `ensimmäiseksi`: ``, `ensimmäisen`: ``, `ensimmäisenä`: ``, `ensimmäiset`: ``, `ensimmäisiksi`: ``, `ensimmäisinä`: ``, `ensimmäisiä`: ``, `ensimmäistä`: ``, `ensin`: ``, `entinen`: ``, `entisen`: ``, `entisiä`: ``, `entisten`: ``, `entistä`: ``, `entonces`: ``, `entre`: ``, `então`: ``, `enää`: ``, `eramos`: ``, `eran`: ``, `eras`: ``, `erem`: ``, `eren`: ``, `eres`: ``, `erittäin`: ``, `erityisesti`: ``, `eräiden`: ``, `eräs`: ``, `eräät`: ``, `esiin`: ``, `esillä`: ``, `esimerkiksi`: ``, `especially`: ``, `essai`: ``, `esta`: ``, `estaba`: ``, `estado`: ``, `estais`: ``, `estamos`: ``, `estan`: ``, `estar`: ``, `estará`: ``, `estat`: ``, `estava`: ``, `este`: ``, `estem`: ``, `estes`: ``, `esteu`: ``, `esteve`: ``, `estic`: ``, `estive`: ``, `estivemos`: ``, `estiveram`: ``, `estoy`: ``, `està`: ``, `está`: ``, `estão`: ``, `et-al`: ``, `eteen`: ``, `etenkin`: ``, `ette`: ``, `ettei`: ``, `ettusen`: ``, `että`: ``, `euer`: ``, `eure`: ``, `even`: ``, `ever`: ``, `every`: ``, `everybody`: ``, `everyone`: ``, `everything`: ``, `everywhere`: ``, `except`: ``, `faig`: ``, `fait`: ``, `faites`: ``, `fanns`: ``, `fare`: ``, `fará`: ``, `fazer`: ``, `fazia`: ``, `femte`: ``, `femtio`: ``, `femtionde`: ``, `femton`: ``, `femtonde`: ``, `fick`: ``, `fifth`: ``, `fine`: ``, `finnas`: ``, `finns`: ``, `fino`: ``, `fire`: ``, `first`: ``, `five`: ``, `fjorton`: ``, `fjortonde`: ``, `fjärde`: ``, `fler`: ``, `flera`: ``, `flere`: ``, `flesta`: ``, `fleste`: ``, `fois`: ``, `folk`: ``, `followed`: ``, `following`: ``, `follows`: ``, `font`: ``, `fora`: ``, `force`: ``, `fordi`: ``, `former`: ``, `formerly`: ``, `forrige`: ``, `forsÛke`: ``, `forth`: ``, `found`: ``, `four`: ``, `fram`: ``, `framför`: ``, `from`: ``, `från`: ``, `fueron`: ``, `fuimos`: ``, `further`: ``, `furthermore`: ``, `fyra`: ``, `fyrtio`: ``, `fyrtionde`: ``, `fÛrst`: ``, `fått`: ``, `följande`: ``, `före`: ``, `förlåt`: ``, `förra`: ``, `första`: ``, `gave`: ``, `gdyby`: ``, `gdzie`: ``, `genast`: ``, `genom`: ``, `gente`: ``, `gets`: ``, `getting`: ``, `gibi`: ``, `gick`: ``, `give`: ``, `given`: ``, `gives`: ``, `giving`: ``, `gjorde`: ``, `gjort`: ``, `gjÛre`: ``, `goda`: ``, `godare`: ``, `godast`: ``, `goes`: ``, `gone`: ``, `gonna`: ``, `gott`: ``, `gotta`: ``, `gotten`: ``, `gueno`: ``, `gälla`: ``, `gäller`: ``, `gällt`: ``, `gärna`: ``, `gått`: ``, `göra`: ``, `hace`: ``, `haceis`: ``, `hacemos`: ``, `hacen`: ``, `hacer`: ``, `haces`: ``, `hadde`: ``, `hade`: ``, `haft`: ``, `hago`: ``, `halua`: ``, `haluaa`: ``, `haluamatta`: ``, `haluamme`: ``, `haluan`: ``, `haluat`: ``, `haluatte`: ``, `haluavat`: ``, `halunnut`: ``, `halusi`: ``, `halusimme`: ``, `halusin`: ``, `halusit`: ``, `halusitte`: ``, `halusivat`: ``, `halutessa`: ``, `haluton`: ``, `hanno`: ``, `hans`: ``, `happens`: ``, `hardly`: ``, `hasn't`: ``, `hatte`: ``, `hatten`: ``, `hattest`: ``, `hattet`: ``, `haut`: ``, `have`: ``, `haven't`: ``, `havent`: ``, `haver`: ``, `having`: ``, `heidän`: ``, `heihin`: ``, `heille`: ``, `heiltä`: ``, `heissä`: ``, `heistä`: ``, `heitä`: ``, `heller`: ``, `hellre`: ``, `helposti`: ``, `helst`: ``, `helt`: ``, `hence`: ``, `hendes`: ``, `henne`: ``, `hennes`: ``, `hepsi`: ``, `here`: ``, `hereafter`: ``, `hereby`: ``, `herein`: ``, `heres`: ``, `hereupon`: ``, `hers`: ``, `herself`: ``, `heti`: ``, `hetkellä`: ``, `hieman`: ``, `hier`: ``, `himself`: ``, `hinter`: ``, `hither`: ``, `hogy`: ``, `home`: ``, `honom`: ``, `horas`: ``, `hors`: ``, `howbeit`: ``, `however`: ``, `http`: ``, `hundra`: ``, `hundraen`: ``, `hundraett`: ``, `hundred`: ``, `huolimatta`: ``, `huomenna`: ``, `hvad`: ``, `hvem`: ``, `hver`: ``, `hvilken`: ``, `hvis`: ``, `hvor`: ``, `hvordan`: ``, `hvorfor`: ``, `hvornår`: ``, `hyvien`: ``, `hyviin`: ``, `hyviksi`: ``, `hyville`: ``, `hyviltä`: ``, `hyvin`: ``, `hyvinä`: ``, `hyvissä`: ``, `hyvistä`: ``, `hyviä`: ``, `hyvä`: ``, `hyvät`: ``, `hyvää`: ``, `häneen`: ``, `hänelle`: ``, `hänellä`: ``, `häneltä`: ``, `hänen`: ``, `hänessä`: ``, `hänestä`: ``, `hänet`: ``, `höger`: ``, `högre`: ``, `högst`: ``, `i'll`: ``, `i've`: ``, `ibland`: ``, `idag`: ``, `igen`: ``, `igår`: ``, `ihan`: ``, `ihre`: ``, `ikke`: ``, `ilman`: ``, `ilmeisesti`: ``, `immediate`: ``, `immediately`: ``, `imorgon`: ``, `importance`: ``, `important`: ``, `incluso`: ``, `inclòs`: ``, `indeed`: ``, `index`: ``, `indietro`: ``, `information`: ``, `inför`: ``, `inga`: ``, `ingen`: ``, `ingenting`: ``, `inget`: ``, `iniciar`: ``, `inicio`: ``, `innan`: ``, `inne`: ``, `innen`: ``, `inny`: ``, `inom`: ``, `instead`: ``, `inte`: ``, `intenta`: ``, `intentais`: ``, `intentamos`: ``, `intentan`: ``, `intentar`: ``, `intentas`: ``, `intento`: ``, `intet`: ``, `into`: ``, `inuti`: ``, `invece`: ``, `invention`: ``, `inward`: ``, `isn't`: ``, `ista`: ``, `iste`: ``, `isto`: ``, `it'll`: ``, `it's`: ``, `itse`: ``, `itself`: ``, `itsensä`: ``, `itseään`: ``, `için`: ``, `jakby`: ``, `jaki`: ``, `jede`: ``, `jedem`: ``, `jeden`: ``, `jeder`: ``, `jedes`: ``, `jedna`: ``, `jedno`: ``, `jego`: ``, `jemu`: ``, `jener`: ``, `jenes`: ``, `jeres`: ``, `jest`: ``, `jestem`: ``, `jetzt`: ``, `jeśli`: ``, `jeżeli`: ``, `johon`: ``, `joiden`: ``, `joihin`: ``, `joiksi`: ``, `joilla`: ``, `joille`: ``, `joilta`: ``, `joissa`: ``, `joista`: ``, `joita`: ``, `joka`: ``, `jokainen`: ``, `jokin`: ``, `joko`: ``, `joku`: ``, `jolla`: ``, `jolle`: ``, `jolloin`: ``, `jolta`: ``, `jompikumpi`: ``, `jonka`: ``, `jonkin`: ``, `jonne`: ``, `jopa`: ``, `joskus`: ``, `jossa`: ``, `josta`: ``, `jota`: ``, `jotain`: ``, `joten`: ``, `jotenkin`: ``, `jotenkuten`: ``, `jotka`: ``, `jotta`: ``, `jouduimme`: ``, `jouduin`: ``, `jouduit`: ``, `jouduitte`: ``, `joudumme`: ``, `joudun`: ``, `joudutte`: ``, `joukkoon`: ``, `joukossa`: ``, `joukosta`: ``, `joutua`: ``, `joutui`: ``, `joutuivat`: ``, `joutumaan`: ``, `joutuu`: ``, `joutuvat`: ``, `just`: ``, `juste`: ``, `juuri`: ``, `jälkeen`: ``, `jälleen`: ``, `jämfört`: ``, `kahdeksan`: ``, `kahdeksannen`: ``, `kahdella`: ``, `kahdelle`: ``, `kahdelta`: ``, `kahden`: ``, `kahdessa`: ``, `kahdesta`: ``, `kahta`: ``, `kahteen`: ``, `kaiken`: ``, `kaikille`: ``, `kaikilta`: ``, `kaikkea`: ``, `kaikki`: ``, `kaikkia`: ``, `kaikkiaan`: ``, `kaikkialla`: ``, `kaikkialle`: ``, `kaikkialta`: ``, `kaikkien`: ``, `kaikkin`: ``, `kaksi`: ``, `kann`: ``, `kannalta`: ``, `kannattaa`: ``, `kannst`: ``, `kanske`: ``, `kanssa`: ``, `kanssaan`: ``, `kanssamme`: ``, `kanssani`: ``, `kanssanne`: ``, `kanssasi`: ``, `katrilyon`: ``, `kauan`: ``, `kauemmas`: ``, `kautta`: ``, `każdy`: ``, `keep`: ``, `keeps`: ``, `kehen`: ``, `keiden`: ``, `keihin`: ``, `keiksi`: ``, `keille`: ``, `keillä`: ``, `keiltä`: ``, `keinä`: ``, `keissä`: ``, `keistä`: ``, `keitten`: ``, `keittä`: ``, `keitä`: ``, `keneen`: ``, `keneksi`: ``, `kenelle`: ``, `kenellä`: ``, `keneltä`: ``, `kenen`: ``, `kenenä`: ``, `kenessä`: ``, `kenestä`: ``, `kenet`: ``, `kenettä`: ``, `kennessästä`: ``, `kept`: ``, `kerran`: ``, `kerta`: ``, `kertaa`: ``, `kesken`: ``, `keskimäärin`: ``, `ketkä`: ``, `ketä`: ``, `kiedy`: ``, `kierunku`: ``, `kiitos`: ``, `kimden`: ``, `kime`: ``, `kimi`: ``, `knappast`: ``, `know`: ``, `known`: ``, `knows`: ``, `kohti`: ``, `koko`: ``, `kokonaan`: ``, `kolmas`: ``, `kolme`: ``, `kolmen`: ``, `kolmesti`: ``, `komma`: ``, `kommer`: ``, `kommit`: ``, `koska`: ``, `koskaan`: ``, `kovin`: ``, `kuin`: ``, `kuinka`: ``, `kuitenkaan`: ``, `kuitenkin`: ``, `kuka`: ``, `kukaan`: ``, `kukin`: ``, `kumpainen`: ``, `kumpainenkaan`: ``, `kumpi`: ``, `kumpikaan`: ``, `kumpikin`: ``, `kunde`: ``, `kunna`: ``, `kunnat`: ``, `kunne`: ``, `kuten`: ``, `kuuden`: ``, `kuusi`: ``, `kuutta`: ``, `kvar`: ``, `kyllä`: ``, `kymmenen`: ``, `kyse`: ``, `können`: ``, `könnt`: ``, `kýrk`: ``, `lage`: ``, `lang`: ``, `largely`: ``, `largo`: ``, `last`: ``, `lately`: ``, `later`: ``, `latter`: ``, `latterly`: ``, `lavoro`: ``, `least`: ``, `legat`: ``, `less`: ``, `lest`: ``, `lesz`: ``, `lets`: ``, `let's`: ``, `leur`: ``, `lidt`: ``, `ligado`: ``, `ligga`: ``, `ligger`: ``, `liian`: ``, `lika`: ``, `like`: ``, `liked`: ``, `likely`: ``, `liki`: ``, `likställd`: ``, `likställda`: ``, `lilla`: ``, `lille`: ``, `line`: ``, `lisäksi`: ``, `lisää`: ``, `lite`: ``, `liten`: ``, `litet`: ``, `little`: ``, `llarg`: ``, `llavors`: ``, `look`: ``, `looking`: ``, `looks`: ``, `loro`: ``, `lungo`: ``, `lähekkäin`: ``, `lähelle`: ``, `lähellä`: ``, `läheltä`: ``, `lähemmäs`: ``, `lähes`: ``, `lähinnä`: ``, `lähtien`: ``, `länge`: ``, `längre`: ``, `längst`: ``, `läpi`: ``, `lätt`: ``, `lättare`: ``, `lättast`: ``, `långsam`: ``, `långsammare`: ``, `långsammast`: ``, `långsamt`: ``, `långt`: ``, `machen`: ``, `made`: ``, `mahdollisimman`: ``, `mahdollista`: ``, `mainly`: ``, `maintenant`: ``, `maioria`: ``, `maiorias`: ``, `mais`: ``, `mają`: ``, `make`: ``, `makes`: ``, `makt`: ``, `mand`: ``, `mange`: ``, `many`: ``, `maybe`: ``, `mean`: ``, `means`: ``, `meantime`: ``, `meanwhile`: ``, `meget`: ``, `meglio`: ``, `meidän`: ``, `meille`: ``, `meillä`: ``, `mein`: ``, `meine`: ``, `melkein`: ``, `melko`: ``, `mellan`: ``, `menee`: ``, `meneet`: ``, `menemme`: ``, `menen`: ``, `menet`: ``, `menette`: ``, `menevät`: ``, `meni`: ``, `menimme`: ``, `menin`: ``, `menit`: ``, `menivät`: ``, `mennessä`: ``, `mennyt`: ``, `menossa`: ``, `mens`: ``, `mentre`: ``, `mera`: ``, `mere`: ``, `merely`: ``, `mesmo`: ``, `mest`: ``, `mientras`: ``, `might`: ``, `mihin`: ``, `mikin`: ``, `miksi`: ``, `mikä`: ``, `mikäli`: ``, `mikään`: ``, `million`: ``, `milloin`: ``, `milyar`: ``, `milyon`: ``, `mina`: ``, `mindre`: ``, `mine`: ``, `minne`: ``, `minst`: ``, `mint`: ``, `minun`: ``, `minut`: ``, `minä`: ``, `miss`: ``, `missä`: ``, `mistä`: ``, `miten`: ``, `mitt`: ``, `mittemot`: ``, `mitä`: ``, `mitään`: ``, `mnie`: ``, `mode`: ``, `modo`: ``, `moins`: ``, `moja`: ``, `moje`: ``, `molemmat`: ``, `molt`: ``, `molta`: ``, `molti`: ``, `molto`: ``, `molts`: ``, `mones`: ``, `monesti`: ``, `monet`: ``, `moni`: ``, `moniaalla`: ``, `moniaalle`: ``, `moniaalta`: ``, `monta`: ``, `more`: ``, `moreover`: ``, `most`: ``, `mostly`: ``, `może`: ``, `muassa`: ``, `much`: ``, `muchos`: ``, `muiden`: ``, `muita`: ``, `muito`: ``, `muitos`: ``, `muka`: ``, `mukaan`: ``, `mukaansa`: ``, `mukana`: ``, `musst`: ``, `must`: ``, `mutta`: ``, `muualla`: ``, `muualle`: ``, `muualta`: ``, `muuanne`: ``, `muulloin`: ``, `muun`: ``, `muut`: ``, `muuta`: ``, `muutama`: ``, `muutaman`: ``, `muuten`: ``, `mußt`: ``, `mycket`: ``, `myself`: ``, `myöhemmin`: ``, `myös`: ``, `myöskin`: ``, `myöskään`: ``, `myötä`: ``, `mÅte`: ``, `många`: ``, `måste`: ``, `même`: ``, `möjlig`: ``, `möjligen`: ``, `möjligt`: ``, `möjligtvis`: ``, `müssen`: ``, `müßt`: ``, `nach`: ``, `nachdem`: ``, `name`: ``, `namely`: ``, `nami`: ``, `nasi`: ``, `nasz`: ``, `nasza`: ``, `nasze`: ``, `nasýl`: ``, `natychmiast`: ``, `navn`: ``, `near`: ``, `nearly`: ``, `necessarily`: ``, `necessary`: ``, `neden`: ``, `nederst`: ``, `nedersta`: ``, `nedre`: ``, `need`: ``, `needs`: ``, `nein`: ``, `neither`: ``, `neljä`: ``, `neljän`: ``, `neljää`: ``, `nella`: ``, `nerde`: ``, `nerede`: ``, `nereye`: ``, `never`: ``, `nevertheless`: ``, `next`: ``, `nich`: ``, `nicht`: ``, `niego`: ``, `niej`: ``, `niemu`: ``, `nigdy`: ``, `niiden`: ``, `niin`: ``, `niistä`: ``, `niitä`: ``, `nimi`: ``, `nine`: ``, `ninety`: ``, `nionde`: ``, `nittio`: ``, `nittionde`: ``, `nitton`: ``, `nittonde`: ``, `niye`: ``, `niçin`: ``, `nobody`: ``, `nogen`: ``, `noget`: ``, `noin`: ``, `noll`: ``, `nome`: ``, `nommés`: ``, `none`: ``, `nonetheless`: ``, `noone`: ``, `nopeammin`: ``, `nopeasti`: ``, `nopeiten`: ``, `normally`: ``, `nosaltres`: ``, `nosotros`: ``, `nosso`: ``, `nostro`: ``, `noted`: ``, `nothing`: ``, `notre`: ``, `nous`: ``, `nouveaux`: ``, `nove`: ``, `novo`: ``, `nowhere`: ``, `nummer`: ``, `nuovi`: ``, `nuovo`: ``, `näiden`: ``, `näin`: ``, `näissä`: ``, `näissähin`: ``, `näissälle`: ``, `näissältä`: ``, `näissästä`: ``, `näitä`: ``, `nämä`: ``, `nästa`: ``, `någon`: ``, `någonting`: ``, `något`: ``, `några`: ``, `næste`: ``, `næsten`: ``, `nödvändig`: ``, `nödvändiga`: ``, `nödvändigt`: ``, `nödvändigtvis`: ``, `obok`: ``, `obtain`: ``, `obtained`: ``, `obviously`: ``, `också`: ``, `oder`: ``, `ofta`: ``, `oftast`: ``, `often`: ``, `ogsÅ`: ``, `oikein`: ``, `okay`: ``, `około`: ``, `olemme`: ``, `olen`: ``, `olet`: ``, `olette`: ``, `oleva`: ``, `olevan`: ``, `olevat`: ``, `olika`: ``, `olikt`: ``, `olimme`: ``, `olin`: ``, `olisi`: ``, `olisimme`: ``, `olisin`: ``, `olisit`: ``, `olisitte`: ``, `olisivat`: ``, `olit`: ``, `olitte`: ``, `olivat`: ``, `olla`: ``, `olleet`: ``, `olli`: ``, `ollut`: ``, `oltre`: ``, `omaa`: ``, `omaan`: ``, `omaksi`: ``, `omalle`: ``, `omalta`: ``, `oman`: ``, `omassa`: ``, `omat`: ``, `omia`: ``, `omien`: ``, `omiin`: ``, `omiksi`: ``, `omille`: ``, `omilta`: ``, `omissa`: ``, `omista`: ``, `omitted`: ``, `once`: ``, `ondan`: ``, `onde`: ``, `ones`: ``, `onkin`: ``, `onko`: ``, `onlar`: ``, `onlardan`: ``, `onlari`: ``, `onlarýn`: ``, `only`: ``, `onto`: ``, `other`: ``, `others`: ``, `otherwise`: ``, `otro`: ``, `otte`: ``, `otto`: ``, `otuz`: ``, `ought`: ``, `ours`: ``, `ourselves`: ``, `outro`: ``, `outside`: ``, `ovat`: ``, `over`: ``, `overall`: ``, `owing`: ``, `owszem`: ``, `page`: ``, `pages`: ``, `paikoittain`: ``, `paitsi`: ``, `pakosti`: ``, `paljon`: ``, `para`: ``, `parce`: ``, `paremmin`: ``, `parempi`: ``, `parhaillaan`: ``, `parhaiten`: ``, `parole`: ``, `part`: ``, `parte`: ``, `particular`: ``, `particularly`: ``, `past`: ``, `pegar`: ``, `peggio`: ``, `pelo`: ``, `perhaps`: ``, `pero`: ``, `perquè`: ``, `persone`: ``, `personnes`: ``, `perusteella`: ``, `peräti`: ``, `però`: ``, `pessoas`: ``, `peut`: ``, `pian`: ``, `pieneen`: ``, `pieneksi`: ``, `pienelle`: ``, `pienellä`: ``, `pieneltä`: ``, `pienempi`: ``, `pienestä`: ``, `pieni`: ``, `pienin`: ``, `pièce`: ``, `placed`: ``, `please`: ``, `plupart`: ``, `plus`: ``, `poco`: ``, `pode`: ``, `podeis`: ``, `podem`: ``, `podemos`: ``, `poden`: ``, `poder`: ``, `poderá`: ``, `podeu`: ``, `podia`: ``, `podria`: ``, `podriais`: ``, `podriamos`: ``, `podrian`: ``, `podrias`: ``, `ponieważ`: ``, `poorly`: ``, `porque`: ``, `possible`: ``, `possibly`: ``, `potentially`: ``, `potser`: ``, `pour`: ``, `pourquoi`: ``, `povo`: ``, `predominantly`: ``, `present`: ``, `previously`: ``, `primarily`: ``, `primer`: ``, `primero`: ``, `primo`: ``, `probably`: ``, `promeiro`: ``, `promesso`: ``, `promptly`: ``, `proud`: ``, `provides`: ``, `przed`: ``, `przedtem`: ``, `puede`: ``, `pueden`: ``, `puedo`: ``, `punkt`: ``, `puolesta`: ``, `puolestaan`: ``, `päälle`: ``, `qual`: ``, `qualquer`: ``, `quan`: ``, `quand`: ``, `quando`: ``, `quant`: ``, `quarto`: ``, `quasi`: ``, `quattro`: ``, `quel`: ``, `quelle`: ``, `quelles`: ``, `quello`: ``, `quels`: ``, `quem`: ``, `questo`: ``, `quickly`: ``, `quien`: ``, `quieto`: ``, `quindi`: ``, `quinto`: ``, `quite`: ``, `rakt`: ``, `rather`: ``, `readily`: ``, `really`: ``, `recent`: ``, `recently`: ``, `redan`: ``, `refs`: ``, `regarding`: ``, `regardless`: ``, `regards`: ``, `related`: ``, `relatively`: ``, `research`: ``, `respectively`: ``, `resulted`: ``, `resulting`: ``, `results`: ``, `rett`: ``, `right`: ``, `riktig`: ``, `rispetto`: ``, `rt's`: ``, `runsaasti`: ``, `rätt`: ``, `saakka`: ``, `sabe`: ``, `sabeis`: ``, `sabem`: ``, `sabemos`: ``, `saben`: ``, `saber`: ``, `sabes`: ``, `sabeu`: ``, `sadam`: ``, `sade`: ``, `sagt`: ``, `said`: ``, `sama`: ``, `samaa`: ``, `samaan`: ``, `samalla`: ``, `samallalta`: ``, `samallassa`: ``, `samallasta`: ``, `saman`: ``, `samat`: ``, `same`: ``, `samma`: ``, `samme`: ``, `samoin`: ``, `sanki`: ``, `sans`: ``, `sant`: ``, `saps`: ``, `sara`: ``, `sata`: ``, `sataa`: ``, `satojen`: ``, `saying`: ``, `says`: ``, `secondo`: ``, `section`: ``, `sedan`: ``, `seeing`: ``, `seem`: ``, `seemed`: ``, `seeming`: ``, `seems`: ``, `seen`: ``, `seid`: ``, `sein`: ``, `seine`: ``, `seitsemän`: ``, `sekiz`: ``, `seks`: ``, `seksen`: ``, `sekä`: ``, `self`: ``, `selves`: ``, `sembra`: ``, `sembrava`: ``, `senare`: ``, `senast`: ``, `senden`: ``, `seni`: ``, `senin`: ``, `sense`: ``, `sent`: ``, `senza`: ``, `sette`: ``, `seulement`: ``, `seuraavat`: ``, `seus`: ``, `seven`: ``, `several`: ``, `sextio`: ``, `sextionde`: ``, `sexton`: ``, `sextonde`: ``, `shall`: ``, `she'll`: ``, `shed`: ``, `shes`: ``, `should`: ``, `shouldn't`: ``, `show`: ``, `showed`: ``, `shown`: ``, `showns`: ``, `shows`: ``, `siamo`: ``, `sich`: ``, `siden`: ``, `siellä`: ``, `sieltä`: ``, `sien`: ``, `siendo`: ``, `siete`: ``, `significant`: ``, `significantly`: ``, `siihen`: ``, `siinä`: ``, `siis`: ``, `siitä`: ``, `sijaan`: ``, `siksi`: ``, `silloin`: ``, `sillä`: ``, `silti`: ``, `similar`: ``, `similarly`: ``, `sina`: ``, `since`: ``, `sind`: ``, `sinne`: ``, `sinua`: ``, `sinulle`: ``, `sinulta`: ``, `sinun`: ``, `sinussa`: ``, `sinusta`: ``, `sinut`: ``, `sinä`: ``, `sist`: ``, `sista`: ``, `siste`: ``, `sisäkkäin`: ``, `sisällä`: ``, `siten`: ``, `sitt`: ``, `sitten`: ``, `sitä`: ``, `sizden`: ``, `sizi`: ``, `sizin`: ``, `sjunde`: ``, `sjuttio`: ``, `sjuttionde`: ``, `sjutton`: ``, `sjuttonde`: ``, `sjätte`: ``, `skall`: ``, `skulle`: ``, `skąd`: ``, `slightly`: ``, `slik`: ``, `slutligen`: ``, `slutt`: ``, `smått`: ``, `snart`: ``, `sobre`: ``, `sois`: ``, `solament`: ``, `solamente`: ``, `soll`: ``, `sollen`: ``, `sollst`: ``, `sollt`: ``, `solo`: ``, `sols`: ``, `some`: ``, `somebody`: ``, `somehow`: ``, `somente`: ``, `someone`: ``, `somethan`: ``, `something`: ``, `sometime`: ``, `sometimes`: ``, `somewhat`: ``, `somewhere`: ``, `somos`: ``, `sono`: ``, `sonst`: ``, `sont`: ``, `soon`: ``, `sopra`: ``, `soprattutto`: ``, `sorry`: ``, `sota`: ``, `sotto`: ``, `sous`: ``, `soweit`: ``, `sowie`: ``, `soyez`: ``, `specifically`: ``, `specified`: ``, `specify`: ``, `specifying`: ``, `start`: ``, `stati`: ``, `stato`: ``, `stesso`: ``, `still`: ``, `stille`: ``, `stop`: ``, `stor`: ``, `stora`: ``, `store`: ``, `stort`: ``, `strongly`: ``, `större`: ``, `störst`: ``, `subito`: ``, `substantially`: ``, `successfully`: ``, `such`: ``, `sufficiently`: ``, `suggest`: ``, `sujet`: ``, `sulla`: ``, `suoraan`: ``, `sure`: ``, `suuntaan`: ``, `suuren`: ``, `suuret`: ``, `suuri`: ``, `suuria`: ``, `suurin`: ``, `suurten`: ``, `szét`: ``, `säga`: ``, `säger`: ``, `sämre`: ``, `sämst`: ``, `taas`: ``, `tack`: ``, `taemmas`: ``, `tahansa`: ``, `takaa`: ``, `takaisin`: ``, `takana`: ``, `taki`: ``, `takia`: ``, `también`: ``, `també`: ``, `também`: ``, `tandis`: ``, `tanto`: ``, `tapauksessa`: ``, `tavalla`: ``, `tavoitteena`: ``, `tellement`: ``, `tels`: ``, `tempo`: ``, `tene`: ``, `teneis`: ``, `tenemos`: ``, `tener`: ``, `tengo`: ``, `tenho`: ``, `tenim`: ``, `tenir`: ``, `teniu`: ``, `tentar`: ``, `tentaram`: ``, `tente`: ``, `tentei`: ``, `terzo`: ``, `teve`: ``, `that`: ``, `than`: ``, `then`: ``, `their`: ``, `theirs`: ``, `there`: ``, `these`: ``, `those`: ``, `they`: ``, `they're`: ``, `this`: ``, `through`: ``, `thru`: ``, `tidig`: ``, `tidigare`: ``, `tidigast`: ``, `tidigt`: ``, `tiempo`: ``, `tiene`: ``, `tienen`: ``, `tietysti`: ``, `tilbake`: ``, `till`: ``, `tills`: ``, `tillsammans`: ``, `tilstand`: ``, `tinc`: ``, `tionde`: ``, `tipo`: ``, `tive`: ``, `tjugo`: ``, `tjugoen`: ``, `tjugoett`: ``, `tjugonde`: ``, `tjugotre`: ``, `tjugotvå`: ``, `tjungo`: ``, `tobie`: ``, `tobą`: ``, `todella`: ``, `todo`: ``, `todos`: ``, `toinen`: ``, `toisaalla`: ``, `toisaalle`: ``, `toisaalta`: ``, `toiseen`: ``, `toiseksi`: ``, `toisella`: ``, `toiselle`: ``, `toiselta`: ``, `toisemme`: ``, `toisen`: ``, `toisensa`: ``, `toisessa`: ``, `toisesta`: ``, `toista`: ``, `toistaiseksi`: ``, `toki`: ``, `tolfte`: ``, `tolv`: ``, `tosin`: ``, `tous`: ``, `tout`: ``, `trabaja`: ``, `trabajais`: ``, `trabajamos`: ``, `trabajan`: ``, `trabajar`: ``, `trabajas`: ``, `trabajo`: ``, `trabalhar`: ``, `trabalho`: ``, `tras`: ``, `tredje`: ``, `trettio`: ``, `trettionde`: ``, `tretton`: ``, `trettonde`: ``, `trilyon`: ``, `triplo`: ``, `trop`: ``, `très`: ``, `tuhannen`: ``, `tuhat`: ``, `tule`: ``, `tulee`: ``, `tulemme`: ``, `tulen`: ``, `tulet`: ``, `tulette`: ``, `tulevat`: ``, `tulimme`: ``, `tulin`: ``, `tulisi`: ``, `tulisimme`: ``, `tulisin`: ``, `tulisit`: ``, `tulisitte`: ``, `tulisivat`: ``, `tulit`: ``, `tulitte`: ``, `tulivat`: ``, `tulla`: ``, `tulleet`: ``, `tullut`: ``, `tuntuu`: ``, `tuolla`: ``, `tuolloin`: ``, `tuolta`: ``, `tuonne`: ``, `tuskin`: ``, `tutaj`: ``, `tuyo`: ``, `tvåhundra`: ``, `twoi`: ``, `twoja`: ``, `twoje`: ``, `twój`: ``, `tykö`: ``, `tähän`: ``, `tällä`: ``, `tällöin`: ``, `tämä`: ``, `tämän`: ``, `tänne`: ``, `tänä`: ``, `tänään`: ``, `tässä`: ``, `tästä`: ``, `täten`: ``, `tätä`: ``, `täysin`: ``, `täytyvät`: ``, `täytyy`: ``, `täällä`: ``, `täältä`: ``, `ultimo`: ``, `umas`: ``, `unas`: ``, `under`: ``, `unes`: ``, `unos`: ``, `unser`: ``, `unsere`: ``, `unter`: ``, `until`: ``, `upon`: ``, `ursäkt`: ``, `usais`: ``, `usamos`: ``, `usan`: ``, `usar`: ``, `usas`: ``, `usea`: ``, `useasti`: ``, `useimmiten`: ``, `usein`: ``, `useita`: ``, `using`: ``, `utan`: ``, `utanför`: ``, `uten`: ``, `uudeksi`: ``, `uudelleen`: ``, `uuden`: ``, `uudet`: ``, `uusi`: ``, `uusia`: ``, `uusien`: ``, `uusinta`: ``, `uuteen`: ``, `uutta`: ``, `vaan`: ``, `vagy`: ``, `vaig`: ``, `vaiheessa`: ``, `vaikea`: ``, `vaikean`: ``, `vaikeat`: ``, `vaikeilla`: ``, `vaikeille`: ``, `vaikeilta`: ``, `vaikeissa`: ``, `vaikeista`: ``, `vaikka`: ``, `vain`: ``, `vais`: ``, `valeur`: ``, `valor`: ``, `vamos`: ``, `vara`: ``, `varför`: ``, `varifrån`: ``, `varit`: ``, `varken`: ``, `varmasti`: ``, `varsin`: ``, `varsinkin`: ``, `varsågod`: ``, `vart`: ``, `varten`: ``, `vasta`: ``, `vastaan`: ``, `vastakkain`: ``, `vaya`: ``, `veja`: ``, `vems`: ``, `verdad`: ``, `verdade`: ``, `verdadeiro`: ``, `verdadera`: ``, `verdadero`: ``, `verdi`: ``, `verkligen`: ``, `verran`: ``, `very`: ``, `veya`: ``, `vidare`: ``, `vielä`: ``, `vierekkäin`: ``, `vieri`: ``, `viiden`: ``, `viime`: ``, `viimeinen`: ``, `viimeisen`: ``, `viimeksi`: ``, `viisi`: ``, `viktig`: ``, `viktigare`: ``, `viktigast`: ``, `viktigt`: ``, `vilka`: ``, `vilken`: ``, `vilket`: ``, `vill`: ``, `ville`: ``, `vissza`: ``, `vite`: ``, `você`: ``, `voidaan`: ``, `voie`: ``, `voient`: ``, `voimme`: ``, `voin`: ``, `voisi`: ``, `voit`: ``, `voitte`: ``, `voivat`: ``, `volt`: ``, `volte`: ``, `vont`: ``, `vosaltres`: ``, `vosotras`: ``, `vosotros`: ``, `vostro`: ``, `votre`: ``, `vous`: ``, `vuoden`: ``, `vuoksi`: ``, `vuosi`: ``, `vuosien`: ``, `vuosina`: ``, `vuotta`: ``, `vÖre`: ``, `vÖrt`: ``, `vähemmän`: ``, `vähintään`: ``, `vähiten`: ``, `vähän`: ``, `välillä`: ``, `vänster`: ``, `vänstra`: ``, `värre`: ``, `våra`: ``, `vårt`: ``, `wami`: ``, `wana`: ``, `wann`: ``, `wanna`: ``, `warum`: ``, `wasi`: ``, `wasz`: ``, `wasza`: ``, `wasze`: ``, `weiter`: ``, `weitere`: ``, `wenn`: ``, `werde`: ``, `werden`: ``, `werdet`: ``, `were`: ``, `we're`: ``, `what`: ``, `what's`: ``, `where`: ``, `wheres`: ``, `where's`: ``, `while`: ``, `whilst`: ``, `weshalb`: ``, `wieder`: ``, `wieso`: ``, `wird`: ``, `wirst`: ``, `with`: ``, `when`: ``, `will`: ``, `well`: ``, `więc`: ``, `woher`: ``, `wohin`: ``, `wouldnt`: ``, `would`: ``, `wouldn't`: ``, `wszystko`: ``, `wtedy`: ``, `yall`: ``, `ya'll`: ``, `yani`: ``, `yedi`: ``, `yetmiþ`: ``, `yhdeksän`: ``, `yhden`: ``, `yhdessä`: ``, `yhteen`: ``, `yhteensä`: ``, `yhteydessä`: ``, `yhteyteen`: ``, `yhtä`: ``, `yhtäälle`: ``, `yhtäällä`: ``, `yhtäältä`: ``, `yhtään`: ``, `yirmi`: ``, `yksi`: ``, `yksin`: ``, `yksittäin`: ``, `yleensä`: ``, `ylemmäs`: ``, `ylös`: ``, `ympäri`: ``, `you're`: ``, `youre`: ``, `your`: ``, `yours`: ``, `youve`: ``, `you've`: ``, `zawsze`: ``, `älköön`: ``, `ännu`: ``, `även`: ``, `åtminstone`: ``, `åtta`: ``, `åttio`: ``, `åttionde`: ``, `åttonde`: ``, `çünkü`: ``, `éssent`: ``, `étaient`: ``, `état`: ``, `étions`: ``, `être`: ``, `össze`: ``, `över`: ``, `övermorgon`: ``, `överst`: ``, `övre`: ``, `últim`: ``, `último`: ``, `über`: ``, `þeyden`: ``, `þeyi`: ``, `þeyler`: ``, `þuna`: ``, `þunda`: ``, `þundan`: ``, `þunu`: ``, `żaden`: ``, `αὐτόσ`: ``, `γάρ`: ``, `γα^`: ``, `δαί`: ``, `δαίσ`: ``, `διά`: ``, `εἰμί`: ``, `εἰσ`: ``, `εἴμι`: ``, `καί`: ``, `κατά`: ``, `μέν`: ``, `μετά`: ``, `οὐδέ`: ``, `οὐδείσ`: ``, `οὐκ`: ``, `οὔτε`: ``, `οὕτωσ`: ``, `οὖν`: ``, `οὗτοσ`: ``, `παρά`: ``, `περί`: ``, `πρόσ`: ``, `σύν`: ``, `τήν`: ``, `τίσ`: ``, `τοί`: ``, `τοιοῦτοσ`: ``, `τούσ`: ``, `τοῦ`: ``, `τόν`: ``, `τῆσ`: ``, `τῶν`: ``, `алло`: ``, `близко`: ``, `более`: ``, `больше`: ``, `будем`: ``, `будет`: ``, `будете`: ``, `будешь`: ``, `будто`: ``, `буду`: ``, `будут`: ``, `будь`: ``, `бывает`: ``, `бывь`: ``, `была`: ``, `были`: ``, `было`: ``, `быть`: ``, `важная`: ``, `важное`: ``, `важные`: ``, `важный`: ``, `вами`: ``, `ваша`: ``, `ваше`: ``, `ваши`: ``, `вверх`: ``, `вдали`: ``, `вдруг`: ``, `ведь`: ``, `везде`: ``, `весь`: ``, `вниз`: ``, `внизу`: ``, `вокруг`: ``, `восемнадцатый`: ``, `восемнадцать`: ``, `восемь`: ``, `восьмой`: ``, `впрочем`: ``, `времени`: ``, `время`: ``, `всегда`: ``, `всего`: ``, `всем`: ``, `всеми`: ``, `всему`: ``, `всех`: ``, `всею`: ``, `всюду`: ``, `второй`: ``, `говорил`: ``, `говорит`: ``, `года`: ``, `году`: ``, `давно`: ``, `даже`: ``, `далеко`: ``, `дальше`: ``, `даром`: ``, `двадцатый`: ``, `двадцать`: ``, `двенадцатый`: ``, `двенадцать`: ``, `двух`: ``, `девятнадцатый`: ``, `девятнадцать`: ``, `девятый`: ``, `девять`: ``, `действительно`: ``, `день`: ``, `десятый`: ``, `десять`: ``, `довольно`: ``, `долго`: ``, `должно`: ``, `другая`: ``, `другие`: ``, `других`: ``, `друго`: ``, `другое`: ``, `другой`: ``, `если`: ``, `есть`: ``, `жизнь`: ``, `занят`: ``, `занята`: ``, `занято`: ``, `заняты`: ``, `затем`: ``, `зато`: ``, `зачем`: ``, `здесь`: ``, `значит`: ``, `именно`: ``, `иметь`: ``, `иногда`: ``, `каждая`: ``, `каждое`: ``, `каждые`: ``, `каждый`: ``, `кажется`: ``, `какая`: ``, `какой`: ``, `когда`: ``, `кого`: ``, `кому`: ``, `конечно`: ``, `которая`: ``, `которого`: ``, `которой`: ``, `которые`: ``, `который`: ``, `которых`: ``, `кроме`: ``, `кругом`: ``, `куда`: ``, `лишь`: ``, `лучше`: ``, `люди`: ``, `мало`: ``, `между`: ``, `меля`: ``, `менее`: ``, `меньше`: ``, `меня`: ``, `миллионов`: ``, `мимо`: ``, `мира`: ``, `много`: ``, `многочисленная`: ``, `многочисленное`: ``, `многочисленные`: ``, `многочисленный`: ``, `мной`: ``, `мною`: ``, `могут`: ``, `может`: ``, `можно`: ``, `можхо`: ``, `мочь`: ``, `наверху`: ``, `надо`: ``, `назад`: ``, `наиболее`: ``, `наконец`: ``, `нами`: ``, `начала`: ``, `наша`: ``, `наше`: ``, `наши`: ``, `него`: ``, `недавно`: ``, `недалеко`: ``, `нельзя`: ``, `немного`: ``, `нему`: ``, `непрерывно`: ``, `нередко`: ``, `несколько`: ``, `нибудь`: ``, `ниже`: ``, `низко`: ``, `никогда`: ``, `никуда`: ``, `ними`: ``, `ничего`: ``, `нужно`: ``, `обычно`: ``, `один`: ``, `одиннадцатый`: ``, `одиннадцать`: ``, `однажды`: ``, `однако`: ``, `одного`: ``, `одной`: ``, `около`: ``, `опять`: ``, `особенно`: ``, `отовсюду`: ``, `отсюда`: ``, `очень`: ``, `первый`: ``, `перед`: ``, `пожалуйста`: ``, `позже`: ``, `пока`: ``, `пора`: ``, `после`: ``, `посреди`: ``, `потом`: ``, `потому`: ``, `почему`: ``, `почти`: ``, `прекрасно`: ``, `просто`: ``, `против`: ``, `процентов`: ``, `пятнадцатый`: ``, `пятнадцать`: ``, `пятый`: ``, `пять`: ``, `разве`: ``, `рано`: ``, `раньше`: ``, `рядом`: ``, `сама`: ``, `сами`: ``, `самим`: ``, `самими`: ``, `самих`: ``, `само`: ``, `самого`: ``, `самой`: ``, `самом`: ``, `самому`: ``, `саму`: ``, `свое`: ``, `своего`: ``, `своей`: ``, `свои`: ``, `своих`: ``, `свою`: ``, `сеаой`: ``, `себе`: ``, `себя`: ``, `сегодня`: ``, `седьмой`: ``, `сейчас`: ``, `семнадцатый`: ``, `семнадцать`: ``, `семь`: ``, `сказал`: ``, `сказала`: ``, `сказать`: ``, `сколько`: ``, `слишком`: ``, `сначала`: ``, `снова`: ``, `собой`: ``, `собою`: ``, `совсем`: ``, `спасибо`: ``, `стал`: ``, `суть`: ``, `такая`: ``, `также`: ``, `такие`: ``, `такое`: ``, `такой`: ``, `твой`: ``, `твоя`: ``, `твоё`: ``, `тебе`: ``, `тебя`: ``, `теми`: ``, `теперь`: ``, `тобой`: ``, `тобою`: ``, `тогда`: ``, `того`: ``, `тоже`: ``, `только`: ``, `тому`: ``, `третий`: ``, `тринадцатый`: ``, `тринадцать`: ``, `туда`: ``, `тысяч`: ``, `уметь`: ``, `хорошо`: ``, `хотеть`: ``, `хоть`: ``, `хотя`: ``, `хочешь`: ``, `часто`: ``, `чаще`: ``, `чего`: ``, `человек`: ``, `чему`: ``, `через`: ``, `четвертый`: ``, `четыре`: ``, `четырнадцатый`: ``, `четырнадцать`: ``, `чтоб`: ``, `чтобы`: ``, `чуть`: ``, `шестнадцатый`: ``, `шестнадцать`: ``, `шестой`: ``, `шесть`: ``, `этим`: ``, `этими`: ``, `этих`: ``, `этого`: ``, `этой`: ``, `этом`: ``, `этому`: ``, `этот`: ``, `اخرى`: ``, `اربعة`: ``, `اطار`: ``, `اعادة`: ``, `اعلنت`: ``, `اكثر`: ``, `الاخيرة`: ``, `الان`: ``, `الاول`: ``, `الاولى`: ``, `التى`: ``, `التي`: ``, `الثاني`: ``, `الثانية`: ``, `الذاتي`: ``, `الذى`: ``, `الذي`: ``, `الذين`: ``, `السابق`: ``, `الماضي`: ``, `المقبل`: ``, `الوقت`: ``, `اليوم`: ``, `امام`: ``, `انها`: ``, `ايار`: ``, `ايام`: ``, `ايضا`: ``, `باسم`: ``, `بسبب`: ``, `بشكل`: ``, `ثلاثة`: ``, `جميع`: ``, `حاليا`: ``, `حوالى`: ``, `خلال`: ``, `زيارة`: ``, `سنوات`: ``, `شخصا`: ``, `صباح`: ``, `عاما`: ``, `عشرة`: ``, `عليه`: ``, `عليها`: ``, `عندما`: ``, `فيها`: ``, `كانت`: ``, `لقاء`: ``, `للامم`: ``, `لوكالة`: ``, `مايو`: ``, `مساء`: ``, `مقابل`: ``, `مليار`: ``, `مليون`: ``, `منها`: ``, `نفسه`: ``, `نهاية`: ``, `هناك`: ``, `واحد`: ``, `واضاف`: ``, `واضافت`: ``, `واكد`: ``, `واوضح`: ``, `وقال`: ``, `وقالت`: ``, `وكان`: ``, `وكانت`: ``, `يكون`: ``, `يمكن`: ``, `ἀλλά`: ``, `ἀλλ’`: ``, `ἀπό`: ``, `ἄλλοσ`: ``, `ἄρα`: ``, `ἐάν`: ``, `ἐγώ`: ``, `ἐμόσ`: ``, `ἐπί`: ``, `ἑαυτοῦ`: ``, `ἔτι`: ``, `ὅδε`: ``, `ὅστισ`: ``, `ὅτι`: ``, `ὑμόσ`: ``, `ὑπέρ`: ``, `ὑπό`: ``, `ὥστε`: ``, `あのかた`: ``, `あります`: ``, `おります`: ``, `貴方方`: ``}

	_, ok := stopWords[word]
	if ok {
		return true
	}
	return false
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

// From https://gist.github.com/seantalts/11266762
// For use with http.Client, to provide some additional options for timeout, etc.

type TimeoutTransport struct {
	http.Transport
	RoundTripTimeout time.Duration
}

type respAndErr struct {
	resp *http.Response
	err  error
}

type netTimeoutError struct {
	error
}

func (ne netTimeoutError) Timeout() bool { return true }

// If you don't set RoundTrip on TimeoutTransport, this will always timeout at 0
func (t *TimeoutTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	timeout := time.After(t.RoundTripTimeout)
	resp := make(chan respAndErr, 1)

	go func() {
		r, e := t.Transport.RoundTrip(req)
		resp <- respAndErr{
			resp: r,
			err:  e,
		}
	}()

	select {
	case <-timeout: // A round trip timeout has occurred.
		t.Transport.CancelRequest(req)
		return nil, netTimeoutError{
			error: fmt.Errorf("timed out after %s", t.RoundTripTimeout),
		}
	case r := <-resp: // Success!
		return r.resp, r.err
	}
}
