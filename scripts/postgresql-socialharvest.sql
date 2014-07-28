-- ----------------------------
--  Table structure for shared_links
-- ----------------------------
DROP TABLE IF EXISTS "public"."shared_links";
CREATE TABLE "public"."shared_links" (
	"time" timestamp(6) NULL,
	"harvest_id" varchar(255) NOT NULL COLLATE "default",
	"territory" varchar(255) COLLATE "default",
	"network" varchar(75) COLLATE "default",
	"contributor_id" varchar(255) COLLATE "default",
	"contributor_screen_name" varchar(255) COLLATE "default",
	"url" varchar(255) COLLATE "default",
	"expanded_url" varchar(255) COLLATE "default",
	"host" varchar(150) COLLATE "default",
	"facebook_shares" int4,
	"contributor_facebook_category" varchar(150) COLLATE "default",
	"message_id" varchar(255) COLLATE "default"
)
WITH (OIDS=FALSE);
ALTER TABLE "public"."shared_links" OWNER TO "upper";

-- ----------------------------
--  Table structure for shared_media
-- ----------------------------
DROP TABLE IF EXISTS "public"."shared_media";
CREATE TABLE "public"."shared_media" (
	"time" timestamp(6) NULL,
	"harvest_id" varchar(255) NOT NULL COLLATE "default",
	"territory" varchar(255) COLLATE "default",
	"network" varchar(75) COLLATE "default",
	"contributor_id" varchar(255) COLLATE "default",
	"contributor_screen_name" varchar(255) COLLATE "default",
	"type" varchar(75) COLLATE "default",
	"preview" varchar(255) COLLATE "default",
	"source" varchar(255) COLLATE "default",
	"url" varchar(255) COLLATE "default",
	"expanded_url" varchar(255) COLLATE "default",
	"host" varchar(150) COLLATE "default",
	"contributor_facebook_category" varchar(150) COLLATE "default",
	"message_id" varchar(255) COLLATE "default"
)
WITH (OIDS=FALSE);
ALTER TABLE "public"."shared_media" OWNER TO "upper";

-- ----------------------------
--  Table structure for questions
-- ----------------------------
DROP TABLE IF EXISTS "public"."questions";
CREATE TABLE "public"."questions" (
	"time" timestamp(6) NULL,
	"harvest_id" varchar(255) NOT NULL COLLATE "default",
	"territory" varchar(255) COLLATE "default",
	"network" varchar(75) COLLATE "default",
	"contributor_id" varchar(255) COLLATE "default",
	"contributor_screen_name" varchar(255) COLLATE "default",
	"iso_language_code" varchar(5) COLLATE "default",
	"longitude" float8,
	"latitude" float8,
	"geohash" varchar(100) COLLATE "default",
	"message_id" varchar(255) COLLATE "default",
	"message" text COLLATE "default"
)
WITH (OIDS=FALSE);
ALTER TABLE "public"."questions" OWNER TO "tom";

-- ----------------------------
--  Table structure for contributors
-- ----------------------------
DROP TABLE IF EXISTS "public"."contributors";
CREATE TABLE "public"."contributors" (
	"time" timestamp(6) NULL,
	"harvest_id" varchar(255) NOT NULL COLLATE "default",
	"territory" varchar(255) COLLATE "default",
	"network" varchar(75) COLLATE "default",
	"contributor_id" varchar(255) COLLATE "default",
	"contributor_screen_name" varchar(255) COLLATE "default",
	"iso_language_code" varchar(5) COLLATE "default",
	"longitude" float8,
	"latitude" float8,
	"geohash" varchar(100) COLLATE "default",
	"gender" int4,
	"name" varchar(150) COLLATE "default",
	"about" text COLLATE "default",
	"checkins" int4,
	"company_overview" text COLLATE "default",
	"description" text COLLATE "default",
	"founded" varchar(150) COLLATE "default",
	"general_info" text COLLATE "default",
	"likes" int4,
	"link" varchar(255) COLLATE "default",
	"street" varchar(150) COLLATE "default",
	"city" varchar(150) COLLATE "default",
	"state" varchar(75) COLLATE "default",
	"zip" varchar(35) COLLATE "default",
	"country" varchar(75) COLLATE "default",
	"phone" varchar(35) COLLATE "default",
	"talking_about_count" int4,
	"were_here_count" int4,
	"url" varchar(255) COLLATE "default",
	"products" text COLLATE "default",
	"contributor_facebook_category" varchar(150) COLLATE "default"
)
WITH (OIDS=FALSE);
ALTER TABLE "public"."contributors" OWNER TO "upper";

-- ----------------------------
--  Table structure for mentions
-- ----------------------------
DROP TABLE IF EXISTS "public"."mentions";
CREATE TABLE "public"."mentions" (
	"time" timestamp(6) NULL,
	"harvest_id" varchar(255) NOT NULL COLLATE "default",
	"territory" varchar(255) COLLATE "default",
	"network" varchar(75) COLLATE "default",
	"contributor_id" varchar(255) COLLATE "default",
	"contributor_screen_name" varchar(255) COLLATE "default",
	"iso_language_code" varchar(5) COLLATE "default",
	"longitude" float8,
	"latitude" float8,
	"geohash" varchar(100) COLLATE "default",
	"mentioned_id" varchar(255) COLLATE "default",
	"mentioned_screen_name" varchar(255) COLLATE "default",
	"mentioned_type" varchar(75) COLLATE "default",
	"contributor_facebook_category" varchar(150) COLLATE "default",
	"message_id" varchar(255) COLLATE "default"
)
WITH (OIDS=FALSE);
ALTER TABLE "public"."mentions" OWNER TO "upper";

-- ----------------------------
--  Table structure for messages
-- ----------------------------
DROP TABLE IF EXISTS "public"."messages";
CREATE TABLE "public"."messages" (
	"time" timestamp(6) NULL,
	"harvest_id" varchar(255) NOT NULL COLLATE "default",
	"territory" varchar(255) COLLATE "default",
	"network" varchar(75) COLLATE "default",
	"contributor_id" varchar(255) COLLATE "default",
	"contributor_screen_name" varchar(255) COLLATE "default",
	"iso_language_code" varchar(5) COLLATE "default",
	"longitude" float8,
	"latitude" float8,
	"geohash" varchar(100) COLLATE "default",
	"facebook_shares" int4,
	"contributor_facebook_category" varchar(150) COLLATE "default",
	"message_id" varchar(255) COLLATE "default"
)
WITH (OIDS=FALSE);
ALTER TABLE "public"."messages" OWNER TO "upper";

-- ----------------------------
--  Table structure for harvest
-- ----------------------------
DROP TABLE IF EXISTS "public"."harvest";
CREATE TABLE "public"."harvest" (
	"territory" varchar(150) COLLATE "default",
	"network" varchar(75) COLLATE "default",
	"action" varchar(255) COLLATE "default",
	"value" text COLLATE "default",
	"last_time_harvested" timestamp(6) NULL,
	"last_id_harvested" varchar(255) COLLATE "default",
	"items_harvested" int4,
	"harvest_time" timestamp(6) NULL
)
WITH (OIDS=FALSE);
ALTER TABLE "public"."harvest" OWNER TO "upper";

-- ----------------------------
--  Primary key structure for table shared_links
-- ----------------------------
ALTER TABLE "public"."shared_links" ADD PRIMARY KEY ("harvest_id") NOT DEFERRABLE INITIALLY IMMEDIATE;

-- ----------------------------
--  Uniques structure for table shared_links
-- ----------------------------
ALTER TABLE "public"."shared_links" ADD CONSTRAINT "shared_links_harvest_id_unique" UNIQUE ("harvest_id") NOT DEFERRABLE INITIALLY IMMEDIATE;

-- ----------------------------
--  Primary key structure for table shared_media
-- ----------------------------
ALTER TABLE "public"."shared_media" ADD PRIMARY KEY ("harvest_id") NOT DEFERRABLE INITIALLY IMMEDIATE;

-- ----------------------------
--  Uniques structure for table shared_media
-- ----------------------------
ALTER TABLE "public"."shared_media" ADD CONSTRAINT "shared_media_harvest_id_unique" UNIQUE ("harvest_id") NOT DEFERRABLE INITIALLY IMMEDIATE;

-- ----------------------------
--  Primary key structure for table questions
-- ----------------------------
ALTER TABLE "public"."questions" ADD PRIMARY KEY ("harvest_id") NOT DEFERRABLE INITIALLY IMMEDIATE;

-- ----------------------------
--  Uniques structure for table questions
-- ----------------------------
ALTER TABLE "public"."questions" ADD CONSTRAINT "questions_harvest_id_unique" UNIQUE ("harvest_id") NOT DEFERRABLE INITIALLY IMMEDIATE;

-- ----------------------------
--  Primary key structure for table contributors
-- ----------------------------
ALTER TABLE "public"."contributors" ADD PRIMARY KEY ("harvest_id") NOT DEFERRABLE INITIALLY IMMEDIATE;

-- ----------------------------
--  Uniques structure for table contributors
-- ----------------------------
ALTER TABLE "public"."contributors" ADD CONSTRAINT "contributors_harvest_id_unique" UNIQUE ("harvest_id") NOT DEFERRABLE INITIALLY IMMEDIATE;

-- ----------------------------
--  Primary key structure for table mentions
-- ----------------------------
ALTER TABLE "public"."mentions" ADD PRIMARY KEY ("harvest_id") NOT DEFERRABLE INITIALLY IMMEDIATE;

-- ----------------------------
--  Uniques structure for table mentions
-- ----------------------------
ALTER TABLE "public"."mentions" ADD CONSTRAINT "mentions_harvest_id_unique" UNIQUE ("harvest_id") NOT DEFERRABLE INITIALLY IMMEDIATE;

-- ----------------------------
--  Primary key structure for table messages
-- ----------------------------
ALTER TABLE "public"."messages" ADD PRIMARY KEY ("harvest_id") NOT DEFERRABLE INITIALLY IMMEDIATE;

-- ----------------------------
--  Uniques structure for table messages
-- ----------------------------
ALTER TABLE "public"."messages" ADD CONSTRAINT "messages_harvest_id_unique" UNIQUE ("harvest_id") NOT DEFERRABLE INITIALLY IMMEDIATE;

