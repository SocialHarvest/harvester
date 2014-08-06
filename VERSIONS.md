# Social Harvest Harvester Version History

Best efforts will be made to keep this up to date, but there are no guarantees before a stable version is released. 
This file will log major feature advancements and bug fixes. Not quite everything will be noted (especially at first). 
Please check GitHub issues.

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