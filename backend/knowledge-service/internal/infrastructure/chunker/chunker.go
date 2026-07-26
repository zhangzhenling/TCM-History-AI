// Package chunker provides a pure-Go text splitter for RAG ingestion.
//
// 将 Markdown / 纯文本按「段落 + token 上限」切分为 chunk，支持滑动窗口
// 重叠以避免边界截断丢失上下文。token 计数以 rune 近似（中文一字 ≈ 1 token，
// 英文一词 ≈ 1 token），不引入外部 tokenizer 依赖，与 ADR-21-01 离线可构建
// 约束一致。
//
// 切分策略（doc/06-RAG设计.md §6.3）：
//  1. 按 Markdown 标题（#/##/###）与空行切出段落
//  2. 段落 token 数 ≤ maxTokens → 整段作为一个 chunk
//  3. 段落 token 数 > maxTokens → 按句子（。！？.!?）二次切分，累积到上限
//  4. 相邻 chunk 之间保留 overlap 个 token 的重叠
package chunker

import (
	"strings"
	"unicode"
)

// Config 控制切片行为。
type Config struct {
	MaxTokens   int // 单个 chunk 的 token 上限（默认 512）
	Overlap     int // 相邻 chunk 的重叠 token 数（默认 50）
	MinTokens   int // chunk 最少 token 数，不足则合并到前一个（默认 50）
}

// DefaultConfig 返回适用于中医古籍 Markdown 的默认配置。
func DefaultConfig() Config {
	return Config{MaxTokens: 512, Overlap: 50, MinTokens: 50}
}

// Chunk 是切分后的一个片段。
type Chunk struct {
	Index      int    // 从 0 开始的序号
	Content    string // chunk 文本
	TokenCount int    // 近似 token 数
}

// Split 将 text 切分为若干 chunk。返回的切片至少包含 1 个元素（除非 text 为空）。
func Split(text string, cfg Config) []Chunk {
	if cfg.MaxTokens <= 0 {
		cfg = DefaultConfig()
	}
	if cfg.Overlap < 0 {
		cfg.Overlap = 0
	}
	if cfg.Overlap >= cfg.MaxTokens {
		cfg.Overlap = cfg.MaxTokens / 4
	}
	if cfg.MinTokens <= 0 {
		cfg.MinTokens = 50
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	paragraphs := splitParagraphs(text)
	var chunks []Chunk
	var buf strings.Builder
	bufTokens := 0

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		paraTokens := countTokens(para)

		// 段落本身超过上限 → 按句子二次切分
		if paraTokens > cfg.MaxTokens {
			// 先把缓冲区已有的内容 flush
			if bufTokens > 0 {
				chunks = append(chunks, finalizeChunk(buf.String(), bufTokens, len(chunks)))
				buf.Reset()
				bufTokens = 0
			}
			sentenceChunks := splitLongParagraph(para, cfg)
			chunks = append(chunks, sentenceChunks...)
			continue
		}

		// 加入缓冲区后会超限 → 先 flush，再尝试滑动窗口重叠
		if bufTokens+paraTokens > cfg.MaxTokens && bufTokens >= cfg.MinTokens {
			chunks = append(chunks, finalizeChunk(buf.String(), bufTokens, len(chunks)))
			// 滑动窗口：从缓冲区末尾取 overlap 个 token 作为下一个 chunk 的开头
			tail := tailTokens(buf.String(), cfg.Overlap)
			buf.Reset()
			buf.WriteString(tail)
			bufTokens = countTokens(tail)
		}

		if buf.Len() > 0 {
			buf.WriteString("\n\n")
			bufTokens += 2 // 换行近似
		}
		buf.WriteString(para)
		bufTokens += paraTokens
	}

	// flush 残余
	if bufTokens > 0 {
		chunks = append(chunks, finalizeChunk(buf.String(), bufTokens, len(chunks)))
	}

	// 合并过短的尾 chunk
	if len(chunks) > 1 {
		last := &chunks[len(chunks)-1]
		prev := &chunks[len(chunks)-2]
		if last.TokenCount < cfg.MinTokens {
			prev.Content += "\n\n" + last.Content
			prev.TokenCount += last.TokenCount
			chunks = chunks[:len(chunks)-1]
		}
	}

	// 重新编号（合并可能导致序号不连续）
	for i := range chunks {
		chunks[i].Index = i
	}
	return chunks
}

// splitParagraphs 按空行与 Markdown 标题切分段落。
func splitParagraphs(text string) []string {
	// 统一换行符
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// 按空行分段
	rawParas := strings.Split(text, "\n\n")

	// 进一步按 Markdown 标题拆分：标题行独立成段
	var paras []string
	for _, p := range rawParas {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// 如果段落以 # 开头，把标题与后续内容分离
		lines := strings.Split(p, "\n")
		var current strings.Builder
		for _, line := range lines {
			if isMarkdownHeading(line) && current.Len() > 0 {
				paras = append(paras, current.String())
				current.Reset()
			}
			if current.Len() > 0 {
				current.WriteString("\n")
			}
			current.WriteString(line)
		}
		if current.Len() > 0 {
			paras = append(paras, current.String())
		}
	}
	return paras
}

// isMarkdownHeading 判断一行是否以 # 开头（1-6 级标题）。
func isMarkdownHeading(line string) bool {
	s := strings.TrimLeft(line, " ")
	count := 0
	for _, r := range s {
		if r == '#' {
			count++
		} else {
			break
		}
	}
	return count >= 1 && count <= 6 && count < len(s) && (s[count] == ' ' || s[count] == '\t')
}

// splitLongParagraph 将超长段落按句子切分，累积到 maxTokens 后输出 chunk。
func splitLongParagraph(para string, cfg Config) []Chunk {
	sentences := splitSentences(para)
	var chunks []Chunk
	var buf strings.Builder
	bufTokens := 0

	for _, sent := range sentences {
		sent = strings.TrimSpace(sent)
		if sent == "" {
			continue
		}
		sentTokens := countTokens(sent)

		if bufTokens+sentTokens > cfg.MaxTokens && bufTokens > 0 {
			chunks = append(chunks, finalizeChunk(buf.String(), bufTokens, len(chunks)))
			tail := tailTokens(buf.String(), cfg.Overlap)
			buf.Reset()
			buf.WriteString(tail)
			bufTokens = countTokens(tail)
		}

		if buf.Len() > 0 {
			buf.WriteString("")
		}
		buf.WriteString(sent)
		bufTokens += sentTokens
	}
	if bufTokens > 0 {
		chunks = append(chunks, finalizeChunk(buf.String(), bufTokens, len(chunks)))
	}
	return chunks
}

// splitSentences 按中英文句号/问号/叹号切分句子，保留分隔符。
func splitSentences(text string) []string {
	var sentences []string
	var buf strings.Builder
	runes := []rune(text)
	for i, r := range runes {
		buf.WriteRune(r)
		// 遇到句末标点且后面是空格/换行/结束 → 断句
		if isSentenceEnd(r) {
			// 检查下一个字符是否是空格或换行（避免切到小数点）
			if i+1 >= len(runes) || unicode.IsSpace(runes[i+1]) || isCJK(runes[i+1]) {
				sentences = append(sentences, strings.TrimSpace(buf.String()))
				buf.Reset()
			}
		}
	}
	if buf.Len() > 0 {
		sentences = append(sentences, strings.TrimSpace(buf.String()))
	}
	return sentences
}

// isSentenceEnd 判断是否为句末标点。
func isSentenceEnd(r rune) bool {
	switch r {
	case '。', '！', '？', '.', '!', '?', '…':
		return true
	}
	return false
}

// isCJK 判断是否为中日韩字符。
func isCJK(r rune) bool {
	return r >= 0x4E00 && r <= 0x9FFF
}

// countTokens 近似计数：中文 1 rune ≈ 1 token，英文按空格分词 ≈ 1 token/词。
func countTokens(text string) int {
	if text == "" {
		return 0
	}
	count := 0
	inWord := false
	for _, r := range text {
		if isCJK(r) {
			if inWord {
				count++
				inWord = false
			}
			count++
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if !inWord {
				count++
				inWord = true
			}
		} else {
			inWord = false
		}
	}
	return count
}

// tailTokens 取 text 末尾 approxTokens 个 token 的文本。
func tailTokens(text string, approxTokens int) string {
	if approxTokens <= 0 || text == "" {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= approxTokens {
		return text
	}
	// 从末尾向前扫描，累积 approxTokens 个 token
	count := 0
	start := len(runes)
	for i := len(runes) - 1; i >= 0 && count < approxTokens; i-- {
		if isCJK(runes[i]) {
			count++
		} else if unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) {
			// 连续字母数字算一个 token，只在词边界计数
			if i == 0 || !(unicode.IsLetter(runes[i-1]) || unicode.IsDigit(runes[i-1])) {
				count++
			}
		}
		start = i
	}
	return string(runes[start:])
}

// finalizeChunk 构造一个 Chunk 并确保 token 计数非零。
func finalizeChunk(content string, tokenCount, index int) Chunk {
	if tokenCount <= 0 {
		tokenCount = countTokens(content)
	}
	return Chunk{
		Index:      index,
		Content:    strings.TrimSpace(content),
		TokenCount: tokenCount,
	}
}
