--
CREATE TABLE `user_tree`(
    `id` bigint NOT NULL AUTO_INCREMENT,
    `user_id` bigint NOT NULL COMMENT '用户ID',
    `parent_id` bigint NOT NULL DEFAULT 0 COMMENT '上级ID,0=无上级',
    'created_at' datetime DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_user_id`(`user_id`),
    KEY `idx_parent_id`(`parent_id`)
) ENGINE=InnoDB Default CHARSET=utf8mb4;

-- 查用户的直接下级
SELECT * FROM user_tree WHERE parent_id = ?;

-- 查用户的所有下级
WITH RECURSIVE cte AS (
    SELECT user_id, parent_id FROM user_tree WHERE parent_id = ?
    UNION ALL
    SELECT ut.user_id, ut.parent_id FROM user_tree ut
    JOIN cte ON ut.parent_id = cte.user_id
)
SELECT * FROM cte;

--用户的上级链（从自己到顶）
WITH RECURSIVE cte AS (
    SELECT user_id, parent_id FROM user_tree WHERE user_id = ?
    UNION ALL
    SELECT ut.user_id, ut.parent_it FROM user_tree ut
    JOIN cte ON cte.parent_id = ut.user_id
)
SELECT * FROM cte;














