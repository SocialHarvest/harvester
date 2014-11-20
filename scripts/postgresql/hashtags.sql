/*
 PostgreSQL
 Date: 08/09/2014 13:33:31 PM
*/

-- ----------------------------
--  Table structure for hashtags
-- ----------------------------
DROP TABLE IF EXISTS "hashtags";
CREATE TABLE "hashtags" (
	"time" timestamp(6) NULL,
	"harvest_id" varchar(255) NOT NULL COLLATE "default",
	"territory" varchar(255) COLLATE "default",
	"network" varchar(75) COLLATE "default",
	"contributor_id" varchar(255) COLLATE "default",
	"contributor_screen_name" varchar(255) COLLATE "default",
	"message_id" varchar(255) COLLATE "default",
	"contributor_lang" varchar(8) COLLATE "default",
	"contributor_gender" int2,
	"contributor_type" varchar(100) COLLATE "default",
	"contributor_longitude" float8,
	"contributor_latitude" float8,
	"contributor_geohash" varchar(100) COLLATE "default",
	"contributor_name" varchar(255) COLLATE "default",
	"tag" varchar(255) COLLATE "default",
	"keyword" varchar(150) COLLATE "default",
	"contributor_country" varchar(6) COLLATE "default",
	"contributor_city" varchar(75) COLLATE "default",
	"contributor_city_pop" int4,
	"contributor_region" varchar(50) COLLATE "default"
)
WITH (OIDS=FALSE);

-- ----------------------------
--  Primary key structure for table hashtags
-- ----------------------------
ALTER TABLE "hashtags" ADD PRIMARY KEY ("harvest_id") NOT DEFERRABLE INITIALLY IMMEDIATE;

-- ----------------------------
--  Uniques structure for table hashtags
-- ----------------------------
ALTER TABLE "hashtags" ADD CONSTRAINT "hashtags_harvest_id_unique" UNIQUE ("harvest_id") NOT DEFERRABLE INITIALLY IMMEDIATE;

-- ----------------------------
--  Indexes structure for table hashtags
-- ----------------------------
CREATE INDEX  "h_contributor_id_key" ON "hashtags" USING btree(contributor_id COLLATE "default" DESC NULLS LAST);
CREATE INDEX  "h_contributor_geohash_key" ON "hashtags" USING btree(contributor_geohash COLLATE "default" ASC NULLS LAST);
CREATE INDEX  "h_message_id_key" ON "hashtags" USING btree(message_id COLLATE "default" DESC NULLS LAST);
CREATE INDEX  "h_tag_key" ON "hashtags" USING btree(tag COLLATE "default" ASC NULLS LAST);
CREATE INDEX  "h_keyword_key" ON "hashtags" USING btree(keyword COLLATE "default" ASC NULLS LAST);
CREATE INDEX  "h_time_key" ON "hashtags" USING btree("time" DESC NULLS LAST);

