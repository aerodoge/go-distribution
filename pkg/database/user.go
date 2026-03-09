package database

import (
	"database/sql"
	"go-distribution/internal/model"
)

type UserReferDAO struct {
	db *sql.DB
}

func NewUserReferDAO(db *sql.DB) *UserReferDAO {
	return &UserReferDAO{
		db: db,
	}
}

// 新增推荐关系
func (d *UserReferDAO) Create(userID, parentID int64) error {
	query := `INSERT INTO user_refer (user_id, parent_id) VALUES ($1, $2) ON CONFLICT (user_id) DO UPDATE SET parent_id = $2`
	_, err := d.db.Exec(query, userID, parentID)
	return err
}

// 根据用户ID查关系
func (d *UserReferDAO) GetByUserID(userID int64) (*model.UserRefer, error) {
	var u model.UserRefer
	err := d.db.QueryRow(`SELECT id, user_id, parent_id, created_at FROM user_refer WHERE user_id = $1`, userID).
		Scan(&u.ID, &u.ParentID, &u.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &u, nil
}

// 删除用户的推荐关系
func (d *UserReferDAO) Delete(userID int64) error {
	_, err := d.db.Exec(`DELETE FROM user_refer WHERE user_id = $1`, userID)
	return err
}

// 查直接下级
func (d *UserReferDAO) ListDirectChildren(parentID int64) ([]model.UserRefer, error) {
	rows, err := d.db.Query(`SELECT id, user_id, parent_id, created_at FROM user_refer WHERE parent_id = $1`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.UserRefer
	for rows.Next() {
		var u model.UserRefer
		if err := rows.Scan(&u.ID, &u.ParentID, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	return users, nil
}

// 查所有下级（递归全层）
func (d *UserReferDAO) ListAllChildren(parentID int64) ([]model.UserRefer, error) {
	rows, err := d.db.Query(`
WITH RECURSIVE down AS (
    SELECT user_id, parent_id, id, created_at FROM user_refer WHERE parent_id = $1
	UNION ALL
	SELECT ur.user_id, ur.parent_id, ur.id, ur.created_at FROM user_refer ur 
	JOIN down ON ur.parent_id = down.user_id
)
SELECT id, user_id, parent_id, created_at FROM down;
`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.UserRefer
	for rows.Next() {
		var u model.UserRefer
		if err := rows.Scan(&u.ID, &u.UserID, &u.ParentID, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// 查整条上级链（从自己往上到顶）
func (d *UserReferDAO) ListAllUpper(userID int64) ([]model.UserRefer, error) {
	rows, err := d.db.Query(`
WITH RECURSIVE up AS ( 
    SELECT user_id, parent_id, id, created_at FROM user_refer WHERE user_id = $1
    UNION ALL
    SELECT ur.user_id, ur.parent_id, ur.id, ur.created_at FROM user_refer ur 
    JOIN up ON up.parent_id = ur.user_id
    WHERE ur.parent_id != 0
) 
SELECT id, user_id, parent_id, created_at FROM up;`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.UserRefer
	for rows.Next() {
		var u model.UserRefer
		if err := rows.Scan(&u.ID, &u.UserID, &u.ParentID, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	return users, nil
}

// 查找所有上级的ID
func (d *UserReferDAO) ListAllUpperIDs(userID int64) ([]int64, error) {
	rows, err := d.db.Query(`
WITH RECURSIVE up AS (
    SELECT user_id, parent_id FROM user_refer WHERE user_id = $1
    UNION ALL 
    SELECT ur.user_id, ur.parent_id FROM user_refer ur 
    JOIN up ON up.parent_id = ur.user_id WHERE ur.parent_id != 0
)
SELECT parent_id FROM up;
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// 查以rootID为根的整棵树
func (d *UserReferDAO) ListTreeByRoot(rootID int64) ([]model.UserRefer, error) {
	rows, err := d.db.Query(`
WITH RECURSIVE tree AS (
    SELECT id, user_id, parent_id, created_at FROM user_refer WHERE user_id = $1
    UNION ALL
    SELECT ur.id, ur.user_id, ur.parent_id, ur.created_at FROM user_refer ur 
    JOIN tree ON ur.parent_id = tree.user_id
)
SELECT id, user_id, parent_id, created_at FROM tree;
`, rootID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.UserRefer
	for rows.Next() {
		var u model.UserRefer
		if err := rows.Scan(&u.ID, &u.UserID, &u.ParentID, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	return users, nil
}

// 将列表转成树形结构
func BuildTree(list []model.UserRefer, rootID int64) *model.UserTreeNode {
	nodeMap := make(map[int64]*model.UserTreeNode)
	// 把所有节点放进map
	for _, ut := range list {
		nodeMap[ut.UserID] = &model.UserTreeNode{
			UserID:   ut.UserID,
			ParentID: ut.ParentID,
			Children: []*model.UserTreeNode{},
		}
	}

	// 挂载子节点到父节点
	for _, node := range nodeMap {
		// 把自己加上父节点的子节点中
		if parent, ok := nodeMap[node.ParentID]; ok {
			parent.Children = append(parent.Children, node)
		}
	}

	return nodeMap[rootID]
}
