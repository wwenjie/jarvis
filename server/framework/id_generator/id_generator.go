package id_generator

import (
	"fmt"
	"sync"
	"time"

	"server/framework/logger"
	"server/framework/mysql"

	"gorm.io/gorm"
)

// ID生成器配置
const (
	// ID步长配置
	UserIDStep     = 100 // 用户ID步长
	SessionIDStep  = 100 // 会话ID步长
	RecordIDStep   = 100 // 记录ID步长
	MemoryIDStep   = 100 // 记忆ID步长
	DocumentIDStep = 100 // 文档ID步长
	ReminderIDStep = 100 // 提醒ID步长

	// ID名称配置
	IDNameUser        = "user_id"         // 用户ID
	IDNameChatSession = "chat_session_id" // 会话ID
	IDNameChatRecord  = "chat_record_id"  // 记录ID
	IDNameChatMemory  = "chat_memory_id"  // 记忆ID
	IDNameDocument    = "document_id"     // 文档ID
	IDNameReminder    = "reminder_id"     // 提醒ID
)

var (
	instance *IDGenerator
	once     sync.Once
)

// IDGenerator ID生成器
type IDGenerator struct {
	db *gorm.DB

	// 用户ID相关
	userIDMutex   sync.Mutex
	userIDCurrent uint64
	userIDMax     uint64

	// 会话ID相关
	sessionIDMutex   sync.Mutex
	sessionIDCurrent uint64
	sessionIDMax     uint64

	// 记录ID相关
	recordIDMutex   sync.Mutex
	recordIDCurrent uint64
	recordIDMax     uint64

	// 记忆ID相关
	memoryIDMutex   sync.Mutex
	memoryIDCurrent uint64
	memoryIDMax     uint64

	// 文档ID相关
	documentIDMutex   sync.Mutex
	documentIDCurrent uint64
	documentIDMax     uint64

	// 提醒ID相关
	reminderIDMutex   sync.Mutex
	reminderIDCurrent uint64
	reminderIDMax     uint64
}

// IDGeneratorRecord ID生成器表记录
type IDGeneratorRecord struct {
	IDName    string `gorm:"column:id_name;type:varchar(50);primaryKey"`
	Sequence  uint64 `gorm:"column:sequence;not null;default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// GetInstance 获取ID生成器实例
func GetInstance() *IDGenerator {
	once.Do(func() {
		instance = &IDGenerator{
			db: mysql.GetDB(),
		}
		if err := instance.init(); err != nil {
			logger.Fatalf("初始化ID生成器失败: %v", err)
		}
	})
	return instance
}

// init 初始化ID生成器
func (g *IDGenerator) init() error {
	// 初始化用户ID
	if err := g.refreshUserID(); err != nil {
		return err
	}

	// 初始化会话ID
	if err := g.refreshSessionID(); err != nil {
		return err
	}

	// 初始化记录ID
	if err := g.refreshRecordID(); err != nil {
		return err
	}

	// 初始化记忆ID
	if err := g.refreshMemoryID(); err != nil {
		return err
	}

	// 初始化文档ID
	if err := g.refreshDocumentID(); err != nil {
		return err
	}

	// 初始化提醒ID
	if err := g.refreshReminderID(); err != nil {
		return err
	}

	return nil
}

// refreshUserID 刷新用户ID段
func (g *IDGenerator) refreshUserID() error {
	g.userIDMutex.Lock()
	defer g.userIDMutex.Unlock()

	var record IDGeneratorRecord
	if err := g.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id_name = ?", IDNameUser).First(&record).Error; err != nil {
			return err
		}

		// 更新数据库中的当前值
		record.Sequence += UserIDStep
		if err := tx.Save(&record).Error; err != nil {
			return err
		}

		return nil
	}); err != nil {
		return fmt.Errorf("刷新用户ID段失败: %v", err)
	}

	g.userIDCurrent = record.Sequence - UserIDStep
	g.userIDMax = record.Sequence

	return nil
}

// refreshSessionID 刷新会话ID段
func (g *IDGenerator) refreshSessionID() error {
	g.sessionIDMutex.Lock()
	defer g.sessionIDMutex.Unlock()

	var record IDGeneratorRecord
	if err := g.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id_name = ?", IDNameChatSession).First(&record).Error; err != nil {
			return err
		}

		// 更新数据库中的当前值
		record.Sequence += SessionIDStep
		if err := tx.Save(&record).Error; err != nil {
			return err
		}

		return nil
	}); err != nil {
		return fmt.Errorf("刷新会话ID段失败: %v", err)
	}

	g.sessionIDCurrent = record.Sequence - SessionIDStep
	g.sessionIDMax = record.Sequence

	return nil
}

// refreshRecordID 刷新记录ID段
func (g *IDGenerator) refreshRecordID() error {
	g.recordIDMutex.Lock()
	defer g.recordIDMutex.Unlock()

	var record IDGeneratorRecord
	if err := g.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id_name = ?", IDNameChatRecord).First(&record).Error; err != nil {
			return err
		}

		// 更新数据库中的当前值
		record.Sequence += RecordIDStep
		if err := tx.Save(&record).Error; err != nil {
			return err
		}

		return nil
	}); err != nil {
		return fmt.Errorf("刷新记录ID段失败: %v", err)
	}

	g.recordIDCurrent = record.Sequence - RecordIDStep
	g.recordIDMax = record.Sequence

	return nil
}

// refreshMemoryID 刷新记忆ID段
func (g *IDGenerator) refreshMemoryID() error {
	g.memoryIDMutex.Lock()
	defer g.memoryIDMutex.Unlock()

	var record IDGeneratorRecord
	if err := g.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id_name = ?", IDNameChatMemory).First(&record).Error; err != nil {
			return err
		}

		// 更新数据库中的当前值
		record.Sequence += MemoryIDStep
		if err := tx.Save(&record).Error; err != nil {
			return err
		}

		return nil
	}); err != nil {
		return fmt.Errorf("刷新记忆ID段失败: %v", err)
	}

	g.memoryIDCurrent = record.Sequence - MemoryIDStep
	g.memoryIDMax = record.Sequence

	return nil
}

// refreshDocumentID 刷新文档ID段
func (g *IDGenerator) refreshDocumentID() error {
	g.documentIDMutex.Lock()
	defer g.documentIDMutex.Unlock()

	var record IDGeneratorRecord
	if err := g.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id_name = ?", IDNameDocument).First(&record).Error; err != nil {
			return err
		}

		// 更新数据库中的当前值
		record.Sequence += DocumentIDStep
		if err := tx.Save(&record).Error; err != nil {
			return err
		}

		return nil
	}); err != nil {
		return fmt.Errorf("刷新文档ID段失败: %v", err)
	}

	g.documentIDCurrent = record.Sequence - DocumentIDStep
	g.documentIDMax = record.Sequence

	return nil
}

// refreshReminderID 刷新提醒ID段
func (g *IDGenerator) refreshReminderID() error {
	g.reminderIDMutex.Lock()
	defer g.reminderIDMutex.Unlock()

	var record IDGeneratorRecord
	if err := g.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id_name = ?", IDNameReminder).First(&record).Error; err != nil {
			return err
		}

		// 更新数据库中的当前值
		record.Sequence += ReminderIDStep
		if err := tx.Save(&record).Error; err != nil {
			return err
		}

		return nil
	}); err != nil {
		return fmt.Errorf("刷新提醒ID段失败: %v", err)
	}

	g.reminderIDCurrent = record.Sequence - ReminderIDStep
	g.reminderIDMax = record.Sequence

	return nil
}

// GetUserID 获取新的用户ID
func (g *IDGenerator) GetUserID() uint64 {
	g.userIDMutex.Lock()
	defer g.userIDMutex.Unlock()

	if g.userIDCurrent >= g.userIDMax {
		if err := g.refreshUserID(); err != nil {
			logger.Errorf("刷新用户ID段失败: %v", err)
			return 0
		}
	}

	g.userIDCurrent++
	return g.userIDCurrent
}

// GetSessionID 获取新的会话ID
func (g *IDGenerator) GetSessionID() uint64 {
	g.sessionIDMutex.Lock()
	defer g.sessionIDMutex.Unlock()

	if g.sessionIDCurrent >= g.sessionIDMax {
		if err := g.refreshSessionID(); err != nil {
			logger.Errorf("刷新会话ID段失败: %v", err)
			return 0
		}
	}

	g.sessionIDCurrent++
	return g.sessionIDCurrent
}

// GetRecordID 获取新的记录ID
func (g *IDGenerator) GetRecordID() uint64 {
	g.recordIDMutex.Lock()
	defer g.recordIDMutex.Unlock()

	if g.recordIDCurrent >= g.recordIDMax {
		if err := g.refreshRecordID(); err != nil {
			logger.Errorf("刷新记录ID段失败: %v", err)
			return 0
		}
	}

	g.recordIDCurrent++
	return g.recordIDCurrent
}

// GetMemoryID 获取新的记忆ID
func (g *IDGenerator) GetMemoryID() uint64 {
	g.memoryIDMutex.Lock()
	defer g.memoryIDMutex.Unlock()

	if g.memoryIDCurrent >= g.memoryIDMax {
		if err := g.refreshMemoryID(); err != nil {
			logger.Errorf("刷新记忆ID段失败: %v", err)
			return 0
		}
	}

	g.memoryIDCurrent++
	return g.memoryIDCurrent
}

// GetDocumentID 获取新的文档ID
func (g *IDGenerator) GetDocumentID() uint64 {
	g.documentIDMutex.Lock()
	defer g.documentIDMutex.Unlock()

	if g.documentIDCurrent >= g.documentIDMax {
		if err := g.refreshDocumentID(); err != nil {
			logger.Errorf("刷新文档ID段失败: %v", err)
			return 0
		}
	}

	g.documentIDCurrent++
	return g.documentIDCurrent
}

// GetReminderID 获取新的提醒ID
func (g *IDGenerator) GetReminderID() uint64 {
	g.reminderIDMutex.Lock()
	defer g.reminderIDMutex.Unlock()

	if g.reminderIDCurrent >= g.reminderIDMax {
		if err := g.refreshReminderID(); err != nil {
			logger.Errorf("刷新提醒ID段失败: %v", err)
			return 0
		}
	}

	g.reminderIDCurrent++
	return g.reminderIDCurrent
}
