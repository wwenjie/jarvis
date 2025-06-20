-- 创建数据库
CREATE DATABASE IF NOT EXISTS jarvis_db DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE jarvis_db;

-- 用户表
CREATE TABLE IF NOT EXISTS `user` (
    `id` bigint unsigned NOT NULL COMMENT '用户ID',
    `username` varchar(50) NOT NULL COMMENT '用户名',
    `email` varchar(100) NOT NULL COMMENT '邮箱',
    `password` varchar(100) NOT NULL COMMENT '密码',
    `status` varchar(20) NOT NULL DEFAULT 'active' COMMENT '用户状态(active/disabled)',
    `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_username` (`username`),
    UNIQUE KEY `uk_email` (`email`),
    KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表';

-- 会话表
CREATE TABLE IF NOT EXISTS `chat_session` (
    `id` bigint unsigned NOT NULL COMMENT '会话ID',
    `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
    `status` varchar(20) NOT NULL DEFAULT 'active' COMMENT '会话状态(active/ended/archived)',
    `title` varchar(200) DEFAULT NULL COMMENT '会话标题',
    `summary` text DEFAULT NULL COMMENT '会话摘要',
    `user_state` json DEFAULT NULL COMMENT '用户状态',
    `system_state` json DEFAULT NULL COMMENT '系统状态',
    `metadata` json DEFAULT NULL COMMENT '元数据',
    `last_active_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最后活跃时间',
    `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_user_status` (`user_id`, `status`),
    KEY `idx_last_active_time` (`last_active_time`),
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='会话表';

-- 对话记录表
CREATE TABLE IF NOT EXISTS `chat_record` (
    `id` bigint unsigned NOT NULL COMMENT '记录ID',
    `session_id` bigint unsigned NOT NULL COMMENT '会话ID',
    `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
    `message` text NOT NULL COMMENT '用户消息',
    `response` text DEFAULT NULL COMMENT '系统回复',
    `message_type` varchar(20) NOT NULL DEFAULT 'text' COMMENT '消息类型',
    `status` varchar(20) NOT NULL DEFAULT 'pending' COMMENT '状态(pending/completed/failed)',
    `context` json DEFAULT NULL COMMENT '上下文信息',
    `function_calls` json DEFAULT NULL COMMENT '函数调用信息',
    `metadata` json DEFAULT NULL COMMENT '元数据',
    `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_session_created` (`session_id`, `created_at`),
    KEY `idx_user_created` (`user_id`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='对话记录表';

-- 对话记忆表
CREATE TABLE IF NOT EXISTS `chat_memory` (
    `id` bigint unsigned NOT NULL COMMENT '记忆ID',
    `session_id` bigint unsigned NOT NULL COMMENT '会话ID',
    `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
    `content` text NOT NULL COMMENT '记忆内容',
    `memory_type` varchar(20) NOT NULL COMMENT '记忆类型(event/reminder/fact)',
    `importance` float NOT NULL DEFAULT 1.0 COMMENT '重要性权重',
    `expire_time` timestamp NULL DEFAULT NULL COMMENT '过期时间',
    `access_count` int NOT NULL DEFAULT 0 COMMENT '访问次数',
    `metadata` json DEFAULT NULL COMMENT '元数据',
    `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_session_user_type` (`session_id`, `user_id`, `memory_type`),
    KEY `idx_expire_time` (`expire_time`),
    KEY `idx_access_count` (`access_count`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='对话记忆表';

-- 提醒表
CREATE TABLE IF NOT EXISTS `reminder` (
    `id` bigint unsigned NOT NULL COMMENT '提醒ID',
    `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
    `content` text NOT NULL COMMENT '提醒内容',
    `remind_time` timestamp NOT NULL COMMENT '提醒时间',
    `status` varchar(20) NOT NULL DEFAULT 'pending' COMMENT '状态(pending/triggered/completed)',
    `metadata` json DEFAULT NULL COMMENT '元数据',
    `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_user_status` (`user_id`, `status`),
    KEY `idx_remind_time` (`remind_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='提醒表';

-- 文档表
CREATE TABLE IF NOT EXISTS `document` (
    `doc_id` bigint unsigned NOT NULL COMMENT '文档ID',
    `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
    `title` varchar(200) NOT NULL COMMENT '标题',
    `status` varchar(20) NOT NULL DEFAULT 'active' COMMENT '文档状态(active/archived/deleted)',
    `metadata` json DEFAULT NULL COMMENT '元数据',
    `paragraph_count` int unsigned NOT NULL DEFAULT 0 COMMENT '段落数',
    `sentence_count` int unsigned NOT NULL DEFAULT 0 COMMENT '句子数',
    `keywords` json DEFAULT NULL COMMENT '文档关键词及权重',
    `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`doc_id`),
    KEY `idx_user_status` (`user_id`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='文档表';

-- 段落表
CREATE TABLE IF NOT EXISTS `document_paragraph` (
    `doc_id` bigint unsigned NOT NULL COMMENT '文档ID',
    `paragraph_id` bigint unsigned NOT NULL COMMENT '段落ID',
    `content` text NOT NULL COMMENT '段落内容',
    `sentence_id_min` bigint unsigned NOT NULL COMMENT '最小句子ID',
    `sentence_id_max` bigint unsigned NOT NULL COMMENT '最大句子ID',
    `keywords` JSON DEFAULT NULL COMMENT '段落关键词及权重',
    `keyword_text` VARCHAR(1024) AS (JSON_UNQUOTE(JSON_EXTRACT(`keywords`, '$[*].word'))) STORED COMMENT '关键词文本（用于全文索引）',
    `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`doc_id`, `paragraph_id`),
    FULLTEXT KEY `idx_keyword_text` (`keyword_text`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='段落表';

-- 句子表
CREATE TABLE IF NOT EXISTS `document_sentence` (
    `doc_id` bigint unsigned NOT NULL COMMENT '文档ID',
    `sentence_id` bigint unsigned NOT NULL COMMENT '句子ID',
    `paragraph_id` bigint unsigned NOT NULL COMMENT '段落ID',
    `content` text NOT NULL COMMENT '句子内容',
    `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`doc_id`, `sentence_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='句子表';

-- 文档块表
CREATE TABLE IF NOT EXISTS `document_chunk` (
    `chunk_id` bigint unsigned NOT NULL COMMENT '块ID',
    `doc_id` bigint unsigned NOT NULL COMMENT '文档ID',
    `paragraph_id` bigint unsigned NOT NULL COMMENT '段落ID',
    `sentence_id_min` bigint unsigned NOT NULL COMMENT '最小句子ID',
    `sentence_id_max` bigint unsigned NOT NULL COMMENT '最大句子ID',
    `keywords` JSON DEFAULT NULL COMMENT '块关键词及权重',
    `keyword_text` VARCHAR(1024) AS (JSON_UNQUOTE(JSON_EXTRACT(`keywords`, '$[*].word'))) STORED COMMENT '关键词文本（用于全文索引）',
    `embedding` blob DEFAULT NULL COMMENT '块的向量嵌入',
    `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`chunk_id`),
    FULLTEXT KEY `idx_keyword_text` (`keyword_text`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='文档块表';

-- ID生成器表
CREATE TABLE IF NOT EXISTS `id_generator` (
    `id_name` varchar(50) NOT NULL COMMENT 'ID名称',
    `sequence` bigint unsigned NOT NULL DEFAULT 0 COMMENT '当前序列值',
    `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='ID生成器表';

-- 初始化ID生成器表
INSERT INTO `id_generator` (`id_name`, `sequence`) VALUES 
('user_id', 0),
('chat_session_id', 0),
('chat_record_id', 0),
('chat_memory_id', 0),
('reminder_id', 0),
('document_id', 0),
('document_chunk_id', 0);
