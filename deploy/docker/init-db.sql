-- TCM-History-AI 数据库初始化脚本
-- PostgreSQL 容器首次启动时自动执行（挂载到 /docker-entrypoint-initdb.d/）
-- POSTGRES_DB=tcm_history 由环境变量创建，此处补充其余业务库

CREATE DATABASE tcm_knowledge;
CREATE DATABASE tcm_graph;
CREATE DATABASE tcm_ai;
CREATE DATABASE tcm_learning;

-- 为所有业务库授予 tcm 用户完整权限
GRANT ALL PRIVILEGES ON DATABASE tcm_knowledge TO tcm;
GRANT ALL PRIVILEGES ON DATABASE tcm_graph    TO tcm;
GRANT ALL PRIVILEGES ON DATABASE tcm_ai       TO tcm;
GRANT ALL PRIVILEGES ON DATABASE tcm_learning TO tcm;
