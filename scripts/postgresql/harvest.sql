/*
 PostgreSQL
 Date: 08/06/2014 22:55:53 PM
*/

-- ----------------------------
--  Table structure for harvest
-- ----------------------------
DROP TABLE IF EXISTS "harvest";
CREATE TABLE "harvest" (
	"territory" varchar(150) COLLATE "default",
	"network" varchar(75) COLLATE "default",
	"action" varchar(255) COLLATE "default",
	"value" text COLLATE "default",
	"last_time_harvested" timestamp(6) NULL,
	"last_id_harvested" varchar(255) COLLATE "default",
	"items_harvested" int4,
	"harvest_time" timestamp(6) NOT NULL
)
WITH (OIDS=FALSE);

-- ----------------------------
--  Primary key structure for table harvest
-- ----------------------------
ALTER TABLE "harvest" ADD PRIMARY KEY ("harvest_time") NOT DEFERRABLE INITIALLY IMMEDIATE;

