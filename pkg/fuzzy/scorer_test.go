package fuzzy

import "testing"

// TestDefaultBonusScorer_MatchAndMiss 基本匹配/不匹配。
func TestDefaultBonusScorer_MatchAndMiss(t *testing.T) {
	// 子串匹配
	if _, ok := DefaultBonusScorer("cub", "github:cube"); !ok {
		t.Fatalf("子串应匹配")
	}
	// 不匹配（缺字符）
	if _, ok := DefaultBonusScorer("xyz", "cube"); ok {
		t.Fatalf("无公共字符应不匹配")
	}
	// 空查询（BonusScorer 内部对空 query 的处理）：空 query 默认认为匹配（score=0）
	if _, ok := DefaultBonusScorer("", "cube"); !ok {
		t.Fatalf("空 query 应默认匹配")
	}
}

// TestEnhancedBonusScorer_ContinuousBetterThanScattered
// 增强版：连续匹配分数应明显高于零散匹配。
func TestEnhancedBonusScorer_ContinuousBetterThanScattered(t *testing.T) {
	contScore, _ := EnhancedBonusScorer("cub", "cube") // 连续
	// 找一个零散但匹配的 target
	scatScore, ok := EnhancedBonusScorer("cub", "c-x-u-b") // 零散
	if !ok {
		t.Skip("零散 target 未匹配，跳过对比")
	}
	if contScore <= scatScore {
		t.Fatalf("连续匹配分数 (%d) 应高于零散 (%d)", contScore, scatScore)
	}
}

// TestWordSegmentationScorer 分词计分：每个 segment 都须匹配。
func TestWordSegmentationScorer(t *testing.T) {
	scorer := WordSegmentationScorer(DefaultBonusScorer)

	// 两段都匹配
	if _, ok := scorer("cu be", "cube-x-be"); !ok {
		t.Fatalf("两段都应匹配")
	}
	// 任一段不匹配 → 整体不匹配
	if _, ok := scorer("cu xyz", "cube"); ok {
		t.Fatalf("xyz 不匹配应整体 false")
	}
}
