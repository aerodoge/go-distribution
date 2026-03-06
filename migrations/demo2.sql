CREATE TABLE user_refer(
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE,
    parent_id BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_user_refer_parent_id ON user_refer(parent_id);

-- INSERT INTO user_refer(user_id,parent_id) values(8,4);
-- INSERT INTO user_refer(user_id,parent_id) values(9,4);
-- INSERT INTO user_refer(user_id,parent_id) values(10,5);
-- INSERT INTO user_refer(user_id,parent_id) values(11,5);
-- INSERT INTO user_refer(user_id,parent_id) values(4,2);
-- INSERT INTO user_refer(user_id,parent_id) values(5,2);
-- INSERT INTO user_refer(user_id,parent_id) values(2,1);
-- INSERT INTO user_refer(user_id,parent_id) values(3,1);
-- INSERT INTO user_refer(user_id,parent_id) values(6,3);
-- INSERT INTO user_refer(user_id,parent_id) values(7,3);
-- INSERT INTO user_refer(user_id,parent_id) values(1,0);

    
--查直接下级
SELECT * FROM user_refer WHERE parent_id = $1;

--查所有下级
WITH RECURSIVE down AS (
    SELECT user_id, parent_id FROM user_refer WHERE parent_id = $1
    UNION ALL
    SELECT ur.user_id, ur.parent_id FROM user_refer ur
    JOIN down ON ur.parent_id = down.user_id
)
SELECT * FROM down;

--查上级链（从自己到根）
WITH RECURSIVE up AS (
    SELECT user_id, parent_id FROM user_refer WHERE user_id = $1
    UNION ALL
    SELECT ur.user_id, ur.parent_id FROM user_refer ur
    JOIN up ON up.parent_id = ur.user_id
    WHERE ur.parent_id != 0 --不包含顶级上面的空
)
SELECT * FROM up;

-- 查找某个节点位根的整棵树
WITH RECURSIVE tree AS (
    SELECT user_id, parent_id FROM user_refer WHERE user_id = $1
    UNION ALL
    SELECT ur.user_id, ur.parent_id FROM user_refer ur
    JOIN tree ON ur.parent_id = tree.user_id
)
SELECT * FROM tree;





