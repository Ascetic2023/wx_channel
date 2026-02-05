package database

import (
	"time"
	"wx_channel/hub_server/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB(path string) error {
	var err error
	DB, err = gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return err
	}

	// Performance Optimization: Enable WAL mode
	// This helps with concurrent reads/writes (MiningService vs Admin Dashboard)
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	// Busy timeout
	_, err = sqlDB.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		return err
	}
	_, err = sqlDB.Exec("PRAGMA busy_timeout=5000;")
	if err != nil {
		return err
	}

	// Critical for SQLite: Limit to 1 open connection to avoid "database is locked" errors
	// effectively serializing access, which is safer for SQLite.
	sqlDB.SetMaxOpenConns(1)

	// Migrate the schema
	err = DB.AutoMigrate(
		&models.User{},
		&models.Node{},
		&models.Task{},
		&models.Transaction{},
		&models.Setting{},
		&models.Subscription{},
		&models.SubscribedVideo{},
	)
	if err != nil {
		return err
	}

	return nil
}

func GetNodes() ([]models.Node, error) {
	var nodes []models.Node
	result := DB.Find(&nodes)
	return nodes, result.Error
}

func UpsertNode(node *models.Node) error {
	// First check if node exists
	var existing models.Node
	if err := DB.First(&existing, "id = ?", node.ID).Error; err != nil {
		// New node - set CreatedAt if not already set
		if node.CreatedAt.IsZero() {
			node.CreatedAt = time.Now()
		}
		return DB.Create(node).Error
	}

	// Update existing fields, but preserve created_at, UserID and BindStatus
	updates := map[string]interface{}{
		"hostname":   node.Hostname,
		"version":    node.Version,
		"ip":         node.IP,
		"status":     node.Status,
		"last_seen":  node.LastSeen,
		"updated_at": time.Now(),
	}

	// Only update UserID and BindStatus if they are set in the new node
	if node.UserID != 0 {
		updates["user_id"] = node.UserID
		updates["bind_status"] = node.BindStatus
	}

	return DB.Model(&models.Node{}).Where("id = ?", node.ID).Updates(updates).Error
}

func UpdateNodeStatus(id string, status string) error {
	return DB.Model(&models.Node{}).Where("id = ?", id).Update("status", status).Error
}

func UpdateNodeBinding(id string, userID uint) error {
	return DB.Model(&models.Node{}).Where("id = ?", id).Updates(map[string]interface{}{
		"user_id":     userID,
		"bind_status": true,
	}).Error
}

// GetNodeByID retrieves a node by its ID
func GetNodeByID(id string) (*models.Node, error) {
	var node models.Node
	err := DB.First(&node, "id = ?", id).Error
	return &node, err
}

// UnbindNode removes the binding between a node and user
func UnbindNode(id string) error {
	return DB.Model(&models.Node{}).Where("id = ?", id).Updates(map[string]interface{}{
		"user_id":     0,
		"bind_status": false,
	}).Error
}

// DeleteNode permanently deletes a node from the database
func DeleteNode(id string) error {
	return DB.Delete(&models.Node{}, "id = ?", id).Error
}

func CreateTask(task *models.Task) error {
	return DB.Create(task).Error
}

func GetTasks(userID uint, nodeID string, offset, limit int) ([]models.Task, int64, error) {
	var tasks []models.Task
	var count int64

	query := DB.Model(&models.Task{})
	if userID != 0 {
		query = query.Where("user_id = ?", userID)
	}
	if nodeID != "" {
		query = query.Where("node_id = ?", nodeID)
	}

	query.Count(&count)

	// Optimization: Select only summary headers to reduce payload size
	// Payload and Result can be very large (e.g. search results), causing slow download times
	query = query.Select("id", "type", "node_id", "user_id", "status", "error", "created_at", "updated_at")

	err := query.Order("created_at desc").Offset(offset).Limit(limit).Find(&tasks).Error
	return tasks, count, err
}

func GetTaskByID(id uint, userID uint) (*models.Task, error) {
	var task models.Task
	err := DB.Where("id = ? AND user_id = ?", id, userID).First(&task).Error
	return &task, err
}

func UpdateTaskResult(id uint, status string, result string, errorMsg string) error {
	return DB.Model(&models.Task{}).Where("id = ?", id).Updates(models.Task{
		Status: status,
		Result: result,
		Error:  errorMsg,
	}).Error
}

func CreateUser(user *models.User) error {
	return DB.Create(user).Error
}

func GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := DB.Where("email = ?", email).First(&user).Error
	return &user, err
}

func GetUserByID(id uint) (*models.User, error) {
	var user models.User
	// Preload devices
	err := DB.Preload("Devices").First(&user, id).Error
	return &user, err
}

func AddCredits(userID uint, amount int64) error {
	return DB.Model(&models.User{}).Where("id = ?", userID).Update("credits", gorm.Expr("credits + ?", amount)).Error
}

func RecordTransaction(transaction *models.Transaction) error {
	return DB.Create(transaction).Error
}

func GetActiveNodes(activeWithin time.Duration) ([]models.Node, error) {
	var nodes []models.Node
	threshold := time.Now().Add(-activeWithin)
	err := DB.Where("status = ? AND last_seen > ? AND user_id > 0", "online", threshold).Find(&nodes).Error
	return nodes, err
}

func GetAllUsers(offset, limit int) ([]models.User, int64, error) {
	var users []models.User
	var count int64
	DB.Model(&models.User{}).Count(&count)
	err := DB.Offset(offset).Limit(limit).Find(&users).Error
	return users, count, err
}

func GetSystemStats() (map[string]interface{}, error) {
	var userCount, deviceCount, txCount int64
	var totalCredits int64

	DB.Model(&models.User{}).Count(&userCount)
	DB.Model(&models.Node{}).Count(&deviceCount)
	DB.Model(&models.Transaction{}).Count(&txCount)

	// Sum total credits in circulation
	row := DB.Model(&models.User{}).Select("sum(credits)").Row()
	row.Scan(&totalCredits)

	return map[string]interface{}{
		"users":         userCount,
		"devices":       deviceCount,
		"transactions":  txCount,
		"total_credits": totalCredits,
	}, nil
}

// UpdateUserRole updates a user's role
func UpdateUserRole(userID uint, role string) error {
	return DB.Model(&models.User{}).Where("id = ?", userID).Update("role", role).Error
}

// DeleteUser permanently deletes a user and all related data
func DeleteUser(userID uint) error {
	// 开始事务
	tx := DB.Begin()

	// 删除用户的设备
	if err := tx.Where("user_id = ?", userID).Delete(&models.Node{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 删除用户的任务
	if err := tx.Where("user_id = ?", userID).Delete(&models.Task{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 删除用户的交易记录
	if err := tx.Where("user_id = ?", userID).Delete(&models.Transaction{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 删除用户的订阅
	if err := tx.Where("user_id = ?", userID).Delete(&models.Subscription{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 删除用户
	if err := tx.Delete(&models.User{}, userID).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 提交事务
	return tx.Commit().Error
}

// GetAllDevices returns all devices in the system
func GetAllDevices() ([]models.Node, error) {
	var nodes []models.Node
	err := DB.Order("last_seen desc").Find(&nodes).Error
	return nodes, err
}

// GetAllTasks returns all tasks in the system
func GetAllTasks() ([]models.Task, int64, error) {
	var tasks []models.Task
	var count int64

	DB.Model(&models.Task{}).Count(&count)

	// Select only summary headers to reduce payload size
	err := DB.Select("id", "type", "node_id", "user_id", "status", "error", "created_at", "updated_at").
		Order("created_at desc").
		Find(&tasks).Error

	return tasks, count, err
}

// DeleteTask permanently deletes a task from the database
func DeleteTask(id uint) error {
	return DB.Delete(&models.Task{}, id).Error
}

// GetAllSubscriptions returns all subscriptions in the system with video count
func GetAllSubscriptions() ([]map[string]interface{}, error) {
	var subscriptions []models.Subscription
	err := DB.Order("created_at desc").Find(&subscriptions).Error
	if err != nil {
		return nil, err
	}

	// 为每个订阅添加视频数量
	result := make([]map[string]interface{}, len(subscriptions))
	for i, sub := range subscriptions {
		var videoCount int64
		DB.Model(&models.SubscribedVideo{}).Where("subscription_id = ?", sub.ID).Count(&videoCount)

		result[i] = map[string]interface{}{
			"id":          sub.ID,
			"user_id":     sub.UserID,
			"finder_id":   sub.WxUsername,
			"nickname":    sub.WxNickname,
			"video_count": videoCount,
			"created_at":  sub.CreatedAt,
			"updated_at":  sub.UpdatedAt,
		}
	}

	return result, nil
}

// DeleteSubscription permanently deletes a subscription and its videos
func DeleteSubscription(id uint) error {
	// 开始事务
	tx := DB.Begin()

	// 删除订阅的视频
	if err := tx.Where("subscription_id = ?", id).Delete(&models.SubscribedVideo{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 删除订阅
	if err := tx.Delete(&models.Subscription{}, id).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 提交事务
	return tx.Commit().Error
}

// GetSetting retrieves a setting value by key
func GetSetting(key string) (string, error) {
	var setting models.Setting
	err := DB.Where("key = ?", key).First(&setting).Error
	if err != nil {
		return "", err
	}
	return setting.Value, nil
}

// SetSetting creates or updates a setting
func SetSetting(key, value string) error {
	return DB.Save(&models.Setting{
		Key:   key,
		Value: value,
	}).Error
}
