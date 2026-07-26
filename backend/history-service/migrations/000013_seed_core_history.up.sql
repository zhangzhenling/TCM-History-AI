-- ============================================================================
-- 000013_seed_core_history.up.sql
-- TCM-History-AI :: history-service 核心种子数据迁移（up）
-- ----------------------------------------------------------------------------
-- 范围：中医发展史核心知识底座，覆盖 12 张表，共 73 条记录
--   1.  history_dynasty           10 条  (id 1-10)        朝代
--   2.  history_person            10 条  (id 1001-1010)   历史人物
--   3.  history_school             4 条  (id 2001-2004)   学派
--   4.  history_book               6 条  (id 3001-3006)   著作
--   5.  history_event              5 条  (id 4001-4005)   历史事件
--   6.  prescription               5 条  (id 5001-5005)   方剂
--   7.  medicine                  10 条  (id 6001-6010)   药物
--   8.  disease                    5 条  (id 7001-7005)   疾病
--   9.  person_school              4 条                    人物-学派
--  10.  book_author                5 条                    著作-作者
--  11.  medicine_prescription     11 条                    方剂-药物组方
--  12.  prescription_disease       3 条                    方剂-主治疾病
-- 幂等策略：
--   - history_dynasty / medicine / disease 依赖 UNIQUE(name) 用 ON CONFLICT (name)
--   - person_school / book_author / medicine_prescription / prescription_disease
--     依赖复合 UNIQUE 约束用 ON CONFLICT (col1, col2)
--   - history_person / history_school / history_book / history_event / prescription
--     无自然 UNIQUE 约束，用 NOT EXISTS 子查询按 id 防重
-- 时间字段统一 now()；中文内容用标准简体；雪花 ID 用固定整数便于引用
-- 依赖迁移：000001~000012 建表脚本（含各表 UNIQUE 约束与索引）
-- 设计文档：/workspace/04-数据库设计.md 第三章 (line 210-470)
-- ============================================================================

BEGIN;

-- ----------------------------------------------------------------------------
-- 1. history_dynasty 朝代（10 条）
-- ----------------------------------------------------------------------------
INSERT INTO history_dynasty (id, name, start_year, end_year, sort_order, description, created_at, updated_at)
VALUES
  (1,  '先秦', -221, -207, 1,  '中医理论奠基期，《黄帝内经》成书奠定阴阳五行与藏象经络学说基础', now(), now()),
  (2,  '汉',   -202, 220,  2,  '张仲景著《伤寒杂病论》确立辨证论治体系，华佗开外科先河', now(), now()),
  (3,  '三国', 220,  280,  3,  '华佗发明麻沸散施行腹腔手术，开中医外科先河', now(), now()),
  (4,  '晋',   265,  420,  4,  '王叔和整理《伤寒论》并著《脉经》，皇甫谧撰《针灸甲乙经》系统化针灸', now(), now()),
  (5,  '隋',   581,  618,  5,  '太医署设立，医学教育制度化起步，《诸病源候论》集病因学之大成', now(), now()),
  (6,  '唐',   618,  907,  6,  '孙思邈《千金要方》集唐以前医学之大成，本草学进入官修阶段', now(), now()),
  (7,  '宋',   960,  1279, 7,  '官修方书与针灸铜人推动医学规范化，钱乙奠基儿科', now(), now()),
  (8,  '金元', 1115, 1368, 8,  '金元四大家各立门户，寒凉、攻下、补土、养阴四派学术争鸣', now(), now()),
  (9,  '明',   1368, 1644, 9,  '李时珍《本草纲目》集本草学之大成，温补学派兴起', now(), now()),
  (10, '清',   1644, 1912, 10, '温病学派形成，叶天士创卫气营血辨证标志外感热病理论成熟', now(), now())
ON CONFLICT (name) DO NOTHING;

-- ----------------------------------------------------------------------------
-- 2. history_person 历史人物（10 条）
--    courtesy_name/alias_name 至少覆盖张仲景、华佗、孙思邈、李时珍、叶天士 5 位
-- ----------------------------------------------------------------------------
INSERT INTO history_person (id, name, courtesy_name, alias_name, dynasty_id, birth_year, death_year, gender, title, biography, achievements, portrait_url, created_at, updated_at)
SELECT t.id, t.name, t.courtesy_name, t.alias_name, t.dynasty_id, t.birth_year, t.death_year, t.gender, t.title, t.biography, t.achievements, t.portrait_url, t.created_at, t.updated_at
FROM (VALUES
  (1001::bigint, '张仲景'::varchar, '仲景'::varchar,  NULL::varchar,    2::bigint, 150::smallint,  219::smallint,  'male'::varchar, '长沙太守'::varchar,        '东汉医学家，官至长沙太守，著《伤寒杂病论》'::text,        '确立六经辨证体系；著《伤寒杂病论》；被后世尊为医圣'::text,                  NULL::varchar, now()::timestamptz, now()::timestamptz),
  (1002,          '华佗',            '元化',            NULL,              2,          145,           208,            'male',           '沛国谯县医者',              '东汉医学家，外科鼻祖，发明麻沸散与五禽戏',                  '发明麻沸散开创全身麻醉；创制五禽戏养生；精于方药针灸外科',     NULL,           now(),              now()),
  (1003,          '王叔和',          '叔和',            NULL,              4,          201,           280,            'male',           '太医令',                    '魏晋医学家，官至太医令，整理《伤寒论》并著《脉经》',        '整理张仲景《伤寒论》使之传世；著《脉经》集脉学之大成',        NULL,           now(),              now()),
  (1004,          '皇甫谧',          '士安',            '玄晏先生',        4,          215,           282,            'male',           '玄晏先生',                  '魏晋医学家，集《素问》《灵枢》《明堂》撰《针灸甲乙经》',    '著《针灸甲乙经》为现存最早针灸学专著；整理医学文献',          NULL,           now(),              now()),
  (1005,          '孙思邈',          NULL,              '药王',            6,          581,           682,            'male',           '药王',                      '唐代医学家，被后世尊为药王，著《千金要方》《千金翼方》',    '著《千金要方》集唐以前医学之大成；倡医德并著《大医精诚》',    NULL,           now(),              now()),
  (1006,          '钱乙',            '仲阳',            NULL,              7,          1032,          1113,           'male',           '太医丞',                    '北宋医学家，官至太医丞，奠基中医儿科',                      '著《小儿药证直诀》为现存最早儿科专著；创制六味地黄丸',       NULL,           now(),              now()),
  (1007,          '刘完素',          '守真',            '河间居士',        8,          1120,          1200,           'male',           '河间居士',                  '金代医学家，金元四大家之一，寒凉派代表',                    '倡火热致病论善用寒凉药；著《素问玄机原病式》；开寒凉学派',    NULL,           now(),              now()),
  (1008,          '张从正',          '子和',            '戴人',            8,          1156,          1228,           'male',           '戴人',                      '金代医学家，金元四大家之一，攻下派代表',                    '倡邪去正安论以汗吐下三法祛邪；著《儒门事亲》；开攻下学派',    NULL,           now(),              now()),
  (1009,          '李时珍',          '东壁',            '濒湖',            9,          1518,          1593,           'male',           '太医院判',                  '明代医学家，历时 27 年编纂《本草纲目》',                    '著《本草纲目》收药 1892 种集本草大成；著《濒湖脉学》',        NULL,           now(),              now()),
  (1010,          '叶天士',          NULL,              '香岩',            10,         1667,          1746,           'male',           '香岩',                      '清代医学家，温病学派代表，创卫气营血辨证',                  '创卫气营血辨证论治外感温热病；著《温热论》；奠基温病学派',    NULL,           now(),              now())
) AS t(id, name, courtesy_name, alias_name, dynasty_id, birth_year, death_year, gender, title, biography, achievements, portrait_url, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM history_person hp WHERE hp.id = t.id);

-- ----------------------------------------------------------------------------
-- 3. history_school 学派（4 条）
-- ----------------------------------------------------------------------------
INSERT INTO history_school (id, name, dynasty_id, founder_person_id, summary, established_year, created_at, updated_at)
SELECT t.id, t.name, t.dynasty_id, t.founder_person_id, t.summary, t.established_year, t.created_at, t.updated_at
FROM (VALUES
  (2001::bigint, '伤寒学派'::varchar, 2::bigint,   1001::bigint, '以《伤寒论》为宗，研究外感热病六经辨证论治'::text,                  200::smallint,  now()::timestamptz, now()::timestamptz),
  (2002,          '寒凉学派',           8,           1007,          '主张火热致病，善用寒凉药物清热泻火，开金元流派之先'::text,           1150,           now(),              now()),
  (2003,          '攻下学派',           8,           1008,          '主张邪去正安，以汗吐下三法攻邪祛病'::text,                           1200,           now(),              now()),
  (2004,          '温病学派',           10,          1010,          '创立卫气营血与三焦辨证，论治外感温热病，标志外感理论成熟'::text,      1700,           now(),              now())
) AS t(id, name, dynasty_id, founder_person_id, summary, established_year, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM history_school hs WHERE hs.id = t.id);

-- ----------------------------------------------------------------------------
-- 4. history_book 著作（6 条）
-- ----------------------------------------------------------------------------
INSERT INTO history_book (id, title, dynasty_id, published_year, category, summary, volume_count, is_extant, file_url, created_at, updated_at)
SELECT t.id, t.title, t.dynasty_id, t.published_year, t.category, t.summary, t.volume_count, t.is_extant, t.file_url, t.created_at, t.updated_at
FROM (VALUES
  (3001::bigint, '黄帝内经'::varchar,     1::bigint, -300::smallint, '经典'::varchar, '中医理论奠基之作，阐述阴阳五行、藏象经络与病机治则'::text,     18::integer, true::boolean,  NULL::varchar, now()::timestamptz, now()::timestamptz),
  (3002,          '伤寒杂病论',            2,          210,            '经典',          '确立六经辨证与辨证论治体系，载方 113 首为方书之祖',             16,          true,           NULL,          now(),              now()),
  (3003,          '金匮要略',              2,          210,            '经典',          '论述杂病证治，载方 205 首，开创内科杂病辨证体系',              3,           true,           NULL,          now(),              now()),
  (3004,          '针灸甲乙经',            4,          282,            '经典',          '现存最早针灸学专著，系统整理经络腧穴与刺灸方法',              12,          true,           NULL,          now(),              now()),
  (3005,          '千金要方',              6,          652,            '方书',          '唐以前医学集大成，载方 5300 余首，首倡医德规范',              30,          true,           NULL,          now(),              now()),
  (3006,          '本草纲目',              9,          1578,           '本草',          '集本草学大成，收药 1892 种附方 11096 首影响东亚医学',          52,          true,           NULL,          now(),              now())
) AS t(id, title, dynasty_id, published_year, category, summary, volume_count, is_extant, file_url, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM history_book hb WHERE hb.id = t.id);

-- ----------------------------------------------------------------------------
-- 5. history_event 历史事件（5 条）
-- ----------------------------------------------------------------------------
INSERT INTO history_event (id, title, dynasty_id, occurred_year, event_type, description, impact, location, created_at, updated_at)
SELECT t.id, t.title, t.dynasty_id, t.occurred_year, t.event_type, t.description, t.impact, t.location, t.created_at, t.updated_at
FROM (VALUES
  (4001::bigint, '《黄帝内经》成书'::varchar,         1::bigint, -300::smallint, '学术'::varchar, '战国至秦汉间中医理论奠基之作成书'::text,                '奠定阴阳五行、藏象经络学说基础，确立中医理论框架'::text, '不详'::varchar, now()::timestamptz, now()::timestamptz),
  (4002,          '张仲景著《伤寒杂病论》',           2,          210,            '出版',          '张仲景总结外感热病证治经验成书',                          '确立辨证论治体系，后世尊张仲景为医圣',                    '长沙',          now(),              now()),
  (4003,          '王叔和整理伤寒论',                 4,          230,            '学术',          '王叔和搜集整理张仲景遗文编次成《伤寒论》',                '使《伤寒论》单独成书并传世，奠定伤寒学派文献基础',        '洛阳',          now(),              now()),
  (4004,          '针灸甲乙经成书',                   4,          282,            '出版',          '皇甫谧集《素问》《灵枢》《明堂孔穴》编撰成书',            '现存最早针灸学专著，系统化经络腧穴理论',                  '不详',          now(),              now()),
  (4005,          '本草纲目成书',                     9,          1578,           '出版',          '李时珍历时 27 年编纂完成《本草纲目》',                    '集本草学之大成，影响东亚医学深远并被译为多国文字',        '蕲州',          now(),              now())
) AS t(id, title, dynasty_id, occurred_year, event_type, description, impact, location, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM history_event he WHERE he.id = t.id);

-- ----------------------------------------------------------------------------
-- 6. prescription 方剂（5 条）
--    source_book_id 为 NULL 时直接传 NULL
-- ----------------------------------------------------------------------------
INSERT INTO prescription (id, name, pinyin, source_book_id, source_person_id, dynasty_id, composition, usage, indications, category, created_at, updated_at)
SELECT t.id, t.name, t.pinyin, t.source_book_id, t.source_person_id, t.dynasty_id, t.composition, t.usage, t.indications, t.category, t.created_at, t.updated_at
FROM (VALUES
  (5001::bigint, '麻黄汤'::varchar,       'mahuangtang'::varchar,       3002::bigint,   1001::bigint, 2::bigint,  '麻黄、桂枝、杏仁、甘草'::text,           '水煎服，温服覆取微似汗'::text,       '外感风寒表实证，恶寒发热无汗身痛'::text, '解表'::varchar, now()::timestamptz, now()::timestamptz),
  (5002,          '桂枝汤',                'guizhitang',                  3002,           1001,         2,           '桂枝、芍药、甘草、生姜、大枣',           '水煎服，温服须臾啜热稀粥助汗',        '外感风寒表虚证，发热汗出恶风',          '解表',           now(),              now()),
  (5003,          '白虎汤',                'baihutang',                   3002,           1001,         2,           '石膏、知母、甘草、粳米',                 '水煎至米熟汤成，温服',                '阳明气分热盛，壮热烦渴汗出脉洪大',      '清热',           now(),              now()),
  (5004,          '六味地黄丸',            'liuweidihuangwan',            NULL::bigint,   1006,         7,           '熟地黄、山茱萸、山药、泽泻、茯苓、牡丹皮','炼蜜为丸，淡盐汤送服',                '肾阴亏损，头晕耳鸣腰膝酸软盗汗',        '补益',           now(),              now()),
  (5005,          '银翘散',                'yinqiaosan',                  NULL::bigint,   1010,         10,          '金银花、连翘、薄荷、桔梗、淡豆豉等',     '煎汤温服，香气大出即取服',            '温病初起，发热微恶风寒头痛口渴',        '解表',           now(),              now())
) AS t(id, name, pinyin, source_book_id, source_person_id, dynasty_id, composition, usage, indications, category, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM prescription p WHERE p.id = t.id);

-- ----------------------------------------------------------------------------
-- 7. medicine 药物（10 条）—— ON CONFLICT (name)
-- ----------------------------------------------------------------------------
INSERT INTO medicine (id, name, pinyin, alias_json, nature, flavor, meridian, efficacy, dosage, toxicity, created_at, updated_at)
VALUES
  (6001, '麻黄',     'mahuang',     '["麻黄草","草麻黄"]'::jsonb,     '温',     '辛',     '肺、膀胱',           '发汗解表、宣肺平喘、利水消肿', '2-9g',    '无毒',   now(), now()),
  (6002, '桂枝',     'guizhi',      '["嫩桂枝","桂枝尖"]'::jsonb,     '温',     '辛、甘', '心、肺、膀胱',       '发汗解肌、温通经脉、助阳化气', '3-9g',    '无毒',   now(), now()),
  (6003, '杏仁',     'xingren',     '["苦杏仁","北杏仁"]'::jsonb,     '微温',   '苦',     '肺、大肠',           '降气止咳平喘、润肠通便',       '5-10g',   '有小毒', now(), now()),
  (6004, '甘草',     'gancao',      '["甜草根","粉甘草"]'::jsonb,     '平',     '甘',     '心、肺、脾、胃',     '补脾益气、清热解毒、调和诸药', '2-10g',   '无毒',   now(), now()),
  (6005, '石膏',     'shigao',      '["生石膏","细石"]'::jsonb,       '寒',     '甘、辛', '肺、胃',             '清热泻火、除烦止渴',           '15-60g',  '无毒',   now(), now()),
  (6006, '知母',     'zhimu',       '["毛知母","光知母"]'::jsonb,     '寒',     '苦、甘', '肺、胃、肾',         '清热泻火、滋阴润燥',           '6-12g',   '无毒',   now(), now()),
  (6007, '熟地黄',   'shudihuang',  '["熟地","大熟地"]'::jsonb,       '微温',   '甘',     '肝、肾',             '补血滋阴、益精填髓',           '12-30g',  '无毒',   now(), now()),
  (6008, '山茱萸',   'shanzhuyu',   '["山萸肉","枣皮"]'::jsonb,       '微温',   '酸、涩', '肝、肾',             '补益肝肾、收敛固涩',           '6-12g',   '无毒',   now(), now()),
  (6009, '山药',     'shanyao',     '["淮山","薯蓣"]'::jsonb,         '平',     '甘',     '脾、肺、肾',         '益气养阴、补脾肺肾、固精止带', '15-30g',  '无毒',   now(), now()),
  (6010, '牡丹皮',   'mudanpi',     '["丹皮","粉丹皮"]'::jsonb,       '微寒',   '苦、辛', '心、肝、肾',         '清热凉血、活血化瘀',           '6-12g',   '无毒',   now(), now())
ON CONFLICT (name) DO NOTHING;

-- ----------------------------------------------------------------------------
-- 8. disease 疾病（5 条）—— ON CONFLICT (name)
-- ----------------------------------------------------------------------------
INSERT INTO disease (id, name, pinyin, category, description, symptoms, tcm_pathogenesis, created_at, updated_at)
VALUES
  (7001, '感冒',     'ganmao',      '外感', '外邪犯肺所致以恶寒发热鼻塞为主证的表证', '鼻塞、流涕、咳嗽、恶寒、发热',           '六淫外邪侵袭肺卫，肺失宣降，卫表失和',       now(), now()),
  (7002, '咳嗽',     'kesou',       '内伤', '肺气上逆所致以咳嗽为主证的病证',         '咳嗽、咯痰、胸闷',                       '外邪犯肺或脏腑失调，肺气上逆',               now(), now()),
  (7003, '喘证',     'chuanzheng',  '杂病', '以呼吸困难气喘为主证的病证',            '气喘、胸闷、不得平卧',                   '外感内伤致肺失宣降，肾不纳气',               now(), now()),
  (7004, '痢疾',     'liji',        '杂病', '以腹痛里急后重下痢赤白为主证的病证',    '腹痛、里急后重、下痢赤白',               '湿热疫毒蕴结肠道，气血凝滞，传导失司',       now(), now()),
  (7005, '肾阴虚',   'shenyinxu',   '内伤', '肾阴亏虚所致虚热内生的证候',            '腰膝酸软、头晕耳鸣、盗汗、五心烦热',     '肾阴亏虚，失于濡养，虚热内生',               now(), now())
ON CONFLICT (name) DO NOTHING;

-- ----------------------------------------------------------------------------
-- 9. person_school 人物-学派（4 条，role=founder）
--    id = person_id * 10000 + school_id
-- ----------------------------------------------------------------------------
INSERT INTO person_school (id, person_id, school_id, role, joined_year, created_at)
VALUES
  (10012001, 1001, 2001, 'founder', 200,  now()),
  (10072002, 1007, 2002, 'founder', 1150, now()),
  (10082003, 1008, 2003, 'founder', 1200, now()),
  (10102004, 1010, 2004, 'founder', 1700, now())
ON CONFLICT (person_id, school_id) DO NOTHING;

-- ----------------------------------------------------------------------------
-- 10. book_author 著作-作者（5 条，author_type=author）
--    《黄帝内经》无明确作者，按任务要求跳过
--    id = book_id * 10000 + person_id
-- ----------------------------------------------------------------------------
INSERT INTO book_author (id, book_id, person_id, author_type, sort_order, created_at)
VALUES
  (30021001, 3002, 1001, 'author', 1, now()),
  (30031001, 3003, 1001, 'author', 1, now()),
  (30041004, 3004, 1004, 'author', 1, now()),
  (30051005, 3005, 1005, 'author', 1, now()),
  (30061009, 3006, 1009, 'author', 1, now())
ON CONFLICT (book_id, person_id) DO NOTHING;

-- ----------------------------------------------------------------------------
-- 11. medicine_prescription 方剂-药物组方（11 条）
--     仅填麻黄汤(5001)、白虎汤(5003)、六味地黄丸(5004) 三方组方关系
--     桂枝汤(含芍药)、银翘散(含银花连翘) 因本种子未含相应药物而省略，留 TODO
--     id = prescription_id * 10000 + medicine_id
-- ----------------------------------------------------------------------------
INSERT INTO medicine_prescription (id, prescription_id, medicine_id, role, dosage, sort_order, created_at)
VALUES
  -- 麻黄汤 5001
  (50016001, 5001, 6001, '君', '9g',  1, now()),
  (50016002, 5001, 6002, '臣', '6g',  2, now()),
  (50016003, 5001, 6003, '佐', '6g',  3, now()),
  (50016004, 5001, 6004, '使', '3g',  4, now()),
  -- 白虎汤 5003（粳米未列入药物表，省略）
  (50036005, 5003, 6005, '君', '50g', 1, now()),
  (50036006, 5003, 6006, '臣', '9g',  2, now()),
  (50036004, 5003, 6004, '使', '3g',  3, now()),
  -- 六味地黄丸 5004（泽泻、茯苓未列入药物表，省略）
  (50046007, 5004, 6007, '君', '24g', 1, now()),
  (50046008, 5004, 6008, '臣', '12g', 2, now()),
  (50046009, 5004, 6009, '臣', '12g', 3, now()),
  (50046010, 5004, 6010, '佐', '9g',  4, now())
ON CONFLICT (prescription_id, medicine_id) DO NOTHING;

-- ----------------------------------------------------------------------------
-- 12. prescription_disease 方剂-主治疾病（3 条）
--     id = prescription_id * 10000 + disease_id
-- ----------------------------------------------------------------------------
INSERT INTO prescription_disease (id, prescription_id, disease_id, efficacy_note, is_primary, created_at)
VALUES
  (50017001, 5001, 7001, '辛温发汗，主治外感风寒表实证',          true,  now()),
  (50037001, 5003, 7001, '清热生津，治气分热盛之壮热烦渴',        false, now()),
  (50047005, 5004, 7005, '滋阴补肾，主治肾阴亏损诸证',            true,  now())
ON CONFLICT (prescription_id, disease_id) DO NOTHING;

COMMIT;

-- ============================================================================
-- TODO（后续迁移补全）：
--   1. 桂枝汤(5002) 组方：缺芍药、生姜、大枣三味药物
--   2. 白虎汤(5003) 组方：缺粳米一味
--   3. 六味地黄丸(5004) 组方：缺泽泻、茯苓二味
--   4. 银翘散(5005) 组方：缺金银花、连翘、薄荷、桔梗、淡豆豉等多味
--   5. 银翘散(5005) 与感冒/温病初起的 prescription_disease 关联
-- ============================================================================
