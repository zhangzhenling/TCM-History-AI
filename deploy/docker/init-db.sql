-- TCM-History-AI 数据库初始化脚本
-- PostgreSQL 容器首次启动时自动执行（挂载到 /docker-entrypoint-initdb.d/）
-- 使用幂等方式创建所有业务库（POSTGRES_DB 已创建 tcm_history，此处做幂等兜底）

-- 创建默认库 tcm_history（user-service + history-service 共用）
SELECT 'CREATE DATABASE tcm_history'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'tcm_history')\gexec

-- 创建其余业务库
SELECT 'CREATE DATABASE tcm_knowledge'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'tcm_knowledge')\gexec

SELECT 'CREATE DATABASE tcm_graph'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'tcm_graph')\gexec

SELECT 'CREATE DATABASE tcm_ai'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'tcm_ai')\gexec

SELECT 'CREATE DATABASE tcm_learning'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'tcm_learning')\gexec

-- 为所有业务库授予 tcm 用户完整权限
GRANT ALL PRIVILEGES ON DATABASE tcm_history   TO tcm;
GRANT ALL PRIVILEGES ON DATABASE tcm_knowledge TO tcm;
GRANT ALL PRIVILEGES ON DATABASE tcm_graph     TO tcm;
GRANT ALL PRIVILEGES ON DATABASE tcm_ai        TO tcm;
GRANT ALL PRIVILEGES ON DATABASE tcm_learning  TO tcm;
