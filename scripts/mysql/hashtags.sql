/*
 Date: 08/06/2014 23:23:41 PM
*/

SET NAMES utf8;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
--  Table structure for `hashtags`
-- ----------------------------
DROP TABLE IF EXISTS `hashtags`;
CREATE TABLE `hashtags` (
  `time` timestamp(6) NULL DEFAULT NULL,
  `harvest_id` varchar(255) NOT NULL,
  `territory` varchar(255) DEFAULT NULL,
  `network` varchar(75) DEFAULT NULL,
  `contributor_id` varchar(255) DEFAULT NULL,
  `contributor_screen_name` varchar(255) DEFAULT NULL,
  `message_id` varchar(255) DEFAULT NULL,
  `contributor_lang` varchar(8) DEFAULT NULL,
  `contributor_gender` smallint(6) DEFAULT NULL,
  `contributor_type` varchar(100) DEFAULT NULL,
  `contributor_longitude` double DEFAULT NULL,
  `contributor_latitude` double DEFAULT NULL,
  `contributor_geohash` varchar(100) DEFAULT NULL,
  `contributor_name` varchar(255) DEFAULT NULL,
  `tag` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`harvest_id`),
  UNIQUE KEY `h_harvest_id_unique` (`harvest_id`),
  KEY `h_time_key` (`time`),
  KEY `h_tag_key` (`tag`),
  KEY `h_message_id_key` (`message_id`),
  KEY `h_contributor_id_key` (`contributor_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

SET FOREIGN_KEY_CHECKS = 1;
