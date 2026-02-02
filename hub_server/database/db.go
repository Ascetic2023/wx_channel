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
		// New node
		return DB.Create(node).Error
	}

	// Update existing fields, but preserve created_at and potentially UserID if not provided
	node.UserID = existing.UserID
	node.BindStatus = existing.BindStatus

	return DB.Save(node).Error
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
