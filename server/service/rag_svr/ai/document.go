package ai

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"server/framework/id_generator"
	"server/framework/milvus"
	"server/framework/mysql"
	"server/framework/redis"

	"github.com/neurosnap/sentences"

	"gorm.io/gorm"
)

const (
	// 文档相关配置
	documentCachePrefix = "doc:"
	documentCacheTTL    = 24 * time.Hour // 文档缓存24小时
	searchCachePrefix   = "search:"
	searchCacheTTL      = 1 * time.Hour // 搜索结果缓存1小时
)

// Document 文档结构
type Document struct {
	ID        uint64          `json:"id"`
	UserID    uint64          `json:"user_id"`
	Title     string          `json:"title"`
	Content   string          `json:"content"`
	Metadata  json.RawMessage `json:"metadata"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// DocumentMetadata 文档元数据
type DocumentMetadata struct {
	Embedding []float32 `json:"embedding"`
}

// SearchResult 搜索结果
type SearchResult struct {
	Document   *mysql.Document `json:"document"`
	Score      float32         `json:"score"`
	Highlights []string        `json:"highlights"`
}

// DocumentSearchParams 文档搜索参数
type DocumentSearchParams struct {
	Query    string  `json:"query"`
	TopK     int     `json:"top_k"`
	MinScore float32 `json:"min_score"`
}

// Keyword 关键词结构体
type Keyword struct {
	Word   string  `json:"word"`
	Weight float32 `json:"weight"`
}

// DocumentService 文档服务
type DocumentService struct {
	db *gorm.DB
}

// NewDocumentService 创建文档服务实例
func NewDocumentService() *DocumentService {
	return &DocumentService{
		db: mysql.GetDB(),
	}
}

// GetKeywords 获取段落关键词
func (s *DocumentService) GetParagraphKeywords(paragraph *mysql.DocumentParagraph) ([]Keyword, error) {
	if paragraph.Keywords == "" {
		return make([]Keyword, 0), nil
	}
	var keywords []Keyword
	if err := json.Unmarshal([]byte(paragraph.Keywords), &keywords); err != nil {
		return nil, fmt.Errorf("解析关键词失败: %v", err)
	}
	return keywords, nil
}

// SetKeywords 设置段落关键词
func (s *DocumentService) SetKeywords(paragraph *mysql.DocumentParagraph, keywords []Keyword) error {
	if keywords == nil {
		paragraph.Keywords = "[]"
		return nil
	}
	data, err := json.Marshal(keywords)
	if err != nil {
		return fmt.Errorf("序列化关键词失败: %v", err)
	}
	paragraph.Keywords = string(data)
	return nil
}

// GetChunkEmbedding 获取块向量
func (s *DocumentService) GetChunkEmbedding(chunk *mysql.DocumentChunk) ([]float32, error) {
	if chunk.Embedding == nil {
		return nil, nil
	}
	vector := make([]float32, len(chunk.Embedding)/4)
	for i := range vector {
		vector[i] = float32(binary.LittleEndian.Uint32(chunk.Embedding[i*4:]))
	}
	return vector, nil
}

// SetChunkEmbedding 设置块向量
func (s *DocumentService) SetChunkEmbedding(chunk *mysql.DocumentChunk, vector []float32) error {
	if vector == nil {
		chunk.Embedding = nil
		return nil
	}
	data := make([]byte, len(vector)*4)
	for i, v := range vector {
		binary.LittleEndian.PutUint32(data[i*4:], uint32(v))
	}
	chunk.Embedding = data
	return nil
}

// AddDocument 添加文档
func (s *DocumentService) AddDocument(ctx context.Context, doc *mysql.Document, content string) error {
	// 开启事务
	tx := s.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("开启事务失败: %v", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. 创建文档
	if err := tx.Create(doc).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("创建文档失败: %v", err)
	}

	// 2. 分割段落
	paragraphs := splitIntoParagraphs(content)
	for i, paraContent := range paragraphs {
		paragraph := &mysql.DocumentParagraph{
			ParagraphID: id_generator.GetInstance().GetDocumentParagraphID(),
			DocID:       doc.DocID,
			Content:     paraContent,
			OrderNum:    uint32(i),
		}
		if err := tx.Create(paragraph).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("创建段落失败: %v", err)
		}

		// 3. 分割句子
		sentences := splitIntoSentences(paraContent)
		sentenceIDs := make([]uint64, len(sentences))

		for j, sentContent := range sentences {
			sentenceID := id_generator.GetInstance().GetDocumentSentenceID()
			sentenceIDs[j] = sentenceID

			sentence := &mysql.DocumentSentence{
				SentenceID:  sentenceID,
				DocID:       doc.DocID,
				ParagraphID: paragraph.ParagraphID,
				Content:     sentContent,
				OrderNum:    uint32(j),
			}
			if err := tx.Create(sentence).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("创建句子失败: %v", err)
			}
		}

		// 4. 创建文档块
		if len(sentences) >= 3 {
			for j := 0; j < len(sentences)-2; j += 2 {
				chunk := &mysql.DocumentChunk{
					ChunkID:     id_generator.GetInstance().GetDocumentChunkID(),
					DocID:       doc.DocID,
					ParagraphID: paragraph.ParagraphID,
					SentenceID1: sentenceIDs[j],
					SentenceID2: sentenceIDs[j+1],
					SentenceID3: sentenceIDs[j+2],
				}

				// 生成块向量
				chunkContent := sentences[j] + " " + sentences[j+1] + " " + sentences[j+2]
				embedding, err := GetEmbedding(chunkContent)
				if err != nil {
					tx.Rollback()
					return fmt.Errorf("生成块向量失败: %v", err)
				}

				// 设置块向量
				data := make([]byte, len(embedding)*4)
				for i, v := range embedding {
					binary.LittleEndian.PutUint32(data[i*4:], uint32(v))
				}
				chunk.Embedding = data

				if err := tx.Create(chunk).Error; err != nil {
					tx.Rollback()
					return fmt.Errorf("创建块失败: %v", err)
				}
			}
		}
	}

	return tx.Commit().Error
}

// IndexDocument 索引文档
func IndexDocument(ctx context.Context, doc *mysql.Document) error {
	// 1. 获取文档的所有段落
	var paragraphs []mysql.DocumentParagraph
	if err := mysql.GetDB().Where("doc_id = ?", doc.DocID).Order("order_num").Find(&paragraphs).Error; err != nil {
		return fmt.Errorf("获取文档段落失败: %v", err)
	}

	// 2. 合并所有段落内容
	var contentBuilder strings.Builder
	for _, para := range paragraphs {
		contentBuilder.WriteString(para.Content)
		contentBuilder.WriteString("\n\n")
	}
	content := contentBuilder.String()

	// 3. 生成文档向量
	embedding, err := GetEmbedding(content)
	if err != nil {
		return fmt.Errorf("生成文档向量失败: %v", err)
	}

	// 4. 更新元数据
	metadata := DocumentMetadata{
		Embedding: embedding,
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %v", err)
	}
	doc.Metadata = string(metadataJSON)

	// 5. 保存到数据库
	if err := mysql.GetDB().Save(doc).Error; err != nil {
		return fmt.Errorf("保存文档失败: %v", err)
	}

	// 6. 保存到 Milvus
	if err := milvus.InsertVector(ctx, milvus.DocumentCollectionName, int64(doc.DocID), embedding); err != nil {
		// 如果 Milvus 插入失败，回滚数据库操作
		mysql.GetDB().Save(doc)
		return fmt.Errorf("保存向量失败: %v", err)
	}

	// 7. 缓存文档
	cacheKey := fmt.Sprintf("%s%d", documentCachePrefix, doc.DocID)
	if docJSON, err := json.Marshal(doc); err == nil {
		redis.Set(ctx, cacheKey, string(docJSON), documentCacheTTL)
	}

	return nil
}

// SearchDocuments 搜索文档
func SearchDocuments(ctx context.Context, params *DocumentSearchParams) ([]*SearchResult, error) {
	// 1. 尝试从缓存获取
	cacheKey := fmt.Sprintf("%s%s:%d:%.2f", searchCachePrefix, params.Query, params.TopK, params.MinScore)
	if cached, err := redis.Get(ctx, cacheKey); err == nil {
		var results []*SearchResult
		if err := json.Unmarshal([]byte(cached), &results); err == nil {
			return results, nil
		}
	}

	// 2. 生成查询向量
	queryEmbedding, err := GetEmbedding(params.Query)
	if err != nil {
		return nil, fmt.Errorf("生成查询向量失败: %v", err)
	}

	// 3. 在 Milvus 中搜索相似向量
	ids, scores, err := milvus.SearchVector(ctx, milvus.DocumentCollectionName, queryEmbedding, params.TopK*2) // 获取更多结果用于重排序
	if err != nil {
		return nil, fmt.Errorf("搜索向量失败: %v", err)
	}

	// 4. 获取对应的文档块记录
	var chunks []*mysql.DocumentChunk
	if err := mysql.GetDB().Table("document_chunk").Where("chunk_id IN ?", ids).Find(&chunks).Error; err != nil {
		return nil, fmt.Errorf("获取文档块记录失败: %v", err)
	}

	// 5. 创建 ID 到分数的映射
	scoreMap := make(map[uint64]float32)
	for i, id := range ids {
		scoreMap[uint64(id)] = scores[i]
	}

	// 6. 构建搜索结果
	results := make([]*SearchResult, 0, len(chunks))
	for _, chunk := range chunks {
		score := scoreMap[chunk.ChunkID]
		if score < params.MinScore {
			continue
		}

		// 获取块中的三个句子
		var sentences []mysql.DocumentSentence
		if err := mysql.GetDB().Where("sentence_id IN ?", []uint64{chunk.SentenceID1, chunk.SentenceID2, chunk.SentenceID3}).
			Order(fmt.Sprintf("FIELD(sentence_id, %d, %d, %d)", chunk.SentenceID1, chunk.SentenceID2, chunk.SentenceID3)).
			Find(&sentences).Error; err != nil {
			continue
		}

		// 提取句子内容
		sentenceContents := make([]string, len(sentences))
		for i, sent := range sentences {
			sentenceContents[i] = sent.Content
		}

		// 合并句子内容用于生成高亮
		content := strings.Join(sentenceContents, " ")

		// 生成高亮文本
		highlights := generateHighlights(content, params.Query)

		// 获取文档信息
		var doc mysql.Document
		if err := mysql.GetDB().First(&doc, chunk.DocID).Error; err != nil {
			continue
		}

		results = append(results, &SearchResult{
			Document:   &doc,
			Score:      score,
			Highlights: highlights,
		})
	}

	// 7. 按分数排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// 8. 只返回前 TopK 个结果
	if len(results) > params.TopK {
		results = results[:params.TopK]
	}

	// 9. 缓存搜索结果
	if resultsJSON, err := json.Marshal(results); err == nil {
		redis.Set(ctx, cacheKey, string(resultsJSON), searchCacheTTL)
	}

	return results, nil
}

// UpdateDocument 更新文档
func UpdateDocument(ctx context.Context, doc *mysql.Document, content string) error {
	// 开启事务
	tx := mysql.GetDB().Begin()
	if tx.Error != nil {
		return fmt.Errorf("开启事务失败: %v", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. 删除旧的段落、句子和块
	if err := tx.Where("doc_id = ?", doc.DocID).Delete(&mysql.DocumentParagraph{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除旧段落失败: %v", err)
	}

	// 2. 分割段落
	paragraphs := splitIntoParagraphs(content)
	for i, paraContent := range paragraphs {
		paragraph := &mysql.DocumentParagraph{
			ParagraphID: id_generator.GetInstance().GetDocumentParagraphID(),
			DocID:       doc.DocID,
			Content:     paraContent,
			OrderNum:    uint32(i),
		}
		if err := tx.Create(paragraph).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("创建段落失败: %v", err)
		}

		// 3. 分割句子
		sentences := splitIntoSentences(paraContent)
		sentenceIDs := make([]uint64, len(sentences))

		for j, sentContent := range sentences {
			sentenceID := id_generator.GetInstance().GetDocumentSentenceID()
			sentenceIDs[j] = sentenceID

			sentence := &mysql.DocumentSentence{
				SentenceID:  sentenceID,
				DocID:       doc.DocID,
				ParagraphID: paragraph.ParagraphID,
				Content:     sentContent,
				OrderNum:    uint32(j),
			}
			if err := tx.Create(sentence).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("创建句子失败: %v", err)
			}
		}

		// 4. 创建文档块
		if len(sentences) >= 3 {
			for j := 0; j < len(sentences)-2; j += 2 {
				chunk := &mysql.DocumentChunk{
					ChunkID:     id_generator.GetInstance().GetDocumentChunkID(),
					DocID:       doc.DocID,
					ParagraphID: paragraph.ParagraphID,
					SentenceID1: sentenceIDs[j],
					SentenceID2: sentenceIDs[j+1],
					SentenceID3: sentenceIDs[j+2],
				}

				// 生成块向量
				chunkContent := sentences[j] + " " + sentences[j+1] + " " + sentences[j+2]
				embedding, err := GetEmbedding(chunkContent)
				if err != nil {
					tx.Rollback()
					return fmt.Errorf("生成块向量失败: %v", err)
				}

				// 设置块向量
				data := make([]byte, len(embedding)*4)
				for i, v := range embedding {
					binary.LittleEndian.PutUint32(data[i*4:], uint32(v))
				}
				chunk.Embedding = data

				if err := tx.Create(chunk).Error; err != nil {
					tx.Rollback()
					return fmt.Errorf("创建块失败: %v", err)
				}
			}
		}
	}

	// 5. 生成新的文档向量
	var contentBuilder strings.Builder
	for _, para := range paragraphs {
		contentBuilder.WriteString(para)
		contentBuilder.WriteString("\n\n")
	}
	content = contentBuilder.String()

	embedding, err := GetEmbedding(content)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("生成文档向量失败: %v", err)
	}

	// 6. 更新元数据
	metadata := DocumentMetadata{
		Embedding: embedding,
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("序列化元数据失败: %v", err)
	}
	doc.Metadata = string(metadataJSON)

	// 7. 更新数据库
	if err := tx.Save(doc).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("更新文档失败: %v", err)
	}

	// 8. 更新 Milvus
	if err := milvus.UpdateVector(ctx, milvus.DocumentCollectionName, int64(doc.DocID), embedding); err != nil {
		// 如果 Milvus 更新失败，回滚数据库操作
		tx.Rollback()
		return fmt.Errorf("更新向量失败: %v", err)
	}

	// 9. 更新缓存
	cacheKey := fmt.Sprintf("%s%d", documentCachePrefix, doc.DocID)
	if docJSON, err := json.Marshal(doc); err == nil {
		redis.Set(ctx, cacheKey, string(docJSON), documentCacheTTL)
	}

	// 10. 删除相关的搜索结果缓存
	pattern := fmt.Sprintf("%s*", searchCachePrefix)
	keys, err := redis.Keys(ctx, pattern)
	if err == nil {
		for _, key := range keys {
			redis.Del(ctx, key)
		}
	}

	return tx.Commit().Error
}

// DeleteDocument 删除文档
func DeleteDocument(ctx context.Context, docID uint64) error {
	// 开启事务
	tx := mysql.GetDB().Begin()
	if tx.Error != nil {
		return fmt.Errorf("开启事务失败: %v", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. 删除文档
	if err := tx.Delete(&mysql.Document{}, docID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除文档失败: %v", err)
	}

	// 2. 删除段落
	if err := tx.Where("doc_id = ?", docID).Delete(&mysql.DocumentParagraph{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除段落失败: %v", err)
	}

	// 3. 删除句子
	if err := tx.Where("doc_id = ?", docID).Delete(&mysql.DocumentSentence{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除句子失败: %v", err)
	}

	// 4. 删除文档块
	if err := tx.Where("doc_id = ?", docID).Delete(&mysql.DocumentChunk{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除文档块失败: %v", err)
	}

	// 5. 从 Milvus 删除
	if err := milvus.DeleteVector(ctx, milvus.DocumentCollectionName, int64(docID)); err != nil {
		// 如果 Milvus 删除失败，回滚数据库操作
		tx.Rollback()
		return fmt.Errorf("删除向量失败: %v", err)
	}

	// 6. 删除缓存
	cacheKey := fmt.Sprintf("%s%d", documentCachePrefix, docID)
	redis.Del(ctx, cacheKey)

	// 7. 删除相关的搜索结果缓存
	pattern := fmt.Sprintf("%s*", searchCachePrefix)
	keys, err := redis.Keys(ctx, pattern)
	if err == nil {
		for _, key := range keys {
			redis.Del(ctx, key)
		}
	}

	return tx.Commit().Error
}

// splitIntoParagraphs 将文本分割成段落
func splitIntoParagraphs(content string) []string {
	paragraphs := strings.Split(content, "\n\n")
	// 过滤空段落
	result := make([]string, 0, len(paragraphs))
	for _, p := range paragraphs {
		if p = strings.TrimSpace(p); p != "" {
			result = append(result, p)
		}
	}
	return result
}

// splitIntoSentences 将文本分割成句子
func splitIntoSentences(content string) []string {
	tokenizer := sentences.NewSentenceTokenizer(nil)
	sentences := tokenizer.Tokenize(content)
	result := make([]string, len(sentences))

	for i, sent := range sentences {
		result[i] = sent.Text
	}

	return result
}

// generateHighlights 生成高亮文本
func generateHighlights(content, query string) []string {
	// TODO: 实现高亮文本生成
	return nil
}
