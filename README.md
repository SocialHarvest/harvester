#Social Harvest (harvester)
[![Gitter chat](https://badges.gitter.im/SocialHarvest/harvester.png)](https://gitter.im/SocialHarvest/harvester) [![Build Status](https://drone.io/github.com/SocialHarvest/harvester/status.png)](https://drone.io/github.com/SocialHarvest/harvester/latest) [![Coverage Status](https://coveralls.io/repos/SocialHarvest/harvester/badge.png?branch=master)](https://coveralls.io/r/SocialHarvest/harvester?branch=master) [![Stories in Ready](https://badge.waffle.io/socialharvest/harvester.png?label=ready&title=Ready)](https://waffle.io/socialharvest/harvester)

http://www.socialharvest.io

Harvests data from Twitter, Facebook, etc. using Go and concurrently stores to a variety of data stores.
Social Harvest also logs to disk and those log files can be used by programs like Fluentd for additional 
flexibility in your data store and workflow.

In addition to harvesting and storing, data can also be retrieved through an API that comes with Social Harvest.
Of course, a separate stand alone integration is also possible since data was stored where ever needed.

For front-end visualizations using the harvested data, be sure to look at the [Social Harvest Dashboard](https://github.com/SocialHarvest/dashboard) project.

This makes Social Harvest a scalable and completely flexible social media analytics platform suitable for any need.

While Social Harvest&reg; is a registered trademark, this software is made publicly available under the GPLv3 license.
"Powered by Social Harvest&reg;" on any rendered web pages (ie. in the footer) and within any documentation, promotional, or sales 
materials would very much be appreciated.

## Configuration

You'll need to create a JSON file for configuring Social Harvest. Ensure this configuration file is named ```social-harvest-conf.json``` 
(unless you pass another location and name when running Social Harvest).

For an example configuration, see ```example-conf.json```

Within the configuration, you can fine tune Social Harvest as well as define the "territories" to be monitored. A territory is just 
a set of criteria for which to search for across several social media networks. You can look for specific keywords, URLs, and even 
track various accounts for growth.

You will also need to provide your application API keys within this configuration file. There is (currently) no OAuth support within 
the RESTful API server. All social media services have an access token you will be able to generate and use within Social Harvest. 
You do not need a web browser to configure Social Harvest. Configurations are porable and can be deployed with each harvester.

Note: If you are working with the Social Harvest Dashboard and are developing locally with ```grunt dev``` then you will likely be
running the dashboard on a Node.js server with a port of ```8881``` (by default) and you will need to configure CORS for that origin. 
You can add as many allowed origins as you like in the configuration.

## Installation

First, you'll need Mercurial and Bazaar since a few packages use those version control systems. On Ubuntu it's as easy as 
```apt-get install bzr``` and ```apt-get install mercurial```.

Installation is pretty simple. You'll need to have Go installed and with your $GOPATH set: ```go get github.com/SocialHarvest/harvester``` 

The dependencies should be handled automatically but you may need to call ```go get```.

You'll need to copy the ```data``` directory (and its contents) to be next to the program you run. So if you build the harvester, ensure where ever 
you put the harvester binary, you have this data directory sitting in the same directory. This will change in the future, but for now it contains 
the data sets for detecting gender.

If you're using a SQL database, be sure to setup your tables using the SQL files in the ```scripts``` directory. It'll save you a lot of trouble. 
However, these will change quite frequently during development until Social Harvest has a stable version released.

Social Harvest makes use of [upper.io](https://upper.io/db) which abstracts some common database calls for a few databases using separate driver packages.
Some of these packages are not directly used within Social Harvest, so you may need to get them separately so that they are in your $GOPATH 
before building.

Be sure to look at the upper.io documentation for additional details. MongoDB, specifically, makes use of ```labix.org/v2/mgo``` which uses the
[bazaar](http://bazaar.canonical.com/en/) version control system. So you'll need that. See [upper.io/db/mongo](https://upper.io/db/mongo) for more information. 
Even if you are not using MongoDB, Social Harvest is going to require this package to be installed in order to run and build. The same is going to be true 
for PostgreSQL, you'll want to run: ```go get github.com/lib/pq``` to get that one if you're using PostgreSQL. Since Social Harest does not directly use this 
package, you shouldn't need it if you don't plan to use PostgreSQL. MySQL support is via ```github.com/go-sql-driver/mysql``` so you may need that as well. 
Again, all of this is explained in upper.io's documentation.

All other dependencies (with the exception of testify, see below) should be obtained easily enough via ```go get```. Then to run Social Harvest before (or without) 
building it (at the package src under your $GOPATH), you can issue the following command because there are multiple files in the main package (and you don't want to run the _test files):

```
go run main.go harvest.go
```

Preferably, you'll just build a Social Harvest binary by running:

```
go build
```

You need not specify the files in this case. It will leave you with a ```harvester``` executable file. Run this. Once running, you should have an API server which 
the Dashboard web application can talk to in order to visualize harvested data. Congratulation, you now have your own social media analytics platform!

## Testing

Social Harvest currently makes use of the testify package which you'll need to get first before running the tests.

```
go get github.com/stretchr/testify
go test ./...
```

Social Harvest also has a few performance benchmarks. Feel free to run tests with benchmarks:

```
go test ./... -bench=".*"
```

Note that while Go performs some operations really, really, fast...Each social network's API has a rate limit which is going to make
a lot of this more of a novelty than something actually required. For example: it's nice to know we can create over a million geohashes 
per second, but we aren't going to have that many results from an API returned to us each second.

Still, speed and concurrency are part of Social Harvest's goals.

## Contributing

Social Harvest is an open-source project and any community contributions are always appreciated. You can write blog posts, tutorials, help 
with documentation, submit bug reports, feature requests, and even pull requests. It's all helpful.

Please keep in mind that Social Harvest is open-source and any contributions must be compatible with the GPLv3 license.

### Bug Reporting
Before submitting any bugs, please be sure to check the [issues list in GitHub](https://github.com/SocialHarvest/harvester/issues?state=open) first. 
Please be sure to provide necessary information with any bugs as it will help expidite the process of fixing them.

### Feature Requests
If you'd like to see a feature implemented in Social Harvest, again first check the issues looking for [feature requests](https://github.com/SocialHarvest/harvester/issues?labels=feature+request&page=1&state=open) to ensure someone else hasn't already suggested it (feel free to +1 away in the comments).

Please keep in mind that Social Harvest has some specific goals in mind and not all features can be put into the platform. To do any work on Social Harvest, 
you will of course need to fork the repository so that you can send pull requests.

## Questions? Chat?
You should hop in the [Gitter.im channel](https://gitter.im/SocialHarvest) and say hi. It's a great place to ask questions for help, suggest ideas, or just chat about Social Harvest. 
The channel is open and chat history is saved so, if you don't get an instant response, be sure to check back regularly.



