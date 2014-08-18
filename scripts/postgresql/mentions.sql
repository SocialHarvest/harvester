/*
 PostgreSQL
 Date: 08/06/2014 22:53:04 PM
*/

-- ----------------------------
--  Table structure for mentions
-- ----------------------------
DROP TABLE IF EXISTS "mentions";
CREATE TABLE "mentions" (
	"time" timestamp(6) NULL,
	"harvest_id" varchar(255) NOT NULL COLLATE "default",
	"territory" varchar(255) COLLATE "default",
	"network" varchar(75) COLLATE "default",
	"contributor_id" varchar(255) COLLATE "default",
	"contributor_screen_name" varchar(255) COLLATE "default",
	"contributor_lang" varchar(8) COLLATE "default",
	"contributor_longitude" float8,
	"contributor_latitude" float8,
	"contributor_geohash" varchar(100) COLLATE "default",
	"mentioned_id" varchar(255) COLLATE "default",
	"mentioned_screen_name" varchar(255) COLLATE "default",
	"mentioned_type" varchar(75) COLLATE "default",
	"message_id" varchar(255) COLLATE "default",
	"contributor_type" varchar(100) COLLATE "default",
	"contributor_gender" int2,
	"contributor_name" varchar(255) COLLATE "default",
	"mentioned_name" varchar(255) COLLATE "default",
	"mentioned_longitude" float8,
	"mentioned_latitude" float8,
	"mentioned_geohash" varchar(100) COLLATE "default",
	"mentioned_lang" varchar(8) COLLATE "default",
	"mentioned_gender" int2
)
WITH (OIDS=FALSE);

-- ----------------------------
--  Primary key structure for table mentions
-- ----------------------------
ALTER TABLE "mentions" ADD PRIMARY KEY ("harvest_id") NOT DEFERRABLE INITIALLY IMMEDIATE;

-- ----------------------------
--  Uniques structure for table mentions
-- ----------------------------
ALTER TABLE "mentions" ADD CONSTRAINT "mentions_harvest_id_unique" UNIQUE ("harvest_id") NOT DEFERRABLE INITIALLY IMMEDIATE;

-- ----------------------------
--  Indexes structure for table mentions
-- ----------------------------
CREATE INDEX  "m_contributor_id_key" ON "mentions" USING btree(contributor_id COLLATE "default" DESC NULLS LAST);
CREATE INDEX  "m_contributor_geohash_key" ON "mentions" USING btree(contributor_geohash COLLATE "default" ASC NULLS LAST);
CREATE INDEX  "m_mentioned_id_key" ON "mentions" USING btree(mentioned_id COLLATE "default" DESC NULLS LAST);
CREATE INDEX  "m_message_id_key" ON "mentions" USING btree(mentioned_id COLLATE "default" DESC NULLS LAST);
CREATE INDEX  "m_time_key" ON "mentions" USING btree("time" DESC NULLS LAST);

