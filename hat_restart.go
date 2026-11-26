package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type hatRestart struct {
	Mode   string `json:"mode"`
	Folder string `json:"folder"`
}

type hatCheckpoint struct {
	Artifacts map[string][32]byte `json:"artifacts"`
}

func hatCheckpointPath(folder string, stage Stage) string {
	return filepath.Join(folder, stage.Input+".verified.json")
}

func hatArtifacts(stage Stage) []string {
	stem := strings.TrimSuffix(stage.Input, filepath.Ext(stage.Input))
	return []string{stage.Input, stem + ".out", stageGeometry(stage)}
}

func hashHATArtifact(folder, name string) ([32]byte, error) {
	var result [32]byte
	if !safeName(name) {
		return result, fmt.Errorf("HAT: 잘못된 파일명")
	}
	path := filepath.Join(folder, name)
	info, err := os.Lstat(path)
	if err != nil {
		return result, err
	}
	if !info.Mode().IsRegular() {
		return result, fmt.Errorf("HAT: 일반 파일이 아닙니다: %s", name)
	}
	f, err := os.Open(path)
	if err != nil {
		return result, err
	}
	defer f.Close()
	digest := sha256.New()
	if _, err = io.Copy(digest, f); err != nil {
		return result, err
	}
	copy(result[:], digest.Sum(nil))
	return result, nil
}

func recordHATCheckpoint(folder string, stage Stage) error {
	if stage.HAT == "" || stage.HAT == "irc" {
		return nil
	}
	record := hatCheckpoint{Artifacts: map[string][32]byte{}}
	for _, name := range hatArtifacts(stage) {
		hash, err := hashHATArtifact(folder, name)
		if err != nil {
			return err
		}
		record.Artifacts[name] = hash
	}
	raw, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return os.WriteFile(hatCheckpointPath(folder, stage), raw, 0600)
}

func writeHATMapping(folder string, req RunRequest) error {
	if req.HATMapping == nil {
		return nil
	}
	raw, err := json.Marshal(req.HATMapping)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(folder, "hat-mapping.json"), raw, 0600)
}

func validateHATRestart(root string, req RunRequest) (string, int, error) {
	restart := req.HATRestart
	if restart == nil {
		return "", 0, nil
	}
	minimum := 0
	switch restart.Mode {
	case "step3":
		minimum = 2
	case "irc":
		minimum = 3
	case "resume":
	default:
		return "", 0, fmt.Errorf("HAT: 알 수 없는 재시작 모드")
	}
	if len(req.Stages) != 4 || req.HATMapping == nil || !filepath.IsAbs(restart.Folder) {
		return "", 0, fmt.Errorf("HAT: 이전 실행의 절대 폴더 경로가 필요합니다")
	}
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", 0, err
	}
	folder, err := filepath.EvalSymlinks(restart.Folder)
	if err != nil {
		return "", 0, err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return "", 0, err
	}
	parent := filepath.Dir(folder)
	if parent != root && (runtime.GOOS != "windows" || !strings.EqualFold(parent, root)) {
		return "", 0, fmt.Errorf("HAT: 작업 폴더 바로 아래의 이전 실행 폴더를 선택하세요")
	}
	var previous hatMapping
	raw, err := readGloriusFile(folder, "hat-mapping.json")
	if err != nil || json.Unmarshal([]byte(raw), &previous) != nil || previous != *req.HATMapping {
		return "", 0, fmt.Errorf("HAT: 이전 실행의 확인된 원자 대응이 없거나 달라졌습니다")
	}
	for _, stage := range req.Stages[:2] {
		for _, f := range req.Files {
			if f.Name != stage.Source {
				continue
			}
			raw, err := readGloriusFile(folder, stage.Source)
			if err != nil || raw != f.Content {
				return "", 0, fmt.Errorf("HAT: 이전 %s XYZ와 현재 구조가 다릅니다", stage.Name)
			}
		}
	}
	limit := 3
	if restart.Mode == "step3" {
		limit = 2
	}
	reusable := 0
	var rejected error
	for _, stage := range req.Stages[:limit] {
		rejected = checkHATCheckpoint(folder, req, stage)
		if rejected != nil {
			break
		}
		reusable++
	}
	if reusable < minimum {
		return "", 0, fmt.Errorf("HAT: 재사용에 필요한 완료 단계가 없습니다: %w", rejected)
	}
	return folder, reusable, nil
}

func checkHATCheckpoint(folder string, req RunRequest, stage Stage) error {
	var record hatCheckpoint
	raw, err := readGloriusFile(folder, stage.Input+".verified.json")
	if err != nil {
		return err
	}
	if err = json.Unmarshal([]byte(raw), &record); err != nil {
		return err
	}
	var expected string
	for _, f := range req.Files {
		if f.Name == stage.Input {
			expected = f.Content
			break
		}
	}
	original, err := readGloriusFile(folder, stage.Input)
	if err != nil {
		return err
	}
	if hatRestartSettings(original) != hatRestartSettings(expected) {
		return fmt.Errorf("%s INP가 변경되었습니다", stage.Name)
	}
	for _, name := range hatArtifacts(stage) {
		digest, err := hashHATArtifact(folder, name)
		want, ok := record.Artifacts[name]
		if err != nil || !ok || digest != want {
			return fmt.Errorf("%s 검증 산출물이 변경되거나 누락되었습니다: %s", stage.Name, name)
		}
	}
	return validateStage(folder, stage)
}

func hatRestartSettings(input string) string {
	// Only ignore canonical scheduling directives. The actual saved INP is
	// verified and copied with its OUT, so resource changes never rewrite history.
	input = autoPALBlock.ReplaceAllString(input, "")
	return strings.TrimSpace(autoMaxCoreLine.ReplaceAllString(input, ""))
}

func copyHATCheckpoints(from, to string, req RunRequest, count int) ([]string, error) {
	var complete []string
	for _, stage := range req.Stages[:count] {
		// The reused report must keep the exact INP that produced it.
		if err := os.Remove(filepath.Join(to, stage.Input)); err != nil {
			return nil, err
		}
		for _, name := range hatArtifacts(stage) {
			input, err := os.Open(filepath.Join(from, name))
			if err != nil {
				return nil, err
			}
			out, err := os.OpenFile(filepath.Join(to, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if err != nil {
				input.Close()
				return nil, err
			}
			_, copyErr := io.Copy(out, input)
			input.Close()
			closeErr := out.Close()
			if copyErr != nil {
				return nil, copyErr
			}
			if closeErr != nil {
				return nil, closeErr
			}
		}
		raw, err := readGloriusFile(from, stage.Input+".verified.json")
		if err != nil {
			return nil, err
		}
		if err = os.WriteFile(hatCheckpointPath(to, stage), []byte(raw), 0600); err != nil {
			return nil, err
		}
		if err = checkHATCheckpoint(to, req, stage); err != nil {
			return nil, err
		}
		complete = append(complete, stage.Input)
	}
	return complete, nil
}
