/*
 Date: 08/09/2014 13:33:03 PM
*/

SET NAMES utf8;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
--  Table structure for `shared_links`
-- ----------------------------
DROP TABLE IF EXISTS `shared_links`;
CREATE TABLE `shared_links` (
  `time` timestamp(6) NULL DEFAULT NULL,
  `harvest_id` varchar(255) NOT NULL,
  `territory` varchar(255) DEFAULT NULL,
  `network` varchar(75) DEFAULT NULL,
  `contributor_id` varchar(255) DEFAULT NULL,
  `contributor_screen_name` varchar(255) DEFAULT NULL,
  `url` varchar(255) DEFAULT NULL,
  `expanded_url` varchar(255) DEFAULT NULL,
  `host` varchar(150) DEFAULT NULL,
  `message_id` varchar(255) DEFAULT NULL,
  `type` varchar(100) DEFAULT NULL,
  `preview` varchar(255) DEFAULT NULL,
  `source` varchar(255) DEFAULT NULL,
  `contributor_lang` varchar(8) DEFAULT NULL,
  `contributor_gender` smallint(6) DEFAULT NULL,
  `contributor_type` varchar(100) DEFAULT NULL,
  `contributor_longitude` double DEFAULT NULL,
  `contributor_latitude` double DEFAULT NULL,
  `contributor_geohash` varchar(100) DEFAULT NULL,
  `contributor_name` varchar(255) DEFAULT NULL,
  `contributor_country` varchar(6) DEFAULT NULL,
  `contributor_city` varchar(75) DEFAULT NULL,
  `contributor_state` varchar(50) DEFAULT NULL,
  `contributor_county` varchar(75) DEFAULT NULL,
  PRIMARY KEY (`harvest_id`),
  UNIQUE KEY `sl_harvest_id_unique` (`harvest_id`),
  KEY `sl_message_id_key` (`message_id`),
  KEY `sl_contributor_geohash_key` (`contributor_geohash`),
  KEY `sl_contributor_id_key` (`contributor_id`),
  KEY `sl_url_key` (`url`),
  KEY `sl_expanded_url_key` (`expanded_url`),
  KEY `sl_time_key` (`time`),
  KEY `sl_host_key` (`host`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

SET FOREIGN_KEY_CHECKS = 1;
