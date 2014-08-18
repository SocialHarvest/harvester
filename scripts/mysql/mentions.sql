/*
 Date: 08/06/2014 23:26:40 PM
*/

SET NAMES utf8;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
--  Table structure for `mentions`
-- ----------------------------
DROP TABLE IF EXISTS `mentions`;
CREATE TABLE `mentions` (
  `time` timestamp(6) NULL DEFAULT NULL,
  `harvest_id` varchar(255) NOT NULL,
  `territory` varchar(255) DEFAULT NULL,
  `network` varchar(75) DEFAULT NULL,
  `contributor_id` varchar(255) DEFAULT NULL,
  `contributor_screen_name` varchar(255) DEFAULT NULL,
  `contributor_lang` varchar(8) DEFAULT NULL,
  `contributor_longitude` double DEFAULT NULL,
  `contributor_latitude` double DEFAULT NULL,
  `contributor_geohash` varchar(100) DEFAULT NULL,
  `mentioned_id` varchar(255) DEFAULT NULL,
  `mentioned_screen_name` varchar(255) DEFAULT NULL,
  `mentioned_type` varchar(75) DEFAULT NULL,
  `message_id` varchar(255) DEFAULT NULL,
  `contributor_type` varchar(100) DEFAULT NULL,
  `contributor_gender` smallint(6) DEFAULT NULL,
  `contributor_name` varchar(255) DEFAULT NULL,
  `mentioned_name` varchar(255) DEFAULT NULL,
  `mentioned_longitude` double DEFAULT NULL,
  `mentioned_latitude` double DEFAULT NULL,
  `mentioned_geohash` varchar(100) DEFAULT NULL,
  `mentioned_lang` varchar(8) DEFAULT NULL,
  `mentioned_gender` smallint(6) DEFAULT NULL,
  PRIMARY KEY (`harvest_id`),
  UNIQUE KEY `m_harvest_id_unique` (`harvest_id`),
  KEY `m_message_id_key` (`message_id`),
  KEY `m_contributor_id_key` (`contributor_id`),
  KEY `m_time_key` (`time`),
  KEY `m_contributor_geohash_key` (`contributor_geohash`),
  KEY `m_mentioned_id_key` (`mentioned_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

SET FOREIGN_KEY_CHECKS = 1;
