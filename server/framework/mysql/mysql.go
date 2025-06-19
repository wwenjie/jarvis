// mysql.go
package mysql

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"server/framework/config"
	"server/framework/logger"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var (
	db *gorm.DB
)

// InitMySQL 初始化MySQL连接
func InitMySQL() error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.GlobalConfig.MySQL.Username,
		config.GlobalConfig.MySQL.Password,
		config.GlobalConfig.MySQL.Host,
		config.GlobalConfig.MySQL.Port,
		config.GlobalConfig.MySQL.Database,
	)

	var err error
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		return fmt.Errorf("连接MySQL失败: %v", err)
	}

	// 设置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取数据库实例失败: %v", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 不再自动迁移表结构
	// if err := autoMigrate(); err != nil {
	// 	return fmt.Errorf("自动迁移表结构失败: %v", err)
	// }
	// 只检查索引
	if err := checkIndexes(); err != nil {
		return fmt.Errorf("检查索引失败: %v", err)
	}

	logger.Infof("MySQL连接成功: %s:%d", config.GlobalConfig.MySQL.Host, config.GlobalConfig.MySQL.Port)
	return nil
}

// GetDB 获取数据库实例
func GetDB() *gorm.DB {
	return db
}

// autoMigrate 自动迁移表结构
// func autoMigrate() error {
// 	// 创建表
// 	if err := db.AutoMigrate(
// 		&IDGenerator{},
// 		&User{},
// 		&ChatSession{},
// 		&ChatRecord{},
// 		&ChatMemory{},
// 		&Reminder{},
// 		&Document{},
// 		&DocumentParagraph{},
// 		&DocumentSentence{},
// 		&DocumentChunk{},
// 	); err != nil {
// 		return err
// 	}

// 	// 创建索引
// 	indexes := []struct {
// 		table   string
// 		columns []string
// 	}{
// 		{"id_generator", []string{"id_name"}},
// 		{"chat_session", []string{"user_id", "status"}},
// 		{"chat_session", []string{"last_active_time"}},
// 		{"chat_record", []string{"session_id", "created_at"}},
// 		{"chat_record", []string{"user_id", "created_at"}},
// 		{"chat_memory", []string{"user_id", "memory_type"}},
// 		{"chat_memory", []string{"expire_time"}},
// 		{"chat_memory", []string{"access_count"}},
// 		{"document", []string{"user_id", "status"}},
// 		{"document_paragraph", []string{"doc_id"}},
// 		{"document_sentence", []string{"paragraph_id"}},
// 		{"document_chunk", []string{"paragraph_id"}},
// 	}

// 	for _, idx := range indexes {
// 		indexName := fmt.Sprintf("idx_%s_%s", idx.table, strings.Join(idx.columns, "_"))

// 		// 检查索引是否存在
// 		var count int64
// 		checkSQL := fmt.Sprintf(`
// 			SELECT COUNT(1)
// 			FROM information_schema.statistics
// 			WHERE table_schema = DATABASE()
// 			AND table_name = '%s'
// 			AND index_name = '%s'`,
// 			idx.table, indexName)

// 		if err := db.Raw(checkSQL).Count(&count).Error; err != nil {
// 			return fmt.Errorf("检查索引 %s 是否存在失败: %v", indexName, err)
// 		}

// 		// 如果索引不存在，则创建
// 		if count == 0 {
// 			createSQL := fmt.Sprintf("CREATE INDEX %s ON %s (%s)",
// 				indexName, idx.table, strings.Join(idx.columns, ", "))
// 			if err := db.Exec(createSQL).Error; err != nil {
// 				return fmt.Errorf("创建索引 %s 失败: %v", indexName, err)
// 			}
// 		}
// 	}

// 	return nil
// }

// checkIndexes 只检查索引
func checkIndexes() error {
	// 创建索引
	indexes := []struct {
		table   string
		columns []string
	}{
		{"id_generator", []string{"id_name"}},
		{"chat_session", []string{"user_id", "status"}},
		{"chat_session", []string{"last_active_time"}},
		{"chat_record", []string{"session_id", "created_at"}},
		{"chat_record", []string{"user_id", "created_at"}},
		{"chat_memory", []string{"user_id", "memory_type"}},
		{"chat_memory", []string{"expire_time"}},
		{"chat_memory", []string{"access_count"}},
		{"document", []string{"user_id", "status"}},
		{"document_paragraph", []string{"doc_id"}},
		{"document_sentence", []string{"paragraph_id"}},
		{"document_chunk", []string{"paragraph_id"}},
	}

	for _, idx := range indexes {
		indexName := fmt.Sprintf("idx_%s_%s", idx.table, strings.Join(idx.columns, "_"))

		// 检查索引是否存在
		var count int64
		checkSQL := fmt.Sprintf(`
			SELECT COUNT(1) 
			FROM information_schema.statistics 
			WHERE table_schema = DATABASE() 
			AND table_name = '%s' 
			AND index_name = '%s'`,
			idx.table, indexName)

		if err := db.Raw(checkSQL).Count(&count).Error; err != nil {
			return fmt.Errorf("检查索引 %s 是否存在失败: %v", indexName, err)
		}

		// 如果索引不存在，则创建
		if count == 0 {
			createSQL := fmt.Sprintf("CREATE INDEX %s ON %s (%s)",
				indexName, idx.table, strings.Join(idx.columns, ", "))
			if err := db.Exec(createSQL).Error; err != nil {
				return fmt.Errorf("创建索引 %s 失败: %v", indexName, err)
			}
		}
	}

	return nil
}

// IDGenerator ID生成器表记录
type IDGenerator struct {
	IDName    string `gorm:"column:id_name;type:varchar(50);primaryKey"`
	Sequence  uint64 `gorm:"column:sequence;not null;default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (IDGenerator) TableName() string {
	return "id_generator"
}

// User 用户表
type User struct {
	ID        uint64 `gorm:"primaryKey"`
	Username  string `gorm:"size:50;not null;unique"`
	Email     string `gorm:"size:100;not null;unique"`
	Password  string `gorm:"size:100;not null"`
	Status    string `gorm:"size:20;not null;default:'active'"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (User) TableName() string {
	return "user"
}

// Session 会话表
type ChatSession struct {
	ID             uint64    `gorm:"primaryKey"`
	UserID         uint64    `gorm:"not null"`
	Status         string    `gorm:"size:20;not null;default:'active'"`
	Title          string    `gorm:"size:200"`
	Summary        string    `gorm:"type:text"`
	UserState      string    `gorm:"type:json"` // 用户状态，JSON格式，包含 last_intent, intent_confidence, last_sentiment 等字段
	SystemState    string    `gorm:"type:json"` // 系统状态，JSON格式
	Metadata       string    `gorm:"type:json"` // 元数据，JSON格式
	LastActiveTime time.Time `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (ChatSession) TableName() string {
	return "chat_session"
}

// ChatRecord 对话记录表
type ChatRecord struct {
	ID            uint64 `gorm:"primaryKey"`
	SessionID     uint64 `gorm:"not null"`
	UserID        uint64 `gorm:"not null"`
	Message       string `gorm:"type:text;not null"`
	Response      string `gorm:"type:text"`
	MessageType   string `gorm:"size:20;not null;default:'text'"`
	Status        string `gorm:"size:20;not null;default:'pending'"`
	Context       string `gorm:"type:json"`
	FunctionCalls string `gorm:"type:json"`
	Metadata      string `gorm:"type:json"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (ChatRecord) TableName() string {
	return "chat_record"
}

// Document 文档表
type Document struct {
	DocID          uint64 `gorm:"column:doc_id;primaryKey"`
	UserID         uint64 `gorm:"column:user_id;not null"`
	Title          string `gorm:"column:title;size:200;not null"`
	Status         string `gorm:"column:status;size:20;not null;default:'active'"`
	Metadata       string `gorm:"column:metadata;type:json"`
	ParagraphCount uint32 `gorm:"column:paragraph_count;not null;default:0"`
	SentenceCount  uint32 `gorm:"column:sentence_count;not null;default:0"`
	Keywords       string `gorm:"column:keywords;type:json"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (Document) TableName() string {
	return "document"
}

// ChatMemory 对话记忆表
type ChatMemory struct {
	ID          uint64  `gorm:"primaryKey"`
	SessionID   uint64  `gorm:"not null"`
	UserID      uint64  `gorm:"not null"`
	Content     string  `gorm:"type:text;not null"`
	MemoryType  string  `gorm:"size:20;not null"`
	Importance  float32 `gorm:"not null;default:1.0"`
	ExpireTime  time.Time
	AccessCount int    `gorm:"not null;default:0"`
	Metadata    string `gorm:"type:json"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (ChatMemory) TableName() string {
	return "chat_memory"
}

// Reminder 提醒记录
type Reminder struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement"`
	Content    string    `gorm:"type:text;not null"`        // 提醒内容
	RemindTime time.Time `gorm:"not null"`                  // 提醒时间
	Status     string    `gorm:"type:varchar(20);not null"` // 状态：pending/triggered/completed
	CreateTime time.Time `gorm:"not null"`                  // 创建时间
	UpdateTime time.Time `gorm:"not null"`                  // 更新时间
}

func (Reminder) TableName() string {
	return "reminder"
}

// GetUserState 获取用户状态
func (s *ChatSession) GetUserState() (map[string]string, error) {
	if s.UserState == "" {
		return make(map[string]string), nil
	}
	var state map[string]string
	if err := json.Unmarshal([]byte(s.UserState), &state); err != nil {
		return nil, fmt.Errorf("解析用户状态失败: %v", err)
	}
	return state, nil
}

// SetUserState 设置用户状态
func (s *ChatSession) SetUserState(state map[string]string) error {
	if state == nil {
		s.UserState = "{}"
		return nil
	}
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("序列化用户状态失败: %v", err)
	}
	s.UserState = string(data)
	return nil
}

// GetSystemState 获取系统状态
func (s *ChatSession) GetSystemState() (map[string]string, error) {
	if s.SystemState == "" {
		return make(map[string]string), nil
	}
	var state map[string]string
	if err := json.Unmarshal([]byte(s.SystemState), &state); err != nil {
		return nil, fmt.Errorf("解析系统状态失败: %v", err)
	}
	return state, nil
}

// SetSystemState 设置系统状态
func (s *ChatSession) SetSystemState(state map[string]string) error {
	if state == nil {
		s.SystemState = "{}"
		return nil
	}
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("序列化系统状态失败: %v", err)
	}
	s.SystemState = string(data)
	return nil
}

// GetMetadata 获取元数据
func (s *ChatSession) GetMetadata() (map[string]interface{}, error) {
	if s.Metadata == "" {
		return make(map[string]interface{}), nil
	}
	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(s.Metadata), &metadata); err != nil {
		return nil, fmt.Errorf("解析元数据失败: %v", err)
	}
	return metadata, nil
}

// SetMetadata 设置元数据
func (s *ChatSession) SetMetadata(metadata map[string]interface{}) error {
	if metadata == nil {
		s.Metadata = "{}"
		return nil
	}
	data, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %v", err)
	}
	s.Metadata = string(data)
	return nil
}

// DocumentParagraph 文档段落表
type DocumentParagraph struct {
	ParagraphID   uint64 `gorm:"column:paragraph_id;primaryKey"`
	DocID         uint64 `gorm:"column:doc_id;primaryKey"`
	Content       string `gorm:"column:content;type:text;not null"`
	SentenceIDMin uint64 `gorm:"column:sentence_id_min;not null"`
	SentenceIDMax uint64 `gorm:"column:sentence_id_max;not null"`
	Keywords      string `gorm:"column:keywords;type:json"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (DocumentParagraph) TableName() string {
	return "document_paragraph"
}

// DocumentSentence 文档句子表
type DocumentSentence struct {
	DocID       uint64 `gorm:"column:doc_id;primaryKey"`
	SentenceID  uint64 `gorm:"column:sentence_id;primaryKey"`
	ParagraphID uint64 `gorm:"column:paragraph_id;not null"`
	Content     string `gorm:"column:content;type:text;not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (DocumentSentence) TableName() string {
	return "document_sentence"
}

// DocumentChunk 文档块表
type DocumentChunk struct {
	ChunkID       uint64 `gorm:"column:chunk_id;primaryKey"`
	DocID         uint64 `gorm:"column:doc_id;not null"`
	ParagraphID   uint64 `gorm:"column:paragraph_id;not null"`
	SentenceIDMin uint64 `gorm:"column:sentence_id_min;not null"`
	SentenceIDMax uint64 `gorm:"column:sentence_id_max;not null"`
	Keywords      string `gorm:"column:keywords;type:json"`
	Embedding     []byte `gorm:"column:embedding;type:blob"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (DocumentChunk) TableName() string {
	return "document_chunk"
}
