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
  `likes` bigint(20) DEFAULT 0,
  `talking_about` bigint(20) DEFAULT 0,
  `were_here` bigint(20) DEFAULT 0,
  `checkins` bigint(20) DEFAULT 0,
  `views` bigint(20) DEFAULT 0,
  `subscribers` bigint(20) DEFAULT 0,
  `status_updates` bigint(20) DEFAULT 0,
  `listed` bigint(20) DEFAULT 0,
  `favorites` bigint(20) DEFAULT 0,
  `followers` bigint(20) DEFAULT 0,
  `following` bigint(20) DEFAULT 0,
  `plus_ones` bigint(20) DEFAULT 0,
  PRIMARY KEY (`harvest_id`),
  UNIQUE KEY `cg_harvest_id_unique` (`harvest_id`),
  KEY `cg_time_key` (`time`),
  KEY `cg_contributor_id_key` (`contributor_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

SET FOREIGN_KEY_CHECKS = 1;
