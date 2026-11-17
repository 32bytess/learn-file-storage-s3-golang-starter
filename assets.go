package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	result := &bytes.Buffer{}
	cmd.Stdout = result
	if err := cmd.Run(); err != nil {
		return "", err
	}
	var output struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(result.Bytes(), &output); err != nil {
		return "", err
	}
	if len(output.Streams) == 0 {
		return "", fmt.Errorf("couldnt get aspect ratio")
	}
	ratio := float64(output.Streams[0].Width) / float64(output.Streams[0].Height)
	tolerance := 0.05

	switch {
	case math.Abs(ratio-(16.0/9.0)) < tolerance:
		return "16:9", nil

	case math.Abs(ratio-(9.0/16.0)) < tolerance:
		return "9:16", nil
	default:
		return "other", nil
	}
}
