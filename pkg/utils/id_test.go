package utils

import (
	"regexp"
	"testing"
)

func TestGenerateQueryID(t *testing.T) {
	// 生成多个ID
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenerateQueryID()
		t.Logf("生成的QueryID: %s", id)

		// 检查格式：qry_YYYYMMDD_xxxxxxxx
		pattern := regexp.MustCompile(`^qry_\d{8}_[a-zA-Z0-9]{8}$`)
		if !pattern.MatchString(id) {
			t.Errorf("生成的QueryID格式不正确: %s", id)
		}

		// 检查唯一性
		if ids[id] {
			t.Errorf("生成的QueryID重复: %s", id)
		}
		ids[id] = true
	}
}

func TestGenerateFeedbackID(t *testing.T) {
	// 生成多个ID
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenerateFeedbackID()
		t.Logf("生成的FeedbackID: %s", id)

		// 检查格式：fb_xxxxxxxxxxxxxxxx（16位十六进制）
		pattern := regexp.MustCompile(`^fb_[a-zA-Z0-9]{16}$`)
		if !pattern.MatchString(id) {
			t.Errorf("生成的FeedbackID格式不正确: %s", id)
		}

		// 检查唯一性
		if ids[id] {
			t.Errorf("生成的FeedbackID重复: %s", id)
		}
		ids[id] = true
	}
}

func TestIDUniqueness(t *testing.T) {
	// 测试QueryID和FeedbackID不会重复
	queryIDs := make(map[string]bool)
	feedbackIDs := make(map[string]bool)

	for i := 0; i < 100; i++ {
		qid := GenerateQueryID()
		fid := GenerateFeedbackID()

		if queryIDs[qid] {
			t.Errorf("QueryID重复: %s", qid)
		}
		queryIDs[qid] = true

		if feedbackIDs[fid] {
			t.Errorf("FeedbackID重复: %s", fid)
		}
		feedbackIDs[fid] = true

		// 确保QueryID和FeedbackID不会冲突
		if queryIDs[fid] {
			t.Errorf("FeedbackID与QueryID冲突: %s", fid)
		}
		if feedbackIDs[qid] {
			t.Errorf("QueryID与FeedbackID冲突: %s", qid)
		}
	}
}
