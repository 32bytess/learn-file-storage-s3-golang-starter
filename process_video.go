package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

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

func processVideoForFastStart(filePath string) (string, error) {
	outputPath := filePath + ".processing"
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputPath)
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return outputPath, nil
}

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	client := s3.NewPresignClient(s3Client)
	objectInput := &s3.GetObjectInput{
		Key:    &key,
		Bucket: &bucket,
	}
	presignReq, err := client.PresignGetObject(
		context.TODO(),
		objectInput,
		s3.WithPresignExpires(expireTime))
	if err != nil {
		return "", err
	}
	return presignReq.URL, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL == nil {
		return video, nil
	}
	args := strings.Split(*video.VideoURL, ",")
	if len(args) != 2 {
		return database.Video{}, fmt.Errorf("invalid video url")
	}
	bucket := args[0]
	key := args[1]
	presignedURL, err := generatePresignedURL(cfg.s3Client, bucket, key, time.Hour*4)
	if err != nil {
		return database.Video{}, fmt.Errorf("couldnt create a presigned url")
	}
	video.VideoURL = &presignedURL
	return video, nil
}
