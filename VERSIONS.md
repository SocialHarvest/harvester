# Social Harvest Harvester Version History

Best efforts will be made to keep this up to date, but there are no guarantees before a stable version is released. 
This file will log major feature advancements and bug fixes. Not quite everything will be noted (especially at first). 
Please check GitHub issues.

## 0.16.0 Alpha
-------------

Unfortunately had to remove support for InfluxDB as a storage engine. InfluxDB is undergoing some major changes and 
the client package has become unstable as a result (in terms of its API and breaking changes). Until InfluxDB is more 
stable, it has been removed. It wasn't working well for analytics anyway and hopefully that will change as well.

InfluxDB can still, of course, be used if setup through Fluentd. Fluentd can read log output from Social Harvest 
in order to store data practically anywhere.

## 0.15.0 Alpha
-------------

With the addition of sentiment analysis, Social Harvest (harvester) is now in a feature complete state. Very little, 
in terms of API and schema for harvested data is expected to change between now and version 1.0. The first version 
of Social Harvest is focusing on analyzing and monitoring a core set of social networks with internal tools. There 
are no 3rd party APIs being used for geolocation, sentiment, or gender detection. This has prompted the "alpha" label.

## 0.14.0
-------------

Adjusted the schema for several series. ```ContibutorState``` is now ```ContributorRegion``` because it can include more 
then just a US state. It is the equivalent to "admin2 code" in the Geonames data set. ```ContributorCounty``` has been 
removed. However, it may be added back in the future. The Geobed package can use yet another data set to decode id number 
values to county names. However, it likely won't be added until needed (or until there's a better time to do so). This 
does not affect plotted points on a map of course. The only benefit would be to group by county or fill a cloropleth 
vector map (like those seen with D3.js). However, those are typically for US counties anyway. So it's limited. We can 
also always get this kind of information after the fact given we have city, region, country.

Also added city population to the series with city location information. This will allow for a whole new dimension in 
reporting. It will now be possible to easily look at and aggregate data from major cities. It will also be possible to 
filter out data from small cities.

## 0.13.0
-------------

Removing depencency on external geocoding APIs. An in memory geodcoer (Geobed) is now used. This is also available as 
a stand-alone package. The ```ContributorState``` field is now more of a state/province/"admin2 code" value. Different 
data sets (Geonames/MaxMind) had different values so it's not strictly for US states anymore. The ```ContributorCounty``` 
field is now not used. It can be in the future though should Geobed add that data set and do all the lookups. The general 
consuses was that county is so rarely used (and could be figured out later anyway based on city, state/region, country).

## 0.12.1
-------------

Fixing an issue with configuration where Postgres connection would be closed after configuring the database. It now will 
be closed once the application exits (like before). Fixing a SQL create table script as well.

Added an API endpoint to test the connection to the Postgres database. Will need to do the same for InfluxDB. 

## 0.12.0
-------------

Lots of configuration enhancements.

Data files (for gender detection, and in the future sentiment analysis) will now be downloaded and installed automatically. 
This makes the installation process much easier for the harvester. It should really just be a matter of downloading a binary 
(or clone from GitHub) and running it. Users won't need to go hunting down files and copying them to specific locations. 
This will become a more robust asset system in the future (allowing for custom replacement data files).

Configuration updates also go into this new ```sh-data``` directory. The harvester does not require a database. So the 
configuration is stored in a JSON file (sure, could have been SQLite or something too). Many users will simply provide this 
config file with each harvester server that they bring online in an automated fashion...But other users will want to change 
the config through the harvester API and have it persist should the harvester crash and restart.

Again, this will become a more robust system in the future because multiple harvesters will be at work in parallel and so 
config updates will need to propagate out to all of those machines as well.

Configuration can now be reloaded, reset, and managed via the harvester API. The application will also exit if there is no 
JSON configuration file available. There must be, at least a minimal, config file.

Also, the database config takes new ```retentionDays``` and ```partitionDays``` values. This helps the harvester with data 
retention. InfluxDB automatically can expire old data, but in the future the harvester will use Postgres' PARTITION feature 
to set partitions and then scheduled tasks will simply drop old tables outside this retention period. For now, only the 
```retentionDays``` setting is used. If greater than 0, it will currently prevent older data from being stored. It is possible 
for the harvester to pick up old data (sometimes months old) and this adds unnecessary strain on InfluxDB because soon after 
the data is added, it's removed by the database. This can happen over and over and over. So this age check on data helps. 
It's also going to help when using Postgres of course. Any gate keeping that prevents un-used data from being inserted helps.

## 0.11.0
-------------

Now supporting InfluxDB again as a data store. InfluxDB has many features and advantages over Postgres for time series data. 
Features include the ability to automatically remove data after an expiration date and many useful aggregation functions. 
As a result, it will allow the API and dashboard to come together sooner. Postgres and InfluxDB will be the front-runners, 
but additional database support may be added in the future.

Additionally, the harvester will now only gather data. A separate reporter application will be responsible for getting the 
data back out for front-end dashboards, etc. This keeps things better organized and allows the harvester to be a smaller binary. 
It also lets the harvester scale without bringing with it a, perhaps, redundant and unused set of functionality.

## 0.10.0
-------------

The harvesting of growth metrics has been added. Not all services offer as much detail through their APIs as one can get 
by using the network's own tools (Facebook's Insights, Google's Analytics, etc.). However, this data is useful for tracking 
the basic growth metrics available and in particular comparing to competitors and other accounts/pages. This will likely 
change over time as new data becomes available through various APIs.

This completes all harvesting for alpha and beta. There are a few more analysis tasks left to be done before beta, such as 
sentiment analysis, but at this point all data to be harvested for the immediate future is being gathered.

## 0.9.2
-------------

Fixed a small bug with HTTP request body being closed in situations where it it was not available to be closed.

## 0.9.1
-------------

Fixed an unclosed HTTP request that eventually lead to the application crashing.

## 0.9.0
-------------

Removed the use of the upper.io/db package. It was causing some major issues unfortunately. This means that, for now, MongoDB 
is no longer a supported database. It can still be used if using Fluentd to watch the log files to then store in MongoDB. 
However, the API, settings, and upcoming reports will not make any effort to support MongoDB for the time being. 
SQL databases are likely going to be the preferred database simply because of cloud hosting options and cost. Even more
specifically, Postgres is likely going to be the preferred SQL database due to performance and features.

A database is now also completely optional. The refactored code with use of the sqlx package over upper.io/db has also 
made it easy to check for the existence of a database connection across the entire application. So Social Harvest will 
start and harvest even if no database configuration is defined.

## 0.8.0
-------------

Added basic authentication (add "apiKeys" to config JSON) which supports multiple values. Added in the support for Bugsnag
under a debug struct in the config that also allows for profiling to be enabled. Profiling may be removed in the future,
but will be available during alpha/beta for sure.

## 0.7.1
-------------

Fixed a bug where group queries were not being limited. This was previously available, but a refactor led to a regression.

## 0.7.0
-------------

Now harvesting content from Google+ by keyword. This wraps up all content harvests through alpha into beta versions. 
Aside from some sentiment analysis and such, this makes Social Harvest a fully functional social media listening platform. 
Hundreds of thousands of messages are easily harvested on a monthly basis without hitting rate limits or taking too much time. 
Harvesting at faster rates will be looked into later on (testing shows it's not necessary now). Growth metrics will be focused on next.

## 0.6.0
-------------

Now harvesting from Instagram by keyword/tag. A few fixes for performance (separate geocode package) and a fix to actually use (and store) 
the reverse geocode during harvests.


## 0.5.1
-------------

Fixed a memory leak! The Facebook harvesting was opening up a lot of goroutines that never returned (or at least not before the program crashed). 
I couldn't figure out exactly what was causing it (I suspect something with the HTTP requests or session stuff in the package I was using). 
So I just replaced that package dependency with my own HTTP requests (with timeouts) and pprof showed goroutine counts increasing and decreasing. 
Since goroutines return now, too many don't seem to be open. It could have been waiting that caused the crash too and not necessarily the 
goroutinues persay.

This leak took 4 evenings to figure out and really delayed some things, but I'm glad to have fixed it (let's hope). 

## 0.5.0 
-------------

Changed logging altogether because it wasn't really the HTTP issues so much as that there was a memory leak. The observer pattern 
being used led to a very rapid memory leak. After removing that, log4go was blocking and created another issue. So a completely new, 
concurrent friendly, logging method was created. This slims down the codebase a bit too. One less dependency.


## 0.4.1
-------------

Fixed a bug with HTTP requests and the geocoder. This was creating issues elsewhere causing a timeout.


## 0.4.0 
-------------

New API methods to get data back out including some streaming API methods.

## 0.3.1
-------------
Fixed log4go package. While not a patch to Social Harvest directly, I'm still incrementing the patch version. 
This is completely backwards compatible and nothing should have changed. This fixes a data race condition (so _kinda_ important).

## 0.3.0
-------------
It's been long enough that the codebase has presented a clear direction and so it has revealed the need for it's first 
specific design pattern (observer pattern).

A fairly big refactor has taken place in order to separate some concerns and reduce unnecessary configurations and 
data being passed through functions. Good use of channels has been made in order to create a good pipeline and increase
performance with regard to concurrency (that -race flag is going to get a workout over the next few versions, but hopefully
an easy one).

The harvester package now no longer knows about the database or log files. It does not write out to anywhere except a channel. 
Yay!

Streaming API! Yay! In fact, it will be possible to filter this stream...Roll your own Gnip anyone?

Each harvester is now greatly simplified too (Facebook needs a refactor).


## 0.2.0
-------------
The scheduler was setup and some API methods implemented. Users can now get the current configuration returned via 
the harvester API. Scheduled jobs can also be retrieved (but not set via the API yet).

Territories can now override the global credentials for the social network APIs (referred to as services as to not
be confused with the harvester API).

Instagram harvester added (configuration and some test calls).

A hypermedia API response format, in JSON, was defined (after spending a LOT of time months ago, and again over the past 
few days, researching various formats).


## 0.1.0
-------------
Proof of concept. This was the first version that included all the beginnings of a harvester.

Multiple services were connected in the code and called manually to ensure they worked (not test cases).

Then Facebook was chosen to be harvested from first. The scheulder was setup but not in use.

Configuration was defined for harvesting, API services, database, etc.

Several databases were configured using upper.io so that 3 (technically 4) databases are supported natively.

Test cases were put in superficially. Need more tests written. Everything was hooked up to Drone.io and Coveralls.io.