#Social Harvest (harvester)
[![wercker status](https://app.wercker.com/status/f2922c2bb4a25b6c5adc65ae41b751bb/s "wercker status")](https://app.wercker.com/project/bykey/f2922c2bb4a25b6c5adc65ae41b751bb) [![Coverage Status](https://coveralls.io/repos/SocialHarvest/harvester/badge.png?branch=master)](https://coveralls.io/r/SocialHarvest/harvester?branch=master)

http://www.socialharvest.io

Social Harvest is a scalable and flexible open-source social media analytics platform.

There are three parts to the platform. This harvester, a reporter API, and the [Social Harvest Dashboard](https://github.com/SocialHarvest/dashboard) 
for front-end visualizations and reporting through a web browser.

This application (harvester) gathers data from Twitter, Facebook, etc. using Go and concurrently stores to a variety of data stores.
Social Harvest also logs to disk and those log files can be used by programs like Fluentd for additional flexibility in your data 
store and workflow. In addition to harvesting and storing, the harvester configuration can also be accessed through an API that comes 
with Social Harvest. 

While Social Harvest&reg; is a registered trademark, this software is made publicly available under the GPLv3 license.
"Powered by Social Harvest&reg;" on any rendered web pages (ie. in the footer) and within any documentation, web sites, or other materials 
would very much be appreciated since this is an open-source project.

## Configuration

You'll need to create a JSON file for configuring Social Harvest. Ensure this configuration file is named ```social-harvest-conf.json``` 
and sits next to the binary Go built or next to the main.go file (unless you pass another location and name when running Social Harvest).

For an example configuration, see ```example-conf.json```

Within the configuration, you can fine tune Social Harvest as well as define the "territories" to be monitored. A territory is just 
a set of criteria for which to search for across several social media networks. You can look for specific keywords, URLs, and even 
track various accounts for growth.

You will also need to provide your application API keys within this configuration file. There is (currently) no OAuth support within 
the RESTful API server. All social media services have an access token you will be able to generate and use within Social Harvest. 
You do not need a web browser to configure Social Harvest. Configurations are portable and can be deployed with each harvester.

Note: If you are working with the Social Harvest Dashboard and are developing locally with ```grunt dev``` then you will likely be
running the dashboard on a Node.js server with a port of ```8881``` (by default) and you will need to configure CORS for that origin. 
You can add as many allowed origins as you like in the configuration.

## Installation

Installation is pretty simple. You'll need to have Go installed and setup, then run: ```go get github.com/SocialHarvest/harvester``` 

Getting the Go packages this application uses is as simple as issueing a ```go get``` command before running or building. Every 3rd party 
package Social Harvest uses has been "vendored" (or forked and available from github.com/SocialHarvestVendors). Even packages that came 
from other revision control systems. So this means everything should be Git and from GitHub.

The data files used for various machine learning and analysis purposes will automatically be copied into an ```sh-data``` directory. 
This directory will be created next to the binary or the source (if you ran without building). The data will be downloaded, if it doesn't 
exist in this directory already, each time the application starts. So if something goes wrong, feel free to remove this directory and restart
the harvester application.

Why the file download? Because ultimately these files could be quite large and they might come from S3 and this process more or less 
installs things for you so you don't need to go wrangling dependencies. This will become more robust over time. Plus, GitHub doesn't
want us storing such large files and getting the actual packages would take forever.

If you're harvesting into a Postgres database, be sure to setup your tables using the SQL files in the ```scripts/postgresql``` directory. 
It'll save you a lot of trouble. However, these will change during development until Social Harvest has a stable version released. 

Then to run Social Harvest before (or without) building it (at the package src under your $GOPATH), you can issue the following command because 
there are multiple files in the main package (and you don't want to run the _test files):

```
go run main.go harvest.go
```

Preferably, you'll just build and use a Social Harvest binary by running:

```
go build
```

You need not specify the files in this case. It will leave you with a ```harvester``` executable file. Run this. Once configured and running, you should 
have a pretty awesome social media data harvester. That is all the harvester is responsible for.

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

## Contributing & Licensing

Social Harvest is an open-source project and any community contributions are always appreciated. You can write blog posts, tutorials, help 
with documentation, submit bug reports, feature requests, and even pull requests. It's all helpful.

Please keep in mind that Social Harvest is open-source and any contributions must be compatible with the GPLv3 license. 
It would also be very much appreciated if you put a "powered by Social Harvest" somewhere on your application/web site (ie. the footer). 
You are free to make money from Social Harvest of course, but you aren't free to modify the source directly and squirrel it away. Sharing is caring. 
If you have proprietary stuff, keep it outside of the Social Harvest package/binary. Social Harvest is designed to gather data on its own 
and not get in the way of other applications.

If GPLv3 does not work for you or your organization, please feel free to get in touch about other commercial licensing options.    

### Bug Reporting
Before submitting any bugs, please be sure to check the [issues list in GitHub](https://github.com/SocialHarvest/harvester/issues?state=open) first. 
Please be sure to provide necessary information with any bugs as it will help expidite the process of fixing them. Operating system, version of Go, etc.

### Feature Requests
If you'd like to see a feature implemented in Social Harvest, again first check the issues looking for [feature requests](https://github.com/SocialHarvest/harvester/issues?labels=feature+request&page=1&state=open) to ensure someone else hasn't already suggested it (feel free to +1 away in the comments).

Please keep in mind that Social Harvest has some specific goals in mind and not all features can be put into the platform. To do any work on Social Harvest, 
you will of course need to fork the repository so that you can send pull requests.

## Questions? Chat?
You should hop in the [Gitter.im channel](https://gitter.im/SocialHarvest) and say hi. It's a great place to ask questions for help, suggest ideas, or just chat about Social Harvest. 
The channel is open and chat history is saved so, if you don't get an instant response, be sure to check back regularly.

Twitter is another good way to get in touch. You can follow [@socialharvest](http://www.twitter.com/socialharvest)



