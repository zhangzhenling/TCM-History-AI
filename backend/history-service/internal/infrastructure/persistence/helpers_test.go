package persistence

import (
	"fmt"
	"path/filepath"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
)

// setupDB returns a temp file-based SQLite DB with the given entity tables
// created using SQLite-compatible DDL.
//
// We use raw CREATE TABLE statements instead of gorm.AutoMigrate because the
// entity structs embed gormutil.BaseModel whose CreatedAt/UpdatedAt are
// tagged `default:now()` (PostgreSQL syntax). SQLite rejects `DEFAULT now()`
// ("near '(': syntax error") — only literal defaults and the keywords
// CURRENT_TIME / CURRENT_DATE / CURRENT_TIMESTAMP may appear unparenthesised
// in a column DEFAULT clause. Mirroring the schemas by hand keeps the test
// DB schema faithful to production while remaining SQLite-friendly
// (datetime instead of timestamptz, text instead of jsonb, CURRENT_TIMESTAMP
// instead of now()).
func setupDB(t *testing.T, models ...interface{}) *gorm.DB {
	t.Helper()
	tmpFile := filepath.Join(t.TempDir(), "test.db")
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_busy_timeout=5000&_loc=UTC", tmpFile)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	for _, m := range models {
		stmt, ok := sqliteSchemaFor(m)
		if !ok {
			t.Fatalf("no schema registered for %T", m)
		}
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("create table for %T: %v", m, err)
		}
	}
	return db
}

// sqliteSchemaFor returns the SQLite-compatible CREATE TABLE statement for
// the given entity pointer. Returns false if the entity is unknown.
func sqliteSchemaFor(model interface{}) (string, bool) {
	switch model.(type) {
	case *entity.Dynasty:
		return dynastySQLiteSchema, true
	case *entity.Person:
		return personSQLiteSchema, true
	case *entity.School:
		return schoolSQLiteSchema, true
	case *entity.Book:
		return bookSQLiteSchema, true
	case *entity.BookAuthor:
		return bookAuthorSQLiteSchema, true
	case *entity.Event:
		return eventSQLiteSchema, true
	case *entity.Medicine:
		return medicineSQLiteSchema, true
	case *entity.Disease:
		return diseaseSQLiteSchema, true
	case *entity.Prescription:
		return prescriptionSQLiteSchema, true
	case *entity.MedicinePrescription:
		return medicinePrescriptionSQLiteSchema, true
	case *entity.PrescriptionDisease:
		return prescriptionDiseaseSQLiteSchema, true
	case *entity.PersonSchool:
		return personSchoolSQLiteSchema, true
	}
	return "", false
}

const dynastySQLiteSchema = `
CREATE TABLE IF NOT EXISTS history_dynasty (
    id          bigint       PRIMARY KEY,
    created_at  datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at  datetime,
    name        varchar(64)  NOT NULL,
    start_year  smallint,
    end_year    smallint,
    sort_order  integer      NOT NULL DEFAULT 0,
    description text
);
CREATE INDEX IF NOT EXISTS idx_history_dynasty_sort_order ON history_dynasty(sort_order);
`

const personSQLiteSchema = `
CREATE TABLE IF NOT EXISTS history_person (
    id           bigint       PRIMARY KEY,
    created_at   datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at   datetime,
    name         varchar(64)  NOT NULL,
    courtesy_name varchar(64),
    alias_name   varchar(128),
    dynasty_id   bigint,
    birth_year   smallint,
    death_year   smallint,
    gender       varchar(16),
    title        varchar(128),
    biography    text,
    achievements text,
    portrait_url varchar(512)
);
`

const schoolSQLiteSchema = `
CREATE TABLE IF NOT EXISTS history_school (
    id                bigint       PRIMARY KEY,
    created_at        datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at        datetime,
    name              varchar(128) NOT NULL,
    dynasty_id        bigint,
    founder_person_id bigint,
    summary           text,
    established_year  smallint
);
`

const bookSQLiteSchema = `
CREATE TABLE IF NOT EXISTS history_book (
    id            bigint       PRIMARY KEY,
    created_at    datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at    datetime,
    title         varchar(255) NOT NULL,
    dynasty_id    bigint,
    published_year smallint,
    category      varchar(64),
    summary       text,
    volume_count  integer,
    is_extant     boolean      NOT NULL DEFAULT 1,
    file_url      varchar(512)
);
`

const bookAuthorSQLiteSchema = `
CREATE TABLE IF NOT EXISTS book_author (
    id          bigint      PRIMARY KEY,
    created_at  datetime    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    book_id     bigint      NOT NULL,
    person_id   bigint      NOT NULL,
    author_type varchar(32) NOT NULL,
    sort_order  integer     NOT NULL DEFAULT 0
);
`

const eventSQLiteSchema = `
CREATE TABLE IF NOT EXISTS history_event (
    id           bigint       PRIMARY KEY,
    created_at   datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at   datetime,
    title        varchar(255) NOT NULL,
    dynasty_id   bigint,
    occurred_year smallint,
    event_type   varchar(32)  NOT NULL,
    description  text,
    impact       text,
    location     varchar(128)
);
`

const medicineSQLiteSchema = `
CREATE TABLE IF NOT EXISTS medicine (
    id         bigint       PRIMARY KEY,
    created_at datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at datetime,
    name       varchar(64)  NOT NULL,
    pinyin     varchar(128),
    alias_json text         NOT NULL DEFAULT '[]',
    nature     varchar(32),
    flavor     varchar(64),
    meridian   varchar(128),
    efficacy   text,
    dosage     varchar(128),
    toxicity   varchar(32)
);
`

const diseaseSQLiteSchema = `
CREATE TABLE IF NOT EXISTS disease (
    id              bigint       PRIMARY KEY,
    created_at      datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at      datetime,
    name            varchar(128) NOT NULL,
    pinyin          varchar(128),
    category        varchar(64),
    description     text,
    symptoms        text,
    tcm_pathogenesis text
);
`

const prescriptionSQLiteSchema = `
CREATE TABLE IF NOT EXISTS prescription (
    id               bigint       PRIMARY KEY,
    created_at       datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at       datetime,
    name             varchar(128) NOT NULL,
    pinyin           varchar(128),
    source_book_id   bigint,
    source_person_id bigint,
    dynasty_id       bigint,
    composition      text,
    usage            text,
    indications      text,
    category         varchar(64)
);
`

const medicinePrescriptionSQLiteSchema = `
CREATE TABLE IF NOT EXISTS medicine_prescription (
    id             bigint      PRIMARY KEY,
    created_at     datetime    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    prescription_id bigint     NOT NULL,
    medicine_id    bigint      NOT NULL,
    role           varchar(32) NOT NULL,
    dosage         varchar(64),
    sort_order     integer     NOT NULL DEFAULT 0
);
`

const prescriptionDiseaseSQLiteSchema = `
CREATE TABLE IF NOT EXISTS prescription_disease (
    id             bigint    PRIMARY KEY,
    created_at     datetime  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    prescription_id bigint   NOT NULL,
    disease_id     bigint    NOT NULL,
    efficacy_note  varchar(255),
    is_primary     boolean   NOT NULL DEFAULT 0
);
`

const personSchoolSQLiteSchema = `
CREATE TABLE IF NOT EXISTS person_school (
    id          bigint      PRIMARY KEY,
    created_at  datetime    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    person_id   bigint      NOT NULL,
    school_id   bigint      NOT NULL,
    role        varchar(32) NOT NULL,
    joined_year smallint
);
`

// allHistoryModels returns every entity type used by the history-service
// persistence layer. Pass each entry to setupDB to migrate all tables.
func allHistoryModels() []interface{} {
	return []interface{}{
		&entity.Dynasty{},
		&entity.Person{},
		&entity.School{},
		&entity.Book{},
		&entity.BookAuthor{},
		&entity.Event{},
		&entity.Medicine{},
		&entity.Disease{},
		&entity.Prescription{},
		&entity.MedicinePrescription{},
		&entity.PrescriptionDisease{},
		&entity.PersonSchool{},
	}
}
