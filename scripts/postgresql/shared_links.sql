/*
 PostgreSQL
 Date: 08/09/2014 13:33:16 PM
*/

-- ----------------------------
--  Table structure for shared_links
-- ----------------------------
DROP TABLE IF EXISTS "shared_links";
CREATE TABLE "shared_links" (
	"time" timestamp(6) NULL,
	"harvest_id" varchar(255) NOT NULL COLLATE "default",
	"territory" varchar(255) COLLATE "default",
	"network" varchar(75) COLLATE "default",
	"contributor_id" varchar(255) COLLATE "default",
	"contributor_screen_name" varchar(255) COLLATE "default",
	"url" varchar(255) COLLATE "default",
	"expanded_url" varchar(255) COLLATE "default",
	"host" varchar(150) COLLATE "default",
	"message_id" varchar(255) COLLATE "default",
	"type" varchar(100) COLLATE "default",
	"preview" varchar(255) COLLATE "default",
	"source" varchar(255) COLLATE "default",
	"contributor_lang" varchar(8) COLLATE "default",
	"contributor_gender" int2,
	"contributor_type" varchar(100) COLLATE "default",
	"contributor_longitude" float8,
	"contributor_latitude" float8,
	"contributor_geohash" varchar(100) COLLATE "default",
	"contributor_name" varchar(255) COLLATE "default",
	"contributor_country" varchar(6) COLLATE "default",
	"contributor_city" varchar(75) COLLATE "default",
	"contributor_state" varchar(50) COLLATE "default",
	"contributor_county" varchar(75) COLLATE "default"
)
WITH (OIDS=FALSE);
ALTER TABLE "shared_links" OWNER TO "upper";

-- ----------------------------
--  Primary key structure for table shared_links
-- ----------------------------
ALTER TABLE "shared_links" ADD PRIMARY KEY ("harvest_id") NOT DEFERRABLE INITIALLY IMMEDIATE;

-- ----------------------------
--  Uniques structure for table shared_links
-- ----------------------------
ALTER TABLE "shared_links" ADD CONSTRAINT "shared_links_harvest_id_unique" UNIQUE ("harvest_id") NOT DEFERRABLE INITIALLY IMMEDIATE;

-- ----------------------------
--  Indexes structure for table shared_links
-- ----------------------------
CREATE INDEX  "sl_contributor_id_key" ON "shared_links" USING btree(contributor_id COLLATE "default" DESC NULLS LAST);
CREATE INDEX  "sl_expanded_url_key" ON "shared_links" USING btree(expanded_url COLLATE "default" DESC NULLS LAST);
CREATE INDEX  "sl_host_key" ON "shared_links" USING btree("host" COLLATE "default" ASC NULLS LAST);
CREATE INDEX  "sl_message_id_key" ON "shared_links" USING btree(message_id COLLATE "default" DESC NULLS LAST);
CREATE INDEX  "sl_time_key" ON "shared_links" USING btree("time" DESC NULLS LAST);
CREATE INDEX  "sl_url_key" ON "shared_links" USING btree(url COLLATE "default" DESC NULLS LAST);

