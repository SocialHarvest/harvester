/*
 Date: 08/09/2014 13:32:56 PM
*/

SET NAMES utf8;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
--  Table structure for `messages`
-- ----------------------------
DROP TABLE IF EXISTS `messages`;
CREATE TABLE `messages` (
  `time` timestamp(6) NULL DEFAULT NULL,
  `harvest_id` varchar(255) NOT NULL,
  `territory` varchar(255) DEFAULT NULL,
  `network` varchar(75) DEFAULT NULL,
  `contributor_id` varchar(255) DEFAULT NULL,
  `contributor_screen_name` varchar(255) DEFAULT NULL,
  `lang` varchar(8) DEFAULT NULL,
  `longitude` double DEFAULT NULL,
  `latitude` double DEFAULT NULL,
  `geohash` varchar(100) DEFAULT NULL,
  `facebook_shares` int(11) DEFAULT NULL,
  `message_id` varchar(255) DEFAULT NULL,
  `message` text,
  `contributor_name` varchar(255) DEFAULT NULL,
  `contributor_gender` smallint(6) DEFAULT NULL,
  `contributor_type` varchar(100) DEFAULT NULL,
  `contributor_longitude` double DEFAULT NULL,
  `contributor_latitude` double DEFAULT NULL,
  `contributor_geohash` varchar(100) DEFAULT NULL,
  `contributor_lang` varchar(8) DEFAULT NULL,
  `contributor_likes` int(11) DEFAULT NULL,
  `contributor_statuses_count` int(11) DEFAULT NULL,
  `contributor_listed_count` int(11) DEFAULT NULL,
  `contributor_followers` int(11) DEFAULT NULL,
  `contributor_verified` smallint(6) DEFAULT NULL,
  `is_question` smallint(6) NOT NULL DEFAULT '0',
  `category` varchar(100) DEFAULT NULL,
  `twitter_retweet_count` int(11) NOT NULL DEFAULT '0',
  `twitter_favorite_count` int(11) NOT NULL DEFAULT '0',
  `like_count` int(11) NOT NULL DEFAULT '0',
  `google_plus_reshares` int(11) NOT NULL DEFAULT '0',
  `google_plus_ones` int(11) NOT NULL DEFAULT '0',
  `contributor_country` varchar(6) DEFAULT NULL,
  `contributor_city` varchar(75) DEFAULT NULL,
  `contributor_region` varchar(50) DEFAULT NULL,
  PRIMARY KEY (`harvest_id`),
  UNIQUE KEY `msg_harvest_id_unique` (`harvest_id`),
  KEY `msg_message_id_key` (`message_id`),
  KEY `msg_contributor_geohash_key` (`contributor_geohash`),
  KEY `msg_contributor_id_key` (`contributor_id`),
  KEY `msg_time_key` (`time`),
  KEY `msg_lang_key` (`lang`),
  KEY `msg_question_key` (`is_question`),
  KEY `msg_category_key` (`category`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

SET FOREIGN_KEY_CHECKS = 1;
