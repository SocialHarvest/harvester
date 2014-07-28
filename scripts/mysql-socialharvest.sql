SET NAMES utf8;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
--  Table structure for `contributors`
-- ----------------------------
DROP TABLE IF EXISTS `contributors`;
CREATE TABLE `contributors` (
  `time` timestamp(6) NULL DEFAULT NULL,
  `harvest_id` varchar(255) NOT NULL,
  `territory` varchar(255) DEFAULT NULL,
  `network` varchar(75) DEFAULT NULL,
  `contributor_id` varchar(255) DEFAULT NULL,
  `contributor_screen_name` varchar(255) DEFAULT NULL,
  `iso_language_code` varchar(5) DEFAULT NULL,
  `longitude` double DEFAULT NULL,
  `latitude` double DEFAULT NULL,
  `geohash` varchar(100) DEFAULT NULL,
  `gender` int(11) DEFAULT NULL,
  `name` varchar(150) DEFAULT NULL,
  `about` text,
  `checkins` int(11) DEFAULT NULL,
  `company_overview` text,
  `description` text,
  `founded` varchar(150) DEFAULT NULL,
  `general_info` text,
  `likes` int(11) DEFAULT NULL,
  `link` varchar(256) DEFAULT NULL,
  `street` varchar(150) DEFAULT NULL,
  `city` varchar(150) DEFAULT NULL,
  `state` varchar(75) DEFAULT NULL,
  `zip` varchar(35) DEFAULT NULL,
  `country` varchar(75) DEFAULT NULL,
  `phone` varchar(35) DEFAULT NULL,
  `talking_about_count` int(11) DEFAULT NULL,
  `were_here_count` int(11) DEFAULT NULL,
  `url` varchar(255) DEFAULT NULL,
  `products` text,
  `contributor_facebook_category` varchar(150) DEFAULT NULL,
  PRIMARY KEY (`harvest_id`),
  UNIQUE KEY `contributors_harvest_id_unique` (`harvest_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- ----------------------------
--  Table structure for `harvest`
-- ----------------------------
DROP TABLE IF EXISTS `harvest`;
CREATE TABLE `harvest` (
  `territory` varchar(150) DEFAULT NULL,
  `network` varchar(75) DEFAULT NULL,
  `action` varchar(255) DEFAULT NULL,
  `value` text,
  `last_time_harvested` timestamp(6) NULL DEFAULT NULL,
  `last_id_harvested` varchar(255) DEFAULT NULL,
  `items_harvested` int(11) DEFAULT NULL,
  `harvest_time` timestamp(6) NULL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- ----------------------------
--  Table structure for `mentions`
-- ----------------------------
DROP TABLE IF EXISTS `mentions`;
CREATE TABLE `mentions` (
  `time` timestamp(6) NULL DEFAULT NULL,
  `harvest_id` varchar(255) NOT NULL,
  `territory` varchar(255) DEFAULT NULL,
  `network` varchar(75) DEFAULT NULL,
  `contributor_id` varchar(256) DEFAULT NULL,
  `contributor_screen_name` varchar(256) DEFAULT NULL,
  `iso_language_code` varchar(5) DEFAULT NULL,
  `longitude` double DEFAULT NULL,
  `latitude` double DEFAULT NULL,
  `geohash` varchar(100) DEFAULT NULL,
  `mentioned_id` varchar(255) DEFAULT NULL,
  `mentioned_screen_name` varchar(255) DEFAULT NULL,
  `mentioned_type` varchar(75) DEFAULT NULL,
  `contributor_facebook_category` varchar(150) DEFAULT NULL,
  `message_id` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`harvest_id`),
  UNIQUE KEY `mentions_harvest_id_unique` (`harvest_id`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- ----------------------------
--  Table structure for `messages`
-- ----------------------------
DROP TABLE IF EXISTS `messages`;
CREATE TABLE `messages` (
  `time` timestamp(6) NULL DEFAULT NULL,
  `harvest_id` varchar(255) NOT NULL,
  `territory` varchar(255) DEFAULT NULL,
  `network` varchar(75) DEFAULT NULL,
  `contributor_id` varchar(256) DEFAULT NULL,
  `contributor_screen_name` varchar(256) DEFAULT NULL,
  `iso_language_code` varchar(5) DEFAULT NULL,
  `longitude` double DEFAULT NULL,
  `latitude` double DEFAULT NULL,
  `geohash` varchar(100) DEFAULT NULL,
  `facebook_shares` int(11) DEFAULT NULL,
  `contributor_facebook_category` varchar(150) DEFAULT NULL,
  `message_id` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`harvest_id`),
  UNIQUE KEY `messages_harvest_id_unique` (`harvest_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- ----------------------------
--  Table structure for `questions`
-- ----------------------------
DROP TABLE IF EXISTS `questions`;
CREATE TABLE `questions` (
  `time` timestamp(6) NULL DEFAULT NULL,
  `harvest_id` varchar(255) NOT NULL,
  `territory` varchar(255) DEFAULT NULL,
  `network` varchar(75) DEFAULT NULL,
  `contributor_id` varchar(256) DEFAULT NULL,
  `contributor_screen_name` varchar(256) DEFAULT NULL,
  `iso_language_code` varchar(5) DEFAULT NULL,
  `longitude` double DEFAULT NULL,
  `latitude` double DEFAULT NULL,
  `geohash` varchar(100) DEFAULT NULL,
  `message_id` varchar(255) DEFAULT NULL,
  `message` text,
  PRIMARY KEY (`harvest_id`),
  UNIQUE KEY `questions_harvest_id_unique` (`harvest_id`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

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
  `facebook_shares` int(11) DEFAULT NULL,
  `contributor_facebook_category` varchar(150) DEFAULT NULL,
  `message_id` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`harvest_id`),
  UNIQUE KEY `shared_links_harvest_id_unique` (`harvest_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- ----------------------------
--  Table structure for `shared_media`
-- ----------------------------
DROP TABLE IF EXISTS `shared_media`;
CREATE TABLE `shared_media` (
  `time` timestamp(6) NULL DEFAULT NULL,
  `harvest_id` varchar(255) NOT NULL,
  `territory` varchar(255) DEFAULT NULL,
  `network` varchar(75) DEFAULT NULL,
  `contributor_id` varchar(255) DEFAULT NULL,
  `contributor_screen_name` varchar(255) DEFAULT NULL,
  `type` varchar(75) DEFAULT NULL,
  `preview` varchar(255) DEFAULT NULL,
  `source` varchar(255) DEFAULT NULL,
  `url` varchar(255) DEFAULT NULL,
  `expanded_url` varchar(255) DEFAULT NULL,
  `host` varchar(150) DEFAULT NULL,
  `contributor_facebook_category` varchar(150) DEFAULT NULL,
  `message_id` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`harvest_id`),
  UNIQUE KEY `shared_media_harvest_id_unique` (`harvest_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

SET FOREIGN_KEY_CHECKS = 1;
