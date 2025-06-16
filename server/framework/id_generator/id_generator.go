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
	UserIDStep              = 100 // 用户ID步长
	SessionIDStep           = 100 // 会话ID步长
	RecordIDStep            = 100 // 记录ID步长
	MemoryIDStep            = 100 // 记忆ID步长
	ReminderIDStep          = 100 // 提醒ID步长
	DocumentIDStep          = 100 // 文档ID步长
	DocumentParagraphIDStep = 100 // 文档段落ID步长
	DocumentSentenceIDStep  = 100 // 文档句子ID步长
	DocumentChunkIDStep     = 100 // 文档块ID步长

	// ID名称配置
	IDNameUser              = "user_id"               // 用户ID
	IDNameChatSession       = "chat_session_id"       // 会话ID
	IDNameChatRecord        = "chat_record_id"        // 记录ID
	IDNameChatMemory        = "chat_memory_id"        // 记忆ID
	IDNameReminder          = "reminder_id"           // 提醒ID
	IDNameDocument          = "document_id"           // 文档ID
	IDNameDocumentParagraph = "document_paragraph_id" // 文档段落ID
	IDNameDocumentSentence  = "document_sentence_id"  // 文档句子ID
	IDNameDocumentChunk     = "document_chunk_id"     // 文档块ID
)

// IDGenerator ID生成器表记录
type IDGenerator struct {
	IDName    string `gorm:"column:id_name;type:varchar(50);primaryKey;table:id_generator"`
	Sequence  uint64 `gorm:"column:sequence;not null;default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName 指定表名
func (IDGenerator) TableName() string {
	return "id_generator"
}

// IDGeneratorManager ID生成器管理器
type IDGeneratorManager struct {
	db *gorm.DB

	// 用户ID相关
	userIDMutex   sync.Mutex
	userIDCurrent uint64
	userIDMax     uint64

	// 会话ID相关
	sessionIDMutex   sync.Mutex
	sessionIDCurrent uint64
	sessionIDMax     uint64

	// 聊天记录ID相关
	chatRecordIDMutex   sync.Mutex
	chatRecordIDCurrent uint64
	chatRecordIDMax     uint64

	// 记忆ID相关
	memoryIDMutex   sync.Mutex
	memoryIDCurrent uint64
	memoryIDMax     uint64

	// 提醒ID相关
	reminderIDMutex   sync.Mutex
	reminderIDCurrent uint64
	reminderIDMax     uint64

	// 文档ID相关
	documentIDMutex   sync.Mutex
	documentIDCurrent uint64
	documentIDMax     uint64

	// 文档段落ID相关
	documentParagraphIDMutex   sync.Mutex
	documentParagraphIDCurrent uint64
	documentParagraphIDMax     uint64

	// 文档句子ID相关
	documentSentenceIDMutex   sync.Mutex
	documentSentenceIDCurrent uint64
	documentSentenceIDMax     uint64

	// 文档块ID相关
	documentChunkIDMutex   sync.Mutex
	documentChunkIDCurrent uint64
	documentChunkIDMax     uint64
}

var (
	instance *IDGeneratorManager
	once     sync.Once
)

// GetInstance 获取ID生成器实例
func GetInstance() *IDGeneratorManager {
	once.Do(func() {
		instance = &IDGeneratorManager{
			db: mysql.GetDB(),
		}
		if err := instance.init(); err != nil {
			logger.Fatalf("初始化ID生成器失败: %v", err)
		}
	})
	return instance
}

// init 初始化ID生成器
func (g *IDGeneratorManager) init() error {
	// 初始化用户ID
	if err := g.refreshUserID(); err != nil {
		return err
	}

	// 初始化会话ID
	if err := g.refreshSessionID(); err != nil {
		return err
	}

	// 初始化记录ID
	if err := g.refreshChatRecordID(); err != nil {
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

	// 初始化文档块ID
	if err := g.refreshDocumentChunkID(); err != nil {
		return err
	}

	return nil
}

// refreshUserID 刷新用户ID段
func (g *IDGeneratorManager) refreshUserID() error {
	g.userIDMutex.Lock()
	defer g.userIDMutex.Unlock()

	var record IDGenerator
	if err := g.db.Transaction(func(tx *gorm.DB) error {
		// 尝试获取记录，如果不存在则创建
		result := tx.Where("id_name = ?", IDNameUser).First(&record)
		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				// 记录不存在，创建新记录
				record = IDGenerator{
					IDName:   IDNameUser,
					Sequence: UserIDStep,
				}
				if err := tx.Create(&record).Error; err != nil {
					return err
				}
			} else {
				return result.Error
			}
		} else {
			// 更新数据库中的当前值
			record.Sequence += UserIDStep
			if err := tx.Save(&record).Error; err != nil {
				return err
			}
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
func (g *IDGeneratorManager) refreshSessionID() error {
	g.sessionIDMutex.Lock()
	defer g.sessionIDMutex.Unlock()

	var record IDGenerator
	if err := g.db.Transaction(func(tx *gorm.DB) error {
		// 尝试获取记录，如果不存在则创建
		result := tx.Where("id_name = ?", IDNameChatSession).First(&record)
		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				// 记录不存在，创建新记录
				record = IDGenerator{
					IDName:   IDNameChatSession,
					Sequence: SessionIDStep,
				}
				if err := tx.Create(&record).Error; err != nil {
					return err
				}
			} else {
				return result.Error
			}
		} else {
			// 更新数据库中的当前值
			record.Sequence += SessionIDStep
			if err := tx.Save(&record).Error; err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("刷新会话ID段失败: %v", err)
	}

	g.sessionIDCurrent = record.Sequence - SessionIDStep
	g.sessionIDMax = record.Sequence

	return nil
}

// refreshChatRecordID 刷新聊天记录ID段
func (g *IDGeneratorManager) refreshChatRecordID() error {
	g.chatRecordIDMutex.Lock()
	defer g.chatRecordIDMutex.Unlock()

	var record IDGenerator
	if err := g.db.Transaction(func(tx *gorm.DB) error {
		// 尝试获取记录，如果不存在则创建
		result := tx.Where("id_name = ?", IDNameChatRecord).First(&record)
		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				// 记录不存在，创建新记录
				record = IDGenerator{
					IDName:   IDNameChatRecord,
					Sequence: RecordIDStep,
				}
				if err := tx.Create(&record).Error; err != nil {
					return err
				}
			} else {
				return result.Error
			}
		} else {
			// 更新数据库中的当前值
			record.Sequence += RecordIDStep
			if err := tx.Save(&record).Error; err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("刷新记录ID段失败: %v", err)
	}

	g.chatRecordIDCurrent = record.Sequence - RecordIDStep
	g.chatRecordIDMax = record.Sequence

	return nil
}

// refreshMemoryID 刷新记忆ID段
func (g *IDGeneratorManager) refreshMemoryID() error {
	g.memoryIDMutex.Lock()
	defer g.memoryIDMutex.Unlock()

	var record IDGenerator
	if err := g.db.Transaction(func(tx *gorm.DB) error {
		// 尝试获取记录，如果不存在则创建
		result := tx.Where("id_name = ?", IDNameChatMemory).First(&record)
		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				// 记录不存在，创建新记录
				record = IDGenerator{
					IDName:   IDNameChatMemory,
					Sequence: MemoryIDStep,
				}
				if err := tx.Create(&record).Error; err != nil {
					return err
				}
			} else {
				return result.Error
			}
		} else {
			// 更新数据库中的当前值
			record.Sequence += MemoryIDStep
			if err := tx.Save(&record).Error; err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("刷新记忆ID段失败: %v", err)
	}

	g.memoryIDCurrent = record.Sequence - MemoryIDStep
	g.memoryIDMax = record.Sequence

	return nil
}

// refreshReminderID 刷新提醒ID段
func (g *IDGeneratorManager) refreshReminderID() error {
	g.reminderIDMutex.Lock()
	defer g.reminderIDMutex.Unlock()

	var record IDGenerator
	if err := g.db.Transaction(func(tx *gorm.DB) error {
		// 尝试获取记录，如果不存在则创建
		result := tx.Where("id_name = ?", IDNameReminder).First(&record)
		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				// 记录不存在，创建新记录
				record = IDGenerator{
					IDName:   IDNameReminder,
					Sequence: ReminderIDStep,
				}
				if err := tx.Create(&record).Error; err != nil {
					return err
				}
			} else {
				return result.Error
			}
		} else {
			// 更新数据库中的当前值
			record.Sequence += ReminderIDStep
			if err := tx.Save(&record).Error; err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("刷新提醒ID段失败: %v", err)
	}

	g.reminderIDCurrent = record.Sequence - ReminderIDStep
	g.reminderIDMax = record.Sequence

	return nil
}

// refreshDocumentID 刷新文档ID段
func (g *IDGeneratorManager) refreshDocumentID() error {
	g.documentIDMutex.Lock()
	defer g.documentIDMutex.Unlock()

	var record IDGenerator
	if err := g.db.Transaction(func(tx *gorm.DB) error {
		// 尝试获取记录，如果不存在则创建
		result := tx.Where("id_name = ?", IDNameDocument).First(&record)
		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				// 记录不存在，创建新记录
				record = IDGenerator{
					IDName:   IDNameDocument,
					Sequence: DocumentIDStep,
				}
				if err := tx.Create(&record).Error; err != nil {
					return err
				}
			} else {
				return result.Error
			}
		} else {
			// 更新数据库中的当前值
			record.Sequence += DocumentIDStep
			if err := tx.Save(&record).Error; err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("刷新文档ID段失败: %v", err)
	}

	g.documentIDCurrent = record.Sequence - DocumentIDStep
	g.documentIDMax = record.Sequence

	return nil
}

func (g *IDGeneratorManager) refreshDocumentParagraphID() error {
	g.documentParagraphIDMutex.Lock()
	defer g.documentParagraphIDMutex.Unlock()

	var record IDGenerator
	if err := g.db.Transaction(func(tx *gorm.DB) error {
		// 尝试获取记录，如果不存在则创建
		result := tx.Where("id_name = ?", IDNameDocumentParagraph).First(&record)
		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				// 记录不存在，创建新记录
				record = IDGenerator{
					IDName:   IDNameDocumentParagraph,
					Sequence: DocumentParagraphIDStep,
				}
				if err := tx.Create(&record).Error; err != nil {
					return err
				}
			} else {
				return result.Error
			}
		} else {
			// 更新数据库中的当前值
			record.Sequence += DocumentParagraphIDStep
			if err := tx.Save(&record).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("刷新文档段落ID段失败: %v", err)
	}

	g.documentParagraphIDCurrent = record.Sequence - DocumentParagraphIDStep
	g.documentParagraphIDMax = record.Sequence

	return nil
}

func (g *IDGeneratorManager) refreshDocumentSentenceID() error {
	g.documentSentenceIDMutex.Lock()
	defer g.documentSentenceIDMutex.Unlock()

	var record IDGenerator
	if err := g.db.Transaction(func(tx *gorm.DB) error {
		// 尝试获取记录，如果不存在则创建
		result := tx.Where("id_name = ?", IDNameDocumentSentence).First(&record)
		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				// 记录不存在，创建新记录
				record = IDGenerator{
					IDName:   IDNameDocumentSentence,
					Sequence: DocumentSentenceIDStep,
				}
				if err := tx.Create(&record).Error; err != nil {
					return err
				}
			} else {
				return result.Error
			}
		} else {
			// 更新数据库中的当前值
			record.Sequence += DocumentSentenceIDStep
			if err := tx.Save(&record).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("刷新文档句子ID段失败: %v", err)
	}
	g.documentSentenceIDCurrent = record.Sequence - DocumentSentenceIDStep
	g.documentSentenceIDMax = record.Sequence

	return nil
}

func (g *IDGeneratorManager) refreshDocumentChunkID() error {
	g.documentChunkIDMutex.Lock()
	defer g.documentChunkIDMutex.Unlock()

	var record IDGenerator
	if err := g.db.Transaction(func(tx *gorm.DB) error {
		// 尝试获取记录，如果不存在则创建
		result := tx.Where("id_name = ?", IDNameDocumentChunk).First(&record)
		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				// 记录不存在，创建新记录
				record = IDGenerator{
					IDName:   IDNameDocumentChunk,
					Sequence: DocumentChunkIDStep,
				}
				if err := tx.Create(&record).Error; err != nil {
					return err
				}
			} else {
				return result.Error
			}
		} else {
			// 更新数据库中的当前值
			record.Sequence += DocumentChunkIDStep
			if err := tx.Save(&record).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("刷新文档块ID段失败: %v", err)
	}

	g.documentChunkIDCurrent = record.Sequence - DocumentChunkIDStep
	g.documentChunkIDMax = record.Sequence

	return nil
}

// GetUserID 获取新的用户ID
func (g *IDGeneratorManager) GetUserID() uint64 {
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
func (g *IDGeneratorManager) GetSessionID() uint64 {
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

// GetChatRecordID 获取新的聊天记录ID
func (g *IDGeneratorManager) GetChatRecordID() uint64 {
	g.chatRecordIDMutex.Lock()
	defer g.chatRecordIDMutex.Unlock()

	if g.chatRecordIDCurrent >= g.chatRecordIDMax {
		if err := g.refreshChatRecordID(); err != nil {
			logger.Errorf("刷新记录ID段失败: %v", err)
			return 0
		}
	}

	g.chatRecordIDCurrent++
	return g.chatRecordIDCurrent
}

// GetMemoryID 获取新的记忆ID
func (g *IDGeneratorManager) GetMemoryID() uint64 {
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

// GetReminderID 获取新的提醒ID
func (g *IDGeneratorManager) GetReminderID() uint64 {
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

// GetDocumentID 获取新的文档ID
func (g *IDGeneratorManager) GetDocumentID() uint64 {
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

func (g *IDGeneratorManager) GetDocumentParagraphID() uint64 {
	g.documentParagraphIDMutex.Lock()
	defer g.documentParagraphIDMutex.Unlock()

	if g.documentParagraphIDCurrent >= g.documentParagraphIDMax {
		if err := g.refreshDocumentParagraphID(); err != nil {
			logger.Errorf("刷新文档段落ID段失败: %v", err)
			return 0
		}
	}

	g.documentParagraphIDCurrent++
	return g.documentParagraphIDCurrent
}

func (g *IDGeneratorManager) GetDocumentSentenceID() uint64 {
	g.documentSentenceIDMutex.Lock()
	defer g.documentSentenceIDMutex.Unlock()

	if g.documentSentenceIDCurrent >= g.documentSentenceIDMax {
		if err := g.refreshDocumentSentenceID(); err != nil {
			logger.Errorf("刷新文档句子ID段失败: %v", err)
			return 0
		}
	}

	g.documentSentenceIDCurrent++
	return g.documentSentenceIDCurrent
}

// GetDocumentChunkID 获取文档块ID
func (m *IDGeneratorManager) GetDocumentChunkID() uint64 {
	m.documentChunkIDMutex.Lock()
	defer m.documentChunkIDMutex.Unlock()

	if m.documentChunkIDCurrent >= m.documentChunkIDMax {
		if err := m.refreshDocumentChunkID(); err != nil {
			logger.Errorf("刷新文档块ID段失败: %v", err)
			return 0
		}
	}

	m.documentChunkIDCurrent++
	return m.documentChunkIDCurrent
}
