package entity

// GraphRelationship 是图谱关系的通用实体。
// 关系方向严格约束在领域语义合理的范围内（详见 doc/05 §5.3）。
// 所有关系均带 uid 与时间戳属性，便于血缘追溯与增量同步。
type GraphRelationship struct {
	UID        string         `json:"uid"`
	Type       string         `json:"type"`
	FromUID    string         `json:"from_uid"`
	ToUID      string         `json:"to_uid"`
	Properties map[string]any `json:"properties"`
}

// 关系 Type 枚举，对应 doc/05 §5.3 的 9 类关系。
const (
	RelAuthored    = "AUTHORED"     // Person → Classic 著作
	RelDiscipled   = "DISCIPLED"    // Person → Person 师承（弟子→师父）
	RelInfluenced  = "INFLUENCED"   // 任意 → 任意 影响
	RelBelongsTo   = "BELONGS_TO"   // Person/Classic/Prescription → School 属于
	RelOccurredIn   = "OCCURRED_IN"  // Person/Classic/HistoricalEvent → Dynasty 发生于
	RelCited       = "CITED"        // Classic → Classic 引用
	RelProposed    = "PROPOSED"     // Person/Classic → Classic(理论) 提出
	RelOpposed     = "OPPOSED"      // Person/Classic → Person/Classic/School 反对
	RelInherited   = "INHERITED"    // Person/School → Person/Classic/School 继承
)

// RelationshipTypes 罗列全部关系 Type，供约束建立与校验复用。
var RelationshipTypes = []string{
	RelAuthored,
	RelDiscipled,
	RelInfluenced,
	RelBelongsTo,
	RelOccurredIn,
	RelCited,
	RelProposed,
	RelOpposed,
	RelInherited,
}

// IsValidRelationshipType reports whether t is one of the 9 known relationship types.
func IsValidRelationshipType(t string) bool {
	for _, r := range RelationshipTypes {
		if r == t {
			return true
		}
	}
	return false
}
