// mongodb.go
package mongodb

import (
	"context"
	"fmt"
	"time"

	"server/framework/config"
	"server/framework/logger"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	client *mongo.Client
	db     *mongo.Database
)

// InitMongoDB 初始化MongoDB连接
func InitMongoDB() error {
	// 设置连接选项
	clientOptions := options.Client().ApplyURI(
		fmt.Sprintf("mongodb://%s:%s@%s:%d",
			config.GlobalConfig.MongoDB.Username,
			config.GlobalConfig.MongoDB.Password,
			config.GlobalConfig.MongoDB.Host,
			config.GlobalConfig.MongoDB.Port,
		),
	).SetMaxPoolSize(100)

	// 连接MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("连接MongoDB失败: %v", err)
	}

	// 测试连接
	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("MongoDB连接测试失败: %v", err)
	}

	// 设置数据库
	db = client.Database(config.GlobalConfig.MongoDB.Database)

	logger.Infof("MongoDB连接成功: %s:%d", config.GlobalConfig.MongoDB.Host, config.GlobalConfig.MongoDB.Port)
	return nil
}

// GetDB 获取数据库实例
func GetDB() *mongo.Database {
	return db
}

// GetCollection 获取集合
func GetCollection(name string) *mongo.Collection {
	return db.Collection(name)
}

// InsertOne 插入单个文档
func InsertOne(ctx context.Context, collection string, document interface{}) (*mongo.InsertOneResult, error) {
	return db.Collection(collection).InsertOne(ctx, document)
}

// InsertMany 插入多个文档
func InsertMany(ctx context.Context, collection string, documents []interface{}) (*mongo.InsertManyResult, error) {
	return db.Collection(collection).InsertMany(ctx, documents)
}

// FindOne 查找单个文档
func FindOne(ctx context.Context, collection string, filter interface{}, opts ...*options.FindOneOptions) *mongo.SingleResult {
	return db.Collection(collection).FindOne(ctx, filter, opts...)
}

// Find 查找多个文档
func Find(ctx context.Context, collection string, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	return db.Collection(collection).Find(ctx, filter, opts...)
}

// UpdateOne 更新单个文档
func UpdateOne(ctx context.Context, collection string, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	return db.Collection(collection).UpdateOne(ctx, filter, update, opts...)
}

// UpdateMany 更新多个文档
func UpdateMany(ctx context.Context, collection string, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	return db.Collection(collection).UpdateMany(ctx, filter, update, opts...)
}

// DeleteOne 删除单个文档
func DeleteOne(ctx context.Context, collection string, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	return db.Collection(collection).DeleteOne(ctx, filter, opts...)
}

// DeleteMany 删除多个文档
func DeleteMany(ctx context.Context, collection string, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	return db.Collection(collection).DeleteMany(ctx, filter, opts...)
}

// CountDocuments 统计文档数量
func CountDocuments(ctx context.Context, collection string, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	return db.Collection(collection).CountDocuments(ctx, filter, opts...)
}

// Aggregate 聚合查询
func Aggregate(ctx context.Context, collection string, pipeline interface{}, opts ...*options.AggregateOptions) (*mongo.Cursor, error) {
	return db.Collection(collection).Aggregate(ctx, pipeline, opts...)
}

// CreateIndex 创建索引
func CreateIndex(ctx context.Context, collection string, keys bson.D, opts ...*options.IndexOptions) (string, error) {
	indexModel := mongo.IndexModel{
		Keys:    keys,
		Options: options.Index().SetBackground(true),
	}
	return db.Collection(collection).Indexes().CreateOne(ctx, indexModel)
}

// DropIndex 删除索引
func DropIndex(ctx context.Context, collection string, name string) error {
	_, err := db.Collection(collection).Indexes().DropOne(ctx, name)
	return err
}

// ListIndexes 列出索引
func ListIndexes(ctx context.Context, collection string, opts ...*options.ListIndexesOptions) error {
	cursor, err := db.Collection(collection).Indexes().List(ctx, opts...)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	// 打印索引信息
	for cursor.Next(ctx) {
		var index bson.Raw
		if err := cursor.Decode(&index); err != nil {
			return err
		}
		logger.Infof("Collection %s index: %v", collection, index)
	}

	if err := cursor.Err(); err != nil {
		return err
	}

	return nil
}
