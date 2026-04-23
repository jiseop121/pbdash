package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func readJSONFile(path string, v any) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return false, nil
	}
	if err := json.Unmarshal(data, v); err != nil {
		return false, err
	}
	return true, nil
}

func writeJSONFile(path string, v any) error {
	// 0o700: 소유자만 접근 가능한 데이터 디렉토리
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	content := append(data, '\n')

	// 부분 쓰기를 방지하기 위해 임시 파일에 먼저 쓴 후 원자적으로 교체
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, content, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("atomic write failed for %s: %w", path, err)
	}
	return nil
}
