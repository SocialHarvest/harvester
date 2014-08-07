/*
 Date: 08/06/2014 23:08:12 PM
*/

SET NAMES utf8;
SET FOREIGN_KEY_CHECKS = 0;

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
  `harvest_time` timestamp(6) NOT NULL DEFAULT '0000-00-00 00:00:00.000000',
  PRIMARY KEY (`harvest_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

SET FOREIGN_KEY_CHECKS = 1;
