-- 种子数据：以张仲景、华佗为核心的东汉医学知识网络（doc/05 §5.4）。
-- uid 命名约定参考 doc/05 §5.2.3 的示例。id 采用稳定占位值，便于本地开发
-- 与集成测试时引用；生产环境由 idgen 重新分配，不影响业务字段。

-- 朝代节点：东汉
INSERT INTO graph_nodes (id, uid, label, name, properties_json, synced_at, created_at, updated_at)
VALUES (
    5001,
    '01HXY7QECDONGHAN',
    'Dynasty',
    '东汉',
    '{"start_year": 25, "end_year": 220, "capital": "洛阳", "intro": "公元25年至220年，医学体系初步形成时期"}'::jsonb,
    now(), now(), now()
) ON CONFLICT DO NOTHING;

-- 人物节点：张仲景
INSERT INTO graph_nodes (id, uid, label, name, properties_json, synced_at, created_at, updated_at)
VALUES (
    5002,
    '01HXY7QK3PERSONZHANGZHONGJING',
    'Person',
    '张仲景',
    '{"courtesy_name": "仲景", "alias": ["张机"], "dynasty_name": "东汉", "birth_year": 150, "death_year": 219, "hometown": "南阳涅阳", "intro": "东汉末年著名医学家，被后世尊为医圣", "achievements": "确立辨证论治体系，著《伤寒杂病论》"}'::jsonb,
    now(), now(), now()
) ON CONFLICT DO NOTHING;

-- 人物节点：华佗
INSERT INTO graph_nodes (id, uid, label, name, properties_json, synced_at, created_at, updated_at)
VALUES (
    5003,
    '01HXY7QK4PERSONHUATUO',
    'Person',
    '华佗',
    '{"courtesy_name": "元化", "dynasty_name": "东汉", "birth_year": 145, "death_year": 208, "hometown": "沛国谯县", "intro": "东汉末年方士、医师，外科麻醉术的早期记载者", "achievements": "发明麻沸散，创五禽戏"}'::jsonb,
    now(), now(), now()
) ON CONFLICT DO NOTHING;

-- 经典节点：伤寒论
INSERT INTO graph_nodes (id, uid, label, name, properties_json, synced_at, created_at, updated_at)
VALUES (
    5004,
    '01HXY7QL8SHANGHANLUN',
    'Classic',
    '伤寒论',
    '{"title": "伤寒论", "category": "著作", "dynasty_name": "东汉", "completion_year": 210, "volumes": 10, "abstract": "论述外感热病六经辨证体系"}'::jsonb,
    now(), now(), now()
) ON CONFLICT DO NOTHING;

-- 经典节点：黄帝内经
INSERT INTO graph_nodes (id, uid, label, name, properties_json, synced_at, created_at, updated_at)
VALUES (
    5005,
    '01HXY7QLAHDNJ',
    'Classic',
    '黄帝内经',
    '{"title": "黄帝内经", "category": "著作", "dynasty_name": "战国", "abstract": "中医理论体系奠基之作，分《素问》《灵枢》各81篇"}'::jsonb,
    now(), now(), now()
) ON CONFLICT DO NOTHING;

-- 学派节点：伤寒派
INSERT INTO graph_nodes (id, uid, label, name, properties_json, synced_at, created_at, updated_at)
VALUES (
    5006,
    '01HXY7QMASHANGHANPAI',
    'School',
    '伤寒派',
    '{"dynasty_name": "东汉", "establish_year": 210, "core_theory": "六经辨证，方证对应", "intro": "以张仲景《伤寒论》为宗，研究外感热病辨证论治的学派"}'::jsonb,
    now(), now(), now()
) ON CONFLICT DO NOTHING;

-- 关系：张仲景 → 伤寒论（AUTHORED）
INSERT INTO graph_edges (id, uid, type, source_uid, target_uid, properties_json, synced_at, created_at, updated_at)
VALUES (
    5101,
    '01HXY7R-AUTH-ZSJ-SHL',
    'AUTHORED',
    '01HXY7QK3PERSONZHANGZHONGJING',
    '01HXY7QL8SHANGHANLUN',
    '{"role": "作者", "completion_year": 210, "description": "张仲景撰《伤寒杂病论》，后世分为伤寒与杂病两部"}'::jsonb,
    now(), now(), now()
) ON CONFLICT DO NOTHING;

-- 关系：张仲景 → 伤寒派（BELONGS_TO）
INSERT INTO graph_edges (id, uid, type, source_uid, target_uid, properties_json, synced_at, created_at, updated_at)
VALUES (
    5102,
    '01HXY7RD-BLT-ZSJ-SHP',
    'BELONGS_TO',
    '01HXY7QK3PERSONZHANGZHONGJING',
    '01HXY7QMASHANGHANPAI',
    '{"join_year": 210, "role": "创始人"}'::jsonb,
    now(), now(), now()
) ON CONFLICT DO NOTHING;

-- 关系：伤寒论 → 黄帝内经（CITED）
INSERT INTO graph_edges (id, uid, type, source_uid, target_uid, properties_json, synced_at, created_at, updated_at)
VALUES (
    5103,
    '01HXY7RF-CIT-SHL-HDNJ',
    'CITED',
    '01HXY7QL8SHANGHANLUN',
    '01HXY7QLAHDNJ',
    '{"chapter": "原序", "context": "撰用《素问》《九卷》", "description": "伤寒论序明示参引内经"}'::jsonb,
    now(), now(), now()
) ON CONFLICT DO NOTHING;

-- 关系：张仲景 → 东汉（OCCURRED_IN）
INSERT INTO graph_edges (id, uid, type, source_uid, target_uid, properties_json, synced_at, created_at, updated_at)
VALUES (
    5104,
    '01HXY7RE-OCC-ZSJ-DH',
    'OCCURRED_IN',
    '01HXY7QK3PERSONZHANGZHONGJING',
    '01HXY7QECDONGHAN',
    '{"description": "生活于东汉末年"}'::jsonb,
    now(), now(), now()
) ON CONFLICT DO NOTHING;

-- 关系：华佗 → 东汉（OCCURRED_IN）
INSERT INTO graph_edges (id, uid, type, source_uid, target_uid, properties_json, synced_at, created_at, updated_at)
VALUES (
    5105,
    '01HXY7RE-OCC-HH-DH',
    'OCCURRED_IN',
    '01HXY7QK4PERSONHUATUO',
    '01HXY7QECDONGHAN',
    '{"description": "生活于东汉末年"}'::jsonb,
    now(), now(), now()
) ON CONFLICT DO NOTHING;
