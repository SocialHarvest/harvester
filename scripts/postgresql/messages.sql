/*
 PostgreSQL
 Date: 08/09/2014 13:33:25 PM
*/

-- ----------------------------
--  Table structure for messages
-- ----------------------------
DROP TABLE IF EXISTS "messages";
CREATE TABLE "messages" (
	"time" timestamp(6) NULL,
	"harvest_id" varchar(255) NOT NULL COLLATE "default",
	"territory" varchar(255) COLLATE "default",
	"network" varchar(75) COLLATE "default",
	"contributor_id" varchar(255) COLLATE "default",
	"contributor_screen_name" varchar(255) COLLATE "default",
	"facebook_shares" int4,
	"message_id" varchar(255) COLLATE "default",
	"message" text COLLATE "default",
	"contributor_name" varchar(255) COLLATE "default",
	"contributor_gender" int2,
	"contributor_type" varchar(100) COLLATE "default",
	"contributor_longitude" float8,
	"contributor_latitude" float8,
	"contributor_geohash" varchar(100) COLLATE "default",
	"contributor_lang" varchar(8) COLLATE "default",
	"contributor_likes" int4,
	"contributor_statuses_count" int4,
	"contributor_listed_count" int4,
	"contributor_followers" int4,
	"contributor_verified" int2,
	"is_question" int2,
	"category" varchar(100) COLLATE "default",
	"twitter_retweet_count" int4,
	"twitter_favorite_count" int4,
	"like_count" int4,
	"contributor_country" varchar(6) COLLATE "default",
	"contributor_city" varchar(75) COLLATE "default",
	"contributor_state" varchar(50) COLLATE "default",
	"contributor_county" varchar(75) COLLATE "default"
)
WITH (OIDS=FALSE);

-- ----------------------------
--  Primary key structure for table messages
-- ----------------------------
ALTER TABLE "messages" ADD PRIMARY KEY ("harvest_id") NOT DEFERRABLE INITIALLY IMMEDIATE;

-- ----------------------------
--  Uniques structure for table messages
-- ----------------------------
ALTER TABLE "messages" ADD CONSTRAINT "messages_harvest_id_unique" UNIQUE ("harvest_id") NOT DEFERRABLE INITIALLY IMMEDIATE;

-- ----------------------------
--  Indexes structure for table messages
-- ----------------------------
CREATE INDEX  "msg_category_key" ON "messages" USING btree(category COLLATE "default" ASC NULLS LAST);
CREATE INDEX  "msg_contributor_id_key" ON "messages" USING btree(contributor_id COLLATE "default" DESC NULLS LAST);
CREATE INDEX  "msg_contributor_geohash_key" ON "messages" USING btree(contributor_geohash COLLATE "default" ASC NULLS LAST);
CREATE INDEX  "msg_message_id_key" ON "messages" USING btree(message_id COLLATE "default" DESC NULLS LAST);
CREATE INDEX  "msg_question_key" ON "messages" USING btree(is_question DESC NULLS LAST);
CREATE INDEX  "msg_time_key" ON "messages" USING btree("time" DESC NULLS LAST);

