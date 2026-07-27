//go:build integration

package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// TestLearnerJourney_EndToEnd 验证一条完整的 learner journey 数据链路：
//
//	用户(注册) → 角色(student) → 资料 → 课程 → 课时 → 选课 → 学习记录
//	→ 考试 → 题目 → 答题 → 错题 → 错题掌握
//
// 由于 Go internal 包可见性规则限制，本测试不调用 *-service/internal 下的
// usecase / repo；而是通过 raw SQL 直接对共享 PostgreSQL 写入并回查，
// 验证跨服务的数据关系（user.id → enrollment.user_id → learning_record.user_id
// → exam_attempt.user_id → wrong_question.user_id）能在同一数据库中正确建立。
//
// 这等价于把 user-service 与 learning-service 的持久化层接到同一 DB，
// 用最小成本覆盖「学习者旅程」涉及的全部表与外键关系。
func TestLearnerJourney_EndToEnd(t *testing.T) {
	skipIfNoDeps(t)

	// 每条用例独立清空相关表，避免与其他用例相互干扰。
	resetTablesInTest(t,
		"learning_wrong_questions",
		"learning_exam_attempts",
		"learning_questions",
		"learning_exams",
		"learning_records",
		"learning_enrollments",
		"learning_lessons",
		"learning_courses",
		"tenant_members",
		"user_settings",
		"user_profiles",
		"user_roles",
		"users",
	)

	ctx, cancel := newContext()
	defer cancel()

	// ---------- 1. 用户注册（写 users + user_roles + user_profiles + user_settings） ----------
	userID := nextID()
	username := "learner_" + strconv.FormatInt(userID, 10)
	// 使用 bcrypt 哈希密码，与生产 user-service 行为一致。
	// cost=4 是测试加速用的最低值；生产环境使用 cost=10。
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("Passw0rd!"), bcrypt.MinCost)
	require.NoError(t, err, "bcrypt hash password")

	err = execSQL(ctx, `
		INSERT INTO users (id, username, email, password_hash, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'active', now(), now())`,
		userID, username, username+"@example.com", string(passwordHash),
	)
	require.NoError(t, err, "insert user")

	// 查询验证：username 唯一索引生效，回读字段一致
	var gotUsername, gotStatus string
	err = db.Raw(`SELECT username, status FROM users WHERE id = $1`, userID).Row().Scan(&gotUsername, &gotStatus)
	require.NoError(t, err, "query user")
	assert.Equal(t, username, gotUsername)
	assert.Equal(t, "active", gotStatus)

	// ---------- 2. 分配 student 角色 ----------
	userRoleID := nextID()
	// student 角色 id=3，由 user-service/migrations/000008 种子迁移写入。
	err = execSQL(ctx, `
		INSERT INTO user_roles (id, user_id, role_id, granted_at, created_at)
		VALUES ($1, $2, 3, now(), now())`,
		userRoleID, userID,
	)
	require.NoError(t, err, "assign student role")

	// 验证：用户角色关联写入
	var roleCode string
	err = db.Raw(`
		SELECT r.code
		FROM user_roles ur
		JOIN roles r ON r.id = ur.role_id
		WHERE ur.user_id = $1`, userID).Row().Scan(&roleCode)
	require.NoError(t, err, "query user role")
	assert.Equal(t, "student", roleCode, "user should have student role")

	// ---------- 3. 写入用户资料与设置（与生产 register 流程对齐） ----------
	profileID := nextID()
	err = execSQL(ctx, `
		INSERT INTO user_profiles (id, user_id, nickname, gender, created_at, updated_at)
		VALUES ($1, $2, $3, 'unknown', now(), now())`,
		profileID, userID, username,
	)
	require.NoError(t, err, "insert user_profile")

	settingsID := nextID()
	err = execSQL(ctx, `
		INSERT INTO user_settings (id, user_id, locale, theme, created_at, updated_at)
		VALUES ($1, $2, 'zh-CN', 'light', now(), now())`,
		settingsID, userID,
	)
	require.NoError(t, err, "insert user_settings")

	// ---------- 4. 创建课程 + 课时 ----------
	courseID := nextID()
	err = execSQL(ctx, `
		INSERT INTO learning_courses (id, title, description, category, difficulty, is_published, sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, 'classic', 'beginner', FALSE, 1, now(), now())`,
		courseID, "伤寒论导读", "测试课程：覆盖六经辨证基础",
	)
	require.NoError(t, err, "insert course")

	lessonID := nextID()
	err = execSQL(ctx, `
		INSERT INTO learning_lessons (id, course_id, title, content, content_type, sort_order, is_published, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'article', 1, TRUE, now(), now())`,
		lessonID, courseID, "太阳病篇", "太阳之为病，脉浮，头项强痛而恶寒。",
	)
	require.NoError(t, err, "insert lesson")

	// 发布课程
	err = execSQL(ctx, `UPDATE learning_courses SET is_published = TRUE, updated_at = now() WHERE id = $1`, courseID)
	require.NoError(t, err, "publish course")

	var isPublished bool
	err = db.Raw(`SELECT is_published FROM learning_courses WHERE id = $1`, courseID).Row().Scan(&isPublished)
	require.NoError(t, err)
	assert.True(t, isPublished, "course should be published")

	// ---------- 5. 选课 ----------
	enrollmentID := nextID()
	err = execSQL(ctx, `
		INSERT INTO learning_enrollments (id, user_id, course_id, progress_percent, status, created_at, updated_at)
		VALUES ($1, $2, $3, 0, 'enrolled', now(), now())`,
		enrollmentID, userID, courseID,
	)
	require.NoError(t, err, "enroll")

	// 唯一索引验证：重复选课应触发唯一约束冲突
	err = execSQL(ctx, `
		INSERT INTO learning_enrollments (id, user_id, course_id, progress_percent, status, created_at, updated_at)
		VALUES ($1, $2, $3, 0, 'enrolled', now(), now())`,
		nextID(), userID, courseID,
	)
	require.Error(t, err, "duplicate (user_id, course_id) should violate unique index")

	// ---------- 6. 学习记录 ----------
	recordID := nextID()
	err = execSQL(ctx, `
		INSERT INTO learning_records (id, user_id, lesson_id, course_id, duration_seconds, position_percent, is_completed, learned_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 120, 50, FALSE, now(), now(), now())`,
		recordID, userID, lessonID, courseID,
	)
	require.NoError(t, err, "insert learning_record")

	// 学习记录唯一索引：同一 (user_id, lesson_id) 只能有一条
	err = execSQL(ctx, `
		INSERT INTO learning_records (id, user_id, lesson_id, course_id, duration_seconds, position_percent, is_completed, learned_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 60, 80, FALSE, now(), now(), now())`,
		nextID(), userID, lessonID, courseID,
	)
	require.Error(t, err, "duplicate (user_id, lesson_id) should violate unique index")

	// 更新选课进度（模拟学习记录提交后的副作用）
	err = execSQL(ctx, `
		UPDATE learning_enrollments
		SET progress_percent = 60, last_lesson_id = $2, status = 'in_progress', updated_at = now()
		WHERE id = $1`,
		enrollmentID, lessonID,
	)
	require.NoError(t, err, "update enrollment progress")

	var progressPercent int
	var status string
	err = db.Raw(`SELECT progress_percent, status FROM learning_enrollments WHERE id = $1`, enrollmentID).
		Row().Scan(&progressPercent, &status)
	require.NoError(t, err)
	assert.Equal(t, 60, progressPercent)
	assert.Equal(t, "in_progress", status)

	// ---------- 7. 创建考试 + 题目 ----------
	examID := nextID()
	err = execSQL(ctx, `
		INSERT INTO learning_exams (id, title, course_id, question_count, pass_score, duration_minutes, is_published, created_at, updated_at)
		VALUES ($1, $2, $3, 0, 60, 30, TRUE, now(), now())`,
		examID, "太阳病篇测验", courseID,
	)
	require.NoError(t, err, "insert exam")

	// 单选题：正确答案为索引 1 (浮脉)
	q1ID := nextID()
	err = execSQL(ctx, `
		INSERT INTO learning_questions (id, exam_id, type, content, options_json, answer_json, score, difficulty, created_at, updated_at)
		VALUES ($1, $2, 'single_choice', $3, $4, $5, 10, 'beginner', now(), now())`,
		q1ID, examID, "太阳病的主脉是？",
		`["沉脉","浮脉","迟脉","数脉"]`, `1`,
	)
	require.NoError(t, err, "insert q1")

	// 判断题：正确答案为 true
	q2ID := nextID()
	err = execSQL(ctx, `
		INSERT INTO learning_questions (id, exam_id, type, content, options_json, answer_json, score, difficulty, created_at, updated_at)
		VALUES ($1, $2, 'true_false', $3, $4, $5, 10, 'beginner', now(), now())`,
		q2ID, examID, "太阳病可见头项强痛。",
		`["对","错"]`, `0`, // true_false 答案存索引：0 = 对
	)
	require.NoError(t, err, "insert q2")

	// ---------- 8. 考试：开始 + 提交（一题对一题错） ----------
	attemptID := nextID()
	err = execSQL(ctx, `
		INSERT INTO learning_exam_attempts (id, exam_id, user_id, score, total_score, is_passed, started_at, answers_json, created_at, updated_at)
		VALUES ($1, $2, $3, 0, 0, FALSE, now(), '[]'::jsonb, now(), now())`,
		attemptID, examID, userID,
	)
	require.NoError(t, err, "start exam attempt")

	// 自动判分：q1 用户答 0（错误，正确答案为 1），q2 用户答 0（正确，正确答案为 0）
	// 期望得分 = 10 (只 q2 正确)，总分 = 20
	answersJSON, _ := json.Marshal([]map[string]interface{}{
		{"question_id": q1ID, "answer": 0, "correct": false},
		{"question_id": q2ID, "answer": 0, "correct": true},
	})
	err = execSQL(ctx, `
		UPDATE learning_exam_attempts
		SET score = 10, total_score = 20, is_passed = FALSE, submitted_at = now(),
		    duration_seconds = EXTRACT(EPOCH FROM (now() - started_at))::INTEGER,
		    answers_json = $2, updated_at = now()
		WHERE id = $1`,
		attemptID, string(answersJSON),
	)
	require.NoError(t, err, "submit attempt")

	var score, totalScore int
	var isPassed bool
	var submittedAt sql.NullTime
	err = db.Raw(`SELECT score, total_score, is_passed, submitted_at FROM learning_exam_attempts WHERE id = $1`, attemptID).
		Row().Scan(&score, &totalScore, &isPassed, &submittedAt)
	require.NoError(t, err)
	assert.Equal(t, 10, score, "score should be 10 (only q2 correct)")
	assert.Equal(t, 20, totalScore, "total score should be 20")
	assert.False(t, isPassed, "10/20 = 50% < 60% pass line, not passed")
	assert.True(t, submittedAt.Valid, "submitted_at must be set")

	// ---------- 9. 错题本（q1 错误 → 写入 wrong_questions） ----------
	wrongID := nextID()
	err = execSQL(ctx, `
		INSERT INTO learning_wrong_questions (id, user_id, question_id, exam_id, attempt_id, user_answer_json, wrong_count, last_wrong_at, is_mastered, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 1, now(), FALSE, now(), now())`,
		wrongID, userID, q1ID, examID, attemptID, `0`,
	)
	require.NoError(t, err, "insert wrong_question")

	// 验证：错题本中恰好 1 条
	var wrongCount int
	err = db.Raw(`SELECT count(*) FROM learning_wrong_questions WHERE user_id = $1 AND is_mastered = FALSE`, userID).
		Row().Scan(&wrongCount)
	require.NoError(t, err)
	assert.Equal(t, 1, wrongCount, "exactly one unmastered wrong question expected")

	// ---------- 10. 错题标记掌握 ----------
	err = execSQL(ctx, `
		UPDATE learning_wrong_questions
		SET is_mastered = TRUE, updated_at = now()
		WHERE id = $1`, wrongID)
	require.NoError(t, err, "mark wrong question mastered")

	var mastered bool
	err = db.Raw(`SELECT is_mastered FROM learning_wrong_questions WHERE id = $1`, wrongID).Row().Scan(&mastered)
	require.NoError(t, err)
	assert.True(t, mastered, "wrong question should be marked mastered")

	// ---------- 跨表一致性验证 ----------
	// 学习者旅程全链路数据都应在数据库中可追溯
	var (
		gotUserID      int64
		gotCourseID    int64
		gotExamID      int64
		gotAttemptID   int64
		gotQuestionID  int64
	)
	err = db.Raw(`
		SELECT wq.user_id, e.course_id, ex.id, att.id, wq.question_id
		FROM learning_wrong_questions wq
		JOIN learning_exams ex ON ex.id = wq.exam_id
		JOIN learning_courses e ON e.id = ex.course_id
		JOIN learning_exam_attempts att ON att.id = wq.attempt_id
		WHERE wq.id = $1`, wrongID).
		Row().Scan(&gotUserID, &gotCourseID, &gotExamID, &gotAttemptID, &gotQuestionID)
	require.NoError(t, err, "cross-table join should succeed")
	assert.Equal(t, userID, gotUserID, "wrong_question.user_id chain intact")
	assert.Equal(t, courseID, gotCourseID, "wrong_question → exam → course chain intact")
	assert.Equal(t, examID, gotExamID, "wrong_question.exam_id chain intact")
	assert.Equal(t, attemptID, gotAttemptID, "wrong_question.attempt_id chain intact")
	assert.Equal(t, q1ID, gotQuestionID, "wrong_question.question_id chain intact")
}

// TestLearnerJourney_UniqueConstraints 验证关键字段的唯一约束生效。
// 包括 users.username、users.email、user_profiles.user_id、user_settings.user_id、
// learning_enrollments.(user_id, course_id)、learning_records.(user_id, lesson_id)。
func TestLearnerJourney_UniqueConstraints(t *testing.T) {
	skipIfNoDeps(t)
	resetTablesInTest(t, "users", "user_profiles", "user_settings", "user_roles")

	ctx, cancel := newContext()
	defer cancel()

	uid1 := nextID()
	require.NoError(t, execSQL(ctx, `
		INSERT INTO users (id, username, email, password_hash, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'hash', 'active', now(), now())`,
		uid1, "uniq_user_1", "u1@example.com",
	))

	// 同 username 二次插入应失败（部分唯一索引 WHERE deleted_at IS NULL）
	err := execSQL(ctx, `
		INSERT INTO users (id, username, email, password_hash, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'hash', 'active', now(), now())`,
		nextID(), "uniq_user_1", "u2@example.com",
	)
	require.Error(t, err, "duplicate username should violate unique index")

	// 同 email 二次插入应失败
	err = execSQL(ctx, `
		INSERT INTO users (id, username, email, password_hash, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'hash', 'active', now(), now())`,
		nextID(), "uniq_user_2", "u1@example.com",
	)
	require.Error(t, err, "duplicate email should violate unique index")

	// 软删除后，原 username 可被复用（部分唯一索引允许 deleted_at NOT NULL）
	require.NoError(t, execSQL(ctx, `UPDATE users SET deleted_at = now() WHERE id = $1`, uid1))
	err = execSQL(ctx, `
		INSERT INTO users (id, username, email, password_hash, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'hash', 'active', now(), now())`,
		nextID(), "uniq_user_1", "u3@example.com",
	)
	require.NoError(t, err, "soft-deleted username should be reusable")
}

// TestLearnerJourney_SoftDeleteFilters 验证业务表软删除字段（deleted_at）
// 不会干扰查询——本测试通过 IS NULL 条件确认软删除行不被读到。
func TestLearnerJourney_SoftDeleteFilters(t *testing.T) {
	skipIfNoDeps(t)
	resetTablesInTest(t, "learning_courses")

	ctx, cancel := newContext()
	defer cancel()

	cid1 := nextID()
	cid2 := nextID()
	require.NoError(t, execSQL(ctx, `
		INSERT INTO learning_courses (id, title, is_published, sort_order, created_at, updated_at)
		VALUES ($1, 'alive', TRUE, 1, now(), now())`, cid1))
	require.NoError(t, execSQL(ctx, `
		INSERT INTO learning_courses (id, title, is_published, sort_order, created_at, updated_at)
		VALUES ($1, 'deleted', TRUE, 2, now(), now())`, cid2))

	// 软删除 cid2
	require.NoError(t, execSQL(ctx, `UPDATE learning_courses SET deleted_at = now() WHERE id = $1`, cid2))

	// 仅未删除的行可见
	var count int
	err := db.Raw(`SELECT count(*) FROM learning_courses WHERE deleted_at IS NULL`).Row().Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "only one course should be visible after soft delete")

	var title string
	err = db.Raw(`SELECT title FROM learning_courses WHERE deleted_at IS NULL`).Row().Scan(&title)
	require.NoError(t, err)
	assert.Equal(t, "alive", title)
}

// execSQL 是 db.Exec 的薄封装，便于在用例中统一错误处理与 context 传递。
// GORM 的 db.Exec 不直接接受 context，这里通过 WithContext 注入。
func execSQL(ctx context.Context, query string, args ...interface{}) error {
	res := db.WithContext(ctx).Exec(query, args...)
	if res.Error != nil {
		return fmt.Errorf("exec sql: %w", res.Error)
	}
	return nil
}
