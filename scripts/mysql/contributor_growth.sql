/*
 Date: 08/06/2014 23:22:50 PM
*/

SET NAMES utf8;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
--  Table structure for `contributor_growth`
-- ----------------------------
DROP TABLE IF EXISTS `contributor_growth`;
CREATE TABLE `contributor_growth` (
  `time` timestamp(6) NULL DEFAULT NULL,
  `harvest_id` varchar(255) NOT NULL,
  `territory` varchar(255) DEFAULT NULL,
  `network` varchar(75) DEFAULT NULL,
  `contributor_id` varchar(255) DEFAULT NULL,
  `contributor_longitude` double DEFAULT NULL,
  `contributor_latitude` double DEFAULT NULL,
  `contributor_geohash` varchar(100) DEFAULT NULL,
  `likes` bigint(20) DEFAULT NULL,
  `talking_about_count` bigint(20) DEFAULT NULL,
  `checkins` bigint(20) DEFAULT NULL,
  `views` bigint(20) DEFAULT NULL,
  `subscribers` bigint(20) DEFAULT NULL,
  `statuses_count` bigint(20) DEFAULT NULL,
  `listed_count` bigint(20) DEFAULT NULL,
  `followers` bigint(20) DEFAULT NULL,
  `following` bigint(20) DEFAULT NULL,
  `were_here_count` bigint(20) DEFAULT NULL,
  PRIMARY KEY (`harvest_id`),
  UNIQUE KEY `cg_harvest_id_unique` (`harvest_id`),
  KEY `cg_time_key` (`time`),
  KEY `cg_contributor_id_key` (`contributor_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

SET FOREIGN_KEY_CHECKS = 1;
